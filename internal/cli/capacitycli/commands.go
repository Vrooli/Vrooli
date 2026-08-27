package capacitycli

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	capacityapp "github.com/vrooli/vrooli/internal/app/capacity"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
)

const (
	commandsParameterA = 3
)

type CommandID string

const (
	CommandClaim     CommandID = "claim"
	CommandHeartbeat CommandID = "heartbeat"
	CommandActivity  CommandID = "activity"
	CommandRelease   CommandID = "release"
	CommandDegrade   CommandID = "degrade"
	CommandResize    CommandID = "resize"
	CommandList      CommandID = "list"
	CommandReconcile CommandID = "reconcile"
	CommandSweep     CommandID = "sweep"
	CommandGC        CommandID = "gc"
	CommandRecommend CommandID = "recommend"
	CommandPolicy    CommandID = "policy"
)

const (
	groupClaims  = "Claims"
	groupObserve = "Observe"
	groupPolicy  = "Policy"
)

func claimIDOption() commandtree.OptionArg {
	return commandtree.OptionArg{Name: "--claim-id", ValueName: "id", Description: "Capacity claim id"}
}

func generationOption() commandtree.OptionArg {
	return commandtree.OptionArg{Name: "--generation", ValueName: "n", Description: "Expected generation (optimistic concurrency guard)"}
}

