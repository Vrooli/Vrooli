package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	platform "github.com/vrooli/platform-go"
)

func TestNativeSystemBrokerBootstrapVerifiesBeforePinGuardAndNativeRuntime(t *testing.T) {
	var events []string
	o := platform.NativeSystemServiceBrokerOptions{OperatorHome: "/operator"}
	e := nativeSystemBrokerEntrypoint{read: func(path string) (platform.NativeSystemServiceBrokerOptions, error) {
		if path != "/installed/descriptor.json" {
			t.Fatal(path)
		}
		events = append(events, "read")
		return o, nil
	}, pin: func() error { events = append(events, "pin"); return nil }, guard: func(actual platform.NativeSystemServiceBrokerOptions) error {
		if actual != o {
			t.Fatal("activation guard used machine home")
		}
		events = append(events, "guard")
		return nil
	}, supervise: func(ctx context.Context, actual platform.NativeSystemServiceBrokerOptions, run func(context.Context) error) error {
		if actual != o {
			t.Fatal("operator substituted")
		}
		events = append(events, "native-runtime")
		return run(ctx)
	}, run: func(context.Context, platform.NativeSystemServiceBrokerOptions) error {
		events = append(events, "supervisor")
		return nil
	}}
	var out bytes.Buffer
	handled, code := e.dispatch(context.Background(), []string{platform.NativeSystemSupervisorVerb, "/installed/descriptor.json"}, &out)
	if !handled || code != 0 || strings.Join(events, ",") != "read,pin,guard,native-runtime,supervisor" {
		t.Fatalf("wrong native bootstrap %v %d %v %s", handled, code, events, out.String())
	}
	events = nil
	denied := errors.New("descriptor not machine owned")
	e.read = func(string) (platform.NativeSystemServiceBrokerOptions, error) {
		events = append(events, "read")
		return o, denied
	}
	if handled, code = e.dispatch(context.Background(), []string{platform.NativeSystemSupervisorVerb, "/installed/descriptor.json"}, &out); !handled || code != 1 || strings.Join(events, ",") != "read" {
		t.Fatalf("forged descriptor reached native effects: %v", events)
	}
}

func TestNativeSystemBrokerBootstrapClaimsOnlyFixedHiddenVerb(t *testing.T) {
	var out bytes.Buffer
	e := nativeSystemBrokerEntrypoint{}
	for _, args := range [][]string{nil, {"runtime", "supervisor", "run"}, {"scenario", "restart", "lighting-control"}} {
		if handled, _ := e.dispatch(context.Background(), args, &out); handled {
			t.Fatalf("intercepted %v", args)
		}
	}
	for _, args := range [][]string{{platform.NativeSystemSupervisorVerb}, {platform.NativeSystemSupervisorVerb, "/descriptor", "--extra"}} {
		if handled, code := e.dispatch(context.Background(), args, &out); !handled || code != 1 {
			t.Fatalf("malformed hidden invocation accepted: %v", args)
		}
	}
}
