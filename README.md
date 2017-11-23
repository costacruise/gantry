# Gantry

Send and receive executable payloads over SQS.

## What Does It Do?

Given a directory of commands and assets, it will create a base64 encoded, gzip
compressed tarball and fire it over SNS on a specific queue. Why? In order that
we can send a shell script and all the relevant data to a remote machine and
have them execute whatever we have sent to them.

Gantry contains a single executable (Go) that can emit and consume messages
over a common queue.

For an example of how an ideal payload directory might look, please see
`./fixtures/happy-path`.

## Cloning this repo

Dependencies are managed with `dep` and can be installed thusly:

    export $GOPATH=./some/where
    git clone git@github.com:costacruise/gantry.git $GOPATH/src/github.com/costacruise/gantry
    (cd $GOPATH/src/github.com/costacruise/gantry && dep ensure)

## Usage:

To publish a directory containing a payload:

    $ AWS_PROFILE=default \
      AWS_REGION=eu-west-1 \
      ./gantry -sqs-queue-url=https://sqs.eu-west-1.amazonaws.com/11111111111/example -dir ./path/to/payload publish

To consume a message (

    $ AWS_PROFILE=default \
      AWS_REGION=eu-west-1 \
      ./gantry -sqs-queue-url=https://sqs.eu-west-1.amazonaws.com/11111111111/example -dir ./path/to/payload publish

