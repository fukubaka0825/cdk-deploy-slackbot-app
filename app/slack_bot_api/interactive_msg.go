package main

import (
	"bitbucket.org/wanocoltd/vk_infra/app/modules/common"
	"bitbucket.org/wanocoltd/vk_infra/app/modules/slack_reporter"
	"bitbucket.org/wanocoltd/vkgo_core/v2/util/log"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/ktrysmt/go-bitbucket"
	"github.com/nlopes/slack"
	"github.com/pkg/errors"
	"net/http"
	"net/url"
	"strings"
)

func makeInteractiveResponse(request events.APIGatewayProxyRequest, config envConfig) (events.APIGatewayProxyResponse, error) {
	response := events.APIGatewayProxyResponse{}

	client := slack.New(config.BotOAuth)

	str, _ := url.QueryUnescape(request.Body)
	str = strings.Replace(str, "payload=", "", 1)
	log.Infof("str:%v type:%T", str, str)

	var callback slack.InteractionCallback
	if err := json.Unmarshal([]byte(str), &callback); err != nil {
		err = errors.New("うまくrequestを変換できなかったみたい。もう一度コマンド入力お願いします。")
		if err := slack_reporter.PostErrMsg(client, err, config.CallBackID, config.ChannelID); err != nil {
			return response, fmt.Errorf("failed to post message: %s", err)
		}
		return events.APIGatewayProxyResponse{Body: "json error", StatusCode: 500}, nil
	}

	if request.HTTPMethod != http.MethodPost {
		response.StatusCode = http.StatusMethodNotAllowed
		return response, errors.New("Invalid method")
	}

	action := callback.ActionCallback.AttachmentActions[0]
	switch ACTION(action.Name) {
	case ACTION_REPO_SELECT:
		value := action.SelectedOptions[0].Value
		log.Infof("%v", value)

		c := bitbucket.NewBasicAuth(config.BitbucketUserID, config.BitbucketPass)
		rbo := &bitbucket.RepositoryBranchOptions{
			Owner:    config.Owner,
			RepoSlug: value,
			Pagelen:  100,
		}
		brs, err := c.Repositories.Repository.ListBranches(rbo)
		if err != nil {
			log.Error(err)
			return response, err
		}

		options := []slack.AttachmentActionOption{}
		for _, br := range brs.Branches {
			options = append(options, slack.AttachmentActionOption{
				Text:  br.Name,
				Value: value + ":" + br.Name,
			})
		}

		message := &slack.Message{}
		attachment := &slack.Attachment{}
		attachment.Color = string(slack_reporter.NOTIFY_COLOR_WHITE)
		attachment.CallbackID = config.CallBackID
		attachment.Text = fmt.Sprintf("*%s のどのブランチ選ぶ？？？*", value)
		attachment.Actions = []slack.AttachmentAction{
			{
				Name:    string(ACTION_BRANCH_SELECT),
				Type:    "select",
				Options: options,
			},
			{
				Name:  string(ACTION_CANCEL),
				Text:  "やっぱいいや",
				Type:  "button",
				Style: "danger",
			},
		}
		message.Attachments = append(message.Attachments, *attachment)
		message.ReplaceOriginal = true

		return makeResponse(&response, message)
	case ACTION_BRANCH_SELECT:
		value := action.SelectedOptions[0].Value
		log.Infof("%v", value)

		options := []slack.AttachmentActionOption{}
		envList := []string{string(common.ENV_VK_STAGE), string(common.ENV_VK_PROD)}
		for _, env := range envList {
			options = append(options, slack.AttachmentActionOption{
				Text:  env,
				Value: value + ":" + env,
			})
		}

		message := &slack.Message{}

		attachment := &slack.Attachment{}
		attachment.Color = string(slack_reporter.NOTIFY_COLOR_WHITE)
		attachment.CallbackID = config.CallBackID
		repoName := strings.Split(value, ":")[0]
		branch := strings.Split(value, ":")[1]
		attachment.Text = fmt.Sprintf("*project:%s* \n *branch:%v* \n *どの環境にdeployする？？？* ", repoName, branch)
		attachment.Actions = []slack.AttachmentAction{
			{
				Name:    string(ACTION_ENV_SELECT),
				Type:    "select",
				Options: options,
			},
			{
				Name:  string(ACTION_CANCEL),
				Text:  "やっぱいいや",
				Type:  "button",
				Style: "danger",
			},
		}

		message.Attachments = append(message.Attachments, *attachment)
		message.ReplaceOriginal = true

		return makeResponse(&response, message)
	case ACTION_ENV_SELECT:
		value := action.SelectedOptions[0].Value
		log.Infof("%v", value)

		message := &slack.Message{}

		attachment := &slack.Attachment{}
		attachment.Color = string(slack_reporter.NOTIFY_COLOR_WHITE)
		attachment.CallbackID = config.CallBackID
		repoName := strings.Split(value, ":")[0]
		branch := strings.Split(value, ":")[1]
		env := strings.Split(value, ":")[2]

		attachment.Text = fmt.Sprintf("*project:%s* \n *branch:%s* \n *env:%s* \n  *本当にdeployしてしまっていい？？？* ", repoName, branch, env)
		attachment.Actions = []slack.AttachmentAction{
			{
				Name:  string(ACTION_START),
				Text:  "はい",
				Type:  "button",
				Style: "primary",
				Value: value,
			},
			{
				Name:  string(ACTION_CANCEL),
				Text:  "やっぱいいや",
				Type:  "button",
				Style: "danger",
			},
		}

		message.Attachments = append(message.Attachments, *attachment)
		message.ReplaceOriginal = true

		return makeResponse(&response, message)
	case ACTION_START:
		value := action.Value
		repoName := strings.Split(value, ":")[0]
		branch := strings.Split(value, ":")[1]
		env := strings.Split(value, ":")[2]
		depReg := &StartDeployRequest{
			owner:         config.Owner,
			repoName:      repoName,
			branch:        branch,
			env:           env,
			circleciToken: config.CirclCIToken,
		}
		if err := startDeploy(depReg); err != nil {
			log.Error(err)
			return response, err
		}
		message := &slack.Message{}

		attachment := &slack.Attachment{}
		attachment.Color = string(slack_reporter.NOTIFY_COLOR_GREEN)
		attachment.CallbackID = config.CallBackID
		attachment.Text = fmt.Sprintf("*user:@%s* \n *project:%s* \n *branch:%s* \n *env:%s* \n *デプロイの開始に成功！*", callback.User.Name, repoName, branch, env)

		message.Attachments = append(message.Attachments, *attachment)
		message.DeleteOriginal = true
		message.ResponseType = "in_channel"
		return makeResponse(&response, message)
	case ACTION_CANCEL:
		message := &slack.Message{}

		attachment := &slack.Attachment{}
		attachment.Color = string(slack_reporter.NOTIFY_COLOR_RED)
		attachment.CallbackID = config.CallBackID
		attachment.Text = fmt.Sprintf(":x: *@%s はデプロイを辞めたようだ*", callback.User.Name)

		message.Attachments = append(message.Attachments, *attachment)
		message.DeleteOriginal = true
		message.ResponseType = "in_channel"
		return makeResponse(&response, message)
	default:
		response.StatusCode = http.StatusInternalServerError
		return response, errors.New("Invalid action was submitted")

	}
}

func makeResponse(response *events.APIGatewayProxyResponse, message *slack.Message) (events.APIGatewayProxyResponse, error) {
	resJson, err := json.Marshal(&message)
	if err != nil {
		log.Error(err)
		return *response, err
	}
	response.Body = string(resJson)
	response.Headers = make(map[string]string)
	response.Headers["Content-Type"] = "application/json"
	response.StatusCode = http.StatusOK
	return *response, nil
}
