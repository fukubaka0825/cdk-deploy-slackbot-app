import cdk = require("@aws-cdk/core")
import { Function, Runtime, Code } from "@aws-cdk/aws-lambda"
import { RestApi, Integration, LambdaIntegration, Resource,
  MockIntegration, PassthroughBehavior, EmptyModel } from "@aws-cdk/aws-apigateway"
import * as iam from '@aws-cdk/aws-iam';

export class hogeSlackBotApiStack extends cdk.Stack {
  constructor(scope: cdk.App, id: string, props?: cdk.StackProps) {
    super(scope, id, props)

    // Lambda Function 作成
    const lambdaFunction: Function = new Function(this, "hogeSlackBotApi", {
      functionName: "hoge-slack-bot-api", // 関数名
      runtime: Runtime.GO_1_X, // ランタイムの指定
      code: Code.asset("./.build/hoge_slack_bot_api"), // ソースコードのディレクトリ
      handler: "main", // handler の指定
      memorySize: 256, // メモリーの指定
      timeout: cdk.Duration.seconds(120), // タイムアウト時間
      environment: {
        "BOT_TOKEN":"NyCAxxxxxxxxxxxxxxxx",
        "CHANNEL_ID":"GSTxxxxxxx",
        "BOT_ID":"UMVxxxxxx",
        "BOT_OAUTH":"xoxb-2901674924-xxxxxxxxxxxxxxxxxxxxxxxxx",
        "SIGNING_SECRETS":"2c3b259xxxxxxxxxxxxxxxxx",
        "HOSTED_ZONE_ID":"Z17I7Yxxxxxxxx",
        "CALL_BACK_ID":"hoge_deploy",
        "OWNER":"wanocoltd",
        "CIRCLECI_TOKEN":"f84f0738xxxxxxxxxxxxx",
        "BITBUCKET_USER_ID":"xxxxxxx",
        "BITBUCKET_PASS":"ws5aWxxxxxxxxxxxxx"
      } // 環境変数
    })

    //Policyを関数に付加
    lambdaFunction.addToRolePolicy(new iam.PolicyStatement({
      resources:["*"],
      actions:[
          "ec2:StartInstances",
          "ec2:StopInstances",
          "ec2:DescribeInstances",
          "ec2:DescribeSecurityGroups",
          "ec2:AuthorizeSecurityGroupIngress",
          "ec2:RevokeSecurityGroupIngress",
          "route53:ChangeResourceRecordSets"
      ]
    }))

    // API Gateway 作成
    const restApi: RestApi = new RestApi(this, "hogeSlackBotApi", {
      restApiName: "hogeSlackBotApi", // API名
      description: "hogeSlackBotApi Deployed by CDK" // 説明
    })

    // Integration 作成
    const integration: Integration = new LambdaIntegration(lambdaFunction)

    // リソースの作成
    const getResouse: Resource = restApi.root.addResource("event")

    // メソッドの作成
    getResouse.addMethod("POST", integration)

  }
}