package recoverycli

import (
	"fmt"
	"io"
	"time"

	recoveryapp "github.com/vrooli/vrooli/internal/app/recovery"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
)

type CommandID string

const (
	CommandCapture   CommandID = "capture"
	CommandRestore   CommandID = "restore"
	CommandWrite     CommandID = "write"
	CommandShow      CommandID = "show"
	CommandList      CommandID = "list"
	CommandTouch     CommandID = "touch"
	CommandSetTTL    CommandID = "set-ttl"
	CommandSetMode   CommandID = "set-mode"
	CommandClean     CommandID = "clean"
	CommandMigrate   CommandID = "migrate"
	CommandNamespace CommandID = "namespace"
)

const (
	groupRestorePoint = "Restore Point"
	groupEngagement   = "Engagement"
	groupMigration    = "Migration"
	groupAddressing   = "Addressing"
)

const recoveryDefaultVariant = "shadow"

func scenarioOption() commandtree.OptionArg {
	return commandtree.OptionArg{Name: "--scenario", ValueName: "name", Description: "Target scenario slug"}
}

func slugOption() commandtree.OptionArg {
	return commandtree.OptionArg{Name: "--slug", ValueName: "slug", Description: "Engagement slug (the baseline-<slug> directory)"}
}

func CommandSpecs() []commandtree.Spec[CommandID] {
	return []commandtree.Spec[CommandID]{
		{
			Name:    string(CommandCapture),
			Summary: "Capture a restore point of a scenario's working tree",
			Group:   groupRestorePoint,
			Args: commandtree.ArgSchema{
				Options: []commandtree.OptionArg{
					scenarioOption(),
					slugOption(),
					{Name: "--source", ValueName: "path", Description: "Working-tree path to capture (default scenarios/<scenario>)"},
					{Name: "--no-reflink", Description: "Force the portable deep-copy floor instead of the CoW fast path"},
					commandtree.JSONOption(),
				},
			},
			Handler: CommandCapture,
		},
		{
			Name:    string(CommandRestore),
			Summary: "Restore a scenario from its captured restore point (the git-free undo)",
			Group:   groupRestorePoint,
			Args: commandtree.ArgSchema{
				Options: []commandtree.OptionArg{
					scenarioOption(),
					slugOption(),
					{Name: "--dest", ValueName: "path", Description: "Destination to overlay the restore point onto (default scenarios/<scenario>)"},
					{Name: "--no-reflink", Description: "Force the portable deep-copy floor instead of the CoW fast path"},
					commandtree.JSONOption(),
				},
			},
			Handler: CommandRestore,
		},
		{
			Name:    string(CommandWrite),
			Summary: "Write or refresh an engagement manifest",
			Group:   groupEngagement,
			Args: commandtree.ArgSchema{
				Options: []commandtree.OptionArg{
					scenarioOption(),
					slugOption(),
					{Name: "--mode", ValueName: "mode", Description: "Execution mode: shadow|live"},
					{Name: "--variant", ValueName: "variant", Description: "Shadow variant (default shadow for shadow mode, live for live mode)"},
					{Name: "--ttl", ValueName: "dur", Description: "Idle TTL (e.g. 3h); omit for orchestrator-heartbeat mode"},
					{Name: "--ambient-var", ValueName: "value", Description: "VROOLI_SHADOW_SCENARIOS value for nested-CLI routing"},
					{Name: "--shadow-instance-key", ValueName: "key", Description: "Registry instance key for the shadow (default <scenario>@<variant>)"},
					{Name: "--anchor", ValueName: "name", Description: "git-control-tower baseline record to diff against"},
					commandtree.JSONOption(),
				},
			},
			Handler: CommandWrite,
		},
		{
			Name:    string(CommandShow),
			Summary: "Show one engagement manifest",
			Group:   groupEngagement,
			Args: commandtree.ArgSchema{
				Options: []commandtree.OptionArg{scenarioOption(), slugOption(), commandtree.JSONOption()},
			},
			Handler: CommandShow,
		},
		{
			Name:    string(CommandList),
			Summary: "List every active engagement (the status source)",
			Group:   groupEngagement,
			Args: commandtree.ArgSchema{
				Options: []commandtree.OptionArg{commandtree.JSONOption()},
			},
			Handler: CommandList,
		},
		{
			Name:    string(CommandTouch),
			Summary: "Renew an engagement lease (touch-on-access)",
			Group:   groupEngagement,
			Args: commandtree.ArgSchema{
				Options: []commandtree.OptionArg{scenarioOption(), slugOption(), commandtree.JSONOption()},
			},
			Handler: CommandTouch,
		},
		{
			Name:    string(CommandSetTTL),
			Summary: "Adjust an engagement's idle TTL",
			Group:   groupEngagement,
			Args: commandtree.ArgSchema{
				Options: []commandtree.OptionArg{
					scenarioOption(),
					slugOption(),
					{Name: "--ttl", ValueName: "dur", Description: "New idle TTL (e.g. 6h); 0 clears it"},
					commandtree.JSONOption(),
				},
			},
			Handler: CommandSetTTL,
		},
		{
			Name:    string(CommandSetMode),
			Summary: "Flip an engagement's mode in place (the non-lossy promote re-point lever; preserves the restore point)",
			Group:   groupEngagement,
			Args: commandtree.ArgSchema{
				Options: []commandtree.OptionArg{
					scenarioOption(),
					slugOption(),
					{Name: "--mode", ValueName: "mode", Description: "New mode (shadow|live)"},
					commandtree.JSONOption(),
				},
			},
			Handler: CommandSetMode,
		},
		{
			Name:    string(CommandClean),
			Summary: "Tear down an engagement (restore point + manifest)",
			Group:   groupEngagement,
			Args: commandtree.ArgSchema{
				Options: []commandtree.OptionArg{scenarioOption(), slugOption(), commandtree.JSONOption()},
			},
			Handler: CommandClean,
		},
		{
			Name:    string(CommandMigrate),
			Summary: "Apply an engagement's managed schema migrations to a database (transactional, dry-run-first)",
			Group:   groupMigration,
			Args: commandtree.ArgSchema{
				Options: []commandtree.OptionArg{
					scenarioOption(),
					slugOption(),
					{Name: "--engine", ValueName: "engine", Description: "Storage engine (default sqlite; v1 supports sqlite only)"},
					{Name: "--db-path", ValueName: "path", Description: "Target database file (SQLite auto-resolves from the scenario's data dir when omitted)"},
					{Name: "--migrations-dir", ValueName: "path", Description: "Override the engagement's managed migrations folder"},
					{Name: "--dry-run", Description: "Validate against a throwaway copy without mutating the real database"},
					commandtree.JSONOption(),
				},
			},
			Handler: CommandMigrate,
		},
		{
			Name:    string(CommandNamespace),
			Summary: "Resolve a scenario variant's SSOT storage namespaces (the shadow-population query)",
			Group:   groupAddressing,
			Args: commandtree.ArgSchema{
				Options: []commandtree.OptionArg{
					scenarioOption(),
					{Name: "--variant", ValueName: "variant", Description: "Instance variant (default shadow)"},
					commandtree.JSONOption(),
				},
			},
			Handler: CommandNamespace,
		},
	}
}

