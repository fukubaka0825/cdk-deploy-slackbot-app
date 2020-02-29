#!/usr/bin/env node
import 'source-map-support/register';
import cdk = require('@aws-cdk/core');
import { AutoStartStopInstanceCdkStack } from '../lib/auto_start_stop_instance';
import { hogeSlackBotApiStack } from '../lib/hoge_slack_bot_api';

const util = require('util');
const exec = util.promisify(require('child_process').exec);

async function deploy(){
    await exec('go get -v -t -d ../../../../../../app/auto_start_stop_instance/... && GOOS=linux GOARCH=amd64 go build -o ./.build/auto_start_stop_instances/main ../../../../../../app/auto_start_stop_instance/**.go')
    await exec('go get -v -t -d ../../../../../../app/hoge_slack_bot_api/... && GOOS=linux GOARCH=amd64 go build -o ./.build/hoge_slack_bot_api/main ../../../../../../app/hoge_slack_bot_api/**.go')
    const app = new cdk.App();
    new AutoStartStopInstanceCdkStack(app, "AutoStartStopInstance");
    new hogeSlackBotApiStack(app, "hogeSlackBotApiStack");
    app.synth()
    await exec('rm ./.build/auto_start_stop_instances/main ./.build/hoge_slack_bot_api/main')
}

deploy()
