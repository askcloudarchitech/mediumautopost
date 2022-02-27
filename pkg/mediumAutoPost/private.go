package mediumautopost

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Medium/medium-sdk-go"
	"github.com/google/go-github/github"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
)

// updateStatusRepository takes the updated array of published articles and commits it back to github so its up to date for the next time this runs.
// I'm not gonna lie, this is a bit complicated. the process is outlined here: http://www.levibotelho.com/development/commit-a-file-with-the-github-api/
// This code does those steps using go-github.
// if STORAGE_TYPE is "file" and STORAGE_FILE_PATH is set, a local file will be used for storing the publish status.
// otherwise github will be used to store the publish status
func updateStatusRepository(PublishedArticles []PublishedArticle, c Config) error {
	log.Println("updating status of posted articles for next use.")
	filebytes, err := json.MarshalIndent(PublishedArticles, "", "  ")
	filebytesSting := string(filebytes)
	if err != nil {
		return err
	}

	if c.StorageType == File && len(c.StorageFile) > 0 {
		err := ioutil.WriteFile(c.StorageFile, filebytes, 0644)
		if err != nil {
			return err
		}
	} else {
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: c.GithubPersonalToken},
		)
		tc := oauth2.NewClient(ctx, ts)
		client := github.NewClient(tc)

		log.Println("fetching main branch")
		branch, _, err := client.Repositories.GetBranch(ctx, c.GithubStatusRepoOwner, c.GithubStatusRepo, "main")
		if err != nil {
			return err
		}

		log.Println("creating blob")
		blob, _, err := client.Git.CreateBlob(ctx, c.GithubStatusRepoOwner, c.GithubStatusRepo, &github.Blob{
			Content: &filebytesSting,
		})
		if err != nil {
			return err
		}

		path := "status.json"
		mode := "100644"
		treetype := "blob"

		log.Println("creating tree")
		tree, _, err := client.Git.CreateTree(ctx, c.GithubStatusRepoOwner, c.GithubStatusRepo, *branch.Commit.Commit.Tree.SHA, []github.TreeEntry{
			{
				Path: &path,
				Mode: &mode,
				Type: &treetype,
				SHA:  blob.SHA,
			},
		})
		if err != nil {
			return err
		}

		log.Println("creating commit")
		message := "update the medium content"
		newCommit, _, err := client.Git.CreateCommit(ctx, c.GithubStatusRepoOwner, c.GithubStatusRepo, &github.Commit{
			Parents: []github.Commit{
				{
					SHA: branch.Commit.SHA,
				},
			},
			Tree:    tree,
			Message: &message,
		})
		if err != nil {
			return err
		}

		log.Println("updating ref")
		branchref := "refs/heads/main"
		reference := github.Reference{
			Object: &github.GitObject{
				SHA: newCommit.SHA,
			},
			Ref: &branchref,
		}
		_, _, err = client.Git.UpdateRef(ctx, c.GithubStatusRepoOwner, c.GithubStatusRepo, &reference, false)
		if err != nil {
			return err
		}
	}

	return nil
}

// postArticleToMedium takes config, a single article's id and URL and an http client. using this info it fetches the
// full article from the website in json format and posts to medium. returns error if failure. on success
// appends to the list of published articles which is passed by reference
func postArticleToMedium(c Config, a ArticleIndexItem, publishedArticles *[]PublishedArticle, client http.Client, mediumClient *medium.Medium) error {
	// fetch the full article json content from site
	resp, err := client.Get(a.URL)
	if err != nil {
		return err
	}

	// get the json data as a byte array
	jsonIndexData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// unmarshal to type
	article := ArticleJSONData{}
	err = json.Unmarshal(jsonIndexData, &article)
	if err != nil {
		return err
	}

	log.Printf("posting article %s to medium", article.Title)
	// post to medium
	result, err := mediumClient.CreatePost(medium.CreatePostOptions{
		UserID:        c.MediumUser.ID,
		Title:         article.Title,
		Content:       article.Content,
		ContentFormat: medium.ContentFormat(article.ContentFormat),
		Tags:          article.Tags,
		CanonicalURL:  article.CanonicalURL,
		PublishStatus: "draft",
	})
	if err != nil {
		return fmt.Errorf("error when posting article %s: %v", article.Title, err)
	}

	// add successful post to the list of published articles
	*publishedArticles = append(*publishedArticles, PublishedArticle{
		URL:                article.CanonicalURL,
		ID:                 a.ID,
		PublishTimestamp:   time.Now().String(),
		MediumPostResponse: *result,
	})

	return nil
}

