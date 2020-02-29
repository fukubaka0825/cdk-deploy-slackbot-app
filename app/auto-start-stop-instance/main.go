package main

import (
	"bitbucket.org/wanocoltd/vk_infra/app/modules/slack_reporter"
	"bitbucket.org/wanocoltd/vk_infra/app/modules/up_down_stage_instance"
	"bitbucket.org/wanocoltd/vkgo_aws/v2/vkgo_aws_v2"
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/labstack/gommon/log"
	"syscall"
	"time"
)

type ACTION_TYPE string

const (
	START_ACTION  ACTION_TYPE = "start"
	STOP_ACTION   ACTION_TYPE = "stop"
	NOTICE_ACTION ACTION_TYPE = "notice"
)

func main() {
	lambda.Start(stopOrStartInstancesHandler)
}

func stopOrStartInstancesHandler(context context.Context, event map[string]string) (e error) {
	//Get ENV
	hostedZoneID, found := syscall.Getenv("HOSTED_ZONE_ID")
	if !found {
		return errors.New("HOSTED_ZONE_ID Not Found")
	}
	webHookURL, found := syscall.Getenv("WEB_HOOK_URL")
	if !found {
		return errors.New("WEB_HOOK_URL Not Found")
	}

	action := ACTION_TYPE(event["Action"])
	if action != START_ACTION && action != STOP_ACTION && action != NOTICE_ACTION {
		err := errors.New("invalid action")
		title := fmt.Sprintf("errorが発生しました。 \n errMsg:%v", err)
		if err := slack_reporter.ReportToSlack(webHookURL, title, "", slack_reporter.SLACK_MESSAGE_LEVEL_NOTIFY); err != nil {
			log.Error(err)
			return err
		}
	}

	ec2Client, err := vkgo_aws_v2.CreateEC2InstanceDefault()
	if err != nil {
		title := fmt.Sprintf("errorが発生しました。 \n errMsg:%v", err)
		if err := slack_reporter.ReportToSlack(webHookURL, title, "", slack_reporter.SLACK_MESSAGE_LEVEL_NOTIFY); err != nil {
			log.Error(err)
			return err
		}
		return err
	}

	route53Client, err := vkgo_aws_v2.CreateRoute53InstanceDefault()
	if err != nil {
		title := fmt.Sprintf("errorが発生しました。 \n errMsg:%v", err)
		if err := slack_reporter.ReportToSlack(webHookURL, title, "", slack_reporter.SLACK_MESSAGE_LEVEL_NOTIFY); err != nil {
			log.Error(err)
			return err
		}
		return err
	}

	switch action {
	case START_ACTION:
		if err := ec2Client.StartEc2InstancesByInstanceNames(up_down_stage_instance.MakeVKStageInstanceNameList()); err != nil {
			log.Error(err)
			return err
		}
		time.Sleep(time.Second * 30)
		//for ssh いちいちpubllic ipに別名つける必要あり
		vkStageDNSNameMap, err := up_down_stage_instance.MakeVKStageDNSNameMap()
		if err != nil {
			title := fmt.Sprintf("errorが発生しました。 \n errMsg:%v", err)
			if err := slack_reporter.ReportToSlack(webHookURL, title, "", slack_reporter.SLACK_MESSAGE_LEVEL_NOTIFY); err != nil {
				log.Error(err)
				return err
			}
			return err
		}
		if route53Client.CreateARecords(vkStageDNSNameMap, aws.String(hostedZoneID)) != nil {
			title := fmt.Sprintf("errorが発生しました。 \n errMsg:%v", err)
			if err := slack_reporter.ReportToSlack(webHookURL, title, "", slack_reporter.SLACK_MESSAGE_LEVEL_NOTIFY); err != nil {
				log.Error(err)
				return err
			}
			return err
		}

	case STOP_ACTION:
		//for ssh いちいちpubllic ipに別名を消す必要あり
		vkStageDNSNameMap, err := up_down_stage_instance.MakeVKStageDNSNameMap()
		if err != nil {
			title := fmt.Sprintf("errorが発生しました。 \n errMsg:%v", err)
			if err := slack_reporter.ReportToSlack(webHookURL, title, "", slack_reporter.SLACK_MESSAGE_LEVEL_NOTIFY); err != nil {
				log.Error(err)
				return err
			}
			return err
		}
		if route53Client.RemoveARecords(vkStageDNSNameMap, aws.String(hostedZoneID)) != nil {
			title := fmt.Sprintf("errorが発生しました。 \n errMsg:%v", err)
			if err := slack_reporter.ReportToSlack(webHookURL, title, "", slack_reporter.SLACK_MESSAGE_LEVEL_NOTIFY); err != nil {
				log.Error(err)
				return err
			}
			return err
		}

		if err := ec2Client.StopEc2InstancesByInstanceNames(up_down_stage_instance.MakeVKStageInstanceNameList()); err != nil {
			title := fmt.Sprintf("errorが発生しました。 \n errMsg:%v", err)
			if err := slack_reporter.ReportToSlack(webHookURL, title, "", slack_reporter.SLACK_MESSAGE_LEVEL_NOTIFY); err != nil {
				log.Error(err)
				return err
			}
			return err
		}
	case NOTICE_ACTION:
		//停止15分前通知
		title := "vk_stageインスタンス停止15分前です。作業を中止してください \n これ以上作業したい場合は@server_god vk_stage_upで再起動してください <@channel>"

		if err := slack_reporter.ReportToSlack(webHookURL, title, "", slack_reporter.SLACK_MESSAGE_LEVEL_NOTIFY); err != nil {
			log.Error(err)
			return err
		}
		return nil
	}

	var title = "*test/stage環境のサーバー再起動開始通知*"
	if action == "stop" {
		title = "*test/stage環境のサーバー停止開始通知*"
	}

	if err := slack_reporter.ReportToSlack(webHookURL, title, "", slack_reporter.SLACK_MESSAGE_LEVEL_NOTIFY); err != nil {
		log.Error(err)
		return err
	}

	return nil
}
