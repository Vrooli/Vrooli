// Package aiproviderapp builds the CLI surface shared by OpenAI-compatible AI
// provider resources. Each resource supplies only its identity, endpoint,
// credential environment, and policy path conventions.
package aiproviderapp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vrooli/ai-provider-go/ensure"
	"github.com/vrooli/ai-provider-go/openai"
	"github.com/vrooli/ai-provider-go/policy"
	"github.com/vrooli/ai-provider-go/policycmd"
	"github.com/vrooli/ai-provider-go/providererror"

	"github.com/vrooli/cli-core/cliapp"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
)

// Config describes one provider resource's CLI identity and conventions.
type Config struct {
	Name               string
	DisplayName        string
	Version            string
	APIBaseEnv         string
	APIBaseDefault     string
	APIKeyEnv          string
	DefaultRoleEnv     string
	DefaultRole        string
	CredentialIdentity string
	CredentialField    string
	PolicyEnv          string
	PolicySourceEnv    string
}

type runtime struct {
	APIBase     string
	APIKey      string
	DefaultRole string
}

// New builds the provider resource CLI.
func New(config Config, buildFingerprint, buildTimestamp, buildSourceRoot string) (*cliapp.ResourceApp, error) {
	if strings.TrimSpace(config.Name) == "" {
		return nil, errors.New("provider name is required")
	}
	if strings.TrimSpace(config.Version) == "" {
		config.Version = "0.1.0"
	}
	if strings.TrimSpace(config.CredentialField) == "" {
		config.CredentialField = "api-key"
	}
	env := cliapp.StandardResourceEnv(config.Name, cliapp.ResourceEnvOptions{})
	app, err := cliapp.NewResourceApp(cliapp.ResourceOptions{
		Name:                config.Name,
		Version:             config.Version,
		Description:         config.DisplayName + " resource CLI",
		SourceRootEnvVars:   env.SourceRootEnvVars,
		ControlPlaneEnvVars: env.ControlPlaneEnvVars,
		BuildFingerprint:    buildFingerprint,
		BuildTimestamp:      buildTimestamp,
		BuildSourceRoot:     buildSourceRoot,
	})
	if err != nil {
		return nil, err
	}

	commands := append(app.StandardLifecycleCommands(), providerCommands(config)...)
	app.SetCommandsWithSubgroups(commands, providerSubgroups(config))
	return app, nil
}

func providerCommands(config Config) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		{
			Title: "Provider",
			Commands: []cliapp.Command{
				{Name: "generate", Description: "Generate a response through a policy role", Run: func(args []string) error { return runGenerate(config, args, os.Stdout, os.Stdin) }},
				{Name: "list-models", Description: "List available provider models", Run: func(args []string) error { return runListModels(config, args, os.Stdout) }},
				{Name: "show-config", Description: "Show resolved provider runtime configuration", Run: func(args []string) error { return runShowConfig(config, args, os.Stdout) }},
				{Name: "configure", Description: "Provision the provider API key through the credential authority", Run: func(args []string) error { return runConfigure(config, args, os.Stdout) }},
				{Name: "ensure", Description: "Validate scenario-declared provider roles", Run: func(args []string) error { return runEnsure(config, args, os.Stdout) }},
			},
		},
	}
}

func providerSubgroups(config Config) []cliapp.SubcommandGroup {
	handlers := &policycmd.Handlers{
		ResourceName: config.Name,
		Load:         func() (policy.Catalog, string, error) { return loadCatalog(config) },
		Stdout:       os.Stdout,
		Stderr:       os.Stderr,
	}
	return []cliapp.SubcommandGroup{
		{
			Name:        "policy",
			Description: "Inspect authoritative provider role and model policy metadata",
			Subcommands: []cliapp.Command{
				{Name: "resolve", Description: "Resolve one role or model", Usage: "policy resolve (--role <role> | --model <model>) [--json] [--field <name>]", Run: handlers.Resolve},
				{Name: "roles", Description: "List roles with resolved model metadata", Run: handlers.Roles},
				{Name: "models", Description: "List catalog models and metadata", Run: handlers.Models},
				{Name: "constraints", Description: "Show policy constraints", Run: handlers.Constraints},
			},
		},
	}
}

