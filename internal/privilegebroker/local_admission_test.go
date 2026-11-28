package privilegebroker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func testAdmissionBinding() LocalAdmissionBinding {
	return LocalAdmissionBinding{ID: "living-room", Scenario: "portal", Listener: "ui", SourceIP: "192.168.10.42", HostIP: "192.168.10.2", Interface: "eth0"}
}

type admissionFirewall struct {
	rules      []admissionRule
	calls      [][]string
	active     bool
	loseAdds   bool
	failDelete bool
}

func (f *admissionFirewall) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	if name != "ufw" {
		return nil, fmt.Errorf("unexpected command %s", name)
	}
	f.calls = append(f.calls, append([]string{}, args...))
	if reflect.DeepEqual(args, []string{"status", "numbered"}) {
		if !f.active {
			return []byte("Status: inactive\n"), nil
		}
		out := "Status: active\n\nTo Action From\n"
		for i, r := range f.rules {
			out += fmt.Sprintf("[%2d] %s %d/tcp on %s  ALLOW IN  %s # %s\n", i+1, r.Host, r.Port, r.Interface, r.Source, r.Tag)
		}
		// An unrelated, broader operator rule must neither satisfy nor be deleted.
		out += "[99] 24965/tcp ALLOW IN Anywhere # operator-rule\n"
		return []byte(out), nil
	}
	remove := len(args) > 1 && args[0] == "--force"
	if remove {
		args = args[2:]
	}
	if len(args) != 14 || args[0] != "allow" || args[1] != "in" || args[2] != "on" || args[4] != "from" || args[6] != "to" || args[8] != "port" || args[10] != "proto" || args[11] != "tcp" || args[12] != "comment" {
		// Fixed argv is intentionally asserted rather than accepting shell syntax.
		return nil, fmt.Errorf("unexpected firewall arguments: %v", args)
	}
	port, _ := strconv.Atoi(args[9])
	rule := admissionRule{Interface: args[3], Source: args[5], Host: args[7], Port: port, Tag: args[13]}
	if remove {
		if f.failDelete {
			return nil, fmt.Errorf("delete failed")
		}
		for i, r := range f.rules {
			if r == rule {
				f.rules = append(f.rules[:i], f.rules[i+1:]...)
				break
			}
		}
	} else if !f.loseAdds {
		f.rules = append(f.rules, rule)
	}
	return nil, nil
}

func admissionFixture(t *testing.T) (*LocalAdmissionManager, *admissionFirewall, *int) {
	t.Helper()
	binding := testAdmissionBinding()
	port := 24965
	firewall := &admissionFirewall{active: true}
	m := &LocalAdmissionManager{Policy: LocalAdmissionPolicy{Version: 1, UID: 1000, Bindings: []LocalAdmissionBinding{binding}}, Executor: firewall, StatePath: filepath.Join(t.TempDir(), "state.json"), Resolve: func(context.Context, LocalAdmissionBinding) (int, error) { return port, nil }}
	return m, firewall, &port
}

func TestLocalAdmissionConvergesOnlyInstalledBindingAndCurrentPort(t *testing.T) {
	m, fw, port := admissionFixture(t)
	ctx := context.Background()
	if r := m.Execute(ctx, ActionLocalAdmissionEnsure, "unknown"); r.Code != "local_binding_not_installed" || len(fw.calls) != 0 {
		t.Fatalf("uninstalled binding reached firewall: %+v", r)
	}
	if r := m.Execute(ctx, ActionLocalAdmissionInspect, "living-room"); r.Status != "pending" || len(fw.rules) != 0 {
		t.Fatalf("inspection mutated: %+v", r)
	}
	r := m.Execute(ctx, ActionLocalAdmissionEnsure, "living-room")
	if r.Status != "changed" || !r.Evidence.Managed || len(fw.rules) != 1 {
		t.Fatalf("first convergence: %+v", r)
	}
	if r := m.Execute(ctx, ActionLocalAdmissionEnsure, "living-room"); r.Changed || r.Status != "verified" {
		t.Fatalf("not idempotent: %+v", r)
	}
	*port = 25012
	if r := m.Execute(ctx, ActionLocalAdmissionEnsure, "living-room"); !r.Changed || len(fw.rules) != 1 || fw.rules[0].Port != 25012 {
		t.Fatalf("port transition: %+v rules=%v", r, fw.rules)
	}
	for _, call := range fw.calls {
		if len(call) > 0 && call[0] == "allow" && strings.Join(call, " ") != "allow in on eth0 from 192.168.10.42 to 192.168.10.2 port 24965 proto tcp comment vrooli-local-v1-1000-living-room" && !strings.Contains(strings.Join(call, " "), "port 25012 proto tcp") {
			t.Fatalf("unexpected exposure %v", call)
		}
	}
}

func TestLocalAdmissionRevocationSurvivesReconciliationAndRestart(t *testing.T) {
	m, fw, _ := admissionFixture(t)
	ctx := context.Background()
	m.Execute(ctx, ActionLocalAdmissionEnsure, "living-room")
	if r := m.Execute(ctx, ActionLocalAdmissionRevoke, "living-room"); r.Status != "verified" || len(fw.rules) != 0 || r.LocalAdmission.Enabled {
		t.Fatalf("revoke=%+v", r)
	}
	restarted := &LocalAdmissionManager{Policy: m.Policy, Executor: fw, Resolve: m.Resolve, StatePath: m.StatePath}
	restarted.Reconcile(ctx)
	if len(fw.rules) != 0 {
		t.Fatal("background convergence undid revocation")
	}
	if r := restarted.Execute(ctx, ActionLocalAdmissionEnsure, "living-room"); !r.LocalAdmission.Enabled || len(fw.rules) != 1 {
		t.Fatalf("explicit re-enable failed: %+v", r)
	}
	restarted.Policy.Bindings = nil
	restarted.Reconcile(ctx)
	if len(fw.rules) != 0 {
		t.Fatal("removed setup binding left an orphan rule")
	}
}

