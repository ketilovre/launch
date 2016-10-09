package cmd

import (
	"fmt"

	"os"
	"strings"

	"github.com/ketilovre/launch/lib"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Setup",
	Long: `Interactive boostrap to set up the minimum configuration
needed to deploy a service to AWS.`,
	Run: doInit,
}

func init() {
	RootCmd.AddCommand(initCmd)
}

func doInit(cmd *cobra.Command, args []string) {
	_, err := os.Open("launch.yml")
	if err != nil && strings.Contains(err.Error(), "no such file") {
		if err := launch.BootstrapConfig(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("Bootstrapped config file")
	}

	_, err = os.Open("server")
	if err != nil && strings.Contains(err.Error(), "no such file") {
		if err = launch.CreateServerFile(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("Bootstrapped server file")
	}
}
