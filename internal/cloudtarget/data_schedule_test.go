package cloudtarget

import (
	"context"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/packages/recoverypoint"
)

// TestDataScheduleReconcilesExplicitBindingsToS3OnTarget [REQ:STC-P0-029]
// verifies the target owner calls DBM and reads the persisted schedule back.
func TestDataScheduleReconcilesExplicitBindingsToS3OnTarget(t *testing.T) {
	store := NewStore(t.TempDir())
	var calls [][]string
	planLists := 0
	runner := func(_ context.Context, args ...string) ([]byte, error) {
		calls = append(calls, append([]string(nil), args...))
		switch strings.Join(args[:2], " ") {
		case "targets register":
			return []byte(`{"target":{"id":"t-db"}}`), nil
		case "destinations list":
			return []byte(`{"destinations":[{"id":"dst-r2","name":"lpbs-r2","backendKind":"BACKEND_KIND_S3"}]}`), nil
		case "plans list":
			planLists++
			if planLists == 1 {
				return []byte(`{"plans":[]}`), nil
			}
			return []byte(`{"plans":[{"id":"p-lpbs","name":"scenario-to-cloud-lpbs-prod","targetIds":["t-db"],"destinationIds":["dst-r2"],"schedule":"24h","enabled":true,"recoveryDrillSchedule":"168h","retention":{"keepDaily":7,"keepWeekly":4,"keepMonthly":3}}]}`), nil
		case "plans create":
			return []byte(`{"plan":{"id":"p-lpbs"}}`), nil
		default:
			t.Fatalf("unexpected target-local DBM call: %v", args)
			return nil, nil
		}
	}
	report, result, err := store.DataSchedule(context.Background(), DataScheduleRequest{
		Effect: EffectRequest{DeploymentID: "lpbs-prod", OperationID: "op-schedule", Step: "data.schedule", Fence: 4}, Scenario: "landing-page-business-suite",
		Policy:   DataSchedulePolicy{DestinationName: "lpbs-r2", Schedule: "24h", KeepDaily: 7, KeepWeekly: 4, KeepMonthly: 3, RecoveryDrillSchedule: "168h"},
		Bindings: []recoverypoint.Binding{{ID: "accounts-db", Kind: recoverypoint.KindSQL, Provider: recoverypoint.ProviderPostgres, Locator: "lpbs"}}, Runner: runner,
	})
	if err != nil {
		t.Fatal(err)
	}
	if report["plan_id"] != "p-lpbs" || result.Receipt.Outcome != OutcomeSucceeded {
		t.Fatalf("report=%v receipt=%+v", report, result.Receipt)
	}
	if len(calls) != 5 {
		t.Fatalf("DBM calls=%v", calls)
	}
	if got := strings.Join(calls[0], " "); !strings.Contains(got, "targets register --owner landing-page-business-suite-lpbs-prod --name accounts-db --kind postgres") {
		t.Fatalf("registration call=%s", got)
	}
	if got := strings.Join(calls[3], " "); !strings.Contains(got, "plans create --name scenario-to-cloud-lpbs-prod") || !strings.Contains(got, "--destinations dst-r2") || !strings.Contains(got, "--schedule 24h") || !strings.Contains(got, "--keep-daily 7") {
		t.Fatalf("create call=%s", got)
	}
	replayed, replayReceipt, err := store.DataSchedule(context.Background(), DataScheduleRequest{
		Effect: EffectRequest{DeploymentID: "lpbs-prod", OperationID: "op-schedule", Step: "data.schedule", Fence: 4}, Scenario: "landing-page-business-suite",
		Policy:   DataSchedulePolicy{DestinationName: "lpbs-r2", Schedule: "24h", KeepDaily: 7, KeepWeekly: 4, KeepMonthly: 3, RecoveryDrillSchedule: "168h"},
		Bindings: []recoverypoint.Binding{{ID: "accounts-db", Kind: recoverypoint.KindSQL, Provider: recoverypoint.ProviderPostgres, Locator: "lpbs"}}, Runner: runner,
	})
	if err != nil || !replayReceipt.Replayed || replayed["plan_id"] != "p-lpbs" || len(calls) != 5 {
		t.Fatalf("replay report=%v result=%+v calls=%d err=%v", replayed, replayReceipt, len(calls), err)
	}
}

