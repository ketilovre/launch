package launch

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	ag "github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/lambda"
)

var (
	proxyPath = "{proxy+}"
)

func GetOrCreateAPI(fn *lambda.FunctionConfiguration, conf *Config) error {
	client := ag.New(conf.Session)

	api, err := getOrCreateRestAPI(client, conf)
	if err != nil {
		return fmt.Errorf("error creating API: %v", err)
	}

	root, err := getResource(client, api, "")
	if err != nil {
		return fmt.Errorf("unable to retrieve root resource: %v", err)
	}

	proxy, err := getOrCreateProxy(client, api, conf)
	if err != nil {
		return fmt.Errorf("error creating proxy resource: %v", err)
	}

	if _, err = getOrCreateMethod(client, api, root); err != nil {
		return fmt.Errorf("error creating ANY method for root resource: %v", err)
	}

	if _, err = getOrCreateMethod(client, api, proxy); err != nil {
		return fmt.Errorf("error creating ANY method for proxy resource: %v", err)
	}

	role, err := GetOrCreateAPIRole(fn, conf)
	if err != nil {
		return fmt.Errorf("error creating AMI role for the API: %v", err)
	}

	if _, err = getOrCreateIntegration(client, api, root, fn, role, conf); err != nil {
		return fmt.Errorf("error creating integration for root resource: %v", err)
	}

	if _, err = getOrCreateIntegration(client, api, proxy, fn, role, conf); err != nil {
		return fmt.Errorf("error creating integration for proxy resource: %v", err)
	}

	return deployAPI(client, api, conf)
}

func GetInvokeUrl(conf *Config) (string, error) {
	client := ag.New(conf.Session)

	api, err := getAPI(client, conf)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("https://%v.execute-api.%v.amazonaws.com/%v", *api.Id, conf.Region, conf.Environment), nil
}

func getOrCreateRestAPI(client *ag.APIGateway, conf *Config) (*ag.RestApi, error) {
	api, err := getAPI(client, conf)
	if err != nil {
		return nil, err
	}

	if api == nil {
		fmt.Printf("Creating API Gateway named '%v'\n", apiName(conf))
		return createAPI(client, conf)
	}

	return api, nil
}

func getAPI(client *ag.APIGateway, conf *Config) (*ag.RestApi, error) {
	apis, err := client.GetRestApis(&ag.GetRestApisInput{
		Limit: aws.Int64(100),
	})
	if err != nil {
		return nil, err
	}

	for _, api := range apis.Items {
		if *api.Name == apiName(conf) {
			return api, nil
		}
	}
	return nil, nil
}

func createAPI(client *ag.APIGateway, conf *Config) (*ag.RestApi, error) {
	return client.CreateRestApi(&ag.CreateRestApiInput{
		Name:        aws.String(apiName(conf)),
		Description: aws.String(conf.Description),
	})
}

func getOrCreateProxy(client *ag.APIGateway, api *ag.RestApi, conf *Config) (*ag.Resource, error) {
	proxy, err := getResource(client, api, proxyPath)
	if err != nil {
		return nil, err
	}

	if proxy == nil {
		fmt.Printf("Creating proxy resource on '%v'\n", apiName(conf))
		return createProxy(client, api, conf)
	}

	return proxy, nil
}

func getResource(client *ag.APIGateway, api *ag.RestApi, path string) (*ag.Resource, error) {
	resources, err := client.GetResources(&ag.GetResourcesInput{
		RestApiId: api.Id,
		Limit:     aws.Int64(100),
	})
	if err != nil {
		return nil, err
	}

	for _, res := range resources.Items {
		if *res.Path == fmt.Sprintf("/%v", path) {
			return res, nil
		}
	}

	return nil, nil
}

func createProxy(client *ag.APIGateway, api *ag.RestApi, conf *Config) (*ag.Resource, error) {
	root, err := getResource(client, api, "")
	if err != nil {
		return nil, err
	}

	if root == nil {
		return nil, fmt.Errorf("the API named '%v' does not have a root resource.", apiName(conf))
	}

	return client.CreateResource(&ag.CreateResourceInput{
		RestApiId: api.Id,
		ParentId:  root.Id,
		PathPart:  &proxyPath,
	})
}

