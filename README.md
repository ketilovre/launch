launch
======

Launch is a tool for creating serverless deployments of web apps on AWS using the new
API Gateway proxy features.

Inspired by [Blueprints for up(1)](https://medium.com/@tjholowaychuk/blueprints-for-up-1-5f8197179275#.6ixfdflgc).

### Getting started

Run `launch init` to bootstrap your `launch.yml` and `server` files.

Your `server` file is responsible for starting your application. For a Node app, `server`
would contain something along these lines:

    #!/usr/bin/env bash
    node app.js

If your application compiles to a binary, you may remove the generated
`server` file and compile your application as `server` instead. I.e. for
a Go app, you'd compile with `GOOS=linux go build -o server`.

Launch is not picky about what `server` contains, as long as it is executable and
somehow starts a server.

Launch operates using the AWS SDK, and requires AWS credentials to be set up properly.
The [AWS CLI Getting Started guide](http://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html)
can help you configure your access keys. Generally speaking, if the AWS CLI works for
you, so should Launch.

Once you're set up, run `launch` to create your first deployment.

Run `launch help` for an overview and `launch help [command]` for details.

### Configuration

Running `launch init` will help you set up the required parameters.

#### Passing values to the app

Any [stage variables](http://docs.aws.amazon.com/apigateway/latest/developerguide/stage-variables.html)
defined in the API Gateway console will be passed to the application as
environment variables.

Launch sets the `environment` stage variable automatically, it
is used to call specific version aliases of the Lambda function and should not be
removed, replaced or edited.

You can add additional variables either in the API Gateway console, or via the config
file.

```yaml
variables:
  dev:
    debug: true
  prod:
    debug: false
```

Variables are environment-specific and must match the `environment` setting or `-e`
flag.

### How it works

These are roughly the steps taken by Launch when creating or updating a
deployment. All resources are checked on every run. Launch will create or
recreate them if they are missing or have been removed.

1. Lambda function.
	1. Create service role.
		1. Add inline policy allowing access to Cloudwatch Logs.
	1. Upload code.
	1. Publish version.
	1. Create or update alias named after the deployment environment, pointing
	to the newly uploaded version.
1. API Gateway.
	1. Create API.
	1. Create `/{proxy?}` resource.
	1. Create `ANY` method on `/` and `/{proxy?}` resources.
	1. Create service role.
		1. Add inline policy allowing execute access on the Lambda function.
	1. Create proxy integration on `/` and `/{proxy?}` resources.
	1. Create deployment to a stage named after the deployment environment.
1. Cloudwatch Events.
	1. Create event to invoke the function once every minute.
	
The proxy integration uses stage variables to call specific aliases of
the Lambda function. The API stage 'dev' would call the Lambda alias 'dev',
and so on.

Cloudwatch events are set up on a per-alias basis, as they each have their own
containers.
