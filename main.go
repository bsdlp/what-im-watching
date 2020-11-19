package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/bsdlp/what-im-watching/twitch"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	OAuthToken          string `envconfig:"OAUTH_TOKEN" required:"true"`
	ClientId            string `split_words:"true" required:"true"`
	TwitchGqlEndpoint   string `split_words:"true" default:"https://gql.twitch.tv/gql"`
	TwitterApiKey       string `required:"true" split_words:"true"`
	TwitterApiSecret    string `required:"true" split_words:"true"`
	TwitterAccessToken  string `required:"true" split_words:"true"`
	TwitterAccessSecret string `required:"true" split_words:"true"`
}

//go:generate go run github.com/Yamashou/gqlgenc
func main() {
	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		log.Fatal(err)
	}

	twitterOAuthConfig := oauth1.NewConfig(cfg.TwitterApiKey, cfg.TwitterApiSecret)
	twitterToken := oauth1.NewToken(cfg.TwitterAccessToken, cfg.TwitterAccessSecret)
	twitterClient := twitter.NewClient(twitterOAuthConfig.Client(oauth1.NoContext, twitterToken))

	client := twitch.NewClient(http.DefaultClient, cfg.TwitchGqlEndpoint, setAuth(cfg.OAuthToken), setClientId(cfg.ClientId))
	ctx := context.Background()
	currentlyWatching, err := client.GetCurrentlyWatching(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if currentlyWatching.CurrentUser == nil {
		log.Fatal("current user is nil")
	}
	if currentlyWatching.CurrentUser.Activity == nil {
		msg := fmt.Sprintf("%s is not currently watching anything", currentlyWatching.CurrentUser.DisplayName)
		_, _, err = twitterClient.Statuses.Update(msg, nil)
		if err != nil {
			log.Printf("error posting: %s", err)
		}
		log.Println(msg)
		return
	}

	msg := fmt.Sprintf("%s is currently watching %s stream %s: %s\n%s", currentlyWatching.CurrentUser.DisplayName, currentlyWatching.CurrentUser.Activity.User.DisplayName, currentlyWatching.CurrentUser.Activity.User.BroadcastSettings.Game.DisplayName, currentlyWatching.CurrentUser.Activity.User.BroadcastSettings.Title, currentlyWatching.CurrentUser.Activity.User.ProfileURL)
	_, _, err = twitterClient.Statuses.Update(msg, nil)
	if err != nil {
		log.Printf("error posting: %s", err)
	}
	log.Println(msg)
}

func setAuth(oauthToken string) func(*http.Request) {
	return func(req *http.Request) {
		req.Header.Set("Authorization", "OAuth "+oauthToken)
	}
}

func setClientId(clientId string) func(*http.Request) {
	return func(req *http.Request) {
		req.Header.Set("Client-Id", clientId)
	}
}
