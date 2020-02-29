# What are these
- hogestageサーバーの自動起動停止の仕組み(lambda,cw)
- hogestageサーバーの全台起動と全台停止とhogeprojectのデプロイ(Slackbot)のためのサーバー(lambda,apigateway)

# How to deploy these cdk stacks
- (hoge stage accountのkeyへcredentialを変更)
- npm ci
- cdk list
- cdk deploy AutoStartStopInstance
    - 自動起動停止の仕組み
- cdk deploy hogeSlackBotApiStack
    - 全台起動と全台停止(Slackbot)とhogeprojectのdeployのためのサーバー