func TestLocalAdmissionRefusesUnverifiedAddsAndCleansStoppedListener(t *testing.T) {
	m, fw, _ := admissionFixture(t)
	ctx := context.Background()
	fw.loseAdds = true
	if r := m.Execute(ctx, ActionLocalAdmissionEnsure, "living-room"); r.Code != "local_rule_not_verified" {
		t.Fatalf("unverified write reported success: %+v", r)
	}
	fw.loseAdds = false
	m.Execute(ctx, ActionLocalAdmissionEnsure, "living-room")
	m.Resolve = func(context.Context, LocalAdmissionBinding) (int, error) { return 0, fmt.Errorf("not running") }
	if r := m.Execute(ctx, ActionLocalAdmissionEnsure, "living-room"); r.Status != "pending" || r.LocalAdmission.Reason != "live_listener_unverified" || len(fw.rules) != 0 {
		t.Fatalf("stopped listener retained admission: %+v", r)
	}
}

func TestLocalAdmissionInactiveFirewallIsNotEnabled(t *testing.T) {
	m, fw, _ := admissionFixture(t)
	fw.active = false
	r := m.Execute(context.Background(), ActionLocalAdmissionEnsure, "living-room")
	if r.Status != "verified" || r.Changed || len(fw.rules) != 0 || r.LocalAdmission.Reason != "firewall_inactive" {
		t.Fatalf("inactive firewall: %+v", r)
	}
}

func TestLocalAdmissionCorruptStateAndUnexpectedRulesFailClosed(t *testing.T) {
	m, fw, _ := admissionFixture(t)
	if err := os.WriteFile(m.StatePath, []byte("broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	if r := m.Execute(context.Background(), ActionLocalAdmissionEnsure, "living-room"); r.Code != "local_state_unavailable" || len(fw.calls) != 0 {
		t.Fatalf("corrupt state: %+v", r)
	}
	if _, _, err := parseAdmissionRules("Status: active\n[ 1] 24965/tcp ALLOW IN Anywhere # vrooli-local-v1-1000-living-room", "vrooli-local-v1-1000-"); err == nil {
		t.Fatal("broad tagged rule accepted")
	}
	_, rules, err := parseAdmissionRules("Status: active\n[ 1] 192.168.10.2 249650/tcp on eth0 ALLOW IN 192.168.10.420 # vrooli-local-v1-1000-living-room", "vrooli-local-v1-1000-")
	if err == nil || len(rules) > 0 {
		t.Fatal("substring port rule accepted")
	}
}

func TestLocalAdmissionConfigRejectsArbitraryTargetsAndPorts(t *testing.T) {
	valid := testAdmissionBinding()
	data, _ := json.Marshal(valid)
	var binding map[string]any
	json.Unmarshal(data, &binding)
	for _, tc := range []struct {
		key   string
		value any
	}{{"port", 22}, {"listener", "ssh"}, {"source_ip", "0.0.0.0/0"}, {"host_ip", "8.8.8.8"}, {"interface", "eth0;id"}, {"scenario", "../portal"}} {
		t.Run(tc.key, func(t *testing.T) {
			bad := map[string]any{}
			for k, v := range binding {
				bad[k] = v
			}
			bad[tc.key] = tc.value
			if _, err := LocalAdmissionBindings(map[string]any{"bindings": []any{bad}}); err == nil {
				t.Fatalf("unsafe config accepted: %v", bad)
			}
		})
	}
	if _, err := LocalAdmissionBindings(map[string]any{"bindings": []any{binding, binding}}); err == nil {
		t.Fatal("duplicate binding accepted")
	}
}

func TestLocalAdmissionBrokerRejectsCrossFamilyAndCallerConfusion(t *testing.T) {
	m, fw, _ := admissionFixture(t)
	broker, err := New(Config{SocketPath: "unused", AllowedUID: 1000, Executor: fw, LocalAdmission: m})
	if err != nil {
		t.Fatal(err)
	}
	req := Request{Version: ProtocolVersion, RequestID: "test", Action: ActionLocalAdmissionEnsure, LocalAdmission: &LocalAdmissionSubject{Binding: "living-room"}}
	if r := broker.Execute(context.Background(), req, 1001); r.Code != "caller_not_authorized" {
		t.Fatalf("wrong uid: %+v", r)
	}
	req.Subject = Subject{Scenario: "portal", CandidateIP: "192.168.10.42", Port: 22}
	if err := Validate(req); err == nil {
		t.Fatal("caller-selected port accepted")
	}
	req.Subject = Subject{}
	req.Action = ActionEdgeUFWAllow
	req.Edge = &EdgeSubject{Port: 443}
	if err := Validate(req); err == nil {
		t.Fatal("cross-family subject accepted")
	}
	if len(fw.calls) != 0 {
		t.Fatal("invalid request reached executor")
	}
}
