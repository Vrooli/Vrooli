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

	"github.com/vrooli/cli-core/cliutil"
	"prompt-manager/cli/internal/appctx"
	"prompt-manager/internal/teamconfig"
)

type finitePreparationInput struct {
	ExpectedRevision string                   `json:"expectedRevision"`
	Binding          *teamconfig.FiniteLeader `json:"binding"`
	EvidenceRef      string                   `json:"evidenceRef"`
}

func cmdHeartbeatAmendEffort(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("heartbeat-amend-effort", flag.ContinueOnError)
	file := fs.String("request-file", "", "JSON amendment: expectedRevision, binding, evidenceRef")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 2 || *file == "" {
		return fmt.Errorf("usage: team heartbeat-amend-effort <team-id> <leader-id> --request-file amendment.json [--json]")
	}
	r, err := os.Open(*file)
	if err != nil {
		return err
	}
	defer r.Close()
	b, err := io.ReadAll(io.LimitReader(r, 16385))
	if err != nil {
		return err
	}
	if len(b) > 16384 {
		return fmt.Errorf("preparation amendment exceeds 16 KiB")
	}
	d := json.NewDecoder(strings.NewReader(string(b)))
	d.DisallowUnknownFields()
	var input finitePreparationInput
	if err = d.Decode(&input); err != nil {
		return err
	}
	if d.Decode(new(any)) != io.EOF {
		return fmt.Errorf("amendment must contain exactly one JSON object")
	}
	if input.ExpectedRevision == "" || input.EvidenceRef == "" || input.Binding == nil {
		return fmt.Errorf("expectedRevision, binding and evidenceRef are required")
	}
	team, agent := fs.Arg(0), fs.Arg(1)
	path := fmt.Sprintf("/teams/%s/heartbeats/%s", url.PathEscape(team), url.PathEscape(agent))
	var current HeartbeatConfig
	if err = ctx.Get(path, &current); err != nil {
		return err
	}
	if current.FiniteLeader == nil || current.TeamID != team || current.AgentID != agent || current.Enabled || current.TeamEnabled || !unusedPreparationResponse(current.FiniteLeaderState) {
		return fmt.Errorf("amendment requires an exact disabled finite binding")
	}
	if err = input.Binding.Validate(current.ProfileKey); err != nil {
		return err
	}
	var result HeartbeatConfig
	transition := &FiniteEffortTransition{Operation: "amend", ExpectedRevision: input.ExpectedRevision, Binding: input.Binding, EvidenceRef: input.EvidenceRef}
	if err = ctx.Put(path, UpdateHeartbeatRequest{FiniteEffortTransition: transition}, &result); err != nil {
		return fmt.Errorf("amend finite preparation: %w", err)
	}
	if result.TeamID != team || result.AgentID != agent || result.Enabled || result.TeamEnabled || result.ProfileKey != current.ProfileKey || result.Schedule != current.Schedule || !reflect.DeepEqual(result.FiniteLeader, input.Binding) || !unusedPreparationResponse(result.FiniteLeaderState) || !confirmedPreparationReceipt(result.PreparationAmendments, input) {
		return fmt.Errorf("owner response did not confirm exact disabled amendment; inspect native state")
	}
	return printFiniteLeaderBinding(result, *jsonOut)
}

func unusedPreparationResponse(raw json.RawMessage) bool {
	var state struct {
		Status          string          `json:"status"`
		ID              string          `json:"id"`
		TaskID          string          `json:"taskId"`
		RunID           string          `json:"runId"`
		TaskStarted     bool            `json:"taskStarted"`
		DispatchStarted bool            `json:"dispatchStarted"`
		Completed       json.RawMessage `json:"completed"`
	}
	return json.Unmarshal(raw, &state) == nil && state.Status == "idle" && state.ID == "" && state.TaskID == "" && state.RunID == "" && !state.TaskStarted && !state.DispatchStarted && (len(state.Completed) == 0 || string(state.Completed) == "null")
}

func confirmedPreparationReceipt(raw json.RawMessage, input finitePreparationInput) bool {
	var history []struct {
		Previous    *teamconfig.FiniteLeader `json:"previous"`
		Replacement *teamconfig.FiniteLeader `json:"replacement"`
		EvidenceRef string                   `json:"evidenceRef"`
		Actor       struct {
			Subject string `json:"subject"`
		} `json:"actor"`
	}
	if json.Unmarshal(raw, &history) != nil || len(history) == 0 {
		return false
	}
	last := history[len(history)-1]
	return last.Previous != nil && last.Previous.AcceptedRevision == input.ExpectedRevision && reflect.DeepEqual(last.Replacement, input.Binding) && last.EvidenceRef == input.EvidenceRef && last.Actor.Subject != ""
}
