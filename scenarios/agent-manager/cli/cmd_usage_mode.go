// This file holds `usage mode start|status|end`: expiring operator policies
// that lower what new work runs on (docs/configuration/usage-modes.md). The API
// owns tier resolution, window expiry, authority and the reach computation;
// this command only carries the request and renders the answer.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"agent-manager/cli/internal/support"
	"agent-manager/internal/usagelabel"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
	measureconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures/measures_v1connect"
)

const usageModeUsage = `agent-manager usage mode start --preset one-tier-down [--model X] [--effort E] [--context-limit N] [--runner R] [--role R] [--profile P] --until 5h-reset|weekly-reset|off|<RFC3339>|<duration like 72h> [--label text] [--json]
  agent-manager usage mode status [--json]
  agent-manager usage mode end <id> | --all [--json]
  agent-manager usage mode extend <id> --until weekly-reset|off|<duration like 2h> [--json]
  agent-manager usage mode rebase <id> [--confirm] [--json]
  agent-manager usage mode ask --runner claude|codex --for 30m [--json]`

const usageModeHelp = `A usage mode lowers what NEW work runs on so quota lasts longer; running sessions and resumed runs keep their settings.

start    needs verified operator authority (the local operator exchange is used automatically; an agent run is refused with 403). The server resolves tiers from the resources' model-policy.json. --preset one-tier-down maps each model to the next tier the resource declares; --model X sends every mapped model to X (needs --runner). --until 5h-reset follows Claude's 5-hour window, weekly-reset the weekly window of --runner (Codex has only the weekly window); both need a quota reading and end early if the window resets sooner. Starting replaces a mode with the identical scope.
status   lists active modes with scope, settings, expiry, time left, the observed quota increase since each started, and a reach table for the current windows. Agents may read it.
end      ends one mode by id, or every mode with --all.
extend   moves one active mode's end and keeps its id, start and burn history. A duration is added to the later of now and the current end. Needs verified operator authority.
rebase   shows which running or resumable runs would switch to the mode's settings: switched_at_next_resume, applies_on_next_launch or unable, with the context the next resume re-sends. It changes nothing until --confirm. Needs verified operator authority.
ask      asks the operator, through a notification-hub decision, to start a fixed-length Save mode on one runner. Any caller may ask; nothing starts until a verified human answers Start, and that human is recorded as started_by.

Reach: shim-launched sessions change on their next launch only; desktop apps are unreachable. Points are estimates, and the burn shown is not a saving.`

func (a *App) usageModeCommand() cliapp.Command {
	command := support.Command("mode", "Lower what new work runs on so quota lasts: start|status|end|extend|rebase [--json]", a.usageMode)
	command.Usage = usageModeUsage
	command.HelpText = usageModeHelp
	return command
}

func (a *App) usageMode(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		fmt.Println("agent-manager usage mode " + strings.TrimPrefix(usageModeUsage, "agent-manager usage mode "))
		fmt.Println()
		fmt.Println(usageModeHelp)
		return nil
	}
	switch args[0] {
	case "start":
		return a.usageModeStart(args[1:])
	case "status", "list":
		return a.usageModeStatus(args[1:])
	case "end":
		return a.usageModeEnd(args[1:])
	case "extend":
		return a.usageModeExtend(args[1:])
	case "rebase":
		return a.usageModeRebase(args[1:])
	case "ask":
		return a.usageModeAsk(args[1:])
	}
	return fmt.Errorf("unknown usage mode subcommand %q; expected start, status, end, extend, rebase or ask", args[0])
}

// modeClient builds the Measures client. Start and end are operator actions:
// outside an identified agent run the local operator exchange supplies the
// verified human the API requires; inside one the ambient agent identity is
// sent and the API refuses it.
func (a *App) modeClient(operator bool) measureconnect.MeasuresServiceClient {
	if modeClientOverride != nil {
		return modeClientOverride(operator)
	}
	h, base := cliapp.NewConnectHTTPClient(a.core)
	if operator && strings.TrimSpace(os.Getenv(cliutil.EnvIdentityToken)) == "" {
		if owner, err := localOwnerAPIForResource(a.services.Maintenance.api, standingGrantResource); err == nil {
			return measureconnect.NewMeasuresServiceClient(ownerHTTPClient{inner: &http.Client{Timeout: 30 * time.Second}, headers: owner.AuthHeaders()}, base)
		}
	}
	return measureconnect.NewMeasuresServiceClient(h, base)
}

