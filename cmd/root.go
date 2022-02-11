package cmd

import (
	"os"

	mediumautopost "github.com/askcloudarchitech/medium-auto-post/pkg/mediumAutoPost"
	"github.com/spf13/cobra"
)

var dotEnvPath string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "medium-auto-post",
	Short: "Auto post your website content to medium.com",
	Long: `
For details on how to set up your site to use this program please visit 
https://askcloudarchitech.com/posts/tutorials/auto-generate-post-payload-medium-com/
Ensure you have set up your env file as shown in the .env.example
Example command: medium-auto-post --envfilepath=.env
	`,
	RunE: func(cmd *cobra.Command, args []string) error {
		mediumautopost.Do(dotEnvPath)
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&dotEnvPath, "envfilepath", "e", "", "Path to your environment file. if left empty, the program will only use system environment variables.")
}
