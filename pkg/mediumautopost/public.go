package mediumautopost

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Medium/medium-sdk-go"
)

// Config is configuration found in .env
type Config struct {
	StorageType           StorageType
	StorageFile           string
	MediumEndpointPrefix  string
	MediumBearerToken     string
	WebsiteJSONIndexURL   string
	GithubPersonalToken   string
	GithubStatusRepoOwner string
	GithubStatusRepo      string
	MediumUser            *medium.User
}

type StorageType int

const (
	File StorageType = iota
	Github
)

// PublishedArticle is a record of how and when an article was published to medium.com.
// Can be seen here: https://github.com/askcloudarchitech/medium-publish-status
type PublishedArticle struct {
	URL                string      `json:"url"`
	ID                 string      `json:"id"`
	PublishTimestamp   string      `json:"publishTimestamp"`
	MediumPostResponse medium.Post `json:"mediumResponse"`
}

// ArticleIndexItem represents one item in the article index produced by the website.
// Can be seen here: https://askcloudarchitech.com/posts/index.json
type ArticleIndexItem struct {
	URL string `json:"url"`
	ID  string `json:"id"`
}

//ArticleJSONData is the JSON data needed for posting an article to medium.com
type ArticleJSONData struct {
	Title         string   `json:"title"`
	ContentFormat string   `json:"contentFormat"`
	Content       string   `json:"content"`
	CanonicalURL  string   `json:"canonicalUrl"`
	Tags          []string `json:"tags"`
}

func Do(dotEnvPath string) {
	client := http.Client{}

	// Populate the config struct
	config, err := getconfig(dotEnvPath)
	if err != nil {
		log.Fatal(err)
	}

	// Fetch the published articles (if they exist)
	publishedArticles, err := fetchPublishedArticles(config)
	if err != nil {
		log.Fatal(err)
	}

	// Fetch the article index from the specified website
	indexOfArticlesOnWebsite, err := fetchArticleIndexFromSite(config, client)
	if err != nil {
		log.Fatal(err)
	}

	// Compare articles on website to list of already published articles.
	// return only the ones that need to be published to medium.com
	articlesThatNeedPostedToMedium := eliminateArticlesThatHaveAlreadyBeenPosted(publishedArticles, indexOfArticlesOnWebsite)

	mediumClient := medium.NewClientWithAccessToken(config.MediumBearerToken)
	config.MediumUser, err = mediumClient.GetUser("")
	if err != nil {
		log.Fatal(err)
	}

	// publish the article(s), return which were successful, log the failures
	for _, article := range articlesThatNeedPostedToMedium {
		err := postArticleToMedium(config, article, &publishedArticles, client, mediumClient)
		if err != nil {
			log.Printf("posting error: %v", err)
		} else {
			log.Printf("successfully posted %s\n", article.URL)
		}
	}

	// update the published articles list to github repo to reflect current state
	err = updateStatusRepository(publishedArticles, config)
	if err != nil {
		log.Fatal(err)
	}

	// return great success
	fmt.Println("Great Success!")
}