func loadCatalog(config Config) (policy.Catalog, string, error) {
	return policy.LoadDefaultFile(os.Getenv, policy.PathOptions{
		ProviderName: config.Name,
		PolicyEnv:    config.PolicyEnv,
		SourceEnv:    config.PolicySourceEnv,
		Validation:   policy.CloudOptions(),
	})
}

func loadRuntime(config Config) runtime {
	base := strings.TrimSpace(os.Getenv(config.APIBaseEnv))
	if base == "" {
		base = config.APIBaseDefault
	}
	defaultRole := strings.TrimSpace(os.Getenv(config.DefaultRoleEnv))
	if defaultRole == "" {
		defaultRole = config.DefaultRole
	}
	return runtime{
		APIBase:     strings.TrimRight(base, "/"),
		APIKey:      strings.TrimSpace(os.Getenv(config.APIKeyEnv)),
		DefaultRole: defaultRole,
	}
}

func newClient(config Config) *openai.Client {
	rt := loadRuntime(config)
	return openai.New(openai.Config{
		ProviderName: config.Name,
		BaseURL:      rt.APIBase,
		APIKey:       rt.APIKey,
		UserAgent:    "vrooli-resource-" + config.Name,
	})
}

func runGenerate(config Config, args []string, stdout io.Writer, stdin io.Reader) error {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	role := fs.String("role", "", "Policy role to resolve")
	model := fs.String("model", "", "Explicit model override (advanced; prefer --role)")
	prompt := fs.String("prompt", "", "Prompt text")
	promptFile := fs.String("prompt-file", "", "Read the prompt from a file")
	inputJSON := fs.Bool("input-json-stdin", false, "Read a multimodal JSON envelope from stdin")
	maxTokens := fs.Int("max-tokens", 0, "Maximum output tokens (0 uses the role default)")
	temperature := fs.Float64("temperature", -1, "Sampling temperature (-1 uses the role default)")
	responseFormat := fs.String("response-format", "", "Raw JSON response_format value")
	asJSON := fs.Bool("json", true, "Emit the provider response as JSON")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintf(stdout, "Usage: resource-%s generate [--role <role>] [--prompt <text>] [--json]\n", config.Name)
			return nil
		}
		return err
	}

	if strings.TrimSpace(loadRuntime(config).APIKey) == "" {
		return &providererror.Error{Code: providererror.CodeUnconfigured, Message: fmt.Sprintf("%s credential is not configured; provision %s:%s", config.DisplayName, config.CredentialIdentity, config.CredentialField)}
	}
	catalog, _, err := loadCatalog(config)
	if err != nil {
		return err
	}
	selected, err := resolveSelection(catalog, *role, *model, loadRuntime(config).DefaultRole)
	if err != nil {
		return err
	}
	if *maxTokens <= 0 {
		*maxTokens = 0
	}
	resolvedTemperature, resolvedMaxTokens := selected.RequestDefaults.ResolveGenerate(*temperature, *maxTokens)

	textPrompt := strings.TrimSpace(*prompt)
	var images []openai.InlineImage
	if *inputJSON {
		raw, err := io.ReadAll(io.LimitReader(stdin, 32<<20))
		if err != nil {
			return fmt.Errorf("read input JSON: %w", err)
		}
		var envelope openai.MultimodalEnvelope
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return fmt.Errorf("decode input JSON: %w", err)
		}
		textPrompt = strings.TrimSpace(envelope.Prompt)
		images = envelope.Images
	} else {
		if textPrompt == "" && strings.TrimSpace(*promptFile) != "" {
			raw, err := os.ReadFile(*promptFile)
			if err != nil {
				return fmt.Errorf("read prompt file: %w", err)
			}
			textPrompt = strings.TrimSpace(string(raw))
		}
		if textPrompt == "" && len(fs.Args()) > 0 {
			textPrompt = strings.TrimSpace(strings.Join(fs.Args(), " "))
		}
		if textPrompt == "" {
			raw, err := io.ReadAll(io.LimitReader(stdin, 8<<20))
			if err != nil {
				return fmt.Errorf("read prompt: %w", err)
			}
			textPrompt = strings.TrimSpace(string(raw))
		}
	}
	if textPrompt == "" {
		return errors.New("prompt is required")
	}

	request := openai.ChatRequest{
		Model:       selected.Model,
		Messages:    openai.MessagesFromEnvelope(textPrompt, images),
		Temperature: resolvedTemperature,
		MaxTokens:   resolvedMaxTokens,
		ExtraBody:   providerRequestBody(config, selected),
	}
	if strings.TrimSpace(*responseFormat) != "" {
		if !json.Valid([]byte(*responseFormat)) {
			return errors.New("--response-format must be valid JSON")
		}
		request.ResponseFormat = json.RawMessage(*responseFormat)
	}

	result, err := newClient(config).Chat(context.Background(), request)
	if err != nil {
		return err
	}
	if *asJSON {
		_, err = fmt.Fprintln(stdout, string(result.Raw))
		return err
	}
	_, err = fmt.Fprintln(stdout, result.Text)
	return err
}

