package vroolicli

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/recovery"
)

type investigationBrokerFixture struct {
	reconciled   bool
	reconcileErr error
	request      recovery.CodingAgentRequest
	record       recovery.CodingAgentRecord
	submitErr    error
}

func (b *investigationBrokerFixture) ReconcileNative(ctx context.Context, limit int) (int, error) {
	if limit != 32 || ctx == nil {
		return 0, errors.New("invalid bounded reconciliation")
	}
	b.reconciled = true
	return 0, b.reconcileErr
}

func (b *investigationBrokerFixture) SubmitContext(ctx context.Context, r recovery.CodingAgentRequest) (recovery.CodingAgentRecord, error) {
	if !b.reconciled || ctx == nil {
		return recovery.CodingAgentRecord{}, errors.New("missing reconciliation")
	}
	b.request = r
	return b.record, b.submitErr
}

func TestCodingInvestigationReturnsAdmittedIdentityOnAmbiguousDispatch(t *testing.T) {
	root := t.TempDir()
	var out strings.Builder
	ctx := &AppContext{Root: root, Stdout: &out, Stderr: &strings.Builder{}}
	ctx.Globals.JSON = true
	wantErr := errors.New("native dispatch acknowledgement unconfirmed")
	b := &investigationBrokerFixture{record: recovery.CodingAgentRecord{ID: "admitted", Scenario: "web-console", Outcome: "scheduled", DeadlineAt: time.Now().Add(time.Minute)}, submitErr: wantErr}
	err := runCodingAgentInvestigation(ctx, []string{"--scenario", "web-console", "--reason", "API unhealthy"}, func() (codingAgentInvestigationBroker, error) { return b, nil })
	if !errors.Is(err, wantErr) {
		t.Fatalf("lost dispatch uncertainty: %v", err)
	}
	var got recovery.CodingAgentRecord
	if err := json.Unmarshal([]byte(out.String()), &got); err != nil || got.ID != "admitted" || got.Outcome != "scheduled" {
		t.Fatalf("lost admission receipt: %s %v", out.String(), err)
	}
	if b.request.Scenario != "web-console" || b.request.Reason != "API unhealthy" || b.request.WorkingDir != root || b.request.Requester != "vrooli-cli" || b.request.Claim != filepath.Join("scenarios", "web-console") {
		t.Fatalf("unexpected native request: %+v", b.request)
	}
}

func TestCodingInvestigationRequiresRootAndExplicitFailureBeforeOpeningBroker(t *testing.T) {
	for _, args := range [][]string{nil, {"--scenario", "web-console"}, {"--scenario", "web-console", "--reason", "failure", "extra"}, {"--scenario", "web-console", "--reason", "failure"}} {
		opened := false
		ctx := &AppContext{Stderr: &strings.Builder{}}
		err := runCodingAgentInvestigation(ctx, args, func() (codingAgentInvestigationBroker, error) {
			opened = true
			return nil, errors.New("unexpected open")
		})
		if err == nil || opened {
			t.Fatalf("invalid request reached broker: args=%q err=%v opened=%v", args, err, opened)
		}
	}
}

func TestCodingInvestigationReconciliationFailurePreventsSubmission(t *testing.T) {
	b := &investigationBrokerFixture{reconcileErr: errors.New("native owner uncertain")}
	ctx := &AppContext{Root: t.TempDir(), Stderr: &strings.Builder{}}
	if err := runCodingAgentInvestigation(ctx, []string{"--scenario", "web-console", "--reason", "failure"}, func() (codingAgentInvestigationBroker, error) { return b, nil }); err == nil || b.request.Scenario != "" {
		t.Fatalf("submitted despite unresolved native owner: %+v %v", b.request, err)
	}
}
