package permissions

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/vrooli/agentharness"
)

// ManagedHooksSchema versions resources/codex/managed-hooks.json.
const ManagedHooksSchema = "vrooli.agent-managed-hooks/v1"

// policyRunnerProgram is the host-owned hook binary. It installs beside the
// other companion binaries in ~/.vrooli/bin.
const policyRunnerProgram = "vrooli-policy-runner"

// policyRunnerEnv overrides the runner command (used verbatim).
const policyRunnerEnv = "VROOLI_AGENT_POLICY_RUNNER"

// defaultHookTimeoutSeconds bounds a hook the declaration gives no timeout:
// Codex waits 600 seconds by default, far too long for a guard.
const defaultHookTimeoutSeconds = 10

// ManagedHook is one hook Vrooli declares for Codex.
type ManagedHook struct {
	ID             string   `json:"id"`
	Event          string   `json:"event"`
	Matcher        string   `json:"matcher"`
	Program        string   `json:"program"`
	Args           []string `json:"args"`
	TimeoutSeconds int      `json:"timeout_seconds"`
	Rationale      string   `json:"rationale"`
}

// ManagedHooksDocument is resources/codex/managed-hooks.json.
type ManagedHooksDocument struct {
	SchemaVersion string        `json:"schema_version"`
	Runner        string        `json:"runner"`
	Description   string        `json:"description"`
	Reference     string        `json:"reference"`
	Hooks         []ManagedHook `json:"hooks"`
}

// ManagedHooksPath is the declaration beside the resource's other data.
func ManagedHooksPath() string {
	return agentharness.ResourceFilePath("codex", "managed-hooks.json")
}