func RenderCommandHelp(w io.Writer) {
	commandtree.RenderHelp(w, commandtree.Help{
		Title:        "Vrooli Recovery Floor Commands",
		Description:  "Trusted-base primitives for Baseline Modes: the git-free restore point and the floor-owned engagement manifest. git-control-tower baseline shells into these.",
		Usage:        "vrooli recovery <subcommand> [options]",
		DefaultGroup: groupEngagement,
	}, CommandSpecs())
}

func ParseCaptureRequest(args []string) (recoveryapp.CaptureRequest, error) {
	parsed, err := parse(CommandCapture, "recovery capture", args)
	if err != nil {
		return recoveryapp.CaptureRequest{}, err
	}
	return recoveryapp.CaptureRequest{
		Scenario:  parsed.FlagValue("--scenario"),
		Slug:      parsed.FlagValue("--slug"),
		Source:    parsed.FlagValue("--source"),
		NoReflink: parsed.HasFlag("--no-reflink"),
	}, nil
}

func ParseRestoreRequest(args []string) (recoveryapp.RestoreRequest, error) {
	parsed, err := parse(CommandRestore, "recovery restore", args)
	if err != nil {
		return recoveryapp.RestoreRequest{}, err
	}
	return recoveryapp.RestoreRequest{
		Scenario:  parsed.FlagValue("--scenario"),
		Slug:      parsed.FlagValue("--slug"),
		Dest:      parsed.FlagValue("--dest"),
		NoReflink: parsed.HasFlag("--no-reflink"),
	}, nil
}

