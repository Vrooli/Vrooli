package recovery

import "fmt"

func CodingAgentPrompt(req CodingAgentRequest) string {
	return fmt.Sprintf(`You are the bounded recovery agent for Vrooli infrastructure.

Investigate and repair the health failure for scenario %q. Autoheal exhausted its mechanical recovery actions and reported: %s

Work from the repository root. Inspect the scenario status, autoheal evidence, recent logs, configuration, and relevant source before changing anything. Repair the underlying cause using supported Vrooli lifecycle and scenario commands. Validate the affected scenario and any directly related core infrastructure after the repair. Keep changes narrowly scoped to the failure; do not disable supervision, weaken health checks, expose credentials, or modify unrelated scenarios. If the issue cannot be safely repaired, leave a concise diagnosis in your final response and do not loop indefinitely.`, req.Scenario, req.Reason)
}

// CodingAgentArguments is the shared headless contract for recovery and CLI
// launches; a prompt must never accidentally open an interactive session.
func CodingAgentArguments(runner, prompt string) []string {
	switch runner {
	case "codex":
		return []string{"exec", prompt}
	case "opencode":
		return []string{"run", prompt}
	case "claude", "grok", "agy", "antigravity":
		return []string{"-p", prompt}
	default:
		return []string{prompt}
	}
}
