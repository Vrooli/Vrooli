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
	"resource-gemini/cli/internal/auth"
	"resource-gemini/cli/internal/config"
	"resource-gemini/cli/internal/health"
	"strings"

	"github.com/vrooli/cli-core/cliapp"

	resourceenv "resource-gemini/cli/internal/env"
)

const (
	appName    = "gemini"
	appVersion = "0.1.0"
)

var errReexeced = errors.New("gemini cli reexeced after rebuild")

// New builds the Gemini resource CLI with explicit native operator commands on
// top of the standard resource lifecycle surface.
func New(buildFingerprint, buildTimestamp, buildSourceRoot string) (*cliapp.ResourceApp, error) {
	env := cliapp.StandardResourceEnv(appName, cliapp.ResourceEnvOptions{})
	app, err := cliapp.NewResourceApp(cliapp.ResourceOptions{
		Name:                appName,
		Version:             appVersion,
		Description:         "Gemini resource CLI",
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
	app.SetCommands(commands)
	return app, nil
}

func commandGroups(app *cliapp.ResourceApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{{
		Title: "Provider",
		Commands: []cliapp.Command{
			{
				Name:        "list-models",
				Description: "List available Gemini models",
				Run: func(args []string) error {
					return runListModels(app, args, os.Stdout)
				},
			},
			{
				Name:        "generate",
				Description: "Generate a Gemini response",
				Run: func(args []string) error {
					return runGenerate(app, args, os.Stdout, os.Stdin)
				},
			},
		},
	}}
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
	var jsonOutput bool
	fs.BoolVar(&jsonOutput, "json", false, "Print raw JSON output")
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

	creds, err := resolver.Resolve(context.Background())
	if err != nil {
		return err
	}

	models, err := health.ListModels(context.Background(), http.DefaultClient, runtime, creds)
	if err != nil {
		return err
	}

	if jsonOutput {
		payload := struct {
			Models []string `json:"models"`
		}{Models: models}
		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(stdout, string(data))
		return err
	}

	for _, model := range models {
		if _, err := fmt.Fprintln(stdout, model); err != nil {
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
		jsonOutput  bool
		temperature float64
	)
	fs.StringVar(&model, "model", "", "Gemini model to use")
	fs.StringVar(&promptFlag, "prompt", "", "Prompt text to send")
	fs.BoolVar(&jsonOutput, "json", false, "Print raw Gemini response JSON")
	fs.Float64Var(&temperature, "temperature", 0.7, "Sampling temperature")
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

	prompt, err := resolvePrompt(promptFlag, fs.Args(), stdin)
	if err != nil {
		return err
	}

	resolver := auth.NewResolver()
	resolver.CredentialsFilePath = runtime.CredentialsFile
	creds, err := resolver.Resolve(context.Background())
	if err != nil {
		return err
	}

	response, err := generate(context.Background(), http.DefaultClient, runtime, creds, model, prompt, temperature)
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

	text := strings.TrimSpace(response.PrimaryText())
	if text == "" {
		return fmt.Errorf("Gemini response did not include text output")
	}
	_, err = fmt.Fprintln(stdout, text)
	return err
}

type generateRequest struct {
	Contents []struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"contents"`
	GenerationConfig struct {
		Temperature float64 `json:"temperature"`
	} `json:"generationConfig"`
}

type generateResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (r generateResponse) PrimaryText() string {
	for _, candidate := range r.Candidates {
		for _, part := range candidate.Content.Parts {
			if trimmed := strings.TrimSpace(part.Text); trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func generate(ctx context.Context, client *http.Client, runtime resourceenv.Runtime, creds auth.Credentials, model, prompt string, temperature float64) (generateResponse, error) {
	if client == nil {
		client = http.DefaultClient
	}

	endpoint := config.GenerateContentEndpoint(runtime.APIBaseURL, model)
	if endpoint == "" {
		return generateResponse{}, fmt.Errorf("Gemini generate endpoint is required")
	}

	reqPayload := generateRequest{}
	reqPayload.Contents = []struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	}{{
		Parts: []struct {
			Text string `json:"text"`
		}{{Text: prompt}},
	}}
	reqPayload.GenerationConfig.Temperature = temperature

	body, err := json.Marshal(reqPayload)
	if err != nil {
		return generateResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"?key="+creds.APIKey, strings.NewReader(string(body)))
	if err != nil {
		return generateResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return generateResponse{}, err
	}
	defer resp.Body.Close()

	var parsed generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return generateResponse{}, err
	}
	if resp.StatusCode >= 400 {
		if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
			return generateResponse{}, fmt.Errorf("Gemini generate failed: %s", parsed.Error.Message)
		}
		return generateResponse{}, fmt.Errorf("Gemini generate failed with status %d", resp.StatusCode)
	}
	return parsed, nil
}

func resolvePrompt(promptFlag string, trailing []string, stdin io.Reader) (string, error) {
	if prompt := strings.TrimSpace(promptFlag); prompt != "" {
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
	fmt.Fprintln(stdout, "  resource-gemini list-models [--json]")
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Options:")
	fmt.Fprintln(stdout, "  --json    Print the model list as JSON")
}

func printGenerateUsage(stdout io.Writer) {
	fmt.Fprintln(stdout, "Usage:")
	fmt.Fprintln(stdout, "  resource-gemini generate [--model <name>] [--temperature <value>] [--json] [--prompt <text>]")
	fmt.Fprintln(stdout, "  resource-gemini generate <prompt text>")
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Prompt input can come from `--prompt`, trailing args, or stdin.")
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Options:")
	fmt.Fprintln(stdout, "  --model <name>          Override the default Gemini model")
	fmt.Fprintln(stdout, "  --temperature <value>   Set the generation temperature")
	fmt.Fprintln(stdout, "  --json                  Print the full Gemini response JSON")
	fmt.Fprintln(stdout, "  --prompt <text>         Provide prompt text explicitly")
}
