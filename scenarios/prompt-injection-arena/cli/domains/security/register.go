package security

import (
	"fmt"
	"os"

	"prompt-injection-arena/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `security` subcommand group. Today the only endpoint is
// POST /api/v1/security/test-agent, which runs an agent config against the
// injection library and returns a robustness report.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "security",
		Description: "Run security tests against agent configurations",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "test-agent", Description: "Test an agent configuration against the injection library", Run: func(args []string) error { return runTestAgent(core, args) }},
		},
	}
}

func runTestAgent(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("security test-agent")
	systemPrompt := fs.String("system-prompt", "", "System prompt for the agent")
	model := fs.String("model", "llama3.2", "Ollama model name")
	temperature := fs.Float64("temperature", 0.7, "Sampling temperature")
	maxTokens := fs.Int("max-tokens", 1000, "Maximum tokens in the response")
	maxExecution := fs.Int("max-execution-time", 30000, "Per-test execution budget in milliseconds")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the full request body (overrides other flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if *bodyFile != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if *systemPrompt == "" {
			return fmt.Errorf("usage: security test-agent --system-prompt TEXT [--model NAME] [--temperature FLOAT] [--max-tokens INT] [--max-execution-time MS] [--body-file PATH]")
		}
		payload = map[string]interface{}{
			"agent_config": map[string]interface{}{
				"system_prompt": *systemPrompt,
				"model_name":    *model,
				"temperature":   *temperature,
				"max_tokens":    *maxTokens,
			},
			"max_execution_time": *maxExecution,
		}
	}

	body, err := core.Request("POST", "/security/test-agent", nil, payload)
	if err != nil {
		return err
	}

	var resp support.TestAgentResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	total := intFromSummary(resp.Summary, "total_tests")
	successful := intFromSummary(resp.Summary, "successful_injections")

	result := []string{
		fmt.Sprintf("Robustness score: %.1f%%", resp.RobustnessScore),
		fmt.Sprintf("Total tests: %d", total),
		fmt.Sprintf("Successful injections: %d", successful),
		fmt.Sprintf("Failed injections: %d", total-successful),
	}

	changes := []string{}
	for _, rec := range resp.Recommendations {
		changes = append(changes, "recommendation: "+rec)
	}
	if len(changes) == 0 {
		changes = []string{"No recommendations returned"}
	}

	report := cliapp.MutationReport{
		Result:  result,
		Changes: changes,
		NextCommand: []string{
			fmt.Sprintf("%s leaderboard agents", support.CLIName),
			fmt.Sprintf("%s statistics", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func intFromSummary(summary map[string]interface{}, key string) int {
	if summary == nil {
		return 0
	}
	if v, ok := summary[key].(float64); ok {
		return int(v)
	}
	return 0
}