func providerRequestBody(_ Config, _ policy.ResolvedPolicyModel) map[string]any {
	// Provider-specific OpenAI-compatible extensions belong here. OpenRouter
	// spend controls stay in the OpenRouter resource; OrcaRouter/abliteration
	// do not need a common extension today.
	return nil
}

func resolveSelection(catalog policy.Catalog, role, model, defaultRole string) (policy.ResolvedPolicyModel, error) {
	role = strings.TrimSpace(role)
	model = strings.TrimSpace(model)
	switch {
	case role != "" && model != "":
		return policy.ResolvedPolicyModel{}, errors.New("--role and --model are mutually exclusive")
	case role != "":
		return policy.ResolveRole(catalog, role)
	case model != "":
		return policy.ResolveModel(catalog, model)
	default:
		if strings.TrimSpace(defaultRole) == "" {
			return policy.ResolvedPolicyModel{}, errors.New("--role is required")
		}
		return policy.ResolveRole(catalog, defaultRole)
	}
}

func runListModels(config Config, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("list-models", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	asJSON := fs.Bool("json", true, "Emit JSON")
	limit := fs.Int("limit", 0, "Maximum models to return")
	search := fs.String("search", "", "Case-insensitive id/name filter")
	provider := fs.String("provider", "", "Provider prefix filter")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(loadRuntime(config).APIKey) == "" {
		return &providererror.Error{Code: providererror.CodeUnconfigured, Message: fmt.Sprintf("%s credential is not configured; provision %s:%s", config.DisplayName, config.CredentialIdentity, config.CredentialField)}
	}
	raw, err := newClient(config).ListModels(context.Background())
	if err != nil {
		return err
	}
	response, err := openai.NormalizeModels(raw, config.Name, "", *limit)
	if err != nil {
		return err
	}
	filtered := response.Models[:0]
	for _, model := range response.Models {
		if *provider != "" && !strings.EqualFold(model.Provider, *provider) {
			continue
		}
		if *search != "" && !strings.Contains(strings.ToLower(model.ID+" "+model.Name+" "+model.Description), strings.ToLower(*search)) {
			continue
		}
		filtered = append(filtered, model)
	}
	response.Models = filtered
	response.Count = len(filtered)
	if *asJSON {
		encoder := json.NewEncoder(stdout)
		encoder.SetEscapeHTML(false)
		return encoder.Encode(response)
	}
	for _, model := range response.Models {
		fmt.Fprintln(stdout, model.ID)
	}
	return nil
}

func runShowConfig(config Config, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("show-config", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	asJSON := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	catalog, path, err := loadCatalog(config)
	if err != nil {
		return err
	}
	rt := loadRuntime(config)
	defaultModel := ""
	if rt.DefaultRole != "" {
		if resolved, err := policy.ResolveRole(catalog, rt.DefaultRole); err == nil {
			defaultModel = resolved.Model
		}
	}
	payload := struct {
		Provider      string `json:"provider"`
		APIBase       string `json:"api_base"`
		DefaultRole   string `json:"default_role,omitempty"`
		DefaultModel  string `json:"default_model,omitempty"`
		PolicyPath    string `json:"policy_path"`
		Credential    string `json:"credential"`
		APIKeySource  string `json:"api_key_source"`
		APIKeyPreview string `json:"api_key_preview,omitempty"`
	}{
		Provider: config.Name, APIBase: rt.APIBase, DefaultRole: rt.DefaultRole,
		DefaultModel: defaultModel, PolicyPath: path, Credential: config.CredentialIdentity + ":" + config.CredentialField,
		APIKeySource: "environment", APIKeyPreview: redact(rt.APIKey),
	}
	if rt.APIKey == "" {
		payload.APIKeySource = "not configured"
	}
	if *asJSON {
		encoder := json.NewEncoder(stdout)
		encoder.SetEscapeHTML(false)
		return encoder.Encode(payload)
	}
	fmt.Fprintf(stdout, "Provider: %s\nAPI Base: %s\nDefault Role: %s\nDefault Model: %s\nPolicy: %s\nCredential: %s\nAPI Key: %s\n",
		payload.Provider, payload.APIBase, payload.DefaultRole, payload.DefaultModel, payload.PolicyPath, payload.Credential, payload.APIKeyPreview)
	return nil
}

func runConfigure(config Config, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("configure", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintf(stdout, "Usage: resource-%s configure\n\nThe key is read from standard input and is never accepted as an argument.\n", config.Name)
			return nil
		}
		return err
	}
	if len(fs.Args()) != 0 {
		return errors.New("configure accepts no positional arguments; provide the value on standard input")
	}
	value, err := io.ReadAll(io.LimitReader(os.Stdin, 64*1024))
	if err != nil {
		return fmt.Errorf("read credential: %w", err)
	}
	authority, err := credentialauthority.Default()
	if err != nil {
		return fmt.Errorf("resolve credential authority: %w", err)
	}
	client, err := credentialclient.NewClient(credentialclient.ClientOptions{Authority: authority})
	if err != nil {
		return fmt.Errorf("resolve credential client: %w", err)
	}
	if _, err := client.Provision(context.Background(), credentialclient.ProvisionRequest{
		Identity: config.CredentialIdentity,
		Field:    config.CredentialField,
		Value:    strings.TrimSpace(string(value)),
	}); err != nil {
		return fmt.Errorf("provision credential through control plane: %w", err)
	}
	_, err = fmt.Fprintf(stdout, "%s credential provisioned.\n", config.DisplayName)
	return err
}

func runEnsure(config Config, args []string, stdout io.Writer) error {
	var configB64 string
	skipCatalog := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--config-base64":
			if i+1 >= len(args) {
				return errors.New("--config-base64 requires a value")
			}
			configB64 = args[i+1]
			i++
		case "--skip-catalog-check":
			skipCatalog = true
		default:
			return fmt.Errorf("unknown flag %q", args[i])
		}
	}
	if configB64 == "" {
		return errors.New("--config-base64 is required")
	}
	raw, err := base64.StdEncoding.DecodeString(configB64)
	if err != nil {
		return fmt.Errorf("decode --config-base64: %w", err)
	}
	cfg, err := ensure.ParseConfig(raw)
	if err != nil {
		return err
	}
	catalog, _, err := loadCatalog(config)
	if err != nil {
		return err
	}
	var checker ensure.CatalogChecker
	if !skipCatalog && strings.TrimSpace(loadRuntime(config).APIKey) != "" {
		checker = liveChecker{client: newClient(config)}
	}
	return ensure.Run(context.Background(), cfg, catalog, checker, stdout, config.Name+"-ensure:")
}

type liveChecker struct {
	client *openai.Client
}

func (c liveChecker) Present(ctx context.Context, _ []string, models []string) (map[string]bool, error) {
	raw, err := c.client.ListModels(ctx)
	if err != nil {
		return nil, err
	}
	response, err := openai.NormalizeModels(raw, "live", "", 0)
	if err != nil {
		return nil, err
	}
	want := map[string]bool{}
	for _, model := range models {
		want[model] = false
	}
	for _, model := range response.Models {
		if _, tracked := want[model.ID]; tracked {
			want[model.ID] = true
		}
	}
	return want, nil
}

func redact(key string) string {
	key = strings.TrimSpace(key)
	switch {
	case key == "":
		return "not configured"
	case len(key) <= 8:
		return "********"
	default:
		return key[:6] + "..." + key[len(key)-4:]
	}
}

// HandleProviderError prints the cross-process marker for a typed failure.
func HandleProviderError(err error) {
	if typed, ok := providererror.As(err); ok {
		if line, markerErr := typed.MarkerLine(); markerErr == nil {
			fmt.Fprintln(os.Stderr, string(line))
		}
	}
}
