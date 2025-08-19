package config

import (
    "bytes"
    "fmt"
    "os"
    "path/filepath"
    "text/template"

    "github.com/spf13/viper"
    "gopkg.in/yaml.v3"
    "github.com/o3-cloud/artifact-specs/cli/internal/logging"
)

type Config struct {
	Model           string  `yaml:"model" mapstructure:"model"`
	BaseURL         string  `yaml:"base_url" mapstructure:"base_url"`
	Provider        string  `yaml:"provider" mapstructure:"provider"`
	SystemPromptFile string `yaml:"system_prompt_file" mapstructure:"system_prompt_file"`
	Prompts         Prompts `yaml:"prompts" mapstructure:"prompts"`
}

type Prompts struct {
	Extraction     string `yaml:"extraction" mapstructure:"extraction"`
	Verbalization  string `yaml:"verbalization" mapstructure:"verbalization"`
}

var (
	defaultConfig = Config{
		Model:            "openai/gpt-4o-mini",
		BaseURL:          "https://openrouter.ai/api/v1",
		Provider:         "openrouter",
		SystemPromptFile: "",
		Prompts: Prompts{
			Extraction: `Extract structured data from the following input according to the "{{.SchemaTitle}}" specification.

Instructions:
- Extract only information that is explicitly present in the input
- Do not invent or infer information not directly stated
- Leave fields empty/null if the information is not available
- Follow the JSON schema structure exactly
- Ensure all required fields are present

Input:
{{.Input}}

JSON Schema Reference:
{{.Schema}}

Provide the extracted data as valid JSON:`,
			Verbalization: `Convert the following structured JSON data into clear, readable Markdown format.

Requirements:
- Use appropriate Markdown formatting (headers, lists, tables, etc.)
- Present the information in a logical, easy-to-read structure
- Include only the information present in the JSON data
- Do not add commentary or extra headings beyond what the data implies
- Maintain accuracy to the original data

JSON Data:
{{.JSONData}}

Generate well-formatted Markdown:`,
		},
	}
	
	globalConfig *Config
)

func Initialize() error {
	configDir, err := getConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}
	
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configDir)

	// Set defaults
	viper.SetDefault("model", defaultConfig.Model)
	viper.SetDefault("base_url", defaultConfig.BaseURL)
	viper.SetDefault("provider", defaultConfig.Provider)
	viper.SetDefault("system_prompt_file", defaultConfig.SystemPromptFile)
	viper.SetDefault("prompts.extraction", defaultConfig.Prompts.Extraction)
	viper.SetDefault("prompts.verbalization", defaultConfig.Prompts.Verbalization)

	// Environment variable bindings
	viper.SetEnvPrefix("ASPEC")
	viper.AutomaticEnv()
	
	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Unmarshal config
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	globalConfig = &config
	return nil
}

func Get() *Config {
	if globalConfig == nil {
		// Return defaults if not initialized
		return &defaultConfig
	}
	return globalConfig
}

func GetAPIKey() string {
	return os.Getenv("OPENROUTER_API_KEY")
}

func CreateDefaultConfig() error {
	configDir, err := getConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	
	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("config file already exists at %s", configPath)
	}

	// Write default config
	configData, err := yaml.Marshal(defaultConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal default config: %w", err)
	}

    if err := os.WriteFile(configPath, configData, 0644); err != nil {
        return fmt.Errorf("failed to write config file: %w", err)
    }

    logging.Info("Created default config", map[string]interface{}{"path": configPath})
    return nil
}

func getConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".artifactspecs"), nil
}

func GetCacheDir() (string, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "specs"), nil
}

// RenderExtractionPrompt renders the extraction prompt template with the given data
func RenderExtractionPrompt(schemaTitle, input, schema string) (string, error) {
	config := Get()
	tmpl, err := template.New("extraction").Parse(config.Prompts.Extraction)
	if err != nil {
		return "", fmt.Errorf("failed to parse extraction prompt template: %w", err)
	}
	
	data := struct {
		SchemaTitle string
		Input       string
		Schema      string
	}{
		SchemaTitle: schemaTitle,
		Input:       input,
		Schema:      schema,
	}
	
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute extraction prompt template: %w", err)
	}
	
	return buf.String(), nil
}

// RenderVerbalizationPrompt renders the verbalization prompt template with the given data
func RenderVerbalizationPrompt(jsonData string) (string, error) {
	config := Get()
	tmpl, err := template.New("verbalization").Parse(config.Prompts.Verbalization)
	if err != nil {
		return "", fmt.Errorf("failed to parse verbalization prompt template: %w", err)
	}
	
	data := struct {
		JSONData string
	}{
		JSONData: jsonData,
	}
	
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute verbalization prompt template: %w", err)
	}
	
	return buf.String(), nil
}