// modeClientOverride replaces the transport in tests.
var modeClientOverride func(operator bool) measureconnect.MeasuresServiceClient

// ownerHTTPClient sends the local operator's token on every request.
type ownerHTTPClient struct {
	inner   connect.HTTPClient
	headers map[string]string
}

func (c ownerHTTPClient) Do(req *http.Request) (*http.Response, error) {
	for name, value := range c.headers {
		req.Header.Set(name, value)
	}
	return c.inner.Do(req)
}

// modeRPCError shows the server's message without the transport wrapper.
func modeRPCError(err error) error {
	var rpcErr *connect.Error
	if errors.As(err, &rpcErr) {
		return errors.New(rpcErr.Message())
	}
	return err
}

func (a *App) usageModeStart(args []string) error {
	fs := flag.NewFlagSet("usage mode start", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	preset := fs.String("preset", "", "one-tier-down")
	model := fs.String("model", "", "send every mapped model to this one (needs --runner)")
	effort := fs.String("effort", "", "lower reasoning effort: low|medium|high|xhigh|max")
	contextLimit := fs.Int64("context-limit", 0, "context limit in tokens (shim launches only; Claude minimum 100000)")
	runner := fs.String("runner", "", "claude or codex; empty covers both")
	role := fs.String("role", "", "portable or resource role the mode applies to")
	profile := fs.String("profile", "", "profile key the mode applies to")
	until := fs.String("until", "", "5h-reset|weekly-reset|off|<RFC3339>|<duration like 72h>")
	label := fs.String("label", "", "operator label")
	fs.Bool("local-owner", false, "accepted for symmetry with other operator commands; the local operator exchange is automatic outside an agent run")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) > 0 {
		return errors.New("usage mode start takes flags only")
	}
	response, err := a.modeClient(true).StartUsageMode(context.Background(), connect.NewRequest(&measurepb.StartUsageModeRequest{
		Preset: strings.TrimSpace(*preset), Model: strings.TrimSpace(*model), Effort: strings.TrimSpace(*effort), ContextLimitTokens: *contextLimit,
		Runner: strings.TrimSpace(*runner), Role: strings.TrimSpace(*role), Profile: strings.TrimSpace(*profile), Until: strings.TrimSpace(*until), Label: strings.TrimSpace(*label),
	}))
	if err != nil {
		return modeRPCError(err)
	}
	if *jsonOut {
		return printProtoJSON(response.Msg)
	}
	mode := response.Msg.GetMode()
	fmt.Printf("Started usage mode %s (started by %s).\n", mode.GetId(), mode.GetStartedBy())
	printModeSummary(mode, "  ")
	if len(response.Msg.GetReplacedIds()) > 0 {
		fmt.Printf("Replaced mode(s) with the identical scope: %s\n", strings.Join(response.Msg.GetReplacedIds(), ", "))
	}
	fmt.Println("New Agent Manager runs with a role use it from now on; shim-launched sessions change on their next launch only; desktop apps are unreachable.")
	return nil
}

