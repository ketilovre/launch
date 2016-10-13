package launch

import (
	"github.com/aws/aws-sdk-go/service/lambda"
	cwe "github.com/aws/aws-sdk-go/service/cloudwatchevents"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
)

func CreateOrUpdateFunctionWarmer(fn *lambda.FunctionConfiguration, conf *Config) error {
	client := cwe.New(conf.Session)

	arn, err := createRule(client, conf)
	if err != nil {
		return fmt.Errorf("unable to create cloudwatch event: %v", err)
	}

	err = addEventPermission(arn, conf)
	if err != nil {
		return fmt.Errorf("unable to give cloudwatch events access to lambda: %v", err)
	}

	err = addTarget(client, fn, conf)
	if err != nil {
		return fmt.Errorf("unable to add lambda as event target: %v", err)
	}

	return nil
}

func createRule(client *cwe.CloudWatchEvents, conf *Config) (*string, error) {
	rule, err := client.PutRule(&cwe.PutRuleInput{
		Name: aws.String(ruleName(conf)),
		ScheduleExpression: aws.String("rate(1 minute)"),
	})

	if err != nil {
		return nil, err
	}
	return rule.RuleArn, err
}

func addTarget(client *cwe.CloudWatchEvents, fn *lambda.FunctionConfiguration, conf *Config) error {
	_, err := client.PutTargets(&cwe.PutTargetsInput{
		Rule: aws.String(ruleName(conf)),
		Targets: []*cwe.Target{
			{
				Id: aws.String(conf.Environment),
				Arn: aws.String(fmt.Sprintf("%v:%v", lambdaRootARN(*fn.FunctionArn, conf), conf.Environment)),
				Input: aws.String(input(conf)),
			},
		},
	})
	return err
}

func ruleName(conf *Config) string {
	return fmt.Sprintf("%v-%v-warmer", conf.Name, conf.Environment)
}

func input(conf *Config) string {
	return fmt.Sprintf(`{
		"resource": "/{proxy+}",
		"path": "/",
		"httpMethod": "GET",
		"stageVariables": {
			"environment": "%v"
		}
	}`, conf.Environment)
}
