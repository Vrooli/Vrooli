package capacitycli

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	capacityapp "github.com/vrooli/vrooli/internal/app/capacity"
	"github.com/vrooli/vrooli/internal/cli/clipolicy"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
)

const (
	commandsParameterA = 3
)

type CommandID string

const (
	CommandClaim            CommandID = "claim"
	CommandHeartbeat        CommandID = "heartbeat"
	CommandActivity         CommandID = "activity"
	CommandRelease          CommandID = "release"
	CommandDegrade          CommandID = "degrade"
	CommandResize           CommandID = "resize"
	CommandList             CommandID = "list"
	CommandReconcile        CommandID = "reconcile"
	CommandSweep            CommandID = "sweep"
	CommandGC               CommandID = "gc"
	CommandRecommend        CommandID = "recommend"
	CommandFootprint        CommandID = "footprint"
	CommandFit              CommandID = "fit"
	CommandPolicy           CommandID = "policy"
	CommandDiskReserve      CommandID = "disk-reserve"
	CommandDiskHeartbeat    CommandID = "disk-heartbeat"
	CommandDiskExtend       CommandID = "disk-extend"
	CommandDiskComplete     CommandID = "disk-complete"
	CommandDiskList         CommandID = "disk-list"
	CommandDiskReservations CommandID = "disk-reservations"
	CommandDiskReconcile    CommandID = "disk-reconcile"
	CommandDiskObserve      CommandID = "disk-observe"
	CommandDiskSample       CommandID = "disk-sample"
	CommandResidency        CommandID = "residency"
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
					{Name: "--owner-id", ValueName: "id", Description: "Required stable owner identity (e.g. whisper, image-tools:job-1)"},
					{Name: "--instance-id", ValueName: "id", Description: "Linked scenarioruntime instance id (optional)"},
					{Name: "--resource-kind", ValueName: "kind", Description: "vram|ram|cpu (default vram)"},
					{Name: "--gpu-index", ValueName: "n", Description: "GPU index for vram (default 0)"},
					{Name: "--preferred", ValueName: "amount", Description: "Preferred amount: bytes/7GiB for memory; millicores for cpu (4000 = 4 cores)"},
					{Name: "--floor", ValueName: "amount", Description: "Minimum amount: bytes/1GiB for memory; millicores for cpu (1000 = 1 core)"},
					{Name: "--priority", ValueName: "tier", Description: "interactive|service|batch (default batch)"},
					{Name: "--no-reclaim", Description: "Use only already-free capacity; never reclaim another owner's claim"},
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
				Options: []commandtree.OptionArg{claimIDOption(), generationOption(), {Name: "--to", ValueName: "label", Description: "Target profile step label"}, {Name: "--amount", ValueName: "amount", Description: "Explicit amount in the claim units: memory bytes or CPU millicores"}, commandtree.JSONOption()},
			},
			Handler: CommandDegrade,
		},
		{
			Name:    string(CommandResize),
			Summary: "Resize a claim in place when the owner's real footprint changes",
			Group:   groupClaims,
			Args: commandtree.ArgSchema{
				Options: []commandtree.OptionArg{claimIDOption(), generationOption(), {Name: "--amount", ValueName: "amount", Description: "New granted and preferred amount: memory bytes or CPU millicores"}, commandtree.JSONOption()},
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
				Options: []commandtree.OptionArg{{Name: "--owner", ValueName: "id", Description: "Filter to one owner id"}, {Name: "--active", Description: "Only active (reserved/granted/degraded) claims"}, {Name: "--all", Description: "Include terminal history (default shows live claims)"}, commandtree.JSONOption()},
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
			Name:    string(CommandFit),
			Summary: "Evaluate enabled resources against host capacity; exit 1 carries an actionable non-fit verdict",
			Group:   groupObserve,
			Args: commandtree.ArgSchema{
				Options: []commandtree.OptionArg{{Name: "--simulate-host", ValueName: "facts", Description: "Override host facts: vram=8GiB[,backends=cuda+cpu][,compute=8.9]"}, commandtree.JSONOption()},
			},
			Handler: CommandFit,
		},
		{
			Name:    string(CommandFootprint),
			Summary: "List or reset durable measured footprint high-water marks",
			Group:   groupObserve,
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "action", Description: "list|reset"}},
				Options:     []commandtree.OptionArg{{Name: "--resource", ValueName: "name", Description: "Filter or reset one resource"}, commandtree.JSONOption()},
			},
			Handler: CommandFootprint,
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
		{
			Name: string(CommandDiskReserve), Summary: "Atomically reserve a multi-filesystem disk job from a JSON request file", Group: groupClaims,
			Args: commandtree.ArgSchema{Options: []commandtree.OptionArg{{Name: "--request-file", ValueName: "path", Description: "JSON DiskJobRequest file"}, {Name: "--lease-file", ValueName: "path", Description: "Private 0600 file for lease credential"}, commandtree.JSONOption()}}, Handler: CommandDiskReserve,
		},
		{
			Name: string(CommandDiskHeartbeat), Summary: "Renew an active disk job lease using its private lease file", Group: groupClaims,
			Args: commandtree.ArgSchema{Options: []commandtree.OptionArg{{Name: "--lease-file", ValueName: "path", Description: "Private disk job lease credential file"}, commandtree.JSONOption()}}, Handler: CommandDiskHeartbeat,
		},
		{
			Name: string(CommandDiskExtend), Summary: "Atomically extend a disk job lease before further growth", Group: groupClaims,
			Args: commandtree.ArgSchema{Options: []commandtree.OptionArg{{Name: "--lease-file", ValueName: "path", Description: "Private disk job lease credential file"}, {Name: "--paths-file", ValueName: "path", Description: "JSON array of added/expanded paths"}, commandtree.JSONOption()}}, Handler: CommandDiskExtend,
		},
		{
			Name: string(CommandDiskComplete), Summary: "Complete a disk job after output and cleanup are verified", Group: groupClaims,
			Args: commandtree.ArgSchema{Options: []commandtree.OptionArg{{Name: "--lease-file", ValueName: "path", Description: "Private disk job lease credential file"}, {Name: "--cleanup-verified", Description: "Assert output publication and temporary cleanup are verified"}, commandtree.JSONOption()}}, Handler: CommandDiskComplete,
		},
		{
			Name: string(CommandDiskList), Summary: "List durable disk jobs or refresh leased-filesystem headroom", Group: groupObserve,
			Args: commandtree.ArgSchema{Options: []commandtree.OptionArg{{Name: "--active", Description: "Only active disk jobs"}, {Name: "--observe", Description: "Refresh aggregate headroom for every filesystem with an active disk lease; observational only, not an admission"}, commandtree.JSONOption()}}, Handler: CommandDiskList,
		},
		{
			Name: string(CommandDiskReservations), Summary: "List legacy single-path disk reservation receipts", Group: groupObserve,
			Args: commandtree.ArgSchema{Options: []commandtree.OptionArg{{Name: "--active", Description: "Only active reservations"}, {Name: "--owner", ValueName: "id", Description: "Filter by reservation owner"}, {Name: "--filesystem", ValueName: "id", Description: "Filter by filesystem id"}, commandtree.JSONOption()}}, Handler: CommandDiskReservations,
		},
		{
			Name: string(CommandDiskReconcile), Summary: "Release a stale disk job after process stop and cleanup verification", Group: groupObserve,
			Args: commandtree.ArgSchema{Options: []commandtree.OptionArg{{Name: "--lease-id", ValueName: "id", Description: "Disk job lease id"}, {Name: "--process-stopped", Description: "Assert the owner process was stopped"}, {Name: "--cleanup-verified", Description: "Assert the owner paths were reconciled"}, commandtree.JSONOption()}}, Handler: CommandDiskReconcile,
		},
		{
			Name: string(CommandDiskObserve), Summary: "Observe free bytes and safe additional lease headroom", Group: groupObserve,
			Args: commandtree.ArgSchema{Options: []commandtree.OptionArg{{Name: "--lease-file", ValueName: "path", Description: "Private disk job lease credential file"}, commandtree.JSONOption()}}, Handler: CommandDiskObserve,
		},
		{
			Name: string(CommandDiskSample), Summary: "Record an owner-measured path baseline/checkpoint, optionally extending atomically", Group: groupClaims,
			Args: commandtree.ArgSchema{Options: []commandtree.OptionArg{{Name: "--lease-file", ValueName: "path", Description: "Private disk job lease credential file"}, {Name: "--path", ValueName: "path", Description: "Exact declared write path"}, {Name: "--bytes", ValueName: "n", Description: "Owner-measured current path bytes"}, {Name: "--extend-peak", ValueName: "n", Description: "Optional new total peak for this path"}, {Name: "--docker-container", ValueName: "name-or-id", Description: "Docker container authoritatively resolving an inaccessible host path"}, {Name: "--docker-destination", ValueName: "path", Description: "Container mount destination (omit only with --docker-upper-dir or --docker-root-dir)"}, {Name: "--docker-upper-dir", Description: "Resolve the container's daemon-reported writable upper directory"}, {Name: "--docker-root-dir", Description: "Resolve Docker daemon root for pre-provision admission"}, commandtree.JSONOption()}}, Handler: CommandDiskSample,
		},
		{
			Name: string(CommandResidency), Summary: "Persist or replay owner-measured model residency transitions", Group: groupObserve,
			Args: commandtree.ArgSchema{Positionals: []commandtree.PositionalArg{{Name: "action", Description: "begin|complete|reconcile|recover|get|history|release"}}, Options: []commandtree.OptionArg{{Name: "--request-file", ValueName: "path", Description: "Owner-authored transition JSON for begin/complete/reconcile/recover; exact owner/claim/model release JSON for release"}, {Name: "--transition-id", ValueName: "id", Description: "Exact transition identity for get/history"}, commandtree.JSONOption()}}, Handler: CommandResidency,
		},
	}
}

func ParseFitRequest(args []string) (capacityapp.FitRequest, error) {
	parsed, err := parse(CommandFit, "vrooli capacity fit", args)
	if err != nil {
		return capacityapp.FitRequest{}, err
	}
	raw := strings.TrimSpace(parsed.FlagValue("--simulate-host"))
	if raw == "" {
		return capacityapp.FitRequest{}, nil
	}
	out := capacityapp.FitRequest{}
	for _, field := range strings.Split(raw, ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(field), "=")
		if !ok || strings.TrimSpace(value) == "" {
			return capacityapp.FitRequest{}, fmt.Errorf("invalid --simulate-host field %q", field)
		}
		switch strings.TrimSpace(key) {
		case "vram":
			bytes, err := parseBytes(value)
			if err != nil {
				return capacityapp.FitRequest{}, fmt.Errorf("invalid simulated vram: %w", err)
			}
			out.SimulatedVRAM = &bytes
		case "backends":
			value = strings.NewReplacer("|", "+", ";", "+").Replace(value)
			for _, backend := range strings.Split(value, "+") {
				if backend = strings.TrimSpace(backend); backend != "" {
					out.SimulatedBackends = append(out.SimulatedBackends, backend)
				}
			}
		case "compute":
			out.SimulatedCompute = strings.TrimSpace(value)
		default:
			return capacityapp.FitRequest{}, fmt.Errorf("unknown --simulate-host fact %q", key)
		}
	}
	if out.SimulatedVRAM == nil {
		return capacityapp.FitRequest{}, fmt.Errorf("--simulate-host requires vram=<size>")
	}
	return out, nil
}

func ParseFootprintRequest(args []string) (capacityapp.FootprintRequest, error) {
	parsed, err := parse(CommandFootprint, "vrooli capacity footprint", args)
	if err != nil {
		return capacityapp.FootprintRequest{}, err
	}
	if len(parsed.Positionals) != 1 || (parsed.Positionals[0] != "list" && parsed.Positionals[0] != "reset") {
		return capacityapp.FootprintRequest{}, fmt.Errorf("footprint requires exactly one action: list|reset")
	}
	return capacityapp.FootprintRequest{Action: parsed.Positionals[0], Resource: parsed.FlagValue("--resource")}, nil
}

func parse(id CommandID, command string, args []string) (commandtree.ParsedArgs, error) {
	command = strings.TrimPrefix(strings.TrimSpace(command), "vrooli ")
	return commandtree.ParseArgs(command, commandHelpText(id), commandSpec(id).Args, args)
}

// ParseClaimRequest parses `vrooli capacity claim`.
func ParseClaimRequest(args []string) (capacityapp.ClaimRequest, error) {
	parsed, err := parse(CommandClaim, "vrooli capacity claim", args)
	if err != nil {
		return capacityapp.ClaimRequest{}, err
	}
	if strings.TrimSpace(parsed.FlagValue("--owner-id")) == "" {
		return capacityapp.ClaimRequest{}, clipolicy.UsageErrorf("capacity claim", "--owner-id is required (for example --owner-kind resource --owner-id image-tools:job-1)")
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
		NoReclaim:      parsed.HasFlag("--no-reclaim"),
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
	command = strings.TrimPrefix(strings.TrimSpace(command), "vrooli ")
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
		return capacityapp.Ref{}, clipolicy.UsageErrorf(command, "--claim-id is required")
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
		AllHistory: parsed.HasFlag("--all"),
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
		Description:  "Host resource claim/lease arbitration (GPU VRAM, RAM, CPU). Resources, scenarios, and the lifecycle claim against a contention-aware ledger; idleness is reported by the work-owner, never inferred. Amounts use bytes for memory and millicores for CPU (4000 = 4 cores).",
		Usage:        "vrooli capacity <subcommand> [options]",
		DefaultGroup: groupClaims,
	}, CommandSpecs())
}
