package cliapp

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

type healthDependencyStatus struct {
	Connected bool `json:"connected"`
}

// healthResponse mirrors the subset of the canonical health payload that
// cli-core consumes when rendering status output.
type healthResponse struct {
	Status       string                            `json:"status"`
	Service      string                            `json:"service"`
	Timestamp    string                            `json:"timestamp"`
	Readiness    bool                              `json:"readiness"`
	Version      string                            `json:"version,omitempty"`
	Dependencies map[string]healthDependencyStatus `json:"dependencies,omitempty"`
}

// StandardBaseCommandOptions customizes the common operational command groups
// that most scenario CLIs should expose.
type StandardBaseCommandOptions struct {
	IncludeStatusCommand    *bool
	IncludeConfigureCommand *bool
	ConfigureAPIBaseKeys    []string
	ConfigureTokenKeys      []string
}

// StatusCommandOptions customizes the built-in status command.
type StatusCommandOptions struct {
	Name        string
	Description string
}

// StandardBaseCommandGroups returns the standard operational command groups
// expected on most scenario CLIs.
func (a *ScenarioApp) StandardBaseCommandGroups(opts ...StandardBaseCommandOptions) []CommandGroup {
	config := StandardBaseCommandOptions{}
	if len(opts) > 0 {
		config = opts[0]
	}

	includeStatus := true
	if config.IncludeStatusCommand != nil {
		includeStatus = *config.IncludeStatusCommand
	}
	includeConfigure := true
	if config.IncludeConfigureCommand != nil {
		includeConfigure = *config.IncludeConfigureCommand
	}

	var groups []CommandGroup
	if includeStatus {
		groups = append(groups, CommandGroup{
			Title: "Health",
			Commands: []Command{
				a.StandardStatusCommand(),
			},
		})
	}
	if includeConfigure {
		groups = append(groups, CommandGroup{
			Title: "Configuration",
			Commands: []Command{
				a.ConfigureCommand(config.ConfigureAPIBaseKeys, config.ConfigureTokenKeys),
			},
		})
	}
	return groups
}

// StandardStatusCommand returns a built-in status command backed by the
// canonical root health endpoint.
func (a *ScenarioApp) StandardStatusCommand(opts ...StatusCommandOptions) Command {
	config := StatusCommandOptions{
		Name:        "status",
		Description: "Check API health",
	}
	if len(opts) > 0 {
		if strings.TrimSpace(opts[0].Name) != "" {
			config.Name = strings.TrimSpace(opts[0].Name)
		}
		if strings.TrimSpace(opts[0].Description) != "" {
			config.Description = strings.TrimSpace(opts[0].Description)
		}
	}

	return Command{
		Name:        config.Name,
		NeedsAPI:    true,
		Description: config.Description,
		Usage:       fmt.Sprintf("%s %s [--json]", a.options.Name, config.Name),
		HelpText:    "Use --json to print the raw health payload instead of the operational summary.",
		Run: func(args []string) error {
			return a.runStandardStatus(args, os.Stdout)
		},
	}
}

func (a *ScenarioApp) runStandardStatus(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var jsonOutput bool
	fs.BoolVar(&jsonOutput, "json", false, "Print raw health JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	body, err := a.fetchHealth()
	if err != nil {
		return err
	}
	if jsonOutput {
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, body, "", "  "); err == nil {
			_, err = fmt.Fprintln(stdout, pretty.String())
			return err
		}
		_, err = fmt.Fprintln(stdout, string(body))
		return err
	}

	var parsed healthResponse
	if err := json.Unmarshal(body, &parsed); err != nil || strings.TrimSpace(parsed.Status) == "" {
		_, err = fmt.Fprintln(stdout, string(body))
		return err
	}

	report := OperationalReport{
		Status: []string{
			fmt.Sprintf("Status: %s", parsed.Status),
			fmt.Sprintf("Ready: %v", parsed.Readiness),
		},
	}
	if parsed.Service != "" {
		report.Status = append(report.Status, fmt.Sprintf("Service: %s", parsed.Service))
	}
	if parsed.Version != "" {
		report.Status = append(report.Status, fmt.Sprintf("Version: %s", parsed.Version))
	}
	if parsed.Timestamp != "" {
		report.Status = append(report.Status, fmt.Sprintf("Timestamp: %s", parsed.Timestamp))
	}
	if len(parsed.Dependencies) > 0 {
		group := TriageGroup{Heading: "Dependencies"}
		keys := make([]string, 0, len(parsed.Dependencies))
		for key := range parsed.Dependencies {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			dep := parsed.Dependencies[key]
			status := "disconnected"
			if dep.Connected {
				status = "connected"
			}
			group.Items = append(group.Items, fmt.Sprintf("%s: %s", key, status))
		}
		report.Triage = append(report.Triage, group)
	}
	if parsed.Readiness {
		report.NextSteps = []string{
			fmt.Sprintf("%s status --json", a.options.Name),
		}
	} else {
		report.NextSteps = []string{
			fmt.Sprintf("%s --auto-start status", a.options.Name),
			fmt.Sprintf("vrooli scenario start %s", a.options.Name),
		}
	}
	return RenderOperationalReport(stdout, report)
}

func (a *ScenarioApp) fetchHealth() ([]byte, error) {
	paths := []string{a.HealthPath()}
	paths = append(paths, a.options.LegacyHealthPaths...)

	var lastErr error
	for _, path := range paths {
		normalized := a.APIRootPath(path)
		body, err := a.rootRequest("GET", normalized, nil, nil)
		if err == nil {
			return body, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("no health paths configured")
}

func (a *ScenarioApp) rootRequest(method, path string, query url.Values, body interface{}) ([]byte, error) {
	rootBase := a.APIRootBase()
	if strings.TrimSpace(rootBase) == "" {
		return nil, fmt.Errorf("api base URL is empty; set --api-base, %s, config api_base, or a port env", strings.Join(append(a.options.APIEnvVars, a.options.APIPortEnvVars...), ", "))
	}

	if a.HTTPClient == nil {
		a.HTTPClient = cliutil.NewHTTPClient(cliutil.HTTPClientOptions{})
	}
	client := *a.HTTPClient
	client.SetBaseOptions(cliutil.APIBaseOptions{Override: rootBase})
	client.SetToken(a.tokenSource())
	return client.Do(method, path, query, body)
}
