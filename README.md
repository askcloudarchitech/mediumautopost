# mediumautopost

## What is It?

Mediumautopost is a CLI tool (and a Go package) that will automatically post your website's articles to medium.com as long as you have your website set up according to this README. Usage is simple, and once it's set up it will work with one single command. 

## How to Install

`brew tap askcloudarchitech/askcloudarchitech && brew install mediumautopost`  
Or download binary from the releases tab.  
Or use as a package with `package github.com/askcloudarchitech/mediumautopost/pkg/mediumautopost`

## Setting up

To use this tool you will need to set up a few things:

1. set up a Github repo where mediumautopost will save the status of your posts (so it doesnt re-post anything)
2. get a Github personal token
3. get your medium.com API key
4. set up your website so it can tell mediumautopost about its articles. To see how to do this, read this article on my website: https://askcloudarchitech.com/posts/tutorials/auto-generate-post-payload-medium-com/ or on medium.com at: https://blog.devgenius.io/auto-generate-a-medium-com-rest-api-payload-to-syndicate-posts-with-hugo-fce630cced67

## Running the tool

After you have your website set up as mentioned above and you have the required tokens an such, create a .env file in the following format:

```bash
MEDIUM_ENDPOINT_PREFIX="https://api.medium.com/v1"
MEDIUM_BEARER_TOKEN="get token from Medium. paste here"
WEBSITE_JSON_INDEX_URL="path to your JSON index file"
GITHUB_PERSONAL_TOKEN="generate a personal access token and paste here"
GITHUB_STATUS_REPO_OWNER="your Github account name"
GITHUB_STATUS_REPO="repo name for storing status of posts to medium.com"
```

Next, run the command and you are all set. 

`mediumautopost -e /path/to/your/.env` and watch the magic happen!

## Contributing

Want to make it better or add a feature? Open a pull request and I will review, test and deploy. 