func getOrCreateMethod(client *ag.APIGateway, api *ag.RestApi, resource *ag.Resource) (*ag.Method, error) {
	method, err := getMethod(client, api, resource)
	if err != nil {
		return nil, err
	}

	if method == nil {
		fmt.Printf("Creating 'ANY' method on '%v'\n", *resource.Path)
		return createMethod(client, api, resource)
	}

	return method, nil
}

func getMethod(client *ag.APIGateway, api *ag.RestApi, proxy *ag.Resource) (*ag.Method, error) {
	method, err := client.GetMethod(&ag.GetMethodInput{
		RestApiId:  api.Id,
		ResourceId: proxy.Id,
		HttpMethod: aws.String("ANY"),
	})

	if err != nil {
		if strings.Contains(err.Error(), "NotFound") {
			return nil, nil
		}
		return nil, err
	}

	return method, nil
}

func createMethod(client *ag.APIGateway, api *ag.RestApi, proxy *ag.Resource) (*ag.Method, error) {
	return client.PutMethod(&ag.PutMethodInput{
		RestApiId:         api.Id,
		ResourceId:        proxy.Id,
		HttpMethod:        aws.String("ANY"),
		AuthorizationType: aws.String("NONE"),
	})
}

func getOrCreateIntegration(
	client *ag.APIGateway,
	api *ag.RestApi,
	resource *ag.Resource,
	fn *lambda.FunctionConfiguration,
	role *iam.Role,
	conf *Config) (*ag.Integration, error) {

	integ, err := getIntegration(api, resource, client)

	if err != nil {
		return nil, err
	}

	if integ == nil {
		fmt.Printf("Creating integration between Lambda and API on '%v'\n", *resource.Path)
		return createIntegration(client, api, resource, fn, role, conf)
	}

	return integ, nil
}

func getIntegration(api *ag.RestApi, proxy *ag.Resource, client *ag.APIGateway) (*ag.Integration, error) {
	integ, err := client.GetIntegration(&ag.GetIntegrationInput{
		HttpMethod: aws.String("ANY"),
		ResourceId: proxy.Id,
		RestApiId:  api.Id,
	})

	if err != nil {
		if strings.Contains(err.Error(), "NotFound") {
			return nil, nil
		}
		return nil, err
	}

	return integ, nil
}

func createIntegration(
	client *ag.APIGateway,
	api *ag.RestApi,
	proxy *ag.Resource,
	fn *lambda.FunctionConfiguration,
	role *iam.Role,
	conf *Config) (*ag.Integration, error) {
	return client.PutIntegration(&ag.PutIntegrationInput{
		HttpMethod:            aws.String("ANY"),
		IntegrationHttpMethod: aws.String("POST"),
		Type:        aws.String(ag.IntegrationTypeAwsProxy),
		Credentials: role.Arn,
		RestApiId:   api.Id,
		ResourceId:  proxy.Id,
		Uri:         aws.String(rewriteLambdaARN(*fn.FunctionArn, conf)),
	})
}

// API Gateway expects the function ARN in a separate format.
func rewriteLambdaARN(arn string, conf *Config) string {
	// The function ARN contains a version number at the end. We use aliases, so it needs to be removed.
	lastRelevantSegment := strings.LastIndex(arn, conf.Name)
	arn = arn[:(lastRelevantSegment + len(conf.Name))]
	return fmt.Sprintf(
		"arn:aws:apigateway:%v:lambda:path/2015-03-31/functions/%v:${stageVariables.environment}/invocations",
		conf.Region,
		arn,
	)
}

func deployAPI(client *ag.APIGateway, api *ag.RestApi, conf *Config) error {
	vars := map[string]*string{
		"environment": aws.String(conf.Environment),
	}

	customVars, defined := conf.Variables[conf.Environment]
	if defined {
		for k, v := range customVars {
			vars[k] = aws.String(v)
		}
	}

	_, err := client.CreateDeployment(&ag.CreateDeploymentInput{
		Description: aws.String(time.Now().Format(time.RFC1123Z)),
		StageName:   aws.String(conf.Environment),
		RestApiId:   api.Id,
		Variables:   vars,
	})
	return err
}

func apiName(conf *Config) string {
	return conf.Name + "-api"
}