// LoadManagedHooks reads and validates a declaration.
func LoadManagedHooks(path string) (ManagedHooksDocument, error) {
	var doc ManagedHooksDocument
	data, err := os.ReadFile(path)
	if err != nil {
		return doc, fmt.Errorf("read managed hooks: %w", err)
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return doc, fmt.Errorf("parse %s: %w", path, err)
	}
	if doc.SchemaVersion != ManagedHooksSchema {
		return doc, fmt.Errorf("%s: schema_version %q, want %q", path, doc.SchemaVersion, ManagedHooksSchema)
	}
	seen := map[string]bool{}
	for i, hook := range doc.Hooks {
		switch {
		case hook.ID == "" || hook.Event == "" || hook.Matcher == "" || hook.Program == "":
			return doc, fmt.Errorf("%s: hook %d needs id, event, matcher and program", path, i)
		case seen[hook.ID]:
			return doc, fmt.Errorf("%s: duplicate hook id %q", path, hook.ID)
		case strings.ContainsAny(hook.Program, `/\`):
			return doc, fmt.Errorf("%s: hook %q program must be a bare name resolved under ~/.vrooli/bin", path, hook.ID)
		}
		seen[hook.ID] = true
	}
	return doc, nil
}

// PolicyRunnerCommand is how a Codex hook invokes the policy runner: the
// environment override verbatim, else the absolute installed path, so a hook
// does not depend on the PATH Codex was started with.
func PolicyRunnerCommand() string {
	if override := strings.TrimSpace(os.Getenv(policyRunnerEnv)); override != "" {
		return override
	}
	return installedProgram(policyRunnerProgram)
}

// installedProgram quotes the absolute path of a companion binary for a shell.
func installedProgram(name string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return name
	}
	return shellQuote(filepath.Join(home, ".vrooli", "bin", name))
}

var shellSafe = regexp.MustCompile(`^[A-Za-z0-9_@%+=:,./-]+$`)

func shellQuote(s string) string {
	if shellSafe.MatchString(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// CommandLine is the hook's shell command.
func (h ManagedHook) CommandLine() string {
	program := installedProgram(h.Program)
	if h.Program == policyRunnerProgram {
		program = PolicyRunnerCommand()
	}
	parts := []string{program}
	for _, arg := range h.Args {
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}

// Entry is the hook object written inside its event group.
func (h ManagedHook) Entry() map[string]any {
	timeout := h.TimeoutSeconds
	if timeout <= 0 {
		timeout = defaultHookTimeoutSeconds
	}
	return map[string]any{"type": "command", "command": h.CommandLine(), "timeout": timeout}
}

// Managed hook states.
const (
	HookOK      = "ok"
	HookMissing = "missing"
	HookDiffers = "differs"
)

// ManagedHookStatus compares one declared hook with hooks.json.
type ManagedHookStatus struct {
	Hook  ManagedHook
	State string
	// Live is the installed matcher and command when State is differs.
	Live string
}

// liveHook is one identified entry found in hooks.json.
type liveHook struct {
	event, matcher string
	entry          map[string]any
}

// readLiveHooks returns the identified entries of hooks.json; a missing file
// has none.
func (a *Adapter) readLiveHooks() (map[string]liveHook, error) {
	snapshot, err := agentharness.ReadPermissionFile(a.HookPath)
	if err != nil {
		return nil, err
	}
	live := map[string]liveHook{}
	if len(snapshot.Data) == 0 {
		return live, nil
	}
	var doc struct {
		Hooks map[string][]struct {
			Matcher string           `json:"matcher"`
			Hooks   []map[string]any `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(snapshot.Data, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", a.HookPath, err)
	}
	for event, groups := range doc.Hooks {
		for _, group := range groups {
			for _, entry := range group.Hooks {
				if id, _ := entry["_id"].(string); id != "" {
					live[event+"/"+id] = liveHook{event: event, matcher: group.Matcher, entry: entry}
				}
			}
		}
	}
	return live, nil
}

// ManagedHookStatuses reads hooks.json and reports each declared hook.
func (a *Adapter) ManagedHookStatuses(doc ManagedHooksDocument) ([]ManagedHookStatus, error) {
	if a.HookPath == "" {
		return nil, errors.New("hooks are not supported in this scope")
	}
	live, err := a.readLiveHooks()
	if err != nil {
		return nil, err
	}
	statuses := make([]ManagedHookStatus, 0, len(doc.Hooks))
	for _, hook := range doc.Hooks {
		status := ManagedHookStatus{Hook: hook, State: HookMissing}
		if found, ok := live[hook.Event+"/"+hook.ID]; ok {
			status.State, status.Live = HookDiffers, fmt.Sprintf("matcher %q, command %v", found.matcher, found.entry["command"])
			if found.matcher == hook.Matcher && liveEntryMatches(found.entry, hook) {
				status.State = HookOK
			}
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

// liveEntryMatches compares an installed entry with the declaration, ignoring
// the ownership markers.
func liveEntryMatches(entry map[string]any, hook ManagedHook) bool {
	stripped := map[string]any{}
	for key, value := range entry {
		if key != "_id" && key != "managedBy" {
			stripped[key] = value
		}
	}
	return reflect.DeepEqual(normalizeJSON(stripped), normalizeJSON(hook.Entry()))
}

func normalizeJSON(v any) any {
	data, _ := json.Marshal(v)
	var out any
	_ = json.Unmarshal(data, &out)
	return out
}

// ApplyManagedHooks writes the declared hooks and nothing else. A hook whose
// matcher changed is replaced, because reconciling an entry in place keeps its
// old group's matcher. It returns the statuses found before the write.
func (a *Adapter) ApplyManagedHooks(doc ManagedHooksDocument) ([]ManagedHookStatus, error) {
	before, err := a.ManagedHookStatuses(doc)
	if err != nil {
		return nil, err
	}
	broker := agentharness.NewHookBroker()
	target := agentharness.HookTarget{Agent: "codex", Path: a.HookPath}
	for _, status := range before {
		if status.State == HookOK {
			continue
		}
		hook := status.Hook
		if status.State == HookDiffers {
			if _, err := broker.Remove(target, hook.Event, hook.ID); err != nil {
				return before, fmt.Errorf("replace hook %s: %w", hook.ID, err)
			}
		}
		if _, err := broker.Reconcile(target, agentharness.HookRegistration{Event: hook.Event, ID: hook.ID, Matcher: hook.Matcher, Hook: hook.Entry()}); err != nil {
			return before, fmt.Errorf("install hook %s: %w", hook.ID, err)
		}
	}
	return before, nil
}
