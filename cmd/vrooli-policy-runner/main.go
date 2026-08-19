// Command vrooli-policy-runner is the portable process boundary for native
// coding-agent hooks. It has no dependency on Agent Manager or live provider
// processes; all enforcement input comes from the local snapshot bundle.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	agentharness "github.com/vrooli/agentharness"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		var decisionExit exitError
		if errors.As(err, &decisionExit) {
			os.Exit(decisionExit.code)
		}
		fmt.Fprintln(os.Stderr, "vrooli-policy-runner:", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: vrooli-policy-runner hook|status|snapshot|canary|diagnose")
	}
	command := args[0]
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(stderr)
	runner := fs.String("runner", "unknown", "native coding-agent runner")
	profile := fs.String("profile", strings.TrimSpace(os.Getenv("VROOLI_AGENT_POLICY_MODE")), "rollout profile")
	storeDir := fs.String("snapshot-dir", "", "snapshot store directory")
	snapshotFile := fs.String("snapshot-file", "", "provider snapshot JSON to publish")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if *profile == "" {
		*profile = string(agentharness.ProfileAdvisory)
	}
	dir := *storeDir
	if dir == "" {
		var err error
		dir, err = agentharness.DefaultDataDir()
		if err != nil {
			return err
		}
	}
	store := agentharness.NewBundleStore(dir)
	runtime := agentharness.Runtime{Profile: agentharness.RolloutProfile(strings.ToLower(*profile)), Store: store}

	switch command {
	case "hook", "canary":
		event, err := readEvent(stdin, *runner)
		if err != nil {
			return err
		}
		decision, err := runtime.Evaluate(event)
		if err != nil {
			return err
		}
		if err := writeJSON(stdout, decision); err != nil {
			return err
		}
		if command == "hook" {
			// Native adapters use these stable classes: zero means continue,
			// ten means ask/confirm, and twenty means deny.
			if code := agentharness.ExitCode(decision.Action); code != 0 {
				return exitError{code: code}
			}
		}
		return nil
	case "status", "diagnose":
		bundle, err := store.Load()
		if err != nil {
			return writeJSON(stdout, map[string]any{"contract_version": agentharness.ContractVersion, "status": "unavailable", "reason": err.Error(), "profile": runtime.Profile})
		}
		return writeJSON(stdout, map[string]any{"contract_version": agentharness.ContractVersion, "status": "ready", "profile": runtime.Profile, "generation": bundle.Generation, "published_at": bundle.PublishedAt, "providers": bundle.Snapshots})
	case "snapshot":
		bundle, err := store.Load()
		if err != nil {
			return err
		}
		return writeJSON(stdout, bundle)
	case "publish":
		if strings.TrimSpace(*snapshotFile) == "" {
			return errors.New("--snapshot-file is required")
		}
		data, err := os.ReadFile(*snapshotFile)
		if err != nil {
			return fmt.Errorf("read provider snapshot: %w", err)
		}
		var snapshot agentharness.ProviderSnapshot
		if err := json.Unmarshal(data, &snapshot); err != nil {
			return fmt.Errorf("parse provider snapshot: %w", err)
		}
		if err := store.PublishProvider(snapshot); err != nil {
			return err
		}
		return writeJSON(stdout, map[string]any{"status": "published", "provider_id": snapshot.ProviderID})
	default:
		return fmt.Errorf("unknown command %q", command)
	}
}

type exitError struct{ code int }

func (e exitError) Error() string { return "decision requires native hook handling" }

func readEvent(reader io.Reader, runner string) (agentharness.ToolEvent, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return agentharness.ToolEvent{}, fmt.Errorf("read hook input: %w", err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return agentharness.ToolEvent{}, errors.New("hook input is empty")
	}
	var event agentharness.ToolEvent
	if err := json.Unmarshal(data, &event); err == nil && (event.Tool != "" || len(event.Arguments) > 0) {
		if event.Runner == "" {
			event.Runner = runner
		}
		return eventWithNow(event)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return agentharness.ToolEvent{}, fmt.Errorf("parse hook input: %w", err)
	}
	event.Runner = firstString(raw, "runner", "agent")
	if event.Runner == "" {
		event.Runner = runner
	}
	event.Tool = firstString(raw, "tool", "tool_name", "name", "command_name")
	event.WorkingDirectory = firstString(raw, "working_directory", "cwd", "workdir")
	event.Shell = firstString(raw, "shell", "command", "cmd")
	event.Target = firstString(raw, "target", "path", "file")
	if arguments, ok := raw["arguments"].([]any); ok {
		for _, argument := range arguments {
			if value, ok := argument.(string); ok {
				event.Arguments = append(event.Arguments, value)
			}
		}
	}
	if event.Tool == "" && event.Shell != "" {
		event.Arguments = []string{event.Shell}
	}
	return eventWithNow(event)
}

func eventWithNow(event agentharness.ToolEvent) (agentharness.ToolEvent, error) {
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}
	if err := event.Normalize(event.OccurredAt); err != nil {
		return agentharness.ToolEvent{}, err
	}
	return event, nil
}

func firstString(raw map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := raw[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
