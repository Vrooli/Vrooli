package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/provenance"
	walkv1 "github.com/vrooli/vrooli/packages/proto/gen/go/command-center/v1/walk"
	journalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal"
	journalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal/journal_v1connect"
)

var walkPhases = []string{"1", "2", "3", "4", "5", "5.3", "5.5", "5.7", "6", "7", "8", "9"}

const walkScope = "team:director-swarm"

func walkKind(channel, kind string) (string, error) {
	switch channel {
	case "", "operator":
		return kind, nil
	case "test":
		return kind + "-test", nil
	}
	return "", connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("channel must be operator or test"))
}
func (s walkConnectService) journal() journalconnect.JournalServiceClient {
	if s.ledger != nil {
		return s.ledger
	}
	return journalconnect.NewJournalServiceClient(&http.Client{Timeout: 90 * time.Second, Transport: provenance.ForwardingTransport{}}, resolveScenarioBaseURL("source-ledger", "SOURCE_LEDGER_BASE_URL", "SOURCE_LEDGER_API_PORT")())
}
func (s walkConnectService) latest(ctx context.Context, kind string) (*walkv1.StoredRecord, error) {
	r, err := s.journal().ListEntries(ctx, connect.NewRequest(&journalv1.ListEntriesRequest{Scope: walkScope, Kind: kind, NewestFirst: true, Limit: 1}))
	if err != nil {
		return nil, err
	}
	if len(r.Msg.Entries) == 0 {
		return nil, nil
	}
	e := r.Msg.Entries[0]
	return &walkv1.StoredRecord{EntryId: e.Id, Body: e.Body, CreatedAt: e.CreatedAt.AsTime().Format(time.RFC3339Nano)}, nil
}
func (s walkConnectService) State(ctx context.Context, r *connect.Request[walkv1.StateRequest]) (*connect.Response[walkv1.StateResponse], error) {
	bk, err := walkKind(r.Msg.Channel, "vision-walk-briefing")
	if err != nil {
		return nil, err
	}
	ck, _ := walkKind(r.Msg.Channel, "walk-checkpoint")
	b, err := s.latest(ctx, bk)
	if err != nil {
		return nil, err
	}
	c, err := s.latest(ctx, ck)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&walkv1.StateResponse{Briefing: b, Checkpoint: c}), nil
}
func badWalk(message string) error {
	return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%s", message))
}
func conflictWalk(message string) error {
	return connect.NewError(connect.CodeAborted, fmt.Errorf("%s", message))
}
func receipt(e *journalv1.Entry, existing bool, channel string) *connect.Response[walkv1.Receipt] {
	return connect.NewResponse(&walkv1.Receipt{EntryId: e.Id, CreatedAt: e.CreatedAt.AsTime().Format(time.RFC3339Nano), Existing: existing, Channel: channel})
}

