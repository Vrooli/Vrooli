package hooks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/cli-core/cliapp"
)

const (
	stopEvent       = "Stop"
	stopID          = "web-console-tts"
	promptEvent     = "UserPromptSubmit"
	promptID        = "web-console-prompt"
	hookScope       = "project"
	defaultAttempts = 5
	// hookTokenFileName matches resolveHookTokenPath in the API.
	hookTokenFileName = "hook-token.txt"
)

// errResourceCLIMissing marks the one failure a caller may reasonably choose to
// tolerate: a host that has never installed the Claude Code resource cannot
// register Claude hooks, and that is a configuration gap rather than a fault.
var errResourceCLIMissing = errors.New("resource-claude-code is not installed, so Claude Code hooks cannot be registered")

type commandRunner func(context.Context, string, ...string) ([]byte, error)

type registrar struct {
	lookup      func(string) (string, error)
	run         commandRunner
	sleep       func(time.Duration)
	stdout      io.Writer
	stderr      io.Writer
	getenv      func(string) string
	userHomeDir func() (string, error)
	workingDir  func() (string, error)
	readFile    func(string) ([]byte, error)
	maxAttempts int
}

type processState struct {
	Port int `json:"port"`
}

type reconcileResult struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

// Register exposes local hook reconciliation. It deliberately does not require
// the Web Console API client: lifecycle invokes it while the API is starting,
// and it resolves that API's managed port and token from local runtime state.
func Register() cliapp.SubcommandGroup {
	r := newRegistrar()
	return cliapp.SubcommandGroup{
		Name:        "hooks",
		Description: "Reconcile Web Console's Claude Code project hooks",
		Subcommands: []cliapp.Command{
			{Name: "register", Description: "Register the Stop and UserPromptSubmit hooks", Run: r.register},
			{Name: "remove", Description: "Remove the Stop and UserPromptSubmit hooks", Run: r.remove},
			{Name: "dispatch", Description: "Dispatch a Claude hook payload to Web Console", Run: dispatch},
		},
	}
}

func newRegistrar() *registrar {
	return &registrar{
		lookup:      exec.LookPath,
		run:         runCommand,
		sleep:       time.Sleep,
		stdout:      os.Stdout,
		stderr:      os.Stderr,
		getenv:      os.Getenv,
		userHomeDir: os.UserHomeDir,
		workingDir:  os.Getwd,
		readFile:    os.ReadFile,
		maxAttempts: defaultAttempts,
	}
}

func runCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, name, args...)
	output, err := command.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("%s: %w", name, err)
	}
	return output, nil
}

func (r *registrar) register(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: hooks register")
	}
	resourceCLI, err := r.resourceCLI()
	if errors.Is(err, exec.ErrNotFound) {
		// This used to return nil. Reporting success when the tool that
		// performs registration is absent meant hooks were never reconciled and
		// nothing ever said so: a stale hook survived every restart, message
		// capture stayed broken, and the only symptom was an empty Messages
		// view. Registration failing is recoverable; registration lying is not.
		return fmt.Errorf("%w: install it, then run 'web-console hooks register'", errResourceCLIMissing)
	}
	if err != nil {
		return err
	}

	apiPort, err := r.apiPort()
	if err != nil {
		fmt.Fprintln(r.stderr, "tts-hook: skipping registration -- API port not available")
		return nil
	}
	hookToken, err := r.awaitHookToken()
	if err != nil {
		fmt.Fprintf(r.stderr, "tts-hook: WARNING -- hook token not available after %d attempts. Auto-TTS will not work until hook is registered.\n", r.maxAttempts)
		fmt.Fprintln(r.stderr, "tts-hook: Run 'web-console hooks register' manually after the API starts.")
		return err
	}
	stopCommand := strings.Join([]string{
		"web-console", "hooks", "dispatch", "--event", shellQuote(stopEvent),
		"--url", shellQuote("http://localhost:" + strconv.Itoa(apiPort) + "/api/v1/hooks/stop"),
		"--token", shellQuote(hookToken),
	}, " ")
	if err := r.reconcile(resourceCLI, stopEvent, stopID, stopCommand, 30, "Stop", apiPort); err != nil {
		return err
	}

	promptCommand := strings.Join([]string{
		"web-console", "hooks", "dispatch", "--event", shellQuote(promptEvent),
		"--url", shellQuote("http://localhost:" + strconv.Itoa(apiPort) + "/api/v1/hooks/prompt-submit"),
		"--token", shellQuote(hookToken),
	}, " ")
	return r.reconcile(resourceCLI, promptEvent, promptID, promptCommand, 10, "UserPromptSubmit", apiPort)
}

