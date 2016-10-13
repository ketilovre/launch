package cmd

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ketilovre/launch/lib"
	"strings"
)

var (
	conf *launch.Config

	cfgFile string
	env     string
	port    int
	region  string
)

var RootCmd = &cobra.Command{
	Use:     "launch",
	Short:   "Deploy serverless applications on AWS",
	Example: "launch\nlaunch -e prod",
	Run:     withValidConfig(launchCommand),
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "CWD/launch.yml", "path to config file")
	RootCmd.PersistentFlags().StringVarP(&env, "environment", "e", "dev", "target environment")
	RootCmd.PersistentFlags().IntVarP(&port, "port", "p", 0, "application port")
	RootCmd.PersistentFlags().StringVarP(&region, "region", "r", "", "AWS region")
}

func initConfig() {
	c := new(launch.Config)

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName("launch")
	viper.SetEnvPrefix("launch")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	viper.SetDefault("environment", "dev")

	if err := viper.ReadInConfig(); err != nil {
		if strings.Contains(err.Error(), "cannot start any token") {
			fmt.Printf("Couldn't read config file: %v\n", err)
			fmt.Println("Check that your launch.yml doesn't contain any tabs. YAML only allows spaces.")
		}
	}

	if err := viper.Unmarshal(c); err != nil {
		fmt.Printf("Malformed config: %v", err)
		os.Exit(1)
	}
	conf = c

	if env != "" {
		c.Environment = env
	}

	if port != 0 {
		c.Port = port
	}

	if region != "" {
		c.Region = region
	}
}

func withValidConfig(action func(*cobra.Command, []string)) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		errs := launch.ValidateConfig(conf)
		if errs != nil {
			for _, err := range errs {
				fmt.Printf("Config error: %v\n", err)
			}
			os.Exit(1)
		}
		action(cmd, args)
	}
}

func launchCommand(cmd *cobra.Command, args []string) {
	conf.Session = session.New(&aws.Config{
		Region: aws.String(conf.Region),
	})

	if err := launch.CheckServerFile(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fn, err := launch.CreateOrUpdateFunction(conf)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err = launch.GetOrCreateAPI(fn, conf); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err = launch.CreateOrUpdateFunctionWarmer(fn, conf); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	url, err := launch.GetInvokeUrl(conf)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("Service deployed to %v\n", url)
}
