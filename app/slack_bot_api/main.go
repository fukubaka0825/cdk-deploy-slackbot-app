package main

import (
	"bitbucket.org/wanocoltd/vk_infra/app/modules/slack_reporter"
	"context"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/kelseyhightower/envconfig"
	"github.com/labstack/gommon/log"
	"github.com/pkg/errors"
	"strings"
)

type ACTION string

const (
	ACTION_REPO_SELECT   ACTION = "repo_select"
	ACTION_BRANCH_SELECT ACTION = "branch_select"
	ACTION_ENV_SELECT    ACTION = "env_select"
	ACTION_START         ACTION = "start"
	ACTION_CANCEL        ACTION = "cancel"
)

type envConfig struct {
	// BotToken is bot user token to access to slack API.
	BotToken string `envconfig:"BOT_TOKEN" required:"true"`

	// BotID is bot user ID.
	BotID string `envconfig:"BOT_ID" required:"true"`

	// BotOAuth used when making slack client
	BotOAuth string `envconfig:"BOT_OAUTH" required:"true"`

	// Used to verify request
	SigningSecrets string `envconfig:"SIGNING_SECRETS" required:"true"`

	// ChannelID is slack channel ID where bot is working.
	// Bot responses to the mention in this channel.
	ChannelID string `envconfig:"CHANNEL_ID" required:"true"`

	// Bitbucket owner
	Owner string `envconfig:"OWNER" required:"true"`

	// CircleCI personal token. Used to start circleCI's deploy project.
	CirclCIToken string `envconfig:"CIRCLECI_TOKEN" required:"true"`

	//used to identify actions
	CallBackID string `envconfig:"CALL_BACK_ID" required:"true"`

	//AWS Route 53 HostedZoneID
	HostedZoneID string `envconfig:"HOSTED_ZONE_ID" required:"true"`

	//Used to get repo's branch list
	BitbucketUserID string `envconfig:"BITBUCKET_USER_ID" required:"true"`
	BitbucketPass   string `envconfig:"BITBUCKET_PASS" required:"true"`
}

func main() {
	lambda.Start(handleRequest)
}

func handleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Info("開始")
	response := events.APIGatewayProxyResponse{}

	// Get ENV
	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		log.Printf("[ERROR] Failed to process env var: %s", err)
		return response, errors.Errorf("[ERROR] Failed to process env var: %s", err)
	}

	// Verify with signingSecrets
	if err := slack_reporter.Verify(env.SigningSecrets, request); err != nil {
		log.Error(err)
		return response, err
	}

	if strings.Contains(request.Body, "payload=") {
		return makeInteractiveResponse(request, env)
	}
	return makeEventResponse(request, env)
}
