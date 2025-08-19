package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available specs from local cache",
	RunE:  runListCommand,
}

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Print resolved raw schema",
	RunE:  runShowCommand,
}

var promptCmd = &cobra.Command{
	Use:   "prompt",
	Short: "Turn a spec into a plain-English prompt",
	RunE:  runPromptCommand,
}

var extractCmd = &cobra.Command{
	Use:   "extract",
	Short: "Extract structured JSON from unstructured input",
	RunE:  runExtractCommand,
}

var renderCmd = &cobra.Command{
	Use:   "render",
	Short: "Render unstructured input to Markdown",
	RunE:  runRenderCommand,
}

var validateCmd = &cobra.Command{
	Use:   "validate <json-file>",
	Short: "Validate JSON against a spec",
	RunE:  runValidateCommand,
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Refresh local cache from GitHub",
	RunE:  runUpdateCommand,
}

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run deterministic tests using mock provider",
	RunE:  runTestCommand,
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration",
	RunE:  runInitCommand,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("aspec %s (%s, %s)\n", version, commit, buildDate)
		os.Exit(0)
	},
}

func init() {
	// list command flags
	listCmd.Flags().Bool("json", false, "output as JSON")
	listCmd.Flags().Bool("yaml", false, "output as YAML")
	listCmd.Flags().String("search", "", "filter specs by search term")
	listCmd.Flags().String("type", "artifacts", "spec type: artifacts or extractors")

	// show command flags
	showCmd.Flags().String("spec", "", "spec slug to show")
	showCmd.Flags().String("type", "artifacts", "spec type: artifacts or extractors")
	showCmd.Flags().String("spec-url", "", "spec URL to show")
	showCmd.Flags().String("spec-path", "", "spec file path to show")
	showCmd.Flags().String("format", "json", "output format: json or yaml")
	showCmd.Flags().String("ref", "main", "git reference")

	// prompt command flags
	promptCmd.Flags().String("spec", "", "spec slug")
	promptCmd.Flags().String("type", "artifacts", "spec type: artifacts or extractors")
	promptCmd.Flags().String("spec-url", "", "spec URL")
	promptCmd.Flags().String("spec-path", "", "spec file path")
	promptCmd.Flags().String("out", "", "output file")
	promptCmd.Flags().String("model", "", "LLM model to use")
	promptCmd.Flags().Bool("stream", true, "stream output")
	promptCmd.Flags().Bool("no-stream", false, "disable streaming")

	// extract command flags
	extractCmd.Flags().String("spec", "", "spec slug")
	extractCmd.Flags().String("type", "artifacts", "spec type: artifacts or extractors")
	extractCmd.Flags().String("spec-url", "", "spec URL")
	extractCmd.Flags().String("spec-path", "", "spec file path")
	extractCmd.Flags().String("in", "", "input file or directory")
	extractCmd.Flags().String("out", "", "output file")
	extractCmd.Flags().String("model", "", "LLM model to use")
	extractCmd.Flags().Int("max-retries", 2, "maximum retry attempts")
	extractCmd.Flags().Bool("no-validate", true, "skip validation")
	extractCmd.Flags().Bool("validate", false, "enable validation (overrides --no-validate)")
	extractCmd.Flags().Bool("compact", false, "compact JSON output")
	extractCmd.Flags().Bool("stats", false, "show stats")
	extractCmd.Flags().Bool("stream", false, "stream output")
	extractCmd.Flags().Bool("no-stream", true, "disable streaming")
	extractCmd.Flags().Int("chunk-size", 20000, "maximum tokens per chunk (enables chunking for large inputs)")
	extractCmd.Flags().String("merge-strategy", "incremental", "merge strategy: incremental, two-pass, template-driven")
	extractCmd.Flags().String("merge-instructions", "", "custom instructions for merging chunks")

		// render command flags
	renderCmd.Flags().String("spec", "", "spec slug")
	renderCmd.Flags().String("type", "artifacts", "spec type: artifacts or extractors")
	renderCmd.Flags().String("spec-url", "", "spec URL")
	renderCmd.Flags().String("spec-path", "", "spec file path")
	renderCmd.Flags().String("in", "", "input file or directory")
	renderCmd.Flags().String("out", "", "output file")
	renderCmd.Flags().String("model", "", "LLM model to use")
	renderCmd.Flags().Bool("save-json", false, "save intermediate JSON")
	renderCmd.Flags().Bool("stats", false, "show stats")
	renderCmd.Flags().Bool("stream", true, "stream output")
	renderCmd.Flags().Bool("no-stream", false, "disable streaming")
	renderCmd.Flags().Bool("no-validate", true, "skip validation")
	renderCmd.Flags().Bool("validate", false, "enable validation (overrides --no-validate)")
	renderCmd.Flags().Int("chunk-size", 20000, "maximum tokens per chunk (enables chunking for large inputs)")
	renderCmd.Flags().String("merge-strategy", "incremental", "merge strategy: incremental, two-pass, template-driven")
	renderCmd.Flags().String("merge-instructions", "", "custom instructions for merging chunks")

	// validate command flags
	validateCmd.Flags().String("spec", "", "spec slug")
	validateCmd.Flags().String("type", "artifacts", "spec type: artifacts or extractors")
	validateCmd.Flags().String("spec-url", "", "spec URL")
	validateCmd.Flags().String("spec-path", "", "spec file path")

	// update command flags
	updateCmd.Flags().String("ref", "main", "git reference")
	updateCmd.Flags().String("repo", "o3-cloud/artifact-specs", "repository")

	// test command flags
	testCmd.Flags().String("spec", "", "spec slug")
	testCmd.Flags().String("fixture", "", "fixture name")
	testCmd.Flags().String("expected", "", "expected output file")
	testCmd.Flags().String("provider", "mock", "provider to use")
	testCmd.Flags().String("mock-fixture", "", "mock response fixture")
}