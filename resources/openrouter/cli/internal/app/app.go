package app

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"

	"resource-openrouter/cli/internal/auth"
	"resource-openrouter/cli/internal/config"
	resourceenv "resource-openrouter/cli/internal/env"
	"resource-openrouter/cli/internal/health"
)

const (
	appName    = "openrouter"
	appVersion = "0.1.0"
)

var errReexeced = errors.New("openrouter cli reexeced after rebuild")

// New builds the OpenRouter resource CLI with explicit native operator commands.
func New(buildFingerprint, buildTimestamp, buildSourceRoot string) (*cliapp.ResourceApp, error) {
	env := cliapp.StandardResourceEnv(appName, cliapp.ResourceEnvOptions{})
	app, err := cliapp.NewResourceApp(cliapp.ResourceOptions{
		Name:                appName,
		Version:             appVersion,
		Description:         "OpenRouter resource CLI",
		SourceRootEnvVars:   env.SourceRootEnvVars,
		ControlPlaneEnvVars: env.ControlPlaneEnvVars,
		BuildFingerprint:    buildFingerprint,
		BuildTimestamp:      buildTimestamp,
		BuildSourceRoot:     buildSourceRoot,
	})
	if err != nil {
		return nil, err
	}

	commands := append(app.StandardLifecycleCommands(), commandGroups(app)...)
	app.SetCommandsWithSubgroups(commands, commandSubgroups(app))
	return app, nil
}

func commandGroups(app *cliapp.ResourceApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		{
			Title: "Provider",
			Commands: []cliapp.Command{
				{
					Name:        "list-models",
					Description: "List available OpenRouter models",
					Run: func(args []string) error {
						return runListModels(app, args, os.Stdout)
					},
				},
				{
					Name:        "generate",
					Description: "Generate an OpenRouter response",
					Run: func(args []string) error {
						return runGenerate(app, args, os.Stdout, os.Stdin)
					},
				},
				{
					Name:        "configure",
					Description: "Store an OpenRouter API key in the native credentials file",
					Run: func(args []string) error {
						return runConfigure(app, args, os.Stdout)
					},
				},
				{
					Name:        "show-config",
					Description: "Show resolved OpenRouter runtime and auth configuration",
					Run: func(args []string) error {
						return runShowConfig(app, args, os.Stdout)
					},
				},
			},
		},
	}
}

func commandSubgroups(app *cliapp.ResourceApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		{
			Name:        "content",
			Description: "OpenRouter content operations",
			Subcommands: []cliapp.Command{
				{
					Name:        "models",
					Description: "List normalized OpenRouter models",
					Run: func(args []string) error {
						return runListModels(app, args, os.Stdout)
					},
				},
			},
		},
	}
}

func runListModels(app *cliapp.ResourceApp, args []string, stdout io.Writer) error {
	if err := checkStale(app); err != nil {
		if errors.Is(err, errReexeced) {
			return nil
		}
		return err
	}

	fs := flag.NewFlagSet("list-models", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var (
		jsonOutput bool
		provider   string
		search     string
		limit      int
	)
	fs.BoolVar(&jsonOutput, "json", false, "Print structured JSON output")
	fs.StringVar(&provider, "provider", "", "Filter by provider prefix")
	fs.StringVar(&search, "search", "", "Filter by model id, name, or description")
	fs.IntVar(&limit, "limit", 0, "Limit number of returned models")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printListModelsUsage(stdout)
			return nil
		}
		return err
	}

	runtime := resourceenv.Load()
	resolver := auth.NewResolver()
	resolver.CredentialsFilePath = runtime.CredentialsFile
	creds, _ := resolver.Resolve(context.Background())

	body, err := fetchModels(context.Background(), http.DefaultClient, runtime, creds)
	if err != nil {
		body = []byte(`{"data":[]}`)
	}
	response, err := config.NormalizeModelsResponse(body, runtime.DefaultModel, time.Now().UTC().Format(time.RFC3339), provider, search, limit, runtime.ManualModelsFile)
	if err != nil {
		return err
	}

	if jsonOutput {
		data, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(stdout, string(data))
		return err
	}

	if len(response.Models) == 0 {
		_, err = fmt.Fprintln(stdout, runtime.DefaultModel)
		return err
	}
	for _, model := range response.Models {
		if _, err := fmt.Fprintln(stdout, model.ID); err != nil {
			return err
		}
	}
	return nil
}

func runGenerate(app *cliapp.ResourceApp, args []string, stdout io.Writer, stdin io.Reader) error {
	if err := checkStale(app); err != nil {
		if errors.Is(err, errReexeced) {
			return nil
		}
		return err
	}

	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var (
		model       string
		promptFlag  string
		promptFile  string
		jsonOutput  bool
		temperature float64
		maxTokens   int
	)
	fs.StringVar(&model, "model", "", "OpenRouter model to use")
	fs.StringVar(&promptFlag, "prompt", "", "Prompt text to send")
	fs.StringVar(&promptFile, "prompt-file", "", "Read prompt text from a file")
	fs.BoolVar(&jsonOutput, "json", false, "Print raw OpenRouter response JSON")
	fs.Float64Var(&temperature, "temperature", 0.7, "Sampling temperature")
	fs.IntVar(&maxTokens, "max-tokens", 0, "Maximum completion tokens")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printGenerateUsage(stdout)
			return nil
		}
		return err
	}

	runtime := resourceenv.Load()
	if strings.TrimSpace(model) == "" {
		model = runtime.DefaultModel
	}
	prompt, err := resolvePrompt(promptFlag, promptFile, fs.Args(), stdin)
	if err != nil {
		return err
	}

	resolver := auth.NewResolver()
	resolver.CredentialsFilePath = runtime.CredentialsFile
	creds, err := resolver.Resolve(context.Background())
	if err != nil {
		return err
	}

	body, err := health.Generate(context.Background(), http.DefaultClient, runtime, creds, model, prompt, temperature, maxTokens)
	if err != nil {
		return err
	}
	if jsonOutput {
		_, err = fmt.Fprintln(stdout, string(body))
		return err
	}

	text, err := extractPrimaryText(body)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, text)
	return err
}

func runConfigure(app *cliapp.ResourceApp, args []string, stdout io.Writer) error {
	if err := checkStale(app); err != nil {
		if errors.Is(err, errReexeced) {
			return nil
		}
		return err
	}

	fs := flag.NewFlagSet("configure", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var apiKey string
	fs.StringVar(&apiKey, "api-key", "", "OpenRouter API key to store")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(stdout, "Usage:\n  resource-openrouter configure --api-key <key>")
			return nil
		}
		return err
	}
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return fmt.Errorf("api key is required")
	}

	runtime := resourceenv.Load()
	if err := runtime.EnsureDirectories(); err != nil {
		return err
	}
	if err := config.SaveCredentialsFile(runtime.CredentialsFile, apiKey); err != nil {
		return err
	}
	_, err := fmt.Fprintf(stdout, "Stored OpenRouter API key in %s\n", runtime.CredentialsFile)
	return err
}

