package cmd

import (
	"fmt"

	"os"

	"github.com/ketilovre/launch/lib"
	"github.com/spf13/cobra"
)

var zipCmd = &cobra.Command{
	Use:   "zip",
	Short: "Package the application and write to disk",
	Long: `
The zip command creates a package as it would have been deployed to Lambda, including
the JS-shim, and writes it to disk.`,
	Run: withValidConfig(func(cmd *cobra.Command, args []string) {
		if err := launch.WriteZipToFile(cmd.Flag("out").Value.String(), conf); err != nil {
			fmt.Printf("Unable to write zip file: %v\n", err)
			os.Exit(1)
		}
	}),
}

func init() {
	RootCmd.AddCommand(zipCmd)

	zipCmd.Flags().StringP("out", "o", "launch", "File name, without extension")
}
