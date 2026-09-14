package teams

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"reflect"
	"strings"

	"prompt-manager/cli/internal/appctx"
	"prompt-manager/internal/teamconfig"

	"github.com/vrooli/cli-core/cliutil"
)

// Binding input is configuration only. Enabled cannot be smuggled into a
// provisioning request; activation remains the normal, separate owner control.
type finiteLeaderBindingInput struct {
	Schedule     string                   `json:"schedule"`
	ProfileKey   string                   `json:"profileKey"`
	FiniteLeader *teamconfig.FiniteLeader `json:"finiteLeader"`
}

func cmdHeartbeatBindEffort(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("heartbeat-bind-effort", flag.ContinueOnError)
	file := fs.String("request-file", "", "JSON binding: schedule, profileKey, finiteLeader (always provisioned disabled)")
	update := fs.Bool("update", false, "Bind an existing unused disabled heartbeat")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 2 || *file == "" {
		return fmt.Errorf("usage: team heartbeat-bind-effort <team-id> <agent-id> --request-file <binding.json> [--update] [--json]")
	}
	reader, err := os.Open(*file)
	if err != nil {
		return err
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, 16385))
	if err != nil {
		return err
	}
	if len(data) > 16384 {
		return fmt.Errorf("finite leader binding exceeds 16 KiB")
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	var input finiteLeaderBindingInput
	if err := decoder.Decode(&input); err != nil {
		return fmt.Errorf("decode finite leader binding: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("finite leader binding must contain exactly one JSON object")
	}
	if input.FiniteLeader == nil || strings.TrimSpace(input.Schedule) == "" {
		return fmt.Errorf("finiteLeader and schedule are required")
	}
	if err := input.FiniteLeader.Validate(input.ProfileKey, nil); err != nil {
		return err
	}
	team, agent := fs.Arg(0), fs.Arg(1)
	path := fmt.Sprintf("/teams/%s/heartbeats/%s", url.PathEscape(team), url.PathEscape(agent))
	disabled := false
	var result HeartbeatConfig
	if *update {
		err = ctx.Put(path, UpdateHeartbeatRequest{FiniteLeader: input.FiniteLeader, Schedule: &input.Schedule, ProfileKey: &input.ProfileKey, Enabled: &disabled}, &result)
	} else {
		err = ctx.Post(path, CreateHeartbeatRequest{FiniteLeader: input.FiniteLeader, Schedule: input.Schedule, ProfileKey: input.ProfileKey, Enabled: &disabled}, &result)
	}
	if err != nil {
		return fmt.Errorf("bind finite leader: %w", err)
	}
	if result.TeamID != team || result.AgentID != agent || result.Enabled || result.ProfileKey != input.ProfileKey || !reflect.DeepEqual(result.FiniteLeader, input.FiniteLeader) {
		return fmt.Errorf("owner response did not confirm the exact disabled binding; inspect heartbeat state before activation")
	}
	return printFiniteLeaderBinding(result, *jsonOut)
}

func cmdHeartbeatRetireEffort(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("heartbeat-retire-effort", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 2 {
		return fmt.Errorf("usage: team heartbeat-retire-effort <team-id> <agent-id> [--json]")
	}
	team, agent := fs.Arg(0), fs.Arg(1)
	path := fmt.Sprintf("/teams/%s/heartbeats/%s", url.PathEscape(team), url.PathEscape(agent))
	var current HeartbeatConfig
	if err := ctx.Get(path, &current); err != nil {
		return fmt.Errorf("read finite leader before retirement: %w", err)
	}
	if current.FiniteLeader == nil || current.TeamID != team || current.AgentID != agent {
		return fmt.Errorf("exact finite leader binding unavailable")
	}
	current.FiniteLeader.Retired = true
	disabled := false
	var result HeartbeatConfig
	if err := ctx.Put(path, UpdateHeartbeatRequest{FiniteLeader: current.FiniteLeader, Enabled: &disabled}, &result); err != nil {
		return fmt.Errorf("retire finite leader: %w", err)
	}
	if result.TeamID != team || result.AgentID != agent || result.Enabled || !reflect.DeepEqual(result.FiniteLeader, current.FiniteLeader) {
		return fmt.Errorf("owner response did not confirm retirement; retain unresolved owner identity")
	}
	return printFiniteLeaderBinding(result, *jsonOut)
}

// finiteEffortTransitionInput is the bounded operator input for a completion,
// reopen, or terminal-run restart. It carries no scheduling or configuration
// changes.
type finiteEffortTransitionInput struct {
	Revision    string `json:"revision"`
	EvidenceRef string `json:"evidenceRef"`
}

func cmdHeartbeatCompleteEffort(ctx appctx.Context, args []string) error {
	return cmdFiniteEffortTransition(ctx, args, "complete")
}

func cmdHeartbeatReopenEffort(ctx appctx.Context, args []string) error {
	return cmdFiniteEffortTransition(ctx, args, "reopen")
}

func cmdHeartbeatRestartEffort(ctx appctx.Context, args []string) error {
	return cmdFiniteEffortTransition(ctx, args, "restart")
}

// cmdFiniteEffortTransition records an explicit authorized lifecycle operation.
// It reads the exact binding first so a completion cannot be sent against an
// unbound or differently-identified member, then writes only the transition.
func cmdFiniteEffortTransition(ctx appctx.Context, args []string, operation string) error {
	fs := flag.NewFlagSet("heartbeat-"+operation+"-effort", flag.ContinueOnError)
	file := fs.String("request-file", "", "JSON transition: revision, evidenceRef")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 2 || *file == "" {
		return fmt.Errorf("usage: team heartbeat-%s-effort <team-id> <agent-id> --request-file <transition.json> [--json]", operation)
	}
	reader, err := os.Open(*file)
	if err != nil {
		return err
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, 16385))
	if err != nil {
		return err
	}
	if len(data) > 16384 {
		return fmt.Errorf("finite effort transition exceeds 16 KiB")
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	var input finiteEffortTransitionInput
	if err := decoder.Decode(&input); err != nil {
		return fmt.Errorf("decode finite effort transition: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("finite effort transition must contain exactly one JSON object")
	}
	if strings.TrimSpace(input.Revision) == "" || strings.TrimSpace(input.EvidenceRef) == "" {
		return fmt.Errorf("finite effort transition requires revision and evidenceRef")
	}
	team, agent := fs.Arg(0), fs.Arg(1)
	path := fmt.Sprintf("/teams/%s/heartbeats/%s", url.PathEscape(team), url.PathEscape(agent))
	var current HeartbeatConfig
	if err := ctx.Get(path, &current); err != nil {
		return fmt.Errorf("read finite leader before %s: %w", operation, err)
	}
	if current.FiniteLeader == nil || current.TeamID != team || current.AgentID != agent {
		return fmt.Errorf("exact finite leader binding unavailable")
	}
	var result HeartbeatConfig
	transition := &FiniteEffortTransition{Operation: operation, Revision: input.Revision, EvidenceRef: input.EvidenceRef}
	if err := ctx.Put(path, UpdateHeartbeatRequest{FiniteEffortTransition: transition}, &result); err != nil {
		return fmt.Errorf("%s finite leader effort: %w", operation, err)
	}
	if result.TeamID != team || result.AgentID != agent || result.FiniteLeader == nil {
		return fmt.Errorf("owner response did not confirm the finite effort transition; inspect heartbeat state")
	}
	return printFiniteLeaderBinding(result, *jsonOut)
}

func printFiniteLeaderBinding(config HeartbeatConfig, jsonOut bool) error {
	if jsonOut {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(config)
	}
	fmt.Printf("Finite effort: %s\nAccepted revision: %s\nCoordinator: %s/%s\nProfile: %s\nEnabled: %t\nRetired: %t\n",
		config.FiniteLeader.EffortRef, config.FiniteLeader.AcceptedRevision, config.TeamID, config.AgentID, config.ProfileKey, config.Enabled, config.FiniteLeader.Retired)
	if len(config.FiniteLeaderState) != 0 {
		fmt.Printf("Leader state: %s\n", config.FiniteLeaderState)
	}
	if config.FiniteLeaderError != "" {
		fmt.Printf("Owner state unavailable: %s\n", config.FiniteLeaderError)
	}
	return nil
}
