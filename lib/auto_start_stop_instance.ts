import events = require('@aws-cdk/aws-events');
import iam = require('@aws-cdk/aws-iam');
import targets = require('@aws-cdk/aws-events-targets');
import { Function, Runtime, Code } from "@aws-cdk/aws-lambda"
import cdk = require('@aws-cdk/core');
import fs = require('fs');

export class AutoStartStopInstanceCdkStack extends cdk.Stack {
  constructor(scope: cdk.Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    const stackConfig = JSON.parse(fs.readFileSync('config.json', {encoding: 'utf-8'}));

    const lambdaFn = new Function(this, 'singleton', {
      functionName: "hoge-stage-auto-ec2-startstop", // 関数名
      runtime: Runtime.GO_1_X, // ランタイムの指定
      code: Code.asset("./.build/auto_start_stop_instances"), // ソースコードのディレクトリ
      handler: "main", // handler の指定
      memorySize: 256, // メモリーの指定
      timeout: cdk.Duration.seconds(120), // タイムアウト時間
      environment: {
        "HOSTED_ZONE_ID":"Z17I7xxxxxxxxxxxxx",
        "WEB_HOOK_URL" : "https://hooks.slack.com/services/T02xxxxxxxxx/BQxxxxxxxxxx/Kxxxxxxxxxxxxxxxxxx"
      } // 環境変数
    });

    lambdaFn.addToRolePolicy(new iam.PolicyStatement({
      actions: [
        'ec2:DescribeInstances',
        'ec2:StartInstances',
        'ec2:StopInstances',
        "ec2:DescribeSecurityGroups",
        "ec2:AuthorizeSecurityGroupIngress",
        "ec2:RevokeSecurityGroupIngress",
        "route53:ChangeResourceRecordSets",
        "rds:StopDBInstance",
        "rds:StartDBInstance",
        "rds:StartDBCluster",
        "rds:StopDBCluster",
      ],
      resources: ['*']
    }));

    // STOP EC2 instances rule
    const stopRule = new events.Rule(this, 'StopRule', {
      schedule: events.Schedule.expression(`cron(${stackConfig.events.cron.stop})`)
    });

    stopRule.addTarget(new targets.LambdaFunction(lambdaFn, {
      event: events.RuleTargetInput.fromObject({Action: 'stop'})
    }));

    // START EC2 instances rule
    const startRule = new events.Rule(this, 'StartRule', {
      schedule: events.Schedule.expression(`cron(${stackConfig.events.cron.start})`)
    });

    startRule.addTarget(new targets.LambdaFunction(lambdaFn, {
      event: events.RuleTargetInput.fromObject({Action: 'start'})
    }));

    // Notify of stopping Instance
    const noticeRule = new events.Rule(this, 'NoticeRule', {
      schedule: events.Schedule.expression(`cron(${stackConfig.events.cron.notice})`)
    });

    noticeRule.addTarget(new targets.LambdaFunction(lambdaFn, {
      event: events.RuleTargetInput.fromObject({Action: 'notice'})
    }));

  }
}
