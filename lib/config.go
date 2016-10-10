package launch

import (
	"os"

	"bufio"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"gopkg.in/yaml.v2"
	"strconv"
	"strings"
)

type Config struct {
	Session     *session.Session `yaml:"-"`
	Name        string
	Description string
	Region      string
	Environment string `yaml:"default-environment"`
	Port        int
	Variables   map[string]map[string]string
}

func BootstrapConfig() error {
	conf := new(Config)
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("Fill in the values to bootstrap your launch.yml")

	fmt.Println("\nThe app name determines the name of all AWS resources created by launch.")
	fmt.Print("App name: ")
	scanner.Scan()
	conf.Name = scanner.Text()

	fmt.Println("\nOptional app description.")
	fmt.Print("Description: ")
	scanner.Scan()
	conf.Description = scanner.Text()
	if conf.Description == "" {
		conf.Description = "No description"
	}

	fmt.Println("\nThe environment determines the name of the API Gateway stage and the Lambda version alias.")
	fmt.Print("Default environment: ")
	scanner.Scan()
	conf.Environment = scanner.Text()

	fmt.Println("\nRegion name in the standard format, e.g. 'us-east-1' or 'eu-central-1'")
	fmt.Print("AWS Region: ")
	scanner.Scan()
	conf.Region = scanner.Text()

	fmt.Println("\nThe port your application binds to.")
	fmt.Print("Application port: ")
	scanner.Scan()
	portStr := scanner.Text()
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("'%v' is not a number: %v", portStr, err)
	}
	if port > 65535 {
		return fmt.Errorf("%v is too large.", port)
	}
	conf.Port = port

	bytes, err := yaml.Marshal(conf)
	if err != nil {
		return fmt.Errorf("couldn't convert config to YAML: %v", err)
	}

	file, err := os.Create("launch.yml")
	if err != nil {
		return fmt.Errorf("couldn't create launch.yml in the current dir: %v", err)
	}

	_, err = file.Write(bytes)
	if err != nil {
		return fmt.Errorf("couldn't write to launch.yml: %v", err)
	}

	return nil
}

func ValidateConfig(conf *Config) []error {
	var errs []error
	if conf.Name == "" {
		errs = append(errs, errors.New("'name' is empty"))
	}
	if conf.Region == "" {
		errs = append(errs, errors.New("'region' is empty"))
	}
	if conf.Port == 0 {
		errs = append(errs, errors.New("'port' is empty"))
	}
	if strings.Contains(conf.Environment, " ") {
		errs = append(errs, errors.New("'environment' cannot contain spaces"))
	}
	return errs
}
