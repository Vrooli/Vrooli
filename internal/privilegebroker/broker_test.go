//nolint:goconst // test data deliberately reuses stable fixture values.
package privilegebroker

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeExecutor struct {
	calls  [][]string
	status []string
}

func (f *fakeExecutor) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	if len(args) >= 2 && args[0] == "status" {
		if len(f.status) == 0 {
			return []byte("Status: active\n"), nil
		}
		out := f.status[0]
		f.status = f.status[1:]
		return []byte(out), nil
	}
	return nil, nil
}

func TestBrokerAllowUsesFixedArgvAndVerifiesManagedRule(t *testing.T) {
	fake := &fakeExecutor{status: []string{"Status: active\n", "Status: active\n18767/tcp ALLOW IN 192.168.1.176 # vrooli-bridge-admission-v1\n"}}
	_, err := New(Config{AllowedUID: 1000, Executor: fake})
	if err == nil {
		t.Fatal("expected socket path validation")
	}
	b, err := New(Config{SocketPath: "/tmp/test.sock", AllowedUID: 1000, Executor: fake})
	if err != nil {
		t.Fatal(err)
	}
	result := b.Execute(context.Background(), validRequest(ActionBridgeUFWAllow), 1000)
	if result.Status != "changed" || !result.Evidence.Managed {
		t.Fatalf("result=%+v", result)
	}
	if got := strings.Join(fake.calls[1], " "); got != "ufw allow from 192.168.1.176 to any port 18767 proto tcp comment vrooli-bridge-admission-v1" {
		t.Fatalf("call=%q", got)
	}
}

func TestBrokerRejectsWrongPeerBeforeExecutor(t *testing.T) {
	fake := &fakeExecutor{}
	b, err := New(Config{SocketPath: "/tmp/test.sock", AllowedUID: 1000, Executor: fake})
	if err != nil {
		t.Fatal(err)
	}
	result := b.Execute(context.Background(), validRequest(ActionBridgeUFWInspect), 1001)
	if result.Code != "caller_not_authorized" || len(fake.calls) != 0 {
		t.Fatalf("result=%+v calls=%v", result, fake.calls)
	}
}

func TestBrokerRevokeRemovesOnlyManagedAdmissionRule(t *testing.T) {
	fake := &fakeExecutor{status: []string{
		"Status: active\n18767/tcp ALLOW IN 192.168.1.176 # vrooli-bridge-admission-v1\n",
		"Status: active\n",
	}}
	b, err := New(Config{SocketPath: "/tmp/test.sock", AllowedUID: 1000, Executor: fake})
	if err != nil {
		t.Fatal(err)
	}
	result := b.Execute(context.Background(), validRequest(ActionBridgeUFWRevoke), 1000)
	if result.Status != "changed" || !result.Changed || result.Evidence.Managed {
		t.Fatalf("result=%+v", result)
	}
	if got := strings.Join(fake.calls[1], " "); got != "ufw delete allow from 192.168.1.176 to any port 18767 proto tcp comment vrooli-bridge-admission-v1" {
		t.Fatalf("call=%q", got)
	}
}

func TestBrokerUnavailableDoesNotClaimFirewallOpen(t *testing.T) {
	b, err := New(Config{SocketPath: "/tmp/test.sock", AllowedUID: 1000, Executor: errExecutor{errors.New("missing")}})
	if err != nil {
		t.Fatal(err)
	}
	result := b.Execute(context.Background(), validRequest(ActionBridgeUFWInspect), 1000)
	if result.Status != "failed" {
		t.Fatalf("result=%+v", result)
	}
}

func TestBrokerRuntimeHomeRepairUsesTypedCallbackAndCallerBinding(t *testing.T) {
	called := false
	b, err := New(Config{SocketPath: "/tmp/test.sock", AllowedUID: 1000, RuntimeHomeRepair: func(_ context.Context, subject RuntimeHomeSubject) Result {
		called = true
		return Result{Version: ProtocolVersion, Action: ActionRuntimeHomeOwnershipRepair, Status: "changed", Evidence: Evidence{Scanned: 3, Repaired: 2}}
	}})
	if err != nil {
		t.Fatal(err)
	}
	req := Request{Version: ProtocolVersion, RequestID: "runtime-1", Action: ActionRuntimeHomeOwnershipRepair, RuntimeHome: &RuntimeHomeSubject{Class: "cache", ExpectedUID: 1000, ExpectedGID: 1000}}
	result := b.Execute(context.Background(), req, 1000)
	if result.Status != "changed" || !called || result.Evidence.Repaired != 2 {
		t.Fatalf("result=%+v called=%v", result, called)
	}
	req.RuntimeHome.ExpectedUID = 1001
	result = b.Execute(context.Background(), req, 1000)
	if result.Code != "runtime_home_identity_not_caller" {
		t.Fatalf("identity mismatch result=%+v", result)
	}
}

type errExecutor struct{ err error }

func (e errExecutor) Run(context.Context, string, ...string) ([]byte, error) { return nil, e.err }
