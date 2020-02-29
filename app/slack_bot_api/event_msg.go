package main

import (
	"bitbucket.org/wanocoltd/vk_infra/app/modules/slack_reporter"
	"bitbucket.org/wanocoltd/vk_infra/app/modules/up_down_stage_instance"
	"bitbucket.org/wanocoltd/vkgo_aws/v2/vkgo_aws_v2"
	"bitbucket.org/wanocoltd/vkgo_core/v2/util/log"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/nlopes/slack"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ApiEvent struct {
	Type      string `json:"type"`
	Text      string `json:"text"`
	Challenge string `json:"challenge"`
	Token     string `json:"token"`
	Event     Event  `json:"event"`
}
type Event struct {
	User    string `json:"user"`
	Type    string `json:"type"`
	Text    string `json:"text"`
	Channel string `json:"channel"`
}

const (
	EVENT_TYPE_URL_VERIFICATION = "url_verification"
	EVENT_TYPE_EVENT_CALLBACK   = "event_callback"
)

type CommandMsg string

const (
	UP_STAGE_SERVER_COMMAND   CommandMsg = "vk_stage_up"
	DOWN_STAGE_SERVER_COMMAND CommandMsg = "vk_stage_down"
	DEPLOY_COMMAND            CommandMsg = "vk_deploy"
)

