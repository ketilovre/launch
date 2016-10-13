package launch

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lambda"
)

func CreateOrUpdateFunction(conf *Config) (*lambda.FunctionConfiguration, error) {
	var fn *lambda.FunctionConfiguration
	client := lambda.New(conf.Session)

	exists, err := getFunction(client, conf)

	if err != nil {
		return nil, err
	}

	if exists {
		fmt.Printf("Updating '%v'\n", conf.Name)
		fn, err = updateFunction(client, conf)
	} else {
		fmt.Printf("Creating '%v'\n", conf.Name)
		fn, err = createFunction(client, conf)
	}

	if err != nil {
		return nil, err
	}

	return fn, createOrUpdateAlias(client, fn, conf)
}

func addEventPermission(eventArn *string, conf *Config) error {
	client := lambda.New(conf.Session)

	client.RemovePermission(&lambda.RemovePermissionInput{
		FunctionName: aws.String(conf.Name),
		StatementId: aws.String(conf.Environment),
		Qualifier: aws.String(conf.Environment),
	})

	_, err := client.AddPermission(&lambda.AddPermissionInput{
		FunctionName: aws.String(conf.Name),
		Action: aws.String("lambda:InvokeFunction"),
		Principal: aws.String("events.amazonaws.com"),
		SourceArn: eventArn,
		StatementId: aws.String(conf.Environment),
		Qualifier: aws.String(conf.Environment),
	})
	return err
}

func getFunction(client *lambda.Lambda, conf *Config) (bool, error) {
	_, err := client.GetFunction(&lambda.GetFunctionInput{
		FunctionName: aws.String(conf.Name),
	})

	if err != nil {
		if strings.Contains(err.Error(), "NotFound") {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func updateFunction(client *lambda.Lambda, conf *Config) (*lambda.FunctionConfiguration, error) {
	bytes, err := ZipWorkingDir(conf)
	if err != nil {
		return nil, err
	}

	fmt.Println("Uploading...")
	return client.UpdateFunctionCode(&lambda.UpdateFunctionCodeInput{
		FunctionName: aws.String(conf.Name),
		Publish:      aws.Bool(true),
		ZipFile:      bytes.Bytes(),
	})
}

func createFunction(client *lambda.Lambda, conf *Config) (*lambda.FunctionConfiguration, error) {
	bytes, err := ZipWorkingDir(conf)
	if err != nil {
		return nil, err
	}

	role, err := GetOrCreateLambdaRole(conf)

	if err != nil {
		return nil, err
	}

	fmt.Println("Uploading...")
	upload := func() (*lambda.FunctionConfiguration, error) {
		return client.CreateFunction(&lambda.CreateFunctionInput{
			FunctionName: aws.String(conf.Name),
			Publish:      aws.Bool(true),
			Description:  aws.String(conf.Description),
			Handler:      aws.String("launch_shim.proxy"),
			Role:         role.Arn,
			Runtime:      aws.String("nodejs4.3"),
			Code: &lambda.FunctionCode{
				ZipFile: bytes.Bytes(),
			},
		})
	}

	fn, err := upload()

	for err != nil && strings.Contains(err.Error(), "cannot be assumed by Lambda") {
		fmt.Printf("Service role '%v' is not ready yet, retrying in 3s...\n", *role.RoleName)
		time.Sleep(time.Second * 3)
		fn, err = upload()
	}

	return fn, err
}

func createOrUpdateAlias(client *lambda.Lambda, fn *lambda.FunctionConfiguration, conf *Config) error {
	alias, err := getAlias(client, conf)
	if err != nil {
		return err
	}

	if alias == nil {
		fmt.Printf("Creating alias '%v' at version %v\n", conf.Environment, *fn.Version)
		return createAlias(client, fn, conf)
	} else {
		fmt.Printf("Updating alias '%v' to point to version %v\n", conf.Environment, *fn.Version)
		return updateAlias(client, fn, conf)
	}
}

func getAlias(client *lambda.Lambda, conf *Config) (*lambda.AliasConfiguration, error) {
	alias, err := client.GetAlias(&lambda.GetAliasInput{
		Name:         aws.String(conf.Environment),
		FunctionName: aws.String(conf.Name),
	})

	if err != nil && strings.Contains(err.Error(), "NotFound") {
		return nil, nil
	}

	return alias, err
}

func updateAlias(client *lambda.Lambda, fn *lambda.FunctionConfiguration, conf *Config) error {
	_, err := client.UpdateAlias(&lambda.UpdateAliasInput{
		Name:            aws.String(conf.Environment),
		FunctionName:    aws.String(conf.Name),
		FunctionVersion: fn.Version,
	})
	return err
}

func createAlias(client *lambda.Lambda, fn *lambda.FunctionConfiguration, conf *Config) error {
	_, err := client.CreateAlias(&lambda.CreateAliasInput{
		Name:            aws.String(conf.Environment),
		FunctionName:    aws.String(conf.Name),
		FunctionVersion: fn.Version,
	})
	return err
}

// lambdaRootARN returns the ARN of the lambda function without any trailing version number.
func lambdaRootARN(arn string, conf *Config) string {
	lastRelevantSegment := strings.LastIndex(arn, conf.Name)
	return arn[:(lastRelevantSegment + len(conf.Name))]
}
