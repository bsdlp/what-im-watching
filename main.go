package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/bsdlp/what-im-watching/twitch"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	TwitchOAuthToken    string `envconfig:"TWITCH_OAUTH_TOKEN" required:"true"`
	TwitchClientId      string `split_words:"true" required:"true"`
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

	client := twitch.NewClient(http.DefaultClient, cfg.TwitchGqlEndpoint, setAuth(cfg.TwitchOAuthToken), setClientId(cfg.TwitchClientId))

	lambda.Start(func(ctx context.Context) error {
		currentlyWatching, err := client.GetCurrentlyWatching(ctx)
		if err != nil {
			log.Println(err)
			return err
		}
		if currentlyWatching.CurrentUser == nil {
			err := errors.New("current user is nil")
			return err
		}
		if currentlyWatching.CurrentUser.Activity == nil {
			msg := fmt.Sprintf("%s is not currently watching anything", currentlyWatching.CurrentUser.DisplayName)
			log.Println(msg)
			return nil
		}

		streamer := currentlyWatching.CurrentUser.Activity.User

		msg := fmt.Sprintf("%s is now watching %s stream %s: %s\n%s", currentlyWatching.CurrentUser.DisplayName, streamer.DisplayName, streamer.BroadcastSettings.Game.DisplayName, streamer.BroadcastSettings.Title, streamer.ProfileURL)
		_, _, err = twitterClient.Statuses.Update(msg, nil)
		if err != nil {
			return fmt.Errorf("error posting: %s", err)
		}

		return nil
	})
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