func (a *App) usageModeEnd(args []string) error {
	fs := flag.NewFlagSet("usage mode end", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	all := fs.Bool("all", false, "end every active mode")
	fs.Bool("local-owner", false, "accepted for symmetry with other operator commands")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	ids := fs.Args()
	if len(ids) == 0 && !*all {
		return errors.New("usage: agent-manager usage mode end <id> | --all [--json]")
	}
	response, err := a.modeClient(true).EndUsageMode(context.Background(), connect.NewRequest(&measurepb.EndUsageModeRequest{Ids: ids, All: *all}))
	if err != nil {
		return modeRPCError(err)
	}
	if *jsonOut {
		return printProtoJSON(response.Msg)
	}
	if len(response.Msg.GetEndedIds()) == 0 {
		fmt.Println("No active usage mode ended.")
	} else {
		fmt.Printf("Ended usage mode(s): %s\n", strings.Join(response.Msg.GetEndedIds(), ", "))
	}
	if missing := response.Msg.GetNotFoundIds(); len(missing) > 0 {
		fmt.Printf("Not active: %s\n", strings.Join(missing, ", "))
	}
	return nil
}

func (a *App) usageModeExtend(args []string) error {
	fs := flag.NewFlagSet("usage mode extend", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	until := fs.String("until", "", "weekly-reset|off|<duration like 2h> added to the later of now and the current end")
	fs.Bool("local-owner", false, "accepted for symmetry with other operator commands")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 || strings.TrimSpace(*until) == "" {
		return errors.New("usage: agent-manager usage mode extend <id> --until weekly-reset|off|<duration like 2h> [--json]")
	}
	response, err := a.modeClient(true).ExtendUsageMode(context.Background(), connect.NewRequest(&measurepb.ExtendUsageModeRequest{Id: fs.Args()[0], To: strings.TrimSpace(*until)}))
	if err != nil {
		return modeRPCError(err)
	}
	if *jsonOut {
		return printProtoJSON(response.Msg)
	}
	mode := response.Msg.GetMode()
	fmt.Printf("Extended usage mode %s.\n", mode.GetId())
	printModeSummary(mode, "  ")
	return nil
}

func (a *App) usageModeRebase(args []string) error {
	fs := flag.NewFlagSet("usage mode rebase", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	confirm := fs.Bool("confirm", false, "switch the runs; without it the command only shows what would switch")
	fs.Bool("local-owner", false, "accepted for symmetry with other operator commands")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return errors.New("usage: agent-manager usage mode rebase <id> [--confirm] [--json]")
	}
	response, err := a.modeClient(true).RebaseUsageModeRuns(context.Background(), connect.NewRequest(&measurepb.RebaseUsageModeRunsRequest{ModeId: fs.Args()[0], DryRun: !*confirm}))
	if err != nil {
		return modeRPCError(err)
	}
	if *jsonOut {
		return printProtoJSON(response.Msg)
	}
	if response.Msg.GetDryRun() {
		fmt.Println("Dry run: nothing changed. Re-run with --confirm to switch these runs.")
	}
	if len(response.Msg.GetResults()) == 0 {
		fmt.Println("No non-terminal run to switch.")
	}
	for _, r := range response.Msg.GetResults() {
		line := fmt.Sprintf("%s  %s", r.GetRunId(), r.GetStatus())
		if r.GetFromModel() != "" || r.GetToModel() != "" {
			line += fmt.Sprintf("  %s -> %s", valueOr(r.GetFromModel(), "?"), valueOr(r.GetToModel(), "?"))
		}
		if r.GetEstimatedRereadTokens() > 0 {
			line += fmt.Sprintf("  re-reads ~%d tokens on resume", r.GetEstimatedRereadTokens())
		}
		fmt.Println(line)
		if r.GetReason() != "" {
			fmt.Println("    " + r.GetReason())
		}
	}
	return nil
}

func (a *App) usageModeStatus(args []string) error {
	fs := flag.NewFlagSet("usage mode status", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	response, err := a.modeClient(false).ListUsageModes(context.Background(), connect.NewRequest(&measurepb.ListUsageModesRequest{}))
	if err != nil {
		return modeRPCError(err)
	}
	if *jsonOut {
		return printProtoJSON(response.Msg)
	}
	printModeStatus(response.Msg)
	return nil
}

func printProtoJSON(message proto.Message) error {
	raw, err := protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: true}.Marshal(message)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(raw)
	return nil
}

func printModeStatus(r *measurepb.ListUsageModesResponse) {
	fmt.Printf("Usage modes as of %s (file %s)\n", r.GetGeneratedAt(), r.GetModeFile())
	if len(r.GetModes()) == 0 {
		fmt.Println("No active usage mode.")
	}
	for _, status := range r.GetModes() {
		mode := status.GetMode()
		fmt.Printf("\n%s  %s\n", mode.GetId(), mode.GetLabel())
		printModeSummary(mode, "  ")
		fmt.Printf("  started %s by %s\n", mode.GetStartedAt(), valueOr(mode.GetStartedBy(), "unknown"))
		if mode.GetExpiresAt() == "" {
			fmt.Println("  expires: when ended (no expiry)")
		} else {
			fmt.Printf("  expires: %s (%s left)\n", mode.GetExpiresAt(), (time.Duration(status.GetSecondsLeft()) * time.Second).Round(time.Minute))
		}
		if len(status.GetBurn()) == 0 {
			fmt.Println("  observed use since start: no current window reading for this mode's provider")
		}
		for _, burn := range status.GetBurn() {
			fmt.Printf("  observed use since start (estimate, not a saving): %s +%.1f points (explained %.1f, unattributed %.1f)\n",
				usagelabel.Title(burn.GetProvider(), burn.GetWindow(), 0), burn.GetObservedIncreasePoints(), burn.GetAttributedPoints(), burn.GetUnattributedPoints())
		}
	}
	fmt.Println("\nReach in the current windows (points are estimates):")
	if len(r.GetReach()) == 0 {
		fmt.Println("  no current window has used-percent readings")
	}
	for _, row := range r.GetReach() {
		fmt.Printf("  %s (resets %s): +%.1f points observed\n", usagelabel.Title(row.GetProvider(), row.GetWindow(), 0), row.GetResetsAt(), row.GetObservedIncreasePoints())
		fmt.Printf("    operator sessions %4.0f%%  (%.1f)  change on next shim launch only; role/profile scopes do not reach them\n", row.GetOperatorShare()*100, row.GetOperatorPoints())
		fmt.Printf("    profile runs      %4.0f%%  (%.1f)  new Agent Manager runs with a role use the mode\n", row.GetProfileShare()*100, row.GetProfilePoints())
		fmt.Printf("    unattributed      %4.0f%%  (%.1f)  not reachable (off-disk, other machines, desktop apps)\n", row.GetUnattributedShare()*100, row.GetUnattributedPoints())
	}
	for _, note := range r.GetReachNotes() {
		fmt.Println("- " + note)
	}
}

func printModeSummary(mode *measurepb.UsageMode, indent string) {
	scope := mode.GetScope()
	fmt.Printf("%sscope: runner=%s role=%s profile=%s\n", indent, valueOr(scope.GetRunner(), "*"), valueOr(scope.GetRole(), "*"), valueOr(scope.GetProfile(), "*"))
	for _, s := range mode.GetSettings() {
		line := fmt.Sprintf("%s%s:", indent, s.GetRunner())
		for from, to := range s.GetModelMap() {
			line += fmt.Sprintf(" %s -> %s;", from, to)
		}
		if s.GetSessionModel() != "" {
			line += " session model " + s.GetSessionModel() + ";"
		}
		if s.GetEffort() != "" {
			line += " effort " + s.GetEffort() + ";"
		}
		if s.GetContextLimitTokens() > 0 {
			line += fmt.Sprintf(" context limit %d (shim launches only);", s.GetContextLimitTokens())
		}
		fmt.Println(line)
	}
	until := mode.GetUntilKind()
	if mode.GetUntilProvider() != "" {
		until += fmt.Sprintf(" (%s, %d-minute window)", mode.GetUntilProvider(), mode.GetUntilWindowMinutes())
	}
	fmt.Printf("%suntil: %s %s\n", indent, until, mode.GetExpiresAt())
	if len(mode.GetExemptRoles()) > 0 {
		fmt.Printf("%sexempt roles: %s\n", indent, strings.Join(mode.GetExemptRoles(), ", "))
	}
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

// usageModeAsk raises the operator decision; the API starts nothing itself.
func (a *App) usageModeAsk(args []string) error {
	fs := flag.NewFlagSet("usage mode ask", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	runner := fs.String("runner", "", "claude or codex")
	length := fs.String("for", "", "how long the mode runs once started, such as 30m")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*runner) == "" || strings.TrimSpace(*length) == "" {
		return errors.New("usage: agent-manager usage mode ask --runner claude|codex --for 30m [--json]")
	}
	body, err := json.Marshal(map[string]string{"runner": strings.TrimSpace(*runner), "length": strings.TrimSpace(*length)})
	if err != nil {
		return err
	}
	client, base := cliapp.NewConnectHTTPClient(a.core)
	request, err := http.NewRequest(http.MethodPost, strings.TrimRight(base, "/")+"/api/v1/usage/modes/ask", bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(response.Body, 1<<16))
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("ask failed (%d): %s", response.StatusCode, strings.TrimSpace(string(raw)))
	}
	if *jsonOut {
		cliutil.PrintJSON(raw)
		return nil
	}
	var out struct{ DecisionID, State string }
	var fields map[string]string
	if json.Unmarshal(raw, &fields) == nil {
		out.DecisionID, out.State = fields["decision_id"], fields["state"]
	}
	fmt.Printf("Decision %s is %s. A mode starts only when a verified operator answers Start.\n", out.DecisionID, out.State)
	return nil
}
