package localserviceadmission

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/privilegebroker"
)

type fakeBroker struct {
	report       *privilegebroker.LocalAdmissionReport
	actions      []string
	ensureResult *privilegebroker.Result
}

func (f *fakeBroker) Do(_ context.Context, req privilegebroker.Request) (privilegebroker.Result, error) {
	f.actions = append(f.actions, req.Action)
	if req.Action == privilegebroker.ActionLocalAdmissionEnsure && f.ensureResult != nil {
		return *f.ensureResult, nil
	}
	return privilegebroker.Result{Status: "verified", LocalAdmission: f.report}, nil
}

func TestFailedApplyRetainsBrokerFailureCode(t *testing.T) {
	h, req, client := fixture(t)
	client.report.Enabled = false
	client.ensureResult = &privilegebroker.Result{Status: "failed", Code: "local_rule_not_verified", Evidence: privilegebroker.Evidence{Detail: "managed rule was not observed"}}
	status := h.Inspect(hostreqkit.Host{OS: "linux"}, req)
	status, err := h.Apply(hostreqkit.Host{OS: "linux"}, status, hostreqkit.EnsureOptions{})
	if err != nil || status.ExecutionState != hostreqkit.ExecutionFailed {
		t.Fatalf("status=%+v err=%v", status, err)
	}
	joined := strings.Join(status.Notes, " ")
	if !strings.Contains(joined, "local_rule_not_verified") || !strings.Contains(joined, "managed rule was not observed") {
		t.Fatalf("failure evidence lost: %v", status.Notes)
	}
}
func fixture(t *testing.T) (handler, hostreqspec.ResolvedRequirement, *fakeBroker) {
	t.Helper()
	binding := privilegebroker.LocalAdmissionBinding{ID: "tv", Scenario: "portal", Listener: "ui", Interface: "eth0", SourceIP: "192.168.10.42", HostIP: "192.168.10.2"}
	data, _ := json.Marshal(binding)
	var item any
	json.Unmarshal(data, &item)
	req := hostreqspec.ResolvedRequirement{Name: "local_service_admission", Kind: hostreqspec.KindSafeguard, Config: map[string]any{"bindings": []any{item}}}
	client := &fakeBroker{report: &privilegebroker.LocalAdmissionReport{Binding: "tv", BindingDigest: binding.Digest(), Enabled: true}}
	return handler{manifest: hostreqkit.SafeguardManifest{Name: "local_service_admission"}, client: client}, req, client
}
func TestInspectIsReadOnlyAndRequiresCurrentInstalledScope(t *testing.T) {
	h, req, client := fixture(t)
	status := h.Inspect(hostreqkit.Host{OS: "linux"}, req)
	if !status.Applied || len(client.actions) != 1 || client.actions[0] != privilegebroker.ActionLocalAdmissionInspect {
		t.Fatalf("inspection=%+v actions=%v", status, client.actions)
	}
	client.report.BindingDigest = "old-policy"
	status = h.Inspect(hostreqkit.Host{OS: "linux"}, req)
	if status.Applied || status.BlockingReason != hostreqkit.BlockingNeedsSudo {
		t.Fatalf("stale policy accepted: %+v", status)
	}
	status, err := h.Apply(hostreqkit.Host{OS: "linux"}, status, hostreqkit.EnsureOptions{})
	if err != nil || status.ExecutionState != hostreqkit.ExecutionManualActionRequired {
		t.Fatalf("unexpected repair: %+v %v", status, err)
	}
	for _, action := range client.actions {
		if action != privilegebroker.ActionLocalAdmissionInspect {
			t.Fatal("stale policy was mutated")
		}
	}
}
func TestApplyRechecksPolicyAndDryRunDoesNotMutate(t *testing.T) {
	h, req, client := fixture(t)
	client.report.Enabled = false
	status := h.Inspect(hostreqkit.Host{OS: "linux"}, req)
	before := len(client.actions)
	result, err := h.Apply(hostreqkit.Host{OS: "linux"}, status, hostreqkit.EnsureOptions{DryRun: true})
	if err != nil || result.ExecutionState != hostreqkit.ExecutionWouldApply || len(client.actions) != before {
		t.Fatal("dry run performed operation")
	}
	result, err = h.Apply(hostreqkit.Host{OS: "linux"}, status, hostreqkit.EnsureOptions{})
	if err != nil || !result.Applied || client.actions[len(client.actions)-1] != privilegebroker.ActionLocalAdmissionEnsure {
		t.Fatalf("apply=%+v %v", result, err)
	}
}
func TestNoBindingOrNonLinuxHasNoHostEffects(t *testing.T) {
	h, req, client := fixture(t)
	for _, platform := range []string{"darwin", "windows"} {
		if s := h.Inspect(hostreqkit.Host{OS: platform}, req); s.SupportClass != hostreqkit.SupportNotApplicable {
			t.Fatalf("platform=%s status=%+v", platform, s)
		}
	}
	req.Config = map[string]any{}
	if s := h.Inspect(hostreqkit.Host{OS: "linux"}, req); s.SupportClass != hostreqkit.SupportNotApplicable {
		t.Fatalf("empty binding: %+v", s)
	}
	if len(client.actions) != 0 {
		t.Fatal("inapplicable safeguard contacted broker")
	}
}
