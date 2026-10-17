package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/resources/openrouter/cli/internal/auth"
	"github.com/vrooli/vrooli/resources/openrouter/cli/internal/config"
	"github.com/vrooli/vrooli/resources/openrouter/cli/internal/ensure"
	"github.com/vrooli/vrooli/resources/openrouter/cli/internal/health"
	"github.com/vrooli/vrooli/resources/openrouter/cli/internal/policy"
	"github.com/vrooli/vrooli/resources/openrouter/cli/internal/policycmd"

	"github.com/vrooli/cli-core/cliapp"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"

	resourceenv "github.com/vrooli/vrooli/resources/openrouter/cli/internal/env"
)

const (
	appName              = "openrouter"
	appVersion           = "0.1.0"
	defaultMaxImageBytes = 10 * 1024 * 1024
	defaultMaxImages     = 4
	envMaxImageBytes     = "OPENROUTER_GATEWAY_MAX_IMAGE_BYTES"
	envMaxImages         = "OPENROUTER_GATEWAY_MAX_IMAGES"
)

var errReexeced = errors.New("openrouter cli reexeced after rebuild")

type imageInputError struct {
	Code    string
	Message string
}

func (e *imageInputError) Error() string { return "image_input." + e.Code + ": " + e.Message }

type imageInputEnvelope struct {
	Prompt string `json:"prompt"`
	Images []struct {
		MediaType string `json:"media_type"`
		DataB64   string `json:"data_b64"`
	} `json:"images"`
}

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
	commands = append(commands, ensure.CommandGroup(ensure.NewCatalogChecker(context.Background())))
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
					Description: "Provision an OpenRouter API key through the Vrooli credential authority",
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
			Name:        "images",
			Description: "OpenRouter image generation operations",
			Subcommands: []cliapp.Command{{
				Name: "generate", Description: "Generate image output through a policy role",
				Run: func(args []string) error { return runImageGenerate(app, args, os.Stdout, os.Stdin) },
			}},
		},
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
		policycmd.Commands(nil),
	}
}

func runImageGenerate(app *cliapp.ResourceApp, args []string, stdout io.Writer, stdin io.Reader) error {
	if err := checkStale(app); err != nil {
		if errors.Is(err, errReexeced) {
			return nil
		}
		return err
	}
	fs := flag.NewFlagSet("images generate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var role, promptFlag, promptFile, inputFile string
	var count int
	fs.StringVar(&role, "role", "image.generate.default", "OpenRouter image policy role")
	fs.StringVar(&promptFlag, "prompt", "", "Image prompt")
	fs.StringVar(&promptFile, "prompt-file", "", "Read image prompt from file")
	fs.StringVar(&inputFile, "input-file", "", "Optional source image for image-to-image or instructed editing")
	fs.IntVar(&count, "output-count", 1, "Number of image outputs")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if count < 1 {
		return fmt.Errorf("output-count must be at least 1")
	}
	model, err := resolveRoleModel(role)
	if err != nil {
		return err
	}
	prompt, err := resolvePrompt(promptFlag, promptFile, fs.Args(), stdin)
	if err != nil {
		return err
	}
	runtime := resourceenv.Load()
	resolver := auth.NewResolver()
	creds, err := resolver.Resolve(context.Background())
	if err != nil {
		return err
	}
	body, err := health.GenerateImage(context.Background(), http.DefaultClient, runtime, creds, model, prompt, inputFile, count)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, string(body))
	return err
}

// resolveDefaultModel resolves the OpenRouter default role to a concrete model
// slug through policy. It is best-effort for display surfaces: a policy load
// failure returns "" rather than aborting the command.
func resolveDefaultModel(role string) string {
	p, _, err := policy.LoadDefaultFile(os.Getenv)
	if err != nil {
		return ""
	}
	resolved, err := p.ResolveRole(strings.TrimSpace(role))
	if err != nil {
		return ""
	}
	return resolved.Model
}

