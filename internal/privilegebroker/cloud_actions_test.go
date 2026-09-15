package privilegebroker

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

type cloudRecordingExecutor struct {
	calls  [][]string
	status string
}

func (r *cloudRecordingExecutor) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, append([]string{name}, args...))
	if name == "ufw" && len(args) > 0 && args[0] == "status" {
		return []byte(r.status), nil
	}
	return nil, nil
}

// [REQ:STC-P0-008] apt.packages.ensure forwards only allowlisted names and
// never a caller-shaped argument.
func TestAptPolicyAcceptsOnlyAllowlistedPackages(t *testing.T) {
	ok := Request{Version: ProtocolVersion, RequestID: "apt-1", Action: ActionAptPackagesEnsure, Apt: &AptSubject{Packages: []string{"jq", "curl", "jq"}}}
	update, install, err := AptArgs(ok)
	if err != nil {
		t.Fatalf("AptArgs: %v", err)
	}
	wantUpdate := []string{"DEBIAN_FRONTEND=noninteractive", "NEEDRESTART_MODE=a", "apt-get", "update", "-qq"}
	wantInstall := []string{"DEBIAN_FRONTEND=noninteractive", "NEEDRESTART_MODE=a", "apt-get", "install", "-y", "-qq", "--no-install-recommends", "curl", "jq"}
	if !reflect.DeepEqual(update, wantUpdate) || !reflect.DeepEqual(install, wantInstall) {
		t.Fatalf("argv = %q / %q", update, install)
	}
	for name, packages := range map[string][]string{
		"unlisted":         {"curl", "netcat"},
		"option smuggling": {"-o", "curl"},
		"shell fragment":   {"curl; echo pwned"},
		"empty":            {},
		"padded":           {" curl"},
	} {
		t.Run(name, func(t *testing.T) {
			req := ok
			req.Apt = &AptSubject{Packages: packages}
			if err := Validate(req); err == nil {
				t.Fatal("Validate accepted an unlisted package subject")
			}
		})
	}
	req := ok
	req.Edge = &EdgeSubject{Port: 80}
	if err := Validate(req); err == nil {
		t.Fatal("Validate accepted an apt request carrying an edge subject")
	}
}

func TestAptExecutionRunsUpdateThenInstallThroughArgvOnly(t *testing.T) {
	executor := &cloudRecordingExecutor{}
	req := Request{Version: ProtocolVersion, RequestID: "apt-2", Action: ActionAptPackagesEnsure, Apt: &AptSubject{Packages: []string{"caddy"}}}
	result := executeApt(context.Background(), executor, req)
	if result.Status != "completed" || !result.Changed {
		t.Fatalf("result = %+v", result)
	}
	if len(executor.calls) != 2 || executor.calls[0][0] != "env" || executor.calls[1][len(executor.calls[1])-1] != "caddy" {
		t.Fatalf("calls = %q", executor.calls)
	}
	for _, call := range executor.calls {
		if strings.Contains(strings.Join(call, " "), "&&") {
			t.Fatalf("argv carries a shell operator: %q", call)
		}
	}
}

// [REQ:STC-P0-008] edge.ufw.allow admits only 80 and 443 and is allow-only.
func TestEdgePolicyAcceptsOnlyPublicEdgePorts(t *testing.T) {
	for _, port := range []int{80, 443} {
		req := Request{Version: ProtocolVersion, RequestID: "edge", Action: ActionEdgeUFWAllow, Edge: &EdgeSubject{Port: port}}
		args, err := EdgeUFWArgs(req)
		if err != nil {
			t.Fatalf("EdgeUFWArgs(%d): %v", port, err)
		}
		if args[0] != "allow" || len(args) != 4 {
			t.Fatalf("argv = %q", args)
		}
	}
	for _, port := range []int{22, 18767, 8080, 0, -1} {
		req := Request{Version: ProtocolVersion, RequestID: "edge", Action: ActionEdgeUFWAllow, Edge: &EdgeSubject{Port: port}}
		if err := Validate(req); err == nil {
			t.Fatalf("Validate accepted edge port %d", port)
		}
	}
	bridge := validRequest(ActionBridgeUFWAllow)
	bridge.Edge = &EdgeSubject{Port: 80}
	if err := Validate(bridge); err == nil {
		t.Fatal("Validate accepted a bridge request carrying an edge subject")
	}
}