func TestDataScheduleFailsWhenDBMDoesNotPersistRequestedPolicy(t *testing.T) {
	store := NewStore(t.TempDir())
	planLists := 0
	runner := func(_ context.Context, args ...string) ([]byte, error) {
		switch strings.Join(args[:2], " ") {
		case "targets register":
			return []byte(`{"target":{"id":"t-db"}}`), nil
		case "destinations list":
			return []byte(`{"destinations":[{"id":"dst-r2","name":"lpbs-r2","backendKind":"BACKEND_KIND_S3"}]}`), nil
		case "plans list":
			planLists++
			if planLists == 1 {
				return []byte(`{"plans":[]}`), nil
			}
			return []byte(`{"plans":[{"id":"p-lpbs","name":"scenario-to-cloud-lpbs-prod","targetIds":["t-db"],"destinationIds":["dst-r2"],"schedule":"24h","enabled":false,"recoveryDrillSchedule":"168h","retention":{"keepDaily":7}}]}`), nil
		case "plans create":
			return []byte(`{"plan":{"id":"p-lpbs"}}`), nil
		default:
			t.Fatalf("unexpected DBM call: %v", args)
			return nil, nil
		}
	}
	_, result, err := store.DataSchedule(context.Background(), DataScheduleRequest{
		Effect: EffectRequest{DeploymentID: "lpbs-prod", OperationID: "op-schedule", Step: "data.schedule", Fence: 4}, Scenario: "landing-page-business-suite",
		Policy:   DataSchedulePolicy{DestinationName: "lpbs-r2", Schedule: "24h", KeepDaily: 7, RecoveryDrillSchedule: "168h"},
		Bindings: []recoverypoint.Binding{{ID: "accounts-db", Kind: recoverypoint.KindSQL, Provider: recoverypoint.ProviderPostgres, Locator: "lpbs"}}, Runner: runner,
	})
	if err == nil || result.Receipt.Outcome != OutcomeFailed {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

// TestDataScheduleRefusesLocalDestinationBeforePlanMutation [REQ:STC-P0-029]
// ensures a healthy status cannot represent local-only storage as off-host.
func TestDataScheduleRefusesLocalDestinationBeforePlanMutation(t *testing.T) {
	store := NewStore(t.TempDir())
	var calls int
	runner := func(_ context.Context, args ...string) ([]byte, error) {
		calls++
		switch strings.Join(args[:2], " ") {
		case "targets register":
			return []byte(`{"target":{"id":"t-db"}}`), nil
		case "destinations list":
			return []byte(`{"destinations":[{"id":"dst-local","name":"lpbs-r2","backendKind":"BACKEND_KIND_FILESYSTEM"}]}`), nil
		default:
			t.Fatalf("plan mutation should not occur: %v", args)
			return nil, nil
		}
	}
	_, _, err := store.DataSchedule(context.Background(), DataScheduleRequest{
		Effect: EffectRequest{DeploymentID: "lpbs-prod", OperationID: "op-schedule", Step: "data.schedule", Fence: 4}, Scenario: "landing-page-business-suite",
		Policy: DataSchedulePolicy{DestinationName: "lpbs-r2", Schedule: "24h", KeepDaily: 7, RecoveryDrillSchedule: "168h"}, Bindings: []recoverypoint.Binding{{ID: "accounts-db", Kind: recoverypoint.KindSQL, Provider: recoverypoint.ProviderPostgres, Locator: "lpbs"}}, Runner: runner,
	})
	if AsError(err).Code != CodeInvalidArgument || calls != 2 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}