// resolveRoleModel resolves a role to its concrete model slug, erroring on an
// unknown role or unreadable policy. This is the authoritative path for the
// generate command.
func resolveRoleModel(role string) (string, error) {
	p, _, err := policy.LoadDefaultFile(os.Getenv)
	if err != nil {
		return "", err
	}
	resolved, err := p.ResolveRole(strings.TrimSpace(role))
	if err != nil {
		return "", err
	}
	return resolved.Model, nil
}

func resolveRolePolicy(role string) (policy.ResolvedPolicyModel, error) {
	p, _, err := policy.LoadDefaultFile(os.Getenv)
	if err != nil {
		return policy.ResolvedPolicyModel{}, err
	}
	return p.ResolveRole(strings.TrimSpace(role))
}

func resolveModelPolicy(model string) (policy.ResolvedPolicyModel, error) {
	p, _, err := policy.LoadDefaultFile(os.Getenv)
	if err != nil {
		return policy.ResolvedPolicyModel{}, err
	}
	return p.ResolveModel(strings.TrimSpace(model))
}

func resolveImageInput(stdin io.Reader, selected policy.ResolvedPolicyModel) (string, []health.ImageInput, error) {
	maxBytes := configuredPositiveInt(envMaxImageBytes, defaultMaxImageBytes)
	maxImages := configuredPositiveInt(envMaxImages, defaultMaxImages)
	limit := int64(maxBytes*2 + 64*1024)
	data, err := io.ReadAll(io.LimitReader(stdin, limit))
	if err != nil {
		return "", nil, fmt.Errorf("read input JSON: %w", err)
	}
	if int64(len(data)) >= limit {
		return "", nil, &imageInputError{Code: "request_too_large", Message: fmt.Sprintf("JSON envelope exceeds %d bytes", limit)}
	}
	var envelope imageInputEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return "", nil, &imageInputError{Code: "invalid_json", Message: err.Error()}
	}
	if strings.TrimSpace(envelope.Prompt) == "" {
		return "", nil, &imageInputError{Code: "prompt_required", Message: "prompt must not be empty"}
	}
	if len(envelope.Images) == 0 {
		return envelope.Prompt, nil, nil
	}
	if len(envelope.Images) > maxImages {
		return "", nil, &imageInputError{Code: "too_many_images", Message: fmt.Sprintf("received %d images; maximum is %d", len(envelope.Images), maxImages)}
	}
	if len(selected.Modalities.Input) > 0 && !hasModality(selected.Modalities.Input, "image") {
		return "", nil, &imageInputError{Code: "capability_mismatch", Message: fmt.Sprintf("model %q does not declare image input modality", selected.Model)}
	}
	images := make([]health.ImageInput, 0, len(envelope.Images))
	for i, image := range envelope.Images {
		mediaType := strings.ToLower(strings.TrimSpace(image.MediaType))
		if mediaType != "image/png" && mediaType != "image/jpeg" && mediaType != "image/webp" {
			return "", nil, &imageInputError{Code: "unsupported_media_type", Message: fmt.Sprintf("image %d has unsupported media_type %q", i, image.MediaType)}
		}
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(image.DataB64))
		if err != nil {
			return "", nil, &imageInputError{Code: "invalid_base64", Message: fmt.Sprintf("image %d: %v", i, err)}
		}
		if len(decoded) == 0 {
			return "", nil, &imageInputError{Code: "empty_image", Message: fmt.Sprintf("image %d is empty", i)}
		}
		if len(decoded) > maxBytes {
			return "", nil, &imageInputError{Code: "image_too_large", Message: fmt.Sprintf("image %d is %d bytes; maximum is %d", i, len(decoded), maxBytes)}
		}
		images = append(images, health.ImageInput{MediaType: mediaType, DataB64: base64.StdEncoding.EncodeToString(decoded)})
	}
	return envelope.Prompt, images, nil
}

func configuredPositiveInt(name string, fallback int) int {
	if value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(name))); err == nil && value > 0 {
		return value
	}
	return fallback
}

