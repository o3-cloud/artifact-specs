package main

import (
    "os"

    "github.com/o3-cloud/artifact-specs/cli/internal/config"
    "github.com/o3-cloud/artifact-specs/cli/internal/logging"
    "github.com/spf13/cobra"
)

var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

func main() {
    if err := rootCmd.Execute(); err != nil {
        logging.Error("Command execution failed", map[string]interface{}{"error": err.Error()})
        os.Exit(1)
    }
}

var rootCmd = &cobra.Command{
	Use:   "aspec",
	Short: "Transform artifact/extractor JSON Schemas into prompts, extractions, and renderings",
	Long: `aspec is a CLI tool that transforms JSON Schemas from artifact-specs into:
- Plain-English prompts
- Validated JSON extraction from unstructured input
- Plain-English Markdown rendering`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize configuration first
		if err := config.Initialize(); err != nil {
			return err
		}
		
		// Configure logging based on flags
		quiet, _ := cmd.Flags().GetBool("quiet")
		logJSON, _ := cmd.Flags().GetBool("log-json")
		verboseCount, _ := cmd.Flags().GetCount("verbose")
		
		if quiet {
			logging.SetQuiet()
		} else if verboseCount > 0 {
			logging.SetVerbose(verboseCount)
		}
		
		if logJSON {
			logging.SetJSONMode(true)
		}
		
		return nil
	},
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().CountP("verbose", "v", "verbose output (repeatable: -v, -vv, -vvv)")
	rootCmd.PersistentFlags().Bool("quiet", false, "quiet mode")
	rootCmd.PersistentFlags().Bool("log-json", false, "output logs as JSON")
	
	// Add all commands
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(promptCmd)
	rootCmd.AddCommand(extractCmd)
	rootCmd.AddCommand(renderCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(testCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(versionCmd)
}