func ParseWriteRequest(args []string) (recoveryapp.WriteRequest, error) {
	parsed, err := parse(CommandWrite, "recovery write", args)
	if err != nil {
		return recoveryapp.WriteRequest{}, err
	}
	ttl, err := parseTTL(parsed.FlagValue("--ttl"))
	if err != nil {
		return recoveryapp.WriteRequest{}, err
	}
	return recoveryapp.WriteRequest{
		Scenario:          parsed.FlagValue("--scenario"),
		Slug:              parsed.FlagValue("--slug"),
		Mode:              parsed.FlagValue("--mode"),
		Variant:           parsed.FlagValue("--variant"),
		TTL:               ttl,
		AmbientVar:        parsed.FlagValue("--ambient-var"),
		ShadowInstanceKey: parsed.FlagValue("--shadow-instance-key"),
		Anchor:            parsed.FlagValue("--anchor"),
	}, nil
}

func ParseMigrateRequest(args []string) (recoveryapp.MigrateRequest, error) {
	parsed, err := parse(CommandMigrate, "recovery migrate", args)
	if err != nil {
		return recoveryapp.MigrateRequest{}, err
	}
	return recoveryapp.MigrateRequest{
		Scenario:      parsed.FlagValue("--scenario"),
		Slug:          parsed.FlagValue("--slug"),
		Engine:        parsed.FlagValue("--engine"),
		DBPath:        parsed.FlagValue("--db-path"),
		MigrationsDir: parsed.FlagValue("--migrations-dir"),
		DryRun:        parsed.HasFlag("--dry-run"),
	}, nil
}

func ParseNamespaceRequest(args []string) (recoveryapp.NamespaceRequest, error) {
	parsed, err := parse(CommandNamespace, "recovery namespace", args)
	if err != nil {
		return recoveryapp.NamespaceRequest{}, err
	}
	// This query exists to build shadow-population mappings, so an omitted
	// --variant defaults to "shadow" rather than the InstanceKey "live" default.
	variant := parsed.FlagValue("--variant")
	if variant == "" {
		variant = recoveryDefaultVariant
	}
	return recoveryapp.NamespaceRequest{
		Scenario: parsed.FlagValue("--scenario"),
		Variant:  variant,
	}, nil
}

func ParseRefRequest(id CommandID, command string, args []string) (recoveryapp.Ref, error) {
	parsed, err := parse(id, command, args)
	if err != nil {
		return recoveryapp.Ref{}, err
	}
	return recoveryapp.Ref{Scenario: parsed.FlagValue("--scenario"), Slug: parsed.FlagValue("--slug")}, nil
}

func ParseSetTTLRequest(args []string) (recoveryapp.SetTTLRequest, error) {
	parsed, err := parse(CommandSetTTL, "recovery set-ttl", args)
	if err != nil {
		return recoveryapp.SetTTLRequest{}, err
	}
	ttl, err := parseTTL(parsed.FlagValue("--ttl"))
	if err != nil {
		return recoveryapp.SetTTLRequest{}, err
	}
	return recoveryapp.SetTTLRequest{
		Scenario: parsed.FlagValue("--scenario"),
		Slug:     parsed.FlagValue("--slug"),
		TTL:      ttl,
	}, nil
}

func ParseSetModeRequest(args []string) (recoveryapp.SetModeRequest, error) {
	parsed, err := parse(CommandSetMode, "recovery set-mode", args)
	if err != nil {
		return recoveryapp.SetModeRequest{}, err
	}
	return recoveryapp.SetModeRequest{
		Scenario: parsed.FlagValue("--scenario"),
		Slug:     parsed.FlagValue("--slug"),
		Mode:     parsed.FlagValue("--mode"),
	}, nil
}

func parse(id CommandID, command string, args []string) (commandtree.ParsedArgs, error) {
	return commandtree.ParseArgs(command, commandHelpText(id), commandSpec(id).Args, args)
}

func parseTTL(raw string) (time.Duration, error) {
	if raw == "" {
		return 0, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid --ttl %q: %w", raw, err)
	}
	return d, nil
}

func commandSpec(id CommandID) commandtree.Spec[CommandID] {
	for _, spec := range CommandSpecs() {
		if spec.Handler == id {
			return spec
		}
	}
	panic("unknown recovery command spec: " + string(id))
}

func commandHelpText(id CommandID) string {
	spec := commandSpec(id)
	return commandtree.SpecHelpText("", "vrooli recovery "+spec.Name, spec)
}