func TestEdgeExecutionAddsRuleOnlyWhenActiveAndAbsent(t *testing.T) {
	req := Request{Version: ProtocolVersion, RequestID: "edge", Action: ActionEdgeUFWAllow, Edge: &EdgeSubject{Port: 443}}
	inactive := &cloudRecordingExecutor{status: "Status: inactive\n"}
	if result := executeEdgeUFW(context.Background(), inactive, req); result.Status != "verified" || len(inactive.calls) != 1 {
		t.Fatalf("inactive result = %+v calls=%q", result, inactive.calls)
	}
	present := &cloudRecordingExecutor{status: "Status: active\n[ 1] 443/tcp ALLOW IN Anywhere # vrooli-edge-http-v1\n"}
	if result := executeEdgeUFW(context.Background(), present, req); result.Status != "already_present" || !result.Evidence.Managed {
		t.Fatalf("present result = %+v", result)
	}
	absent := &cloudRecordingExecutor{status: "Status: active\n[ 1] 22/tcp ALLOW IN Anywhere\n"}
	result := executeEdgeUFW(context.Background(), absent, req)
	if len(absent.calls) < 2 || absent.calls[1][1] != "allow" {
		t.Fatalf("calls = %q", absent.calls)
	}
	if result.Status != "failed" || result.Code != "rule_not_verified" {
		t.Fatalf("unverified add result = %+v", result)
	}
	for _, call := range absent.calls {
		if call[1] == "delete" || call[1] == "reload" {
			t.Fatalf("edge policy issued %q", call)
		}
	}
}

// [REQ:STC-P0-008] process.stop.scoped stops one scenario through the
// lifecycle owner and never accepts a process pattern.
func TestProcessStopPolicyBindsOneScenarioAndWorkdir(t *testing.T) {
	ok := Request{Version: ProtocolVersion, RequestID: "stop", Action: ActionProcessStopScoped, Process: &ProcessSubject{Scenario: "landing-app", Workdir: "/opt/vrooli"}}
	name, args, err := ProcessStopArgs(ok)
	if err != nil {
		t.Fatalf("ProcessStopArgs: %v", err)
	}
	if name != "vrooli" || !reflect.DeepEqual(args, []string{"scenario", "stop", "landing-app", "--json"}) {
		t.Fatalf("argv = %s %q", name, args)
	}
	for name, subject := range map[string]ProcessSubject{
		"pattern":        {Scenario: "landing-*", Workdir: "/opt/vrooli"},
		"pkill shape":    {Scenario: "-f api", Workdir: "/opt/vrooli"},
		"uppercase":      {Scenario: "Landing", Workdir: "/opt/vrooli"},
		"relative dir":   {Scenario: "landing-app", Workdir: "opt/vrooli"},
		"dotdot dir":     {Scenario: "landing-app", Workdir: "/opt/../etc"},
		"space dir":      {Scenario: "landing-app", Workdir: "/opt/vrooli; echo"},
		"uncleaned dir":  {Scenario: "landing-app", Workdir: "/opt//vrooli/"},
		"empty scenario": {Scenario: "", Workdir: "/opt/vrooli"},
	} {
		t.Run(name, func(t *testing.T) {
			req := ok
			subject := subject
			req.Process = &subject
			if err := Validate(req); err == nil {
				t.Fatal("Validate accepted an unbounded process subject")
			}
		})
	}
}

