package hostpresentation

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/shell/shelltest"
)

type fakeProbe struct {
	env map[string]string
	*shelltest.Fake
}

type fakeResult struct {
	output string
	err    error
}

func (p *fakeProbe) Env(name string) string { return p.env[name] }

func probe(env map[string]string, run map[string]fakeResult) *fakeProbe {
	results := make(map[string]shelltest.Result, len(run))
	for key, result := range run {
		results[key] = shelltest.Result{Output: []byte(result.output), Err: result.err}
	}
	return &fakeProbe{env: env, Fake: &shelltest.Fake{Results: results, LookPathFunc: func(string) (string, error) { return "/fake/tool", nil }}}
}

func TestDetectUniversalOverrides(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{name: "ci", env: map[string]string{"CI": "true"}, want: "CI environment"},
		{name: "container env", env: map[string]string{"container": "podman"}, want: "container environment"},
		{name: "docker marker", want: "Docker container"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := probe(tt.env, map[string]fakeResult{"file-exists /.dockerenv": {output: "true"}})
			got := detectWithOS(context.Background(), p, "linux")
			if got.Kind != KindHeadless || got.Reachable || got.Reason != tt.want {
				t.Fatalf("capability = %#v, want headless/unreachable reason %q", got, tt.want)
			}
		})
	}
}

func TestDetectLinuxDecisionTable(t *testing.T) {
	base := map[string]fakeResult{"file-exists /.dockerenv": {output: "false"}, "file-read /proc/version": {output: "Linux version"}}
	tests := []struct {
		name      string
		env       map[string]string
		run       map[string]fakeResult
		uid       int
		wantKind  Kind
		reachable bool
	}{
		{name: "local graphical", env: map[string]string{"DISPLAY": ":0"}, run: merge(base, map[string]fakeResult{"loginctl show-session self -p Remote --value": {output: "no"}}), wantKind: KindLocalGraphical, reachable: true},
		{name: "forwarded graphical", env: map[string]string{"DISPLAY": "localhost:10.0", "SSH_CONNECTION": "client 22 host 22"}, run: merge(base, map[string]fakeResult{"loginctl show-session self -p Remote --value": {output: "yes"}}), wantKind: KindForwardedGraphical, reachable: true},
		{name: "wsl graphical", env: map[string]string{"DISPLAY": ":0"}, run: merge(base, map[string]fakeResult{"file-read /proc/version": {output: "Linux microsoft WSL"}}), wantKind: KindWSLGraphical, reachable: true},
		{name: "elevated desktop", env: map[string]string{"SUDO_USER": "operator"}, run: merge(base, map[string]fakeResult{"loginctl show-seat seat0 -p ActiveSession --value": {output: "2"}, "loginctl show-session 2 -p Name,Type,Remote": {output: "Name=operator\nType=wayland\nRemote=no"}}), uid: 0, wantKind: KindLocalGraphical, reachable: true},
		{name: "remote shell", env: map[string]string{"SSH_CONNECTION": "client 22 host 22", "SSH_TTY": "/dev/pts/1"}, run: base, wantKind: KindRemoteShell},
		{name: "headless", env: map[string]string{}, run: base, wantKind: KindHeadless},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := probe(tt.env, tt.run)
			got := detectLinuxWithUID(context.Background(), p, tt.uid)
			if got.Kind != tt.wantKind || got.Reachable != tt.reachable {
				t.Fatalf("capability = %#v, want kind=%q reachable=%t", got, tt.wantKind, tt.reachable)
			}
		})
	}
}

func TestDetectDarwinDecisionTable(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		out  string
		want Kind
	}{
		{name: "aqua", out: "Aqua", want: KindLocalGraphical},
		{name: "ssh", env: map[string]string{"SSH_CONNECTION": "client 22 host 22"}, out: "Background", want: KindRemoteShell},
		{name: "headless", out: "Background", want: KindHeadless},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := probe(tt.env, map[string]fakeResult{"file-exists /.dockerenv": {output: "false"}, "launchctl managername": {output: tt.out}})
			got := detectWithOS(context.Background(), p, "darwin")
			if got.Kind != tt.want {
				t.Fatalf("kind = %q, want %q (%#v)", got.Kind, tt.want, got)
			}
		})
	}
}

func TestDetectWindowsDecisionTable(t *testing.T) {
	tests := []struct {
		name string
		out  string
		want Kind
	}{
		{name: "console", out: "process=3 console=3", want: KindLocalGraphical},
		{name: "remote desktop", out: "process=7 console=3", want: KindRemoteDesktop},
		{name: "session zero", out: "process=0 console=3", want: KindHeadless},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := probe(nil, map[string]fakeResult{"file-exists /.dockerenv": {output: "false"}, "windows-session": {output: tt.out}})
			got := detectWithOS(context.Background(), p, "windows")
			if got.Kind != tt.want {
				t.Fatalf("kind = %q, want %q (%#v)", got.Kind, tt.want, got)
			}
		})
	}
}

func TestDetectOtherIsUnknown(t *testing.T) {
	p := probe(nil, map[string]fakeResult{"file-exists /.dockerenv": {output: "false"}})
	got := detectWithOS(context.Background(), p, "freebsd")
	if got.Kind != KindUnknown || got.Reachable {
		t.Fatalf("capability = %#v, want unknown/unreachable", got)
	}
}

func TestDetectTimeoutIsDegradedAndUnreachable(t *testing.T) {
	p := probe(map[string]string{"DISPLAY": ":0"}, map[string]fakeResult{
		"file-exists /.dockerenv": {output: "false"},
		"file-read /proc/version": {err: context.DeadlineExceeded},
	})
	got := detectWithOS(context.Background(), p, "linux")
	if !got.Degraded || got.Reachable {
		t.Fatalf("capability = %#v, want degraded/unreachable", got)
	}
}

func TestDetectUsesOnlyDeclaredProbeOperations(t *testing.T) {
	p := probe(nil, map[string]fakeResult{"file-exists /.dockerenv": {output: "false"}, "file-read /proc/version": {output: "Linux"}})
	_ = detectWithOS(context.Background(), p, "linux")
	for _, call := range p.Calls() {
		if !strings.HasPrefix(call, "file-exists ") && !strings.HasPrefix(call, "file-read ") && !strings.HasPrefix(call, "loginctl ") {
			t.Fatalf("unexpected probe call %q", call)
		}
	}
}

func TestProbeBudgetPropagatesCancellation(t *testing.T) {
	p := &blockingProbe{Fake: &shelltest.Fake{RunFunc: func(ctx context.Context, _ string, _ ...string) ([]byte, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}}}
	got := detectWithOS(context.Background(), p, "darwin")
	if !got.Degraded || got.Reachable {
		t.Fatalf("capability = %#v, want degraded/unreachable", got)
	}
}

type blockingProbe struct{ *shelltest.Fake }

func (*blockingProbe) Env(string) string { return "" }

func merge(left, right map[string]fakeResult) map[string]fakeResult {
	result := map[string]fakeResult{}
	for key, value := range left {
		result[key] = value
	}
	for key, value := range right {
		result[key] = value
	}
	return result
}

func TestEvidenceIsStableAndCopied(t *testing.T) {
	evidence := []string{"one"}
	got := capability(KindHeadless, false, "test", evidence...)
	evidence[0] = "changed"
	if !reflect.DeepEqual(got.Evidence, []string{"one"}) {
		t.Fatal("evidence was not copied")
	}
}
