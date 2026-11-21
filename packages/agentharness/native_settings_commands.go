package agentharness

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/upstreamcheck"
)

// NativeSettingsConfig binds one resource's declared document to its native
// file format. Decode and Encode own the format (TOML, JSON); everything else
// is shared.
type NativeSettingsConfig struct {
	Runner       string
	DocumentPath string
	// SettingsPath returns the native file. A symlinked file is resolved to
	// its target, so a per-session home that links to the shared config
	// updates the shared file instead of failing.
	SettingsPath func() (string, error)
	Decode       func([]byte) (map[string]any, error)
	Encode       func(map[string]any) ([]byte, error)
	// InstalledVersion reports the harness version for verified-version notes;
	// optional.
	InstalledVersion func() string
	WriterVersion    string
	Stdout           io.Writer
	Stderr           io.Writer
}

// NativeSettingsReport is the machine-readable result of plan and apply.
type NativeSettingsReport struct {
	NativeSettingsPlan
	Applied          bool     `json:"applied"`
	Backup           string   `json:"backup,omitempty"`
	InstalledVersion string   `json:"installed_version,omitempty"`
	Notes            []string `json:"notes,omitempty"`
}

// NativeSettingsCommands is the shared `settings` group.
func NativeSettingsCommands(cfg NativeSettingsConfig) cliapp.SubcommandGroup {
	cfg = cfg.withDefaults()
	return cliapp.SubcommandGroup{
		Name:        "settings",
		Description: "Reconcile Vrooli-declared native " + cfg.Runner + " settings (resources/" + cfg.Runner + "/native-settings.json)",
		Subcommands: []cliapp.Command{
			{Name: "plan", Description: "Show what apply would change (read-only)", Run: func(args []string) error { return cfg.run("settings plan", args, false) }},
			{Name: "apply", Description: "Apply declared settings; user-owned values are left alone unless --force (mutating)", Run: func(args []string) error { return cfg.run("settings apply", args, true) }},
			{Name: "decline", Description: "Stop managing one setting ID on this machine (mutating)", Run: func(args []string) error { return cfg.decline(args, true) }},
			{Name: "accept", Description: "Manage a previously declined setting ID again (mutating)", Run: func(args []string) error { return cfg.decline(args, false) }},
		},
	}
}

// EnsureNativeSettings applies the declaration without force. It is what a
// resource's `ensure` calls: conflicts are reported, never overwritten.
func EnsureNativeSettings(cfg NativeSettingsConfig) (NativeSettingsReport, error) {
	return cfg.withDefaults().reconcile(true, false)
}

func (cfg NativeSettingsConfig) withDefaults() NativeSettingsConfig {
	if cfg.Stdout == nil {
		cfg.Stdout = os.Stdout
	}
	if cfg.Stderr == nil {
		cfg.Stderr = os.Stderr
	}
	if cfg.DocumentPath == "" {
		cfg.DocumentPath = ResourceFilePath(cfg.Runner, "native-settings.json")
	}
	return cfg
}

// ResourceFilePath resolves a file owned by a resource in the repository.
func ResourceFilePath(resource, name string) string {
	return filepath.Join(filepath.Dir(ResourceCatalogPath(resource)), name)
}