func runShowConfig(app *cliapp.ResourceApp, args []string, stdout io.Writer) error {
	if err := checkStale(app); err != nil {
		if errors.Is(err, errReexeced) {
			return nil
		}
		return err
	}

	fs := flag.NewFlagSet("show-config", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var jsonOutput bool
	fs.BoolVar(&jsonOutput, "json", false, "Print structured JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(stdout, "Usage:\n  resource-openrouter show-config [--json]")
			return nil
		}
		return err
	}

	runtime := resourceenv.Load()
	resolver := auth.NewResolver()
	resolver.CredentialsFilePath = runtime.CredentialsFile
	creds, _ := resolver.Resolve(context.Background())

	payload := struct {
		APIBaseURL      string `json:"api_base_url"`
		DefaultModel    string `json:"default_model"`
		CredentialsFile string `json:"credentials_file"`
		ManualModels    string `json:"manual_models_file"`
		APIKeySource    string `json:"api_key_source,omitempty"`
		APIKeyPreview   string `json:"api_key_preview,omitempty"`
	}{
		APIBaseURL:      runtime.APIBaseURL,
		DefaultModel:    runtime.DefaultModel,
		CredentialsFile: runtime.CredentialsFile,
		ManualModels:    runtime.ManualModelsFile,
		APIKeySource:    creds.Source,
		APIKeyPreview:   creds.RedactedAPIKey(),
	}

	if jsonOutput {
		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(stdout, string(data))
		return err
	}

	_, err := fmt.Fprintf(stdout, "API Base: %s\nDefault Model: %s\nCredentials File: %s\nManual Models File: %s\nAPI Key Source: %s\nAPI Key Preview: %s\n",
		payload.APIBaseURL,
		payload.DefaultModel,
		payload.CredentialsFile,
		payload.ManualModels,
		firstNonEmpty(payload.APIKeySource, "not configured"),
		firstNonEmpty(payload.APIKeyPreview, "not configured"),
	)
	return err
}

func fetchModels(ctx context.Context, client *http.Client, runtime resourceenv.Runtime, creds auth.Credentials) ([]byte, error) {
	endpoint := config.ModelsEndpoint(runtime.APIBaseURL)
	if endpoint == "" {
		return nil, fmt.Errorf("OpenRouter models endpoint is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if creds.Valid() {
		req.Header.Set("Authorization", "Bearer "+creds.APIKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("OpenRouter model fetch failed with status %d", resp.StatusCode)
	}
	return body, nil
}

func extractPrimaryText(body []byte) (string, error) {
	var payload struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	for _, choice := range payload.Choices {
		if content := strings.TrimSpace(choice.Message.Content); content != "" {
			return content, nil
		}
	}
	return "", fmt.Errorf("OpenRouter response did not include message content")
}

func resolvePrompt(promptFlag, promptFile string, trailing []string, stdin io.Reader) (string, error) {
	if prompt := strings.TrimSpace(promptFlag); prompt != "" {
		return prompt, nil
	}
	if path := strings.TrimSpace(promptFile); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		prompt := strings.TrimSpace(string(data))
		if prompt == "" {
			return "", fmt.Errorf("prompt file is empty: %s", path)
		}
		return prompt, nil
	}
	if len(trailing) > 0 {
		return strings.TrimSpace(strings.Join(trailing, " ")), nil
	}
	data, err := io.ReadAll(stdin)
	if err != nil {
		return "", err
	}
	prompt := strings.TrimSpace(string(data))
	if prompt == "" {
		return "", fmt.Errorf("prompt is required; pass text as an argument, --prompt, or stdin")
	}
	return prompt, nil
}

func checkStale(app *cliapp.ResourceApp) error {
	if app == nil || app.StaleChecker == nil {
		return nil
	}
	app.StaleChecker.ReexecArgs = append([]string(nil), os.Args[1:]...)
	if restarted := app.StaleChecker.CheckAndMaybeRebuild(); restarted {
		return errReexeced
	}
	return nil
}

func printListModelsUsage(stdout io.Writer) {
	fmt.Fprintln(stdout, "Usage:")
	fmt.Fprintln(stdout, "  resource-openrouter list-models [--provider <id>] [--search <term>] [--limit <n>] [--json]")
	fmt.Fprintln(stdout, "  resource-openrouter content models [--provider <id>] [--search <term>] [--limit <n>] [--json]")
}

func printGenerateUsage(stdout io.Writer) {
	fmt.Fprintln(stdout, "Usage:")
	fmt.Fprintln(stdout, "  resource-openrouter generate [--model <name>] [--temperature <value>] [--max-tokens <n>] [--json] [--prompt <text>]")
	fmt.Fprintln(stdout, "  resource-openrouter generate [--model <name>] [--temperature <value>] [--max-tokens <n>] [--json] --prompt-file <path>")
	fmt.Fprintln(stdout, "  resource-openrouter generate <prompt text>")
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Prompt input can come from `--prompt`, `--prompt-file`, trailing args, or stdin.")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