// eliminateArticlesThatHaveAlreadyBeenPosted takes the index of all articles from the website
// as well as the list of articles which have already been posted to medium.com and then
// returns a new set of ArticleIndexItems which are only the ones that actually need to be published to medium.com
func eliminateArticlesThatHaveAlreadyBeenPosted(alreadyPublished []PublishedArticle, allArticlesOnWebsite []ArticleIndexItem) []ArticleIndexItem {
	articlesThatNeedPostedToMedium := []ArticleIndexItem{}

	for _, article := range allArticlesOnWebsite {
		if article.ID == "" {
			continue
		}
		found := false
		for _, alreadyPostedValue := range alreadyPublished {
			if alreadyPostedValue.ID == article.ID {
				found = true
				break
			}
		}
		if !found {
			articlesThatNeedPostedToMedium = append(articlesThatNeedPostedToMedium, article)
		}
	}

	log.Printf("after removing duplicates, %v articles will be published to medium", len(articlesThatNeedPostedToMedium))

	return articlesThatNeedPostedToMedium
}

// fetchArticleIndexFromSite pulls the index from the origin website.
// as can be seen here: https://askcloudarchitech.com/posts/index.json
func fetchArticleIndexFromSite(c Config, client http.Client) ([]ArticleIndexItem, error) {
	resp, err := client.Get(c.WebsiteJSONIndexURL)
	if err != nil {
		return []ArticleIndexItem{}, err
	}
	jsonIndexData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []ArticleIndexItem{}, err
	}
	indexOfArticlesOnWebsite := []ArticleIndexItem{}
	json.Unmarshal(jsonIndexData, &indexOfArticlesOnWebsite)
	if err != nil {
		return []ArticleIndexItem{}, err
	}
	log.Printf("found index containing a total of %v articles", len(indexOfArticlesOnWebsite))
	return indexOfArticlesOnWebsite, nil
}

// fetchPublishedArticles fetches a json file from the github repo or from file which stores the status of which articles
// have been already published to medium.com
// if STORAGE_TYPE is "file" and STORAGE_FILE_PATH is set, a local file will be used for storing the publish status.
// otherwise github will be used to store the publish status
func fetchPublishedArticles(c Config) ([]PublishedArticle, error) {
	publishedArticles := []PublishedArticle{}
	var filedata []byte
	var err error
	if c.StorageType == File && len(c.StorageFile) > 0 {
		log.Println("FILE storage type detected. Using local file for status storage")
		filedata, err = ioutil.ReadFile(c.StorageFile)
		if err != nil {
			return publishedArticles, err
		}
	} else {
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: c.GithubPersonalToken},
		)
		tc := oauth2.NewClient(ctx, ts)

		client := github.NewClient(tc)
		gitoptions := github.RepositoryContentGetOptions{
			Ref: "main",
		}
		log.Println("Pulling list of already published articles")
		rc, err := client.Repositories.DownloadContents(ctx, c.GithubStatusRepoOwner, c.GithubStatusRepo, "status.json", &gitoptions)
		if err != nil && strings.Contains(err.Error(), "No file named") {
			log.Println("no status.json file found. starting from scratch")
			return publishedArticles, nil
		}
		if err != nil {
			return publishedArticles, err
		}
		defer rc.Close()
		filedata, err = ioutil.ReadAll(rc)
		if err != nil {
			return publishedArticles, err
		}
	}

	log.Println("unmarshaling status.json")
	err = json.Unmarshal(filedata, &publishedArticles)
	if err != nil {
		return publishedArticles, err
	}
	return publishedArticles, nil
}

// getconfig reads in an environment variable file then populates the config with the necessary values
func getconfig(filename string) (Config, error) {
	log.Println("loading config")

	if filename != "" {
		// use the godotenv package to read the contents of the .env file.
		err := godotenv.Load(filename)
		if err != nil {
			return Config{}, err
		}
	}

	// use the values imported from .env to populate an instance of the config type declared above.
	config := Config{
		MediumEndpointPrefix:  os.Getenv("MEDIUM_ENDPOINT_PREFIX"),
		MediumBearerToken:     os.Getenv("MEDIUM_BEARER_TOKEN"),
		WebsiteJSONIndexURL:   os.Getenv("WEBSITE_JSON_INDEX_URL"),
		GithubPersonalToken:   os.Getenv("GITHUB_PERSONAL_TOKEN"),
		GithubStatusRepoOwner: os.Getenv("GITHUB_STATUS_REPO_OWNER"),
		GithubStatusRepo:      os.Getenv("GITHUB_STATUS_REPO"),
		StorageFile:           os.Getenv("STORAGE_FILE_PATH"),
	}
	if config.MediumEndpointPrefix == "" {
		config.MediumEndpointPrefix = "https://api.medium.com/v1"
	}
	switch os.Getenv("STORAGE_TYPE") {
	case "FILE":
		config.StorageType = File
	default:
		config.StorageType = Github
	}

	return config, nil
}
