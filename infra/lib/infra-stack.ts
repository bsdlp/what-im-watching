import * as cdk from '@aws-cdk/core';
import assets = require("@aws-cdk/aws-s3-assets");
import dynamodb = require("@aws-cdk/aws-dynamodb");
import events = require("@aws-cdk/aws-events");
import events_targets = require("@aws-cdk/aws-events-targets");
import lambda = require("@aws-cdk/aws-lambda");
import logs = require("@aws-cdk/aws-logs");
import path = require("path");
import secrets_manager = require("@aws-cdk/aws-secretsmanager");

export class InfraStack extends cdk.Stack {
  constructor(scope: cdk.Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    const whatImWatchingTable = new dynamodb.Table(this, "whatImPreviouslyWatching", {
      partitionKey: {
        name: "UserId",
        type: dynamodb.AttributeType.STRING,
      },
      timeToLiveAttribute: "ExpirationTime",
    })

    const whatImWatchingAsset = new assets.Asset(this, "whatImWatchingAsset", {
      path: path.join(__dirname, "../../build/"),
    })

    const secrets = secrets_manager.Secret.fromSecretCompleteArn(this, "whatImWatchingSecrets", "arn:aws:secretsmanager:us-west-2:635281304921:secret:what-im-watching/prod/secrets-IfbaaY")

    const whatImWatchingLambda = new lambda.Function(this, "whatImWatchingLambda", {
      code: lambda.Code.fromBucket(
        whatImWatchingAsset.bucket,
        whatImWatchingAsset.s3ObjectKey,
      ),
      runtime: lambda.Runtime.GO_1_X,
      handler: "what-im-watching",
      timeout: cdk.Duration.seconds(45),
      environment: {
        "TWITCH_CLIENT_ID": secrets.secretValueFromJson("TWITCH_CLIENT_ID").toString(),
        "TWITCH_OAUTH_TOKEN": secrets.secretValueFromJson("TWITCH_OAUTH_TOKEN").toString(),
        "TWITTER_API_KEY": secrets.secretValueFromJson("TWITTER_API_KEY").toString(),
        "TWITTER_API_SECRET": secrets.secretValueFromJson("TWITTER_API_SECRET").toString(),
        "TWITTER_ACCESS_TOKEN": secrets.secretValueFromJson("TWITTER_ACCESS_TOKEN").toString(),
        "TWITTER_ACCESS_SECRET": secrets.secretValueFromJson("TWITTER_ACCESS_SECRET").toString(),
        "PREVIOUSLY_WATCHING_TABLE_NAME": whatImWatchingTable.tableName,
        "PREVIOUSLY_WATCHING_EVENT_TTL": "2h",
      },
      logRetention: logs.RetentionDays.THREE_DAYS,
    })

    whatImWatchingTable.grantReadWriteData(whatImWatchingLambda)

    new events.Rule(this, "everyFiveMinutes", {
      schedule: events.Schedule.rate(cdk.Duration.minutes(5)),
      targets: [
        new events_targets.LambdaFunction(whatImWatchingLambda),
      ],
    })
  }
}
