package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil/mocks"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/cli-core/cliutil"
)

type backupTestPublisher struct{ events []eventbus.DomainEvent }

func (p *backupTestPublisher) PublishDomainEvent(_ context.Context, ev eventbus.DomainEvent) error {
	p.events = append(p.events, ev)
	return nil
}

func TestCodingBackupRequiresCurrentConsentAndSwitchesOnce(t *testing.T) {
	for _, tc := range []struct {
		name             string
		mutate           func(*RunExecutor, *cliutil.CodingBackupRecord)
		changeAfterIssue bool
		want             bool
	}{
		{name: "automatic consent", want: true},
		{name: "explicit subscription", mutate: func(e *RunExecutor, _ *cliutil.CodingBackupRecord) {
			e.customEnv = map[string]string{cliutil.ModelSourceEnvironment: `{"kind":"subscription"}`}
		}, want: true},
		{name: "different explicit source", mutate: func(e *RunExecutor, _ *cliutil.CodingBackupRecord) {
			e.customEnv = map[string]string{cliutil.ModelSourceEnvironment: `{"kind":"own_key"}`}
		}},
		{name: "no automatic consent", mutate: func(_ *RunExecutor, r *cliutil.CodingBackupRecord) { r.Consent.Automatic = false }},
		{name: "wrong owner", mutate: func(_ *RunExecutor, r *cliutil.CodingBackupRecord) { r.Owner = "other" }},
		{name: "expired", mutate: func(_ *RunExecutor, r *cliutil.CodingBackupRecord) {
			r.Consent.ExpiresAt = time.Now().Add(-time.Minute)
		}},
		{name: "stale limit", mutate: func(e *RunExecutor, _ *cliutil.CodingBackupRecord) {
			e.result.CodingLimit.ObservedAt = time.Now().Add(-6 * time.Minute)
		}},
		{name: "not resumable", mutate: func(e *RunExecutor, _ *cliutil.CodingBackupRecord) { e.run.Phase = domain.RunPhaseCompleted }},
		{name: "interactive", mutate: func(e *RunExecutor, _ *cliutil.CodingBackupRecord) {
			e.run.ExecutionMode = domain.ExecutionModeInteractive
		}},
		{name: "revoked while issuing", changeAfterIssue: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			issued, revoked := 0, 0
			door := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer parent-test-only" {
					t.Error("parent was not used for authority operation")
				}
				switch r.Method {
				case http.MethodPost:
					issued++
					_ = json.NewEncoder(w).Encode(map[string]any{"id": "child", "key": "child-test-only", "expires_at": time.Now().Add(10 * time.Minute).Unix()})
				case http.MethodDelete:
					revoked++
					w.WriteHeader(http.StatusNoContent)
				default:
					t.Errorf("unexpected method %s", r.Method)
				}
			}))
			defer door.Close()
			cfg := domain.DefaultRunConfig()
			cfg.RunnerType = domain.RunnerTypeCodex
			run := &domain.Run{ID: uuid.New(), OwnerSubject: "owner", RunMode: domain.RunModeInPlace, Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting, ResolvedConfig: cfg}
			store := mocks.NewFakeEventStore()
			e := NewRunExecutor(nil, nil, nil, store, run, nil, nil, "original task", "")
			e.workDir, e.runStateRoot = t.TempDir(), t.TempDir()
			e.result = &runner.ExecuteResult{CodingLimit: &cliutil.CodingLimit{Harness: "codex", Kind: "provider_limit", ObservedAt: time.Now()}}
			record := cliutil.CodingBackupRecord{Owner: "owner", BaseURL: door.URL, ParentID: "parent", ParentKey: "parent-test-only", Consent: cliutil.CodingBackupConsent{Harness: "codex", Account: "account", Model: "model", MaxCredits: 5000, ExpiresAt: time.Now().Add(time.Hour), Automatic: true}}
			if tc.mutate != nil {
				tc.mutate(e, &record)
			}
			reads := 0
			e.codingBackupReader = func(owner, harness string) (cliutil.CodingBackupRecord, error) {
				if owner != "owner" || harness != "codex" {
					return cliutil.CodingBackupRecord{}, fmt.Errorf("wrong authority scope")
				}
				reads++
				if tc.changeAfterIssue && reads == 2 {
					return cliutil.CodingBackupRecord{}, fmt.Errorf("removed")
				}
				return record, nil
			}
			pub := &backupTestPublisher{}
			e.hooks = &Orchestrator{approvalPublisher: pub}
			agent := runner.NewMockRunner(domain.RunnerTypeCodex)
			agent.SetCapabilities(runner.Capabilities{SupportsContinuation: true})
			launches := 0
			agent.ExecuteFunc = func(_ context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
				launches++
				if req.Environment["VROOLI_CODING_SESSION_KEY"] != "child-test-only" || strings.Contains(fmt.Sprint(req.Environment), "parent-test-only") {
					t.Error("child environment is incorrect")
				}
				if req.ResolvedConfig.Model != "model" || req.ResolvedConfig.Billing.Mode != domain.BillingModeMetered {
					t.Error("backup model or billing was not applied")
				}
				_ = req.EventSink.Emit(domain.NewLifecycleEvent(run.ID, domain.LifecycleEventData{Phase: domain.LifecyclePhaseRunnerAcquired}))
				return &runner.ExecuteResult{Success: true}, nil
			}
			switched, out := e.tryCodingBackup(context.Background(), agent, e.createEventSink())
			if out.RunState != nil {
				defer out.RunState.Close()
			}
			if switched != tc.want || (launches == 1) != tc.want {
				t.Fatalf("switch=%v launches=%d want=%v", switched, launches, tc.want)
			}
			if tc.want && (out.ExecErr != nil || len(pub.events) != 1 || pub.events[0].Payload["category"] != "coding-backup-switched") {
				t.Fatalf("out=%+v notifications=%+v", out, pub.events)
			}
			if !tc.want && len(pub.events) != 0 {
				t.Fatal("notification without launch")
			}
			if issued != revoked {
				t.Fatalf("issued %d, revoked %d", issued, revoked)
			}
		})
	}
}
