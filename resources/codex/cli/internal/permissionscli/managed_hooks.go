package permissionscli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/resources/codex/cli/internal/permissions"
	"github.com/vrooli/vrooli/resources/internal/agentapp"
)

// The hooks Vrooli depends on are declared in resources/codex/managed-hooks.json
// and projected into hooks.json by `hooks apply` and by `ensure`. Both only add
// or correct those identified entries; operator hooks are never touched.

func (h *Handlers) declaration() (permissions.ManagedHooksDocument, error) {
	path := h.ManagedHooksPath
	if path == "" {
		path = permissions.ManagedHooksPath()
	}
	return permissions.LoadManagedHooks(path)
}

func (h *Handlers) trustProbe() permissions.TrustProbe {
	if h.TrustProbe != nil {
		return h.TrustProbe
	}
	return permissions.AppServerTrustProbe
}

// managedHookCommands are the `hooks plan` and `hooks apply` verbs.
func (h *Handlers) managedHookCommands() []cliapp.Command {
	return []cliapp.Command{
		{Name: "plan", Description: "Show the declared Vrooli hooks, what apply would write, and whether Codex trusts them (read-only)", Run: func(args []string) error { return h.runManagedHooks("hooks plan", args, false) }},
		{Name: "apply", Description: "Install or correct the declared Vrooli hooks in hooks.json (mutating; trust is never written)", Run: func(args []string) error { return h.runManagedHooks("hooks apply", args, true) }},
	}
}

func (h *Handlers) runManagedHooks(name string, args []string, apply bool) error {
	fs, scope := h.flagSet(name)
	authorized := fs.Bool("i-was-explicitly-authorized", false, "Override agent gate only when a human explicitly authorized this call")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if apply {
		if err := h.gate(name, true, *authorized); err != nil {
			return err
		}
	}
	a, err := h.adapter(*scope)
	if err != nil {
		return err
	}
	return h.reconcileManagedHooks(a, apply, h.Stdout)
}

// reconcileManagedHooks plans or applies the declaration and writes a report
// that ends with the trust state Codex gives each installed hook.
func (h *Handlers) reconcileManagedHooks(a *permissions.Adapter, apply bool, out io.Writer) error {
	doc, err := h.declaration()
	if err != nil {
		return err
	}
	if a.HookPath == "" {
		return errors.New("managed hooks apply to the user scope only")
	}
	var statuses []permissions.ManagedHookStatus
	if apply {
		statuses, err = a.ApplyManagedHooks(doc)
	} else {
		statuses, err = a.ManagedHookStatuses(doc)
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "managed hooks: %s -> %s\n", h.declarationLabel(), a.HookPath)
	var trust []permissions.HookTrust
	var trustErr error
	if apply || !needsWrite(statuses) {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		trust, trustErr = h.trustProbe()(ctx, hooksHome(a))
	}
	for _, status := range statuses {
		hook := status.Hook
		verb := map[string]string{permissions.HookOK: "installed", permissions.HookMissing: "would add", permissions.HookDiffers: "would replace (installed: " + status.Live + ")"}[status.State]
		if apply && status.State != permissions.HookOK {
			verb = map[string]string{permissions.HookMissing: "added", permissions.HookDiffers: "replaced"}[status.State]
		}
		fmt.Fprintf(out, "  %s  %s[%s]  %s\n    command: %s\n", hook.ID, hook.Event, hook.Matcher, verb, hook.CommandLine())
		if apply || status.State == permissions.HookOK {
			fmt.Fprintf(out, "    trust: %s\n", permissions.TrustAdvice(trustOf(trust, trustErr, hook)))
		}
	}
	if trustErr != nil {
		fmt.Fprintf(out, "  (Codex was not asked about trust: %v)\n", trustErr)
	}
	if runner := runnerPath(); runner != "" {
		if _, err := os.Stat(runner); err != nil {
			fmt.Fprintf(out, "  note: %s is not installed yet; the hooks are inert and fail open until `make install` places it\n", runner)
		}
	}
	return nil
}

func needsWrite(statuses []permissions.ManagedHookStatus) bool {
	for _, status := range statuses {
		if status.State != permissions.HookOK {
			return true
		}
	}
	return false
}

func trustOf(trust []permissions.HookTrust, err error, hook permissions.ManagedHook) string {
	if err != nil {
		return permissions.TrustUnknown
	}
	return permissions.TrustOf(trust, hook.CommandLine())
}

func (h *Handlers) declarationLabel() string {
	if h.ManagedHooksPath != "" {
		return h.ManagedHooksPath
	}
	return permissions.ManagedHooksPath()
}

// hooksHome is the directory Codex reads hooks.json from.
func hooksHome(a *permissions.Adapter) string {
	return strings.TrimSuffix(a.HookPath, "/hooks.json")
}

// runnerPath is the policy runner file the hooks invoke, when it is a path.
func runnerPath() string {
	command := permissions.PolicyRunnerCommand()
	if strings.HasPrefix(command, "/") {
		return strings.Trim(command, "'")
	}
	return ""
}

// ManagedHooksStep is the `ensure` step that keeps the declared hooks installed.
func ManagedHooksStep(h *Handlers) agentapp.EnsureStep {
	return agentapp.EnsureStep{Name: "managed-hooks", Run: func(stdout, _ io.Writer) error {
		a, err := h.AdapterFor(permissions.ScopeUser)
		if err != nil {
			return err
		}
		return h.reconcileManagedHooks(a, true, stdout)
	}}
}

// managedHookDrift reports the first declared hook that is not installed as declared.
func (h *Handlers) managedHookDrift(a *permissions.Adapter) error {
	if a.HookPath == "" {
		return nil // admin scope has no hooks.json
	}
	doc, err := h.declaration()
	if err != nil {
		return err
	}
	statuses, err := a.ManagedHookStatuses(doc)
	if err != nil {
		return err
	}
	for _, status := range statuses {
		if status.State != permissions.HookOK {
			return fmt.Errorf("managed hook %s is %s in %s; run `resource-codex hooks apply`", status.Hook.ID, status.State, a.HookPath)
		}
	}
	return nil
}