func hasModality(values []string, want string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), want) {
			return true
		}
	}
	return false
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
	creds, _ := resolver.Resolve(context.Background())

	body, err := fetchModels(context.Background(), http.DefaultClient, runtime, creds)
	if err != nil {
		body = []byte(`{"data":[]}`)
	}
	defaultModel := resolveDefaultModel(runtime.DefaultRole)
	response, err := config.NormalizeModelsResponse(body, defaultModel, time.Now().UTC().Format(time.RFC3339), provider, search, limit, runtime.ManualModelsFile)
	if err != nil {
		return err
	}

	if jsonOutput {
		data, err := cliout.MarshalIndent(response)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(stdout, string(data))
		return err
	}

	if len(response.Models) == 0 {
		_, err = fmt.Fprintln(stdout, firstNonEmpty(defaultModel, runtime.DefaultRole))
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
		role           string
		model          string
		promptFlag     string
		promptFile     string
		jsonOutput     bool
		temperature    float64
		maxTokens      int
		responseFormat string
		inputJSONStdin bool
	)
	fs.StringVar(&role, "role", "", "OpenRouter policy role to resolve (e.g. chat.default)")
	fs.StringVar(&model, "model", "", "Concrete OpenRouter model slug (advanced override; prefer --role)")
	fs.StringVar(&promptFlag, "prompt", "", "Prompt text to send")
	fs.StringVar(&promptFile, "prompt-file", "", "Read prompt text from a file")
	fs.BoolVar(&jsonOutput, "json", false, "Print raw OpenRouter response JSON")
	// -1 is the "unset" sentinel, mirroring resource-ollama's gateway flags. An
	// absent flag falls through to the resolved role's request_defaults, and an
	// absent default omits the parameter from the request entirely rather than
	// pinning a resource-invented value the role never asked for.
	fs.Float64Var(&temperature, "temperature", -1, "Sampling temperature; omitted when < 0 and the role declares no default")
	fs.IntVar(&maxTokens, "max-tokens", 0, "Maximum completion tokens; omitted when <= 0 and the role declares no default")
	fs.StringVar(&responseFormat, "response-format", "", "OpenAI-compatible response_format JSON object")
	fs.BoolVar(&inputJSONStdin, "input-json-stdin", false, "Read {prompt,images:[{media_type,data_b64}]} from stdin")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printGenerateUsage(stdout)
			return nil
		}
		return err
	}

	runtime := resourceenv.Load()
	// Greenfield: the model is selected by policy role, not a concrete default.
	// --model is an explicit advanced override; absent it, --role (defaulting to
	// the resource default role) is resolved through the policy authority.
	var resolvedPolicy policy.ResolvedPolicyModel
	if strings.TrimSpace(model) == "" {
		if strings.TrimSpace(role) == "" {
			role = runtime.DefaultRole
		}
		var err error
		resolvedPolicy, err = resolveRolePolicy(role)
		if err != nil {
			return err
		}
		model = resolvedPolicy.Model
	} else if resolved, err := resolveModelPolicy(model); err == nil {
		resolvedPolicy = resolved
	}
	var prompt string
	var images []health.ImageInput
	var err error
	if inputJSONStdin {
		if promptFlag != "" || promptFile != "" || len(fs.Args()) > 0 {
			return fmt.Errorf("--input-json-stdin cannot be combined with --prompt, --prompt-file, or positional prompt")
		}
		prompt, images, err = resolveImageInput(stdin, resolvedPolicy)
	} else {
		prompt, err = resolvePrompt(promptFlag, promptFile, fs.Args(), stdin)
	}
	if err != nil {
		return err
	}

	resolver := auth.NewResolver()
	creds, err := resolver.Resolve(context.Background())
	if err != nil {
		return err
	}

	var responseFormatJSON json.RawMessage
	if strings.TrimSpace(responseFormat) != "" {
		if !json.Valid([]byte(responseFormat)) {
			return fmt.Errorf("--response-format must be valid JSON")
		}
		responseFormatJSON = json.RawMessage(responseFormat)
	}
	// The role's request_defaults are the policy authority for sampling and the
	// output cap. Applying them here — not only in validation — is what makes a
	// declared per-role temperature real on the generate path.
	effectiveTemperature, effectiveMaxTokens := resolvedPolicy.RequestDefaults.ResolveGenerate(temperature, maxTokens)
	body, err := health.Generate(context.Background(), http.DefaultClient, runtime, creds, model, prompt, effectiveTemperature, effectiveMaxTokens, responseFormatJSON, images)
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
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(stdout, "Usage:\n  resource-openrouter configure\n\nThe key is read from standard input and is never accepted as an argument.")
			return nil
		}
		return err
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("configure accepts no positional arguments; provide the value on standard input")
	}
	value, err := io.ReadAll(io.LimitReader(os.Stdin, 64*1024))
	if err != nil {
		return fmt.Errorf("read OpenRouter credential: %w", err)
	}
	authority, err := credentialauthority.Default()
	if err != nil {
		return fmt.Errorf("resolve credential authority: %w", err)
	}
	client, err := credentialclient.NewClient(credentialclient.ClientOptions{Authority: authority})
	if err != nil {
		return fmt.Errorf("resolve credential client: %w", err)
	}
	if _, err := client.Provision(context.Background(), credentialclient.ProvisionRequest{Identity: "vrooli/openrouter", Field: "api-key", Value: strings.TrimSpace(string(value))}); err != nil {
		return fmt.Errorf("provision OpenRouter credential through control plane: %w", err)
	}
	_, err = fmt.Fprintln(stdout, "OpenRouter credential provisioned.")
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
	creds, _ := resolver.Resolve(context.Background())

	payload := struct {
		APIBaseURL       string `json:"api_base_url"`
		DefaultRole      string `json:"default_role"`
		DefaultRoleModel string `json:"default_role_model,omitempty"`
		ManualModels     string `json:"manual_models_file"`
		APIKeySource     string `json:"api_key_source,omitempty"`
		APIKeyPreview    string `json:"api_key_preview,omitempty"`
	}{
		APIBaseURL:       runtime.APIBaseURL,
		DefaultRole:      runtime.DefaultRole,
		DefaultRoleModel: resolveDefaultModel(runtime.DefaultRole),
		ManualModels:     runtime.ManualModelsFile,
		APIKeySource:     creds.Source,
		APIKeyPreview:    creds.RedactedAPIKey(),
	}

	if jsonOutput {
		data, err := cliout.MarshalIndent(payload)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(stdout, string(data))
		return err
	}

	_, err := fmt.Fprintf(stdout, "API Base: %s\nDefault Role: %s\nDefault Role Model: %s\nManual Models File: %s\nAPI Key Source: %s\nAPI Key Preview: %s\n",
		payload.APIBaseURL,
		payload.DefaultRole,
		firstNonEmpty(payload.DefaultRoleModel, "unresolved"),
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
	fmt.Fprintln(stdout, "  resource-openrouter generate [--role <role>] [--temperature <value>] [--max-tokens <n>] [--response-format <json>] [--json] [--prompt <text>]")
	fmt.Fprintln(stdout, "  resource-openrouter generate --model <slug> ...   (advanced override; prefer --role)")
	fmt.Fprintln(stdout, "  resource-openrouter generate <prompt text>")
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Model selection: --role is resolved through `resource-openrouter policy`; absent both --role and")
	fmt.Fprintln(stdout, "--model the resource default role is used. Prompt input can come from --prompt, --prompt-file,")
	fmt.Fprintln(stdout, "trailing args, or stdin.")
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Sampling: --temperature defaults to the -1 \"unset\" sentinel. Absent the flag the resolved role's")
	fmt.Fprintln(stdout, "request_defaults.temperature applies; absent that too, the parameter is omitted from the request so")
	fmt.Fprintln(stdout, "the upstream provider's own default applies. --max-tokens follows the same precedence with 0 as its")
	fmt.Fprintln(stdout, "sentinel. This resource never invents a sampling value the role did not declare.")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