func makeEventResponse(request events.APIGatewayProxyRequest, config envConfig) (events.APIGatewayProxyResponse, error) {

	vals, _ := url.ParseQuery(request.Body)
	log.Infof("vals:%v type:%T", vals, vals)

	response := events.APIGatewayProxyResponse{}

	client := slack.New(config.BotOAuth)

	route53Client, err := vkgo_aws_v2.CreateRoute53InstanceDefault()
	if err != nil {
		if err := slack_reporter.PostErrMsg(client, err, config.CallBackID, config.ChannelID); err != nil {
			return response, fmt.Errorf("failed to post message: %s", err)
		}
		return response, err
	}

	apiEvent := &ApiEvent{}
	for key, _ := range vals {
		log.Infof("json:%v", key)
		err := json.Unmarshal([]byte(key), apiEvent)
		if err != nil {
			err = errors.New("うまくrequestを変換できなかったみたい。もう一度コマンド入力お願いします。")
			if err := slack_reporter.PostErrMsg(client, err, config.CallBackID, config.ChannelID); err != nil {
				return response, fmt.Errorf("failed to post message: %s", err)
			}
			return response, err
		}
	}

	switch apiEvent.Type {
	case EVENT_TYPE_URL_VERIFICATION:
		log.Info("url_verification")
		response.Headers = make(map[string]string)
		response.Headers["Content-Type"] = "text/plain"
		response.Body = apiEvent.Challenge
		response.StatusCode = http.StatusOK
		return response, nil
	case EVENT_TYPE_EVENT_CALLBACK:

		event := apiEvent.Event
		//input validate
		if event.Type != "app_mention" {
			err := errors.New("Not target eventType")
			if err := slack_reporter.PostErrMsg(client, err, config.CallBackID, config.ChannelID); err != nil {
				return response, fmt.Errorf("failed to post message: %s", err)
			}
			return response, err
		}

		if !strings.HasPrefix(event.Text, fmt.Sprintf("<@%s>", config.BotID)) {
			err := errors.New("Bot Id Not Found")
			if err := slack_reporter.PostErrMsg(client, err, config.CallBackID, config.ChannelID); err != nil {
				return response, fmt.Errorf("failed to post message: %s", err)
			}
			return response, err
		}
		if event.Channel != config.ChannelID {
			err := errors.New("Channel Id Not Found")
			if err := slack_reporter.PostErrMsg(client, err, config.CallBackID, config.ChannelID); err != nil {
				return response, fmt.Errorf("failed to post message: %s", err)
			}
			return response, err
		}
		if apiEvent.Token != config.BotToken {
			err := errors.New("botToken Not Found")
			if err := slack_reporter.PostErrMsg(client, err, config.CallBackID, config.ChannelID); err != nil {
				return response, fmt.Errorf("failed to post message: %s", err)
			}
			return response, err
		}

		m := strings.Split(strings.TrimSpace(event.Text), " ")[1:]

		ec, err := vkgo_aws_v2.CreateEC2InstanceDefault()
		if err != nil {
			if err := slack_reporter.PostErrMsg(client, err, config.CallBackID, config.ChannelID); err != nil {
				return response, fmt.Errorf("failed to post message: %s", err)
			}
			return response, err
		}

		msgOptText := slack.MsgOptionText("", true)

		if len(m) == 0 {
			attachment := slack.Attachment{
				Text:       "*何も問いかけがないようだぞ。サーバー神は以下のmessageにしか答えない* \n *vk_stage_up* : stageのインスタンスを緊急対応であげたい場合使用　 \n *vk_stage_down* : stageのインスタンスを緊急対応後で下げたい場合使用 \n *vk_deploy* :vkのプロジェクトのデプロイ(対話型)",
				Color:      string(slack_reporter.NOTIFY_COLOR_RED),
				CallbackID: config.CallBackID,
			}
			msgOptAttachment := slack.MsgOptionAttachments(attachment)

			if _, _, err := client.PostMessage(config.ChannelID, msgOptText, msgOptAttachment); err != nil {
				return response, fmt.Errorf("failed to post message: %s", err)
			}
			response.StatusCode = http.StatusOK
			return response, nil
		}

		if m[0] != string(UP_STAGE_SERVER_COMMAND) && m[0] != string(DOWN_STAGE_SERVER_COMMAND) && m[0] != string(DEPLOY_COMMAND) {
			attachment := slack.Attachment{
				Text:       "*サーバー神は以下のmessageにしか答えない* \n *vk_stage_up* : stageのインスタンスを緊急対応であげたい場合使用　 \n *vk_stage_down* : stageのインスタンスを緊急対応後で下げたい場合使用 \n *vk_deploy* :vkのプロジェクトのデプロイ(対話型) ",
				Color:      string(slack_reporter.NOTIFY_COLOR_RED),
				CallbackID: config.CallBackID,
			}
			msgOptAttachment := slack.MsgOptionAttachments(attachment)

			if _, _, err := client.PostMessage(config.ChannelID, msgOptText, msgOptAttachment); err != nil {
				return response, fmt.Errorf("failed to post message: %s", err)
			}
			response.StatusCode = http.StatusOK
			return response, nil
		}

		if m[0] == string(DOWN_STAGE_SERVER_COMMAND) {
			//for ssh いちいちpubllic ipに別名を消す必要あり
			vkStageDNSNameMap, err := up_down_stage_instance.MakeVKStageDNSNameMap()
			if err != nil {
				log.Error(err)
				return response, err
			}
			if err := route53Client.RemoveARecords(vkStageDNSNameMap, aws.String(config.HostedZoneID)); err != nil {
				return response, fmt.Errorf("failed to post message: %s", err)
			}

			if err := ec.StopEc2InstancesByInstanceNames(up_down_stage_instance.MakeVKStageInstanceNameList()); err != nil {
				if err := slack_reporter.PostErrMsg(client, err, config.CallBackID, config.ChannelID); err != nil {
					return response, fmt.Errorf("failed to post message: %s", err)
				}
				return response, err
			}

			attachment := slack.Attachment{
				Text:       "*vk stageのEC2インスタンス停止開始*",
				Color:      string(slack_reporter.NOTIFY_COLOR_GREEN),
				CallbackID: config.CallBackID,
			}

			msgOptAttachment := slack.MsgOptionAttachments(attachment)

			if _, _, err := client.PostMessage(config.ChannelID, msgOptText, msgOptAttachment); err != nil {
				return response, fmt.Errorf("failed to post message: %s", err)
			}
			response.StatusCode = http.StatusOK
			return response, nil
		}
		if m[0] == string(UP_STAGE_SERVER_COMMAND) {

			if err := ec.StartEc2InstancesByInstanceNames(up_down_stage_instance.MakeVKStageInstanceNameList()); err != nil {
				if err := slack_reporter.PostErrMsg(client, err, config.CallBackID, config.ChannelID); err != nil {
					return response, fmt.Errorf("failed to post message: %s", err)
				}
				return response, err
			}
			time.Sleep(time.Second * 30)
			////for ssh いちいちpubllic ipに別名つける必要あり
			vkStageDNSNameMap, err := up_down_stage_instance.MakeVKStageDNSNameMap()
			if err != nil {
				if err := slack_reporter.PostErrMsg(client, err, config.CallBackID, config.ChannelID); err != nil {
					return response, fmt.Errorf("failed to post message: %s", err)
				}

				return response, err
			}

			if err := route53Client.CreateARecords(vkStageDNSNameMap, aws.String(config.HostedZoneID)); err != nil {
				return response, fmt.Errorf("failed to post message: %s", err)
			}

			attachment := slack.Attachment{
				Text:       "*vk stageのEC2インスタンス再起動開始*",
				Color:      string(slack_reporter.NOTIFY_COLOR_GREEN),
				CallbackID: config.CallBackID,
			}
			msgOptAttachment := slack.MsgOptionAttachments(attachment)

			if _, _, err := client.PostMessage(config.ChannelID, msgOptText, msgOptAttachment); err != nil {
				return response, fmt.Errorf("failed to post message: %s", err)
			}
			response.StatusCode = http.StatusOK
			return response, nil
		}
		if m[0] == string(DEPLOY_COMMAND) {
			options := []slack.AttachmentActionOption{}
			deployRepos := []string{
				INTERNAL_REPO,
				INTERNAL_PROXY_REPO,
				STORE_REVIEW_REPO,
				VIDEOCROP_REPO,
				POST_PRCESS_BATCH_REPO,
				WEB_REPO,
				DAISENDAN_REPO,
				REVIEW_REPO,
				DELIVERY_HANDLER_REPO,
			}
			for _, repo := range deployRepos {
				options = append(options, slack.AttachmentActionOption{
					Text:  repo,
					Value: repo,
				})
			}
			attachment := slack.Attachment{
				Text:       "*どのprojectが対象かの?*",
				Color:      string(slack_reporter.NOTIFY_COLOR_WHITE),
				CallbackID: config.CallBackID,
				Actions: []slack.AttachmentAction{
					{
						Name:    string(ACTION_REPO_SELECT),
						Type:    "select",
						Options: options,
					},

					{
						Name:  string(ACTION_CANCEL),
						Text:  "やっぱ何でもないわ",
						Type:  "button",
						Style: "danger",
					},
				},
			}

			msgOptAttachment := slack.MsgOptionAttachments(attachment)

			if _, err := client.PostEphemeral(config.ChannelID, event.User, msgOptAttachment); err != nil {
				return response, fmt.Errorf("メッセージ送信に失敗: %s", err)
			}

			response.StatusCode = http.StatusOK
			return response, nil

		}
		response.StatusCode = http.StatusOK
		return response, nil
	default:
		response.StatusCode = http.StatusOK
		return response, nil
	}
}
