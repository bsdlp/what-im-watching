package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/bsdlp/what-im-watching/twitch"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	TwitchOAuthToken            string        `envconfig:"TWITCH_OAUTH_TOKEN" required:"true"`
	TwitchClientId              string        `split_words:"true" required:"true"`
	TwitchGqlEndpoint           string        `split_words:"true" default:"https://gql.twitch.tv/gql"`
	TwitterApiKey               string        `required:"true" split_words:"true"`
	TwitterApiSecret            string        `required:"true" split_words:"true"`
	TwitterAccessToken          string        `required:"true" split_words:"true"`
	TwitterAccessSecret         string        `required:"true" split_words:"true"`
	PreviouslyWatchingTableName string        `required:"true" split_words:"true"`
	PreviouslyWatchingEventTtl  time.Duration `default:"2h" split_words:"true"`
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

	kvs := &keyValueStore{
		ddb:       dynamodb.New(session.Must(session.NewSession())),
		tableName: cfg.PreviouslyWatchingTableName,
	}

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

		previouslyWatching, err := kvs.GetPreviouslyWatching(currentlyWatching.CurrentUser.ID)
		if err != nil {
			log.Printf("error getting previously watching: %s", err)
		}

		if previouslyWatching == nil {
			log.Println("not previously watching anything")
		}

		streamer := currentlyWatching.CurrentUser.Activity.User

		err = kvs.SetPreviouslyWatching(currentlyWatching.CurrentUser.ID, streamer.ID, cfg.PreviouslyWatchingEventTtl)
		if err != nil {
			log.Printf("error setting previously watching: %s", err)
		}

		if previouslyWatching != nil {
			if previouslyWatching.StreamUserId == streamer.ID {
				log.Printf("%s is still watching %s", currentlyWatching.CurrentUser.DisplayName, streamer.DisplayName)
				return nil
			}
		}

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

type PreviouslyWatchingEvent struct {
	UserId         string
	Timestamp      dynamodbattribute.UnixTime
	StreamUserId   string
	ExpirationTime dynamodbattribute.UnixTime
}

type keyValueStore struct {
	ddb       dynamodbiface.DynamoDBAPI
	tableName string
}

func (kv *keyValueStore) GetPreviouslyWatching(userId string) (*PreviouslyWatchingEvent, error) {
	params := &dynamodb.GetItemInput{
		TableName: aws.String(kv.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"UserId": {S: aws.String(userId)},
		},
	}

	output, err := kv.ddb.GetItem(params)
	if err != nil {
		return nil, err
	}

	var event PreviouslyWatchingEvent
	err = dynamodbattribute.UnmarshalMap(output.Item, &event)
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (kv *keyValueStore) SetPreviouslyWatching(userId, streamId string, ttl time.Duration) error {
	now := time.Now()
	av, err := dynamodbattribute.MarshalMap(&PreviouslyWatchingEvent{
		UserId:         userId,
		StreamUserId:   streamId,
		Timestamp:      dynamodbattribute.UnixTime(now),
		ExpirationTime: dynamodbattribute.UnixTime(now.Add(ttl)),
	})
	if err != nil {
		return err
	}

	params := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(kv.tableName),
	}
	_, err = kv.ddb.PutItem(params)
	return err
}
