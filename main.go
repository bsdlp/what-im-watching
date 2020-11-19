package main

import (
	"context"
	"log"
	"net/http"

	"github.com/bsdlp/what-im-watching/twitch"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	OAuthToken        string `envconfig:"OAUTH_TOKEN" required:"true"`
	ClientId          string `split_words:"true" required:"true"`
	TwitchGqlEndpoint string `split_words:"true" default:"https://gql.twitch.tv/gql"`
}

//go:generate go run github.com/Yamashou/gqlgenc
func main() {
	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		log.Fatal(err)
	}

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
		log.Printf("%s is not currently watching anything", currentlyWatching.CurrentUser.DisplayName)
		return
	}
	log.Printf("%s is currently watching %s", currentlyWatching.CurrentUser.DisplayName, currentlyWatching.CurrentUser.Activity.User.DisplayName)
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