// [REQ:STC-P0-036] edge.caddy.validate / edge.caddy.reload accept only a
// deployment id subject and build a fixed argv that never carries a path,
// so a snippet writer cannot point validation or reload at another config.
func TestCaddyPolicyBindsDeploymentAndFixedArgv(t *testing.T) {
	validate := Request{Version: ProtocolVersion, RequestID: "caddy-1", Action: ActionEdgeCaddyValidate, Caddy: &CaddySubject{DeploymentID: "dep-1"}}
	name, args, err := CaddyArgs(validate)
	if err != nil || name != "caddy" || !reflect.DeepEqual(args, []string{"validate", "--config", CaddyMainConfigPath, "--adapter", "caddyfile"}) {
		t.Fatalf("validate argv = %s %q err=%v", name, args, err)
	}
	reload := validate
	reload.Action = ActionEdgeCaddyReload
	name, args, err = CaddyArgs(reload)
	if err != nil || name != "systemctl" || !reflect.DeepEqual(args, []string{"reload", "caddy"}) {
		t.Fatalf("reload argv = %s %q err=%v", name, args, err)
	}
	for label, req := range map[string]Request{
		"missing subject":    {Version: ProtocolVersion, RequestID: "c", Action: ActionEdgeCaddyValidate},
		"path-shaped id":     {Version: ProtocolVersion, RequestID: "c", Action: ActionEdgeCaddyReload, Caddy: &CaddySubject{DeploymentID: "../etc"}},
		"option-shaped id":   {Version: ProtocolVersion, RequestID: "c", Action: ActionEdgeCaddyReload, Caddy: &CaddySubject{DeploymentID: "--config"}},
		"empty id":           {Version: ProtocolVersion, RequestID: "c", Action: ActionEdgeCaddyValidate, Caddy: &CaddySubject{}},
		"caddy plus edge":    {Version: ProtocolVersion, RequestID: "c", Action: ActionEdgeCaddyValidate, Caddy: &CaddySubject{DeploymentID: "dep"}, Edge: &EdgeSubject{Port: 80}},
		"edge carries caddy": {Version: ProtocolVersion, RequestID: "c", Action: ActionEdgeUFWAllow, Edge: &EdgeSubject{Port: 80}, Caddy: &CaddySubject{DeploymentID: "dep"}},
		"apt carries caddy":  {Version: ProtocolVersion, RequestID: "c", Action: ActionAptPackagesEnsure, Apt: &AptSubject{Packages: []string{"jq"}}, Caddy: &CaddySubject{DeploymentID: "dep"}},
	} {
		if err := Validate(req); err == nil {
			t.Fatalf("%s: Validate accepted %+v", label, req)
		}
	}
}

type caddyFailingExecutor struct {
	calls [][]string
	fail  string
}

func (r *caddyFailingExecutor) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, append([]string{name}, args...))
	if name == r.fail {
		return []byte("Error: adapting config using caddyfile: /etc/caddy/conf.d/x.caddy:3: unrecognized directive"), errExec
	}
	return []byte("Valid configuration"), nil
}

var errExec = fmt.Errorf("exit status 1")

func TestCaddyExecutionReportsTypedFailureWithBoundedDetail(t *testing.T) {
	req := Request{Version: ProtocolVersion, RequestID: "caddy-2", Action: ActionEdgeCaddyValidate, Caddy: &CaddySubject{DeploymentID: "dep-1"}}
	ok := &caddyFailingExecutor{}
	if result := executeCaddy(context.Background(), ok, req); result.Status != "completed" || result.Changed || len(ok.calls) != 1 {
		t.Fatalf("ok result = %+v calls=%q", result, ok.calls)
	}
	bad := &caddyFailingExecutor{fail: "caddy"}
	result := executeCaddy(context.Background(), bad, req)
	if result.Status != "failed" || result.Code != "caddy_validate_failed" || !strings.Contains(result.Evidence.Detail, "unrecognized directive") {
		t.Fatalf("failed result = %+v", result)
	}
	req.Action = ActionEdgeCaddyReload
	badReload := &caddyFailingExecutor{fail: "systemctl"}
	if result := executeCaddy(context.Background(), badReload, req); result.Code != "caddy_reload_failed" {
		t.Fatalf("reload result = %+v", result)
	}
}