// CommandSpecs declares the `vrooli capacity` verb surface (plan §8.4).
func CommandSpecs() []commandtree.Spec[CommandID] {
	return []commandtree.Spec[CommandID]{
		{
			Name:    string(CommandClaim),
			Summary: "Claim host capacity (vram/ram/cpu); returns the admission verdict",
			Group:   groupClaims,
			Args: commandtree.ArgSchema{
				Options: []commandtree.OptionArg{
					{Name: "--owner-kind", ValueName: "kind", Description: "resource|scenario|op"},
					{Name: "--owner-id", ValueName: "id", Description: "Owner identity (e.g. whisper, image-tools:job-1)"},
					{Name: "--instance-id", ValueName: "id", Description: "Linked scenarioruntime instance id (optional)"},
					{Name: "--resource-kind", ValueName: "kind", Description: "vram|ram|cpu (default vram)"},
					{Name: "--gpu-index", ValueName: "n", Description: "GPU index for vram (default 0)"},
					{Name: "--preferred", ValueName: "bytes", Description: "Preferred amount (bytes or e.g. 7GiB)"},
					{Name: "--floor", ValueName: "bytes", Description: "Minimum viable amount (bytes or e.g. 1GiB)"},
					{Name: "--priority", ValueName: "tier", Description: "interactive|service|batch (default batch)"},
					{Name: "--protected", Description: "Never preempt/degrade while active"},
					{Name: "--yield-when-idle", Description: "When idle beyond grace, yield capacity to active work at/above idle_yield_floor"},
					{Name: "--idle-grace", ValueName: "dur", Description: "Demand-reclaim warm-idle dwell before this claim can become cold-idle (e.g. 15m; default policy idle_grace)"},
					{Name: "--idle-unload-ttl", ValueName: "dur", Description: "Autonomously unload to floor after this much idle dwell (e.g. 30m; 0 = off)"},
					{Name: "--profile", ValueName: "json", Description: "Degradation profile JSON"},
					{Name: "--ttl", ValueName: "dur", Description: "Heartbeat TTL (e.g. 30s)"},
					commandtree.JSONOption(),
				},
			},
			Handler: CommandClaim,
		},
		{
			Name:    string(CommandHeartbeat),
			Summary: "Renew a claim's liveness (does not change activity)",
			Group:   groupClaims,
			Args: commandtree.ArgSchema{
				Options: []commandtree.OptionArg{claimIDOption(), generationOption(), {Name: "--ttl", ValueName: "dur", Description: "Heartbeat TTL"}, commandtree.JSONOption()},
			},
			Handler: CommandHeartbeat,
		},
		{
			Name:    string(CommandActivity),
			Summary: "Report work-owner activity state (the idleness truth source)",
			Group:   groupClaims,
			Args: commandtree.ArgSchema{
				Options: []commandtree.OptionArg{claimIDOption(), generationOption(), {Name: "--state", ValueName: "state", Description: "active|idle"}, commandtree.JSONOption()},
			},
			Handler: CommandActivity,
		},
		{
			Name:    string(CommandDegrade),
			Summary: "Step a claim down to a profile rung (broker->adopter callback target)",
			Group:   groupClaims,
			Args: commandtree.ArgSchema{
				Options: []commandtree.OptionArg{claimIDOption(), generationOption(), {Name: "--to", ValueName: "label", Description: "Target profile step label"}, {Name: "--amount", ValueName: "bytes", Description: "Explicit amount (overrides profile lookup)"}, commandtree.JSONOption()},
			},
			Handler: CommandDegrade,
		},
		{
			Name:    string(CommandResize),
			Summary: "Resize a claim in place when the owner's real footprint changes",
			Group:   groupClaims,
			Args: commandtree.ArgSchema{
				Options: []commandtree.OptionArg{claimIDOption(), generationOption(), {Name: "--amount", ValueName: "bytes", Description: "New granted and preferred amount in bytes"}, commandtree.JSONOption()},
			},
			Handler: CommandResize,
		},
		{
			Name:    string(CommandRelease),
			Summary: "Release a claim",
			Group:   groupClaims,
			Args: commandtree.ArgSchema{
				Options: []commandtree.OptionArg{claimIDOption(), commandtree.JSONOption()},
			},
			Handler: CommandRelease,
		},
		{
			Name:    string(CommandList),
			Summary: "List capacity claims",
			Group:   groupObserve,
			Args: commandtree.ArgSchema{
				Options: []commandtree.OptionArg{{Name: "--owner", ValueName: "id", Description: "Filter to one owner id"}, {Name: "--active", Description: "Only active (reserved/granted/degraded) claims"}, commandtree.JSONOption()},
			},
			Handler: CommandList,
		},
		{
			Name:    string(CommandReconcile),
			Summary: "Classify observed GPU consumers against the ledger (warns on unclaimed)",
			Group:   groupObserve,
			Args: commandtree.ArgSchema{
				Options: []commandtree.OptionArg{commandtree.JSONOption()},
			},
			Handler: CommandReconcile,
		},
		{
			Name:    string(CommandSweep),
			Summary: "Heartbeat resident claims still observed on the GPU; expire the rest (the resident-claim liveness driver)",
			Group:   groupObserve,
			Args: commandtree.ArgSchema{
				Options: []commandtree.OptionArg{commandtree.JSONOption()},
			},
			Handler: CommandSweep,
		},
		{
			Name:    string(CommandGC),
			Summary: "Prune terminal (released/expired/preempted) claims past terminal_retention",
			Group:   groupObserve,
			Args: commandtree.ArgSchema{
				Options: []commandtree.OptionArg{commandtree.JSONOption()},
			},
			Handler: CommandGC,
		},
		{
			Name:    string(CommandRecommend),
			Summary: "Advisory right-sizing: flag claims whose observed peak is well below their reservation",
			Group:   groupObserve,
			Args: commandtree.ArgSchema{
				Options: []commandtree.OptionArg{{Name: "--owner", ValueName: "id", Description: "Filter to one owner id"}, commandtree.JSONOption()},
			},
			Handler: CommandRecommend,
		},
		{
			Name:    string(CommandPolicy),
			Summary: "Get or set tunable policy levers: policy {get,set} <key> [value]",
			Group:   groupPolicy,
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{
					{Name: "action", Description: "get|set"},
					{Name: "key", Description: "policy lever key (optional for get)"},
					{Name: "value", Description: "new value (set only)", Repeatable: true},
				},
				Options: []commandtree.OptionArg{commandtree.JSONOption()},
			},
			Handler: CommandPolicy,
		},
	}
}

func parse(id CommandID, command string, args []string) (commandtree.ParsedArgs, error) {
	return commandtree.ParseArgs(command, commandHelpText(id), commandSpec(id).Args, args)
}

// ParseClaimRequest parses `vrooli capacity claim`.
func ParseClaimRequest(args []string) (capacityapp.ClaimRequest, error) {
	parsed, err := parse(CommandClaim, "vrooli capacity claim", args)
	if err != nil {
		return capacityapp.ClaimRequest{}, err
	}
	preferred, err := parseBytes(parsed.FlagValue("--preferred"))
	if err != nil {
		return capacityapp.ClaimRequest{}, err
	}
	floor, err := parseBytes(parsed.FlagValue("--floor"))
	if err != nil {
		return capacityapp.ClaimRequest{}, err
	}
	gpuIndex, err := parseOptionalInt(parsed.FlagValue("--gpu-index"))
	if err != nil {
		return capacityapp.ClaimRequest{}, err
	}
	ttl, err := parseTTL(parsed.FlagValue("--ttl"))
	if err != nil {
		return capacityapp.ClaimRequest{}, err
	}
	idleUnloadTTL, err := parseTTL(parsed.FlagValue("--idle-unload-ttl"))
	if err != nil {
		return capacityapp.ClaimRequest{}, err
	}
	idleGrace, err := parseTTL(parsed.FlagValue("--idle-grace"))
	if err != nil {
		return capacityapp.ClaimRequest{}, err
	}
	return capacityapp.ClaimRequest{
		OwnerKind:      parsed.FlagValue("--owner-kind"),
		OwnerID:        parsed.FlagValue("--owner-id"),
		InstanceID:     parsed.FlagValue("--instance-id"),
		ResourceKind:   parsed.FlagValue("--resource-kind"),
		GPUIndex:       gpuIndex,
		PreferredBytes: preferred,
		FloorBytes:     floor,
		PriorityTier:   parsed.FlagValue("--priority"),
		Protected:      parsed.HasFlag("--protected"),
		YieldWhenIdle:  parsed.HasFlag("--yield-when-idle"),
		IdleUnloadTTL:  idleUnloadTTL,
		IdleGrace:      idleGrace,
		ProfileJSON:    parsed.FlagValue("--profile"),
		TTL:            ttl,
	}, nil
}

// ParseRefRequest parses a claim-id + generation + verb-specific flags.
func ParseRefRequest(id CommandID, command string, args []string) (capacityapp.Ref, error) {
	parsed, err := parse(id, command, args)
	if err != nil {
		return capacityapp.Ref{}, err
	}
	ref := capacityapp.Ref{
		ClaimID: parsed.FlagValue("--claim-id"),
		State:   parsed.FlagValue("--state"),
		ToStep:  parsed.FlagValue("--to"),
		Reason:  parsed.FlagValue("--reason"),
	}
	if strings.TrimSpace(ref.ClaimID) == "" {
		return capacityapp.Ref{}, fmt.Errorf("--claim-id is required")
	}
	if gen := parsed.FlagValue("--generation"); gen != "" {
		n, convErr := strconv.ParseInt(gen, 10, 64)
		if convErr != nil {
			return capacityapp.Ref{}, fmt.Errorf("invalid --generation %q: %w", gen, convErr)
		}
		ref.Generation = n
	}
	if amt := parsed.FlagValue("--amount"); amt != "" {
		n, convErr := parseBytes(amt)
		if convErr != nil {
			return capacityapp.Ref{}, convErr
		}
		ref.AmountByte = n
	}
	if ttl := parsed.FlagValue("--ttl"); ttl != "" {
		d, convErr := parseTTL(ttl)
		if convErr != nil {
			return capacityapp.Ref{}, convErr
		}
		ref.TTL = d
	}
	return ref, nil
}

// ParseListRequest parses `vrooli capacity list`.
func ParseListRequest(args []string) (capacityapp.ListRequest, error) {
	parsed, err := parse(CommandList, "vrooli capacity list", args)
	if err != nil {
		return capacityapp.ListRequest{}, err
	}
	return capacityapp.ListRequest{
		OwnerID:    parsed.FlagValue("--owner"),
		ActiveOnly: parsed.HasFlag("--active"),
	}, nil
}

