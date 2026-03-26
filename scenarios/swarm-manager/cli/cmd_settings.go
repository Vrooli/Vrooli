package main

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdSettingsGet(args []string) error {
	fs := flag.NewFlagSet("settings get", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := a.getV1("/settings", nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	var response struct {
		Settings struct {
			Theme                    string `json:"theme"`
			DefaultMode              string `json:"default_mode"`
			DefaultDelaySeconds      int64  `json:"default_delay_seconds"`
			AutoFixup                bool   `json:"auto_fixup"`
			MaxFixupAttempts         int    `json:"max_fixup_attempts"`
			AutoInitializeWorkshop   bool   `json:"auto_initialize_workshop"`
			AutoAdvanceWorkshop      bool   `json:"auto_advance_workshop"`
			AutoCascadeWorkshop      bool   `json:"auto_cascade_workshop"`
			MaxAutoRounds            int    `json:"max_auto_rounds"`
			AgentMaxTurns            int    `json:"agent_max_turns"`
			AgentTimeoutSeconds      int    `json:"agent_timeout_seconds"`
			AgentRequiresApproval    bool   `json:"agent_requires_approval"`
			SearchDebounceMs         int    `json:"search_debounce_ms"`
			ToastDurationMs          int    `json:"toast_duration_ms"`
			ConfirmDestructiveActions bool   `json:"confirm_destructive_actions"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	s := response.Settings

	printSection("General")
	fmt.Printf("  Theme: %s\n", s.Theme)

	printSection("Execution Defaults")
	fmt.Printf("  Default mode: %s\n", s.DefaultMode)
	fmt.Printf("  Default delay seconds: %d\n", s.DefaultDelaySeconds)
	fmt.Printf("  Auto fixup: %t\n", s.AutoFixup)
	fmt.Printf("  Max fixup attempts: %d\n", s.MaxFixupAttempts)

	printSection("Workshop")
	fmt.Printf("  Auto-initialize workshop: %t\n", s.AutoInitializeWorkshop)
	fmt.Printf("  Auto-advance workshop:    %t\n", s.AutoAdvanceWorkshop)
	fmt.Printf("  Auto-cascade workshop:    %t\n", s.AutoCascadeWorkshop)
	fmt.Printf("  Max auto rounds:          %d\n", s.MaxAutoRounds)

	printSection("Agent Behavior")
	fmt.Printf("  Agent max turns: %d\n", s.AgentMaxTurns)
	fmt.Printf("  Agent timeout seconds: %d\n", s.AgentTimeoutSeconds)
	fmt.Printf("  Agent requires approval: %t\n", s.AgentRequiresApproval)

	printSection("UI Preferences")
	fmt.Printf("  Search debounce ms: %d\n", s.SearchDebounceMs)
	fmt.Printf("  Toast duration ms: %d\n", s.ToastDurationMs)
	fmt.Printf("  Confirm destructive actions: %t\n", s.ConfirmDestructiveActions)

	printCommandListSection("Next Steps", []string{
		cliCommand("settings", "update", "--data", `'{"default_mode":"yolo"}'`),
		cliCommand("execution", "policy-get"),
		cliCommand("status"),
	})
	return nil
}

func (a *App) cmdSettingsUpdate(args []string) error {
	fs := flag.NewFlagSet("settings update", flag.ContinueOnError)
	data := fs.String("data", "", "JSON payload (inline or @file)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("data", *data); err != nil {
		return fmt.Errorf("usage: settings update --data JSON [--json]\n\n%s", err)
	}

	payload, err := parseJSONString(*data)
	if err != nil {
		return fmt.Errorf("usage: settings update --data JSON [--json]\n\n%s", err)
	}

	body, err := a.requestV1("PUT", "/settings", nil, payload)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	printSection("Result")
	fmt.Println("  Updated settings")
	cliutil.PrintJSON(body)
	printCommandListSection("Next Steps", []string{
		cliCommand("settings", "get"),
		cliCommand("status"),
	})
	return nil
}
