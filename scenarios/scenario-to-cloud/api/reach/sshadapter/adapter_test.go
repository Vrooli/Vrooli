package sshadapter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"scenario-to-cloud/identity"
	"scenario-to-cloud/reach"
)

type fakeRunner struct {
	commands []string
	stdin    [][]byte
	result   Result
	err      error
}

func (f *fakeRunner) Run(_ context.Context, _ ConnectionConfig, command string, opts RunOptions) (Result, error) {
	f.commands = append(f.commands, command)
	f.stdin = append(f.stdin, opts.Stdin)
	return f.result, f.err
}

func target() identity.TargetRef {
	return identity.TargetRef{Transport: identity.TransportSSH, Locator: identity.TargetLocator{Host: "203.0.113.10", Port: 22, User: "root", Workdir: "/root/Vrooli"}}
}

func adapter(runner *fakeRunner) *Adapter {
	return &Adapter{Runner: runner, Config: func(context.Context, identity.TargetRef) (ConnectionConfig, error) {
		return NewConfig("203.0.113.10", 22, "root", ""), nil
	}}
}

// [REQ:STC-P0-024] Every argument is single-quoted by one policy and the
// remote string runs the bound workdir's vrooli binary; stdin never enters
// the command string.
func TestExecRendersQuotedArgvAndKeepsStdinOut(t *testing.T) {
	runner := &fakeRunner{result: Result{ExitCode: 0, Stdout: `{"outcome":"succeeded"}`}}
	a := adapter(runner)
	res, err := a.Exec(context.Background(), target(), reach.Command{Verb: "cloud-target credential ingest", Args: []string{"--deployment", "dep-1", "--binding", "cb_abc", "--fence", "3"}, Stdin: []byte("canary-secret-9f3a")})
	if err != nil {
		t.Fatal(err)
	}
	if res.Transport != identity.TransportSSH || res.Stdout == "" {
		t.Fatalf("result = %+v", res)
	}
	cmd := runner.commands[0]
	for _, want := range []string{"cd '/root/Vrooli'", "'cloud-target' 'credential' 'ingest'", "'--deployment' 'dep-1'", "'--fence' '3'"} {
		if !strings.Contains(cmd, want) {
			t.Fatalf("remote command %q lacks %q", cmd, want)
		}
	}
	if strings.Contains(cmd, "canary-secret") {
		t.Fatalf("stdin value leaked into the command string: %q", cmd)
	}
	if string(runner.stdin[0]) != "canary-secret-9f3a" {
		t.Fatalf("stdin not forwarded")
	}
}

func TestExecRefusesShellSyntaxBeforeRunning(t *testing.T) {
	runner := &fakeRunner{}
	a := adapter(runner)
	_, err := a.Exec(context.Background(), target(), reach.Command{Verb: "cloud-target receipt get", Args: []string{"--deployment", "dep;id"}})
	if !reach.IsKind(err, reach.KindInvalidArgument) {
		t.Fatalf("expected invalid_request, got %v", err)
	}
	if len(runner.commands) != 0 {
		t.Fatalf("runner must not be called, got %v", runner.commands)
	}
}

// Exit 255 is the ssh client's own failure (unreachable host); exit 127 is
// a missing vrooli binary; any other non-zero exit is the verb's typed
// answer and is returned without a transport error.
func TestExecClassifiesTransportFailuresByExitCode(t *testing.T) {
	cases := []struct {
		exit int
		kind reach.Kind
	}{
		{255, reach.KindTargetOffline},
		{127, reach.KindProtocolUnsupported},
	}
	for _, c := range cases {
		runner := &fakeRunner{result: Result{ExitCode: c.exit}, err: errors.New("ssh failed")}
		_, err := adapter(runner).Exec(context.Background(), target(), reach.Command{Verb: "cloud-target receipt get", Args: []string{"--deployment", "dep-1"}})
		if !reach.IsKind(err, c.kind) {
			t.Fatalf("exit %d: expected %s, got %v", c.exit, c.kind, err)
		}
	}
	runner := &fakeRunner{result: Result{ExitCode: 2, Stdout: `{"code":"fence_stale"}`}, err: errors.New("ssh failed")}
	res, err := adapter(runner).Exec(context.Background(), target(), reach.Command{Verb: "cloud-target release activate", Args: []string{"--fence", "1"}})
	if err != nil || res.ExitCode != 2 {
		t.Fatalf("verb refusal must be returned as the target's answer: res=%+v err=%v", res, err)
	}
	runner = &fakeRunner{err: errors.New("boom")}
	if _, err := adapter(runner).Exec(context.Background(), target(), reach.Command{Verb: "cloud-target receipt get", Args: []string{"--deployment", "dep-1"}}); !reach.IsKind(err, reach.KindTransport) {
		t.Fatalf("expected transport_failure, got %v", err)
	}
}

// scriptedRunner answers per command substring so the platform probe and the
// native-CLI probe can differ.
type scriptedRunner struct {
	fakeRunner
	answers map[string]Result
	errs    map[string]error
	copies  []string
}