func (cfg NativeSettingsConfig) run(name string, args []string, apply bool) error {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)
	jsonOut := fs.Bool("json", false, "Emit JSON")
	force := false
	if apply {
		fs.BoolVar(&force, "force", false, "Replace user-owned values that differ from the declaration")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	report, err := cfg.reconcile(apply, force)
	if err != nil {
		return err
	}
	if *jsonOut {
		enc := json.NewEncoder(cfg.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}
	WriteNativeSettingsReport(cfg.Stdout, report)
	return nil
}

func (cfg NativeSettingsConfig) settingsPath() (string, error) {
	if cfg.SettingsPath == nil {
		return "", errors.New("native settings path is not configured")
	}
	path, err := cfg.SettingsPath()
	if err != nil {
		return "", err
	}
	return ResolveNativeFilePath(path)
}

// maxPolicyBytes bounds the model policy read for value_from settings.
const maxPolicyBytes = 1 << 20

// desired reads the declaration and resolves its value_from settings from the
// model policy beside it. A policy file is read only when a setting needs it.
// A policy that is missing, oversized, or invalid JSON is not an error: the
// value_from settings are held as they are (see NativeSettingHeld), the rest of
// the declaration still applies, and the problem comes back as a note.
func (cfg NativeSettingsConfig) desired() (doc NativeSettingsDocument, notes []string, err error) {
	doc, err = ReadNativeSettingsDocument(cfg.DocumentPath)
	if err != nil {
		return doc, nil, fmt.Errorf("read %s: %w", cfg.DocumentPath, err)
	}
	if doc.Runner != cfg.Runner {
		return doc, nil, fmt.Errorf("%s declares runner %q, want %q", cfg.DocumentPath, doc.Runner, cfg.Runner)
	}
	if !slices.ContainsFunc(doc.Settings, func(s NativeSetting) bool { return s.ValueFrom != "" }) {
		return doc, nil, nil
	}
	policyPath := filepath.Join(filepath.Dir(cfg.DocumentPath), "model-policy.json")
	policy, err := readPolicyDocument(policyPath)
	if err != nil {
		return doc, []string{fmt.Sprintf("model policy unusable, so settings that follow it were left as they are: %v", err)}, nil
	}
	return ResolveNativeSettings(doc, policy), nil, nil
}

func readPolicyDocument(path string) (map[string]any, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, maxPolicyBytes+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > maxPolicyBytes {
		return nil, fmt.Errorf("%s is larger than %d bytes", path, maxPolicyBytes)
	}
	policy := map[string]any{}
	if err := json.Unmarshal(raw, &policy); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return policy, nil
}

func (cfg NativeSettingsConfig) reconcile(apply, force bool) (NativeSettingsReport, error) {
	desired, policyNotes, err := cfg.desired()
	if err != nil {
		return NativeSettingsReport{}, err
	}
	path, err := cfg.settingsPath()
	if err != nil {
		return NativeSettingsReport{}, err
	}
	var report NativeSettingsReport
	err = NewHookBroker().WithLock(path, func() error {
		before, err := ReadPermissionFile(path)
		if err != nil {
			return err
		}
		live := map[string]any{}
		if len(before.Data) > 0 {
			if live, err = cfg.Decode(before.Data); err != nil {
				return fmt.Errorf("parse %s: %w", path, err)
			}
		}
		statePath := NativeSettingsStatePath(path)
		state, err := LoadNativeSettingsState(statePath)
		if err != nil {
			return err
		}
		plan := PlanNativeSettings(live, desired, state, force)
		plan.SettingsPath = path
		report = NativeSettingsReport{NativeSettingsPlan: plan}
		if cfg.InstalledVersion != nil {
			report.InstalledVersion = cfg.InstalledVersion()
		}
		report.Notes = append(policyNotes, verifiedVersionNotes(plan, report.InstalledVersion)...)
		if !apply {
			return nil
		}
		if plan.Mutates() {
			if err := ApplyNativeSettings(live, plan); err != nil {
				return err
			}
			data, err := cfg.Encode(live)
			if err != nil {
				return fmt.Errorf("encode %s: %w", path, err)
			}
			if report.Backup, err = BackupNativeFile(before, "native-settings"); err != nil {
				return err
			}
			if err := PublishPermissionFile(before, data); err != nil {
				return err
			}
		}
		if plan.ChangesState() || state == nil {
			if err := WriteNativeSettingsState(statePath, NextNativeSettingsState(state, plan, path, cfg.WriterVersion)); err != nil {
				return fmt.Errorf("native settings applied but state publication failed: %w", err)
			}
		}
		report.Applied = true
		return nil
	})
	return report, err
}

func (cfg NativeSettingsConfig) decline(args []string, decline bool) error {
	verb := "accept"
	if decline {
		verb = "decline"
	}
	if len(args) != 1 || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("usage: settings %s <setting-id>", verb)
	}
	desired, err := ReadNativeSettingsDocument(cfg.DocumentPath)
	if err != nil {
		return err
	}
	known := false
	for _, s := range desired.Settings {
		known = known || s.ID == args[0]
	}
	if !known {
		return fmt.Errorf("unknown setting id %q in %s", args[0], cfg.DocumentPath)
	}
	path, err := cfg.settingsPath()
	if err != nil {
		return err
	}
	statePath := NativeSettingsStatePath(path)
	// Same lock as reconcile, so a concurrent ensure cannot drop the decision.
	err = NewHookBroker().WithLock(path, func() error {
		state, err := LoadNativeSettingsState(statePath)
		if err != nil {
			return err
		}
		next := DeclineNativeSetting(state, args[0], decline)
		next.SettingsPath = path
		next.WrittenByVer = cfg.WriterVersion
		return WriteNativeSettingsState(statePath, next)
	})
	if err != nil {
		return err
	}
	if decline {
		fmt.Fprintf(cfg.Stdout, "%s: declined; its live value is left as it is and no longer managed\n", args[0])
	} else {
		fmt.Fprintf(cfg.Stdout, "%s: accepted; the next apply manages it again\n", args[0])
	}
	return nil
}

func verifiedVersionNotes(plan NativeSettingsPlan, installed string) []string {
	var notes []string
	for _, change := range plan.Changes {
		verified := change.VerifiedVersion()
		if verified == "" || installed == "" {
			continue
		}
		switch upstreamcheck.Compare(upstreamcheck.ExtractSemver(installed), verified) {
		case upstreamcheck.StatusUpToDate, upstreamcheck.StatusAhead:
			continue
		case upstreamcheck.StatusBehind:
			notes = append(notes, fmt.Sprintf("%s was verified on %s; installed %s is older, so it may have no effect until the harness is updated", change.ID, verified, installed))
		default:
			notes = append(notes, fmt.Sprintf("%s was verified on %s; installed version %q could not be compared", change.ID, verified, installed))
		}
	}
	return notes
}

// WriteNativeSettingsReport prints a short human summary.
func WriteNativeSettingsReport(w io.Writer, report NativeSettingsReport) {
	fmt.Fprintf(w, "%s native settings: %s\n", report.Runner, report.SettingsPath)
	for _, change := range report.Changes {
		line := fmt.Sprintf("  %-10s %s (%s)", change.Action, change.ID, change.Path)
		if change.Reason != "" {
			line += ": " + change.Reason
		}
		fmt.Fprintln(w, line)
	}
	for _, note := range report.Notes {
		fmt.Fprintln(w, "  note: "+note)
	}
	if report.Backup != "" {
		fmt.Fprintln(w, "  backup: "+report.Backup)
	}
	if !report.Applied {
		fmt.Fprintln(w, "  (plan only; run `settings apply` to write)")
	}
}
