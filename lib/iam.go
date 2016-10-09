package launch

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	l "github.com/aws/aws-sdk-go/service/lambda"
)

func GetOrCreateLambdaRole(conf *Config) (*iam.Role, error) {
	client := iam.New(conf.Session)

	role, err := getRole(client, lambdaRoleName(conf))
	if err != nil {
		return nil, err
	}

	if role != nil {
		return role, nil
	}

	fmt.Printf("Creating service role named '%v'\n", lambdaRoleName(conf))
	return createLambdaRole(client, conf)
}

func GetOrCreateAPIRole(fn *l.FunctionConfiguration, conf *Config) (*iam.Role, error) {
	client := iam.New(conf.Session)

	role, err := getRole(client, apiRoleName(conf))
	if err != nil {
		return nil, err
	}

	if role != nil {
		return role, nil
	}

	fmt.Printf("Creating service role named '%v'\n", apiRoleName(conf))
	return createAPIRole(client, fn, conf)
}

func getRole(client *iam.IAM, roleName string) (*iam.Role, error) {
	role, err := client.GetRole(&iam.GetRoleInput{
		RoleName: aws.String(roleName),
	})
	if err != nil && !strings.Contains(err.Error(), "cannot be found") {
		return nil, err
	}

	if role == nil {
		return nil, nil
	}

	return role.Role, nil
}

func createLambdaRole(client *iam.IAM, conf *Config) (*iam.Role, error) {
	role, err := client.CreateRole(&iam.CreateRoleInput{
		RoleName: aws.String(lambdaRoleName(conf)),
		Path:     aws.String("/service-role/"),
		AssumeRolePolicyDocument: aws.String(`{
  		"Version": "2012-10-17",
  		"Statement": [
    		{
      		"Effect": "Allow",
      		"Principal": {
        		"Service": "lambda.amazonaws.com"
      		},
      		"Action": "sts:AssumeRole"
    		}
  		]
		}`),
	})

	if err != nil {
		return nil, err
	}

	_, err = client.PutRolePolicy(&iam.PutRolePolicyInput{
		PolicyName: aws.String(lambdaPolicyName(conf)),
		RoleName:   role.Role.RoleName,
		PolicyDocument: aws.String(fmt.Sprintf(`{
    	"Version": "2012-10-17",
    	"Statement": [
        {
					"Effect": "Allow",
					"Action": "logs:CreateLogGroup",
					"Resource": "arn:aws:logs:%v:*:*"
        },
        {
					"Effect": "Allow",
					"Action": [
						"logs:CreateLogStream",
						"logs:PutLogEvents"
					],
					"Resource": [
						"arn:aws:logs:%v:*:log-group:/aws/lambda/%v:*"
					]
        }
    	]
		}`, conf.Region, conf.Region, conf.Name)),
	})

	return role.Role, err
}

func createAPIRole(client *iam.IAM, fn *l.FunctionConfiguration, conf *Config) (*iam.Role, error) {
	role, err := client.CreateRole(&iam.CreateRoleInput{
		RoleName: aws.String(apiRoleName(conf)),
		Path:     aws.String("/service-role/"),
		AssumeRolePolicyDocument: aws.String(`{
  		"Version": "2012-10-17",
  		"Statement": [
    		{
      		"Effect": "Allow",
      		"Principal": {
        		"Service": "apigateway.amazonaws.com"
      		},
      		"Action": "sts:AssumeRole"
    		}
  		]
		}`),
	})

	if err != nil {
		return nil, err
	}

	_, err = client.PutRolePolicy(&iam.PutRolePolicyInput{
		RoleName:   role.Role.RoleName,
		PolicyName: aws.String(apiPolicyName(conf)),
		PolicyDocument: aws.String(fmt.Sprintf(`{
  		"Version": "2012-10-17",
  		"Statement": [
    		{
      		"Effect": "Allow",
      		"Resource": [
        		"%v:*"
      		],
      		"Action": [
        		"lambda:InvokeFunction"
      		]
    		}
  		]
		}`, *fn.FunctionArn)),
	})

	return role.Role, err
}

func lambdaRoleName(conf *Config) string {
	return conf.Name + "-lambda-role"
}

func apiRoleName(conf *Config) string {
	return conf.Name + "-api-role"
}

func lambdaPolicyName(conf *Config) string {
	return conf.Name + "-log-access"
}

func apiPolicyName(conf *Config) string {
	return conf.Name + "-lambda-invoke-access"
}