func (s *scriptedRunner) Run(ctx context.Context, cfg ConnectionConfig, command string, opts RunOptions) (Result, error) {
	s.commands = append(s.commands, command)
	s.stdin = append(s.stdin, opts.Stdin)
	for needle, err := range s.errs {
		if strings.Contains(command, needle) {
			return s.answers[needle], err
		}
	}
	for needle, res := range s.answers {
		if strings.Contains(command, needle) {
			return res, nil
		}
	}
	return s.result, s.err
}

func (s *scriptedRunner) Copy(_ context.Context, _ ConnectionConfig, local, remote string, _ SCPOptions) error {
	s.copies = append(s.copies, local+" -> "+remote)
	return nil
}

func scripted(runner *scriptedRunner) *Adapter {
	return &Adapter{Runner: runner, SCP: runner, Config: func(context.Context, identity.TargetRef) (ConnectionConfig, error) {
		return NewConfig("203.0.113.10", 22, "root", ""), nil
	}}
}

// [REQ:STC-P0-024] Negotiation learns the platform from the transport probe
// before any native binary exists, so a fresh host is online with a known
// platform and an absent binary is protocol_unsupported, never offline.
func TestNegotiateReportsPlatformAndNativeCLI(t *testing.T) {
	runner := &scriptedRunner{answers: map[string]Result{"uname": {ExitCode: 0, Stdout: "Linux\nx86_64\n"}}}
	caps, err := scripted(runner).Negotiate(context.Background(), target())
	if err != nil || !caps.Online || !caps.NativeCLI || caps.Transport != identity.TransportSSH || caps.Platform != "linux/amd64" {
		t.Fatalf("caps=%+v err=%v", caps, err)
	}
	runner = &scriptedRunner{answers: map[string]Result{"uname": {ExitCode: 0, Stdout: "Linux\naarch64\n"}, "cloud-target": {ExitCode: 127}}, errs: map[string]error{"cloud-target": errors.New("ssh failed")}}
	caps, err = scripted(runner).Negotiate(context.Background(), target())
	if !reach.IsKind(err, reach.KindProtocolUnsupported) || !caps.Online || caps.NativeCLI || caps.Platform != "linux/arm64" {
		t.Fatalf("absent binary must be protocol_unsupported with online=true and a platform: caps=%+v err=%v", caps, err)
	}
	runner = &scriptedRunner{answers: map[string]Result{"uname": {ExitCode: 255}}, errs: map[string]error{"uname": errors.New("ssh failed")}}
	if caps, err := scripted(runner).Negotiate(context.Background(), target()); !reach.IsKind(err, reach.KindTargetOffline) || caps.Online {
		t.Fatalf("unreachable host must be target_offline: caps=%+v err=%v", caps, err)
	}
}

// [REQ:STC-P0-024] Delivery places bytes through scp only, proves the remote
// digest equals the local bytes before reporting, and applies the requested
// mode; a digest mismatch is a transport failure.
func TestDeliverVerifiesDigestAndAppliesMode(t *testing.T) {
	dir := t.TempDir()
	local := filepath.Join(dir, "vrooli-linux-amd64")
	if err := os.WriteFile(local, []byte("binary-bytes"), 0o755); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte("binary-bytes"))
	digest := hex.EncodeToString(sum[:])
	runner := &scriptedRunner{answers: map[string]Result{"sha256sum": {ExitCode: 0, Stdout: digest + "  /root/Vrooli/.vrooli/bin/vrooli\n"}}}
	receipt, err := scripted(runner).Deliver(context.Background(), target(), reach.Delivery{Files: []reach.ArtifactFile{{Role: "native_cli", LocalPath: local, RemotePath: "/root/Vrooli/.vrooli/bin/vrooli", Mode: 0o755}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(receipt.Files) != 1 || receipt.Files[0].SHA256 != digest || receipt.Transport != identity.TransportSSH {
		t.Fatalf("receipt = %+v", receipt)
	}
	if len(runner.copies) != 1 || !strings.HasSuffix(runner.copies[0], "-> /root/Vrooli/.vrooli/bin/vrooli") {
		t.Fatalf("copies = %v", runner.copies)
	}
	joined := strings.Join(runner.commands, "\n")
	for _, want := range []string{"mkdir -p '/root/Vrooli/.vrooli/bin'", "sha256sum -- '/root/Vrooli/.vrooli/bin/vrooli'", "chmod 0755 '/root/Vrooli/.vrooli/bin/vrooli'"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("commands %q lack %q", joined, want)
		}
	}
	runner = &scriptedRunner{answers: map[string]Result{"sha256sum": {ExitCode: 0, Stdout: strings.Repeat("0", 64) + "  x\n"}}}
	if _, err := scripted(runner).Deliver(context.Background(), target(), reach.Delivery{Files: []reach.ArtifactFile{{LocalPath: local, RemotePath: "/root/Vrooli/.vrooli/bin/vrooli"}}}); !reach.IsKind(err, reach.KindTransport) {
		t.Fatalf("digest mismatch must be a transport failure, got %v", err)
	}
	if _, err := scripted(&scriptedRunner{}).Deliver(context.Background(), target(), reach.Delivery{Files: []reach.ArtifactFile{{LocalPath: local, RemotePath: "relative/path"}}}); !reach.IsKind(err, reach.KindInvalidArgument) {
		t.Fatalf("relative remote path must be refused, got %v", err)
	}
}
