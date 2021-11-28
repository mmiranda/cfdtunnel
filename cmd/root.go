package cmd

import (
	"fmt"

	"github.com/mmiranda/cfdtunnel/cfdtunnel"
	"github.com/spf13/cobra"
	goVersion "go.hein.dev/go-version"
)

var (
	profile, output string
	debug           bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cfdtunnel",
	Short: "Manage multiple cloudflared clients for you",
	Long: `cfdtunnel creates your cloudflare tunnel clients
on the fly only when you need to use them.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfdtunnel := cfdtunnel.Arguments{
			Profile: profile,
			Command: args[0],
			Args:    args[1:],
		}

		cfdtunnel.Execute()
	},
}

var (
	shortened     = false
	version       = "dev"
	commit        = "none"
	date          = "unknown"
	versionOutput = "json"
	versionCmd    = &cobra.Command{
		Use:   "version",
		Short: "Version will output the current build information",
		Long:  ``,
		Run: func(_ *cobra.Command, _ []string) {
			resp := goVersion.FuncWithOutput(shortened, version, commit, date, versionOutput)
			fmt.Print(resp)
		},
	}
)

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	versionCmd.Flags().BoolVarP(&shortened, "short", "s", false, "Print just the version number.")
	versionCmd.Flags().StringVarP(&output, "output", "o", "json", "Output format. One of 'yaml' or 'json'.")
	rootCmd.AddCommand(versionCmd)
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.PersistentFlags().StringVar(&profile, "profile", "", "Which cfdtunnel profile to use")
	_ = rootCmd.MarkPersistentFlagRequired("profile")
	rootCmd.PersistentFlags().BoolVarP(&debug, "short", "d", false, "Enable Debug Mode.")

}