func (r *registrar) remove(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: hooks remove")
	}
	resourceCLI, err := r.resourceCLI()
	if errors.Is(err, exec.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, hook := range []struct{ event, id, label string }{
		{stopEvent, stopID, "Stop"},
		{promptEvent, promptID, "UserPromptSubmit"},
	} {
		if _, err := r.run(context.Background(), resourceCLI, "hooks", "remove", "--event", hook.event, "--id", hook.id, "--scope", hookScope); err != nil {
			return fmt.Errorf("tts-hook: deregister %s hook: %w", hook.label, err)
		}
		fmt.Fprintf(r.stdout, "tts-hook: deregistered %s hook\n", hook.label)
	}
	return nil
}

func (r *registrar) reconcile(resourceCLI, event, id, command string, timeout int, label string, apiPort int) error {
	payload, err := json.Marshal(map[string]any{"type": "command", "command": command, "timeout": timeout})
	if err != nil {
		return fmt.Errorf("encode %s hook: %w", label, err)
	}
	output, err := r.run(context.Background(), resourceCLI, "hooks", "reconcile", "--event", event, "--id", id, "--hook-json", string(payload), "--scope", hookScope)
	if err != nil {
		return fmt.Errorf("tts-hook: reconcile %s hook: %w: %s", label, err, strings.TrimSpace(string(output)))
	}
	var result reconcileResult
	if err := json.Unmarshal(output, &result); err != nil {
		return fmt.Errorf("tts-hook: decode %s reconciliation: %w", label, err)
	}
	switch result.Status {
	case "applied":
		fmt.Fprintf(r.stdout, "tts-hook: registered %s hook -> localhost:%d\n", label, apiPort)
	case "unchanged":
		fmt.Fprintf(r.stdout, "tts-hook: %s hook already healthy -> localhost:%d\n", label, apiPort)
	default:
		return fmt.Errorf("tts-hook: unexpected %s reconcile result status %q: %s", label, result.Status, result.Reason)
	}
	return nil
}

func (r *registrar) resourceCLI() (string, error) {
	if path, err := r.lookup("resource-claude-code"); err == nil {
		return path, nil
	}
	if binDir := strings.TrimSpace(r.getenv("VROOLI_BIN_DIR")); binDir != "" {
		candidate := filepath.Join(binDir, "resource-claude-code")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return candidate, nil
		}
	}
	return "", exec.ErrNotFound
}

func (r *registrar) apiPort() (int, error) {
	home, err := r.userHomeDir()
	if err == nil {
		contents, readErr := r.readFile(filepath.Join(home, ".vrooli", "processes", "scenarios", "web-console", "start-api.json"))
		if readErr == nil {
			var state processState
			if json.Unmarshal(contents, &state) == nil && state.Port > 0 {
				return state.Port, nil
			}
		}
	}
	if value := strings.TrimSpace(r.getenv("API_PORT")); value != "" {
		port, parseErr := strconv.Atoi(value)
		if parseErr == nil && port > 0 {
			return port, nil
		}
	}
	return 0, errors.New("API port unavailable")
}

func (r *registrar) awaitHookToken() (string, error) {
	attempts := r.maxAttempts
	if attempts < 1 {
		attempts = 1
	}
	for attempt := 1; attempt <= attempts; attempt++ {
		if token, err := r.hookToken(); err == nil {
			return token, nil
		}
		if attempt < attempts {
			fmt.Fprintf(r.stderr, "tts-hook: hook token not found, retrying in 2s (%d/%d)...\n", attempt, attempts)
			r.sleep(2 * time.Second)
		}
	}
	return "", errors.New("hook token unavailable")
}

// hookToken reads the shared secret the API generates for hook callers.
//
// The path comes from the same api-core storage resolver the API writes with.
// These two used to disagree — the API wrote to the scenario state class while
// the registrar read $XDG_STATE_HOME — and the only reason hooks kept working
// was that a years-old file at the legacy path happened to hold the same value.
// Any token rotation would have silently registered hooks that authenticate
// against nothing, so the path is resolved once, from one authority.
func (r *registrar) hookToken() (string, error) {
	path, err := r.hookTokenPath()
	if err != nil {
		return "", err
	}
	contents, err := r.readFile(path)
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(string(contents))
	if token == "" {
		return "", fmt.Errorf("hook token file %s is empty", path)
	}
	return token, nil
}

func (r *registrar) hookTokenPath() (string, error) {
	if override := strings.TrimSpace(r.getenv("WC_HOOK_TOKEN_PATH")); override != "" {
		return override, nil
	}
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", fmt.Errorf("resolve hook token path: %w", err)
	}
	// Path resolves without creating anything: the registrar only ever reads
	// this file, and the API owns its creation.
	path, err := resolver.Path(storage.Options{ScenarioID: "web-console"}, storage.ClassState, hookTokenFileName)
	if err != nil {
		return "", fmt.Errorf("resolve hook token path: %w", err)
	}
	return path, nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