// ParseSweepRequest validates `vrooli capacity sweep` (only --json/help).
func ParseSweepRequest(args []string) error {
	_, err := parse(CommandSweep, "vrooli capacity sweep", args)
	return err
}

// ParseGCRequest validates `vrooli capacity gc` (only --json/help).
func ParseGCRequest(args []string) error {
	_, err := parse(CommandGC, "vrooli capacity gc", args)
	return err
}

// ParseRecommendRequest parses `vrooli capacity recommend [--owner X]`.
func ParseRecommendRequest(args []string) (capacityapp.RecommendRequest, error) {
	parsed, err := parse(CommandRecommend, "vrooli capacity recommend", args)
	if err != nil {
		return capacityapp.RecommendRequest{}, err
	}
	return capacityapp.RecommendRequest{OwnerID: parsed.FlagValue("--owner")}, nil
}

// PolicyArgs is the parsed policy action.
type PolicyArgs struct {
	Action string
	Key    string
	Value  string
}

// ParsePolicyRequest parses `vrooli capacity policy {get,set} <key> [value]`.
func ParsePolicyRequest(args []string) (PolicyArgs, error) {
	parsed, err := parse(CommandPolicy, "vrooli capacity policy", args)
	if err != nil {
		return PolicyArgs{}, err
	}
	pos := parsed.Positionals
	if len(pos) == 0 {
		return PolicyArgs{}, fmt.Errorf("policy requires an action: get|set")
	}
	out := PolicyArgs{Action: pos[0]}
	switch out.Action {
	case "get":
		if len(pos) > 1 {
			out.Key = pos[1]
		}
	case "set":
		if len(pos) < commandsParameterA {
			return PolicyArgs{}, fmt.Errorf("policy set requires <key> <value>")
		}
		out.Key = pos[1]
		out.Value = strings.Join(pos[2:], " ")
	default:
		return PolicyArgs{}, fmt.Errorf("unknown policy action %q (want get|set)", out.Action)
	}
	return out, nil
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

func parseOptionalInt(raw string) (*int, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("invalid integer %q: %w", raw, err)
	}
	return &n, nil
}

// parseBytes accepts a raw byte count or a human suffix (KiB/MiB/GiB/TiB and
// their decimal KB/MB/GB/TB forms). An empty string is 0.
func parseBytes(raw string) (int64, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, nil
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n, nil
	}
	upper := strings.ToUpper(s)
	multipliers := []struct {
		suffix string
		mult   int64
	}{
		{"TIB", 1 << 40},
		{"GIB", 1 << 30},
		{"MIB", 1 << 20},
		{"KIB", 1 << 10},
		{"TB", 1e12},
		{"GB", 1e9},
		{"MB", 1e6},
		{"KB", 1e3},
		{"B", 1},
	}
	for _, m := range multipliers {
		if strings.HasSuffix(upper, m.suffix) {
			numPart := strings.TrimSpace(upper[:len(upper)-len(m.suffix)])
			f, err := strconv.ParseFloat(numPart, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid byte amount %q: %w", raw, err)
			}
			return int64(f * float64(m.mult)), nil
		}
	}
	return 0, fmt.Errorf("invalid byte amount %q", raw)
}

func commandSpec(id CommandID) commandtree.Spec[CommandID] {
	for _, spec := range CommandSpecs() {
		if spec.Handler == id {
			return spec
		}
	}
	panic("unknown capacity command spec: " + string(id))
}

func commandHelpText(id CommandID) string {
	spec := commandSpec(id)
	return commandtree.SpecHelpText("", "vrooli capacity "+spec.Name, spec)
}

// RenderCommandHelp renders `vrooli capacity` group help.
func RenderCommandHelp(w io.Writer) {
	commandtree.RenderHelp(w, commandtree.Help{
		Title:        "Vrooli Capacity Broker Commands",
		Description:  "Host resource claim/lease arbitration (GPU VRAM, RAM, CPU). Resources, scenarios, and the lifecycle claim against a contention-aware ledger; idleness is reported by the work-owner, never inferred.",
		Usage:        "vrooli capacity <subcommand> [options]",
		DefaultGroup: groupClaims,
	}, CommandSpecs())
}