// A replay is checked before transition validation so a delayed retry still returns
// its original receipt after another valid transition has committed.
func (s walkConnectService) replay(ctx context.Context, kind, key, body, channel string) (*connect.Response[walkv1.Receipt], error) {
	if len(key) < 1 || len(key) > 160 {
		return nil, badWalk("request_key must contain 1-160 characters")
	}
	e, err := s.journal().GetEntry(ctx, connect.NewRequest(&journalv1.GetEntryRequest{Scope: walkScope, RequestKey: "command-center/" + kind + "/" + key}))
	if connect.CodeOf(err) == connect.CodeNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if e.Msg.Entry.Body != body || e.Msg.Entry.Kind != kind {
		return nil, conflictWalk("request key already names different content")
	}
	return receipt(e.Msg.Entry, true, channel), nil
}
func (s walkConnectService) appendWalk(ctx context.Context, kind, key, previous, body, channel string) (*connect.Response[walkv1.Receipt], error) {
	r, err := s.journal().AppendEntry(ctx, connect.NewRequest(&journalv1.AppendEntryRequest{Scope: walkScope, Kind: kind, Body: body, RequestKey: "command-center/" + kind + "/" + key, ExpectedLatestId: &previous}))
	if err != nil {
		return nil, err
	}
	check, err := s.journal().GetEntry(ctx, connect.NewRequest(&journalv1.GetEntryRequest{Scope: walkScope, Id: r.Msg.Entry.Id}))
	if err != nil {
		return nil, err
	}
	if check.Msg.Entry.Body != body {
		return nil, connect.NewError(connect.CodeDataLoss, fmt.Errorf("receipt readback differs"))
	}
	return receipt(check.Msg.Entry, r.Msg.Existing, channel), nil
}
func (s walkConnectService) Publish(ctx context.Context, r *connect.Request[walkv1.PublishRequest]) (*connect.Response[walkv1.Receipt], error) {
	m := r.Msg
	kind, err := walkKind(m.Channel, "vision-walk-briefing")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(m.Briefing) == "" || len(m.Briefing) > 30000 || !strings.HasPrefix(m.ProgramId, "prog_") {
		return nil, badWalk("program_id and nonempty briefing (at most 30000 bytes) required")
	}
	if len(m.EnvelopeJson) > 60000 {
		return nil, badWalk("envelope exceeds 60000 bytes")
	}
	var e struct {
		Program string `json:"program"`
		Status  string `json:"status"`
		Signals struct {
			Generated string `json:"generated_at"`
			Phases    []struct {
				Phase string `json:"phase"`
			} `json:"phases"`
			Checkpoint json.RawMessage `json:"checkpoint"`
		} `json:"signals"`
	}
	if json.Unmarshal([]byte(m.EnvelopeJson), &e) != nil || e.Program != "command-center.vision-walk-prep" || (e.Status != "ok" && e.Status != "partial") {
		return nil, badWalk("usable canonical prep envelope required")
	}
	if len(e.Signals.Phases) != len(walkPhases) {
		return nil, badWalk("all twelve phases required")
	}
	for i, p := range e.Signals.Phases {
		if p.Phase != walkPhases[i] {
			return nil, badWalk("phase order differs")
		}
	}
	var cp map[string]any
	if json.Unmarshal(e.Signals.Checkpoint, &cp) != nil || cp["status"] == nil {
		return nil, badWalk("checkpoint evidence required")
	}
	allowedCheckpoint := map[string]bool{"none": true, "active": true, "completed": true, "abandoned": true, "legacy": true, "unavailable": true, "invalid": true}
	cpStatus, _ := cp["status"].(string)
	if !allowedCheckpoint[cpStatus] {
		return nil, badWalk("unknown checkpoint evidence state")
	}
	var supplied struct {
		Inputs struct {
			Channel string `json:"channel"`
		} `json:"inputs"`
	}
	_ = json.Unmarshal([]byte(m.EnvelopeJson), &supplied)
	actualChannel := m.Channel
	if actualChannel == "" {
		actualChannel = "operator"
	}
	if supplied.Inputs.Channel != "" && supplied.Inputs.Channel != actualChannel {
		return nil, badWalk("envelope and publication channels differ")
	}
	var fleet map[string]any
	if json.Unmarshal([]byte(m.FleetHealthJson), &fleet) != nil || len(fleet) == 0 {
		return nil, badWalk("fleet-health aggregate or explicit unavailable evidence required")
	}
	if fleet["status"] == "unavailable" {
		if reason, ok := fleet["reason"].(string); !ok || strings.TrimSpace(reason) == "" {
			return nil, badWalk("fleet gap requires a reason")
		}
	} else {
		if _, ok := fleet["meetsThreshold"].(bool); !ok {
			return nil, badWalk("fleet aggregate requires exact owner verdict")
		}
		for _, key := range []string{"enabledMembers", "producedMembers", "succeededMembers", "blockedMembers", "failedMembers", "membersWithTwoFailures", "successPercent", "thresholdPercent", "windowHours"} {
			if _, ok := fleet[key].(float64); !ok {
				return nil, badWalk("fleet aggregate missing " + key)
			}
		}
		if at, ok := fleet["generatedAt"].(string); !ok || at == "" {
			return nil, badWalk("fleet aggregate timestamp required")
		}
	}
	bodyBytes, _ := json.Marshal(map[string]any{"schema_version": 1, "program_id": m.ProgramId, "envelope": json.RawMessage(m.EnvelopeJson), "briefing": m.Briefing, "fleet_health": fleet, "expected_previous_id": m.ExpectedPreviousId})
	body := string(bodyBytes)
	if replay, err := s.replay(ctx, kind, m.RequestKey, body, m.Channel); replay != nil || err != nil {
		return replay, err
	}
	at, err := time.Parse(time.RFC3339Nano, e.Signals.Generated)
	if err != nil || time.Since(at) > 36*time.Hour || time.Until(at) > 5*time.Minute {
		return nil, badWalk("prep generation time is invalid or stale")
	}
	return s.appendWalk(ctx, kind, m.RequestKey, m.ExpectedPreviousId, body, m.Channel)
}
func (s walkConnectService) Checkpoint(ctx context.Context, r *connect.Request[walkv1.CheckpointRequest]) (*connect.Response[walkv1.Receipt], error) {
	m := r.Msg
	kind, err := walkKind(m.Channel, "walk-checkpoint")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(m.WalkId) == "" || len(m.WalkId) > 160 {
		return nil, badWalk("walk_id required")
	}
	if m.State != "active" && m.State != "completed" && m.State != "abandoned" {
		return nil, badWalk("invalid state")
	}
	if len(m.Content) > 40000 {
		return nil, badWalk("checkpoint content exceeds 40000 bytes")
	}
	if m.State != "active" && (m.Content != "" || m.ResumePhase != "") {
		return nil, badWalk("terminal transitions carry no active content or resume phase")
	}
	if m.State == "active" {
		valid := false
		for _, p := range walkPhases {
			valid = valid || p == m.ResumePhase
		}
		if !valid || strings.TrimSpace(m.Content) == "" || len(m.Content) > 40000 {
			return nil, badWalk("valid phase and exact content of at most 40000 bytes required")
		}
	}
	bodyBytes, _ := json.Marshal(map[string]string{"walk_id": m.WalkId, "state": m.State, "resume_phase": m.ResumePhase, "content": m.Content, "expected_previous_id": m.ExpectedPreviousId})
	body := string(bodyBytes)
	if replay, err := s.replay(ctx, kind, m.RequestKey, body, m.Channel); replay != nil || err != nil {
		return replay, err
	}
	last, err := s.latest(ctx, kind)
	if err != nil {
		return nil, err
	}
	if last == nil {
		if m.ExpectedPreviousId != "" || m.State != "active" {
			return nil, conflictWalk("new walk requires active state and empty predecessor")
		}
	} else {
		if last.EntryId != m.ExpectedPreviousId {
			return nil, conflictWalk("checkpoint predecessor changed")
		}
		var prior map[string]string
		if json.Unmarshal([]byte(last.Body), &prior) != nil || prior["walk_id"] == "" {
			return nil, conflictWalk("prior checkpoint is invalid")
		}
		if prior["state"] == "active" {
			if prior["walk_id"] != m.WalkId {
				return nil, conflictWalk("another walk is active")
			}
		} else if prior["state"] == "completed" || prior["state"] == "abandoned" {
			if m.State != "active" || m.WalkId == prior["walk_id"] {
				return nil, conflictWalk("finished walk cannot be resurrected")
			}
		} else {
			return nil, conflictWalk("prior state is invalid")
		}
	}
	return s.appendWalk(ctx, kind, m.RequestKey, m.ExpectedPreviousId, body, m.Channel)
}
