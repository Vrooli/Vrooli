package setupprivilege

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/apparmorprofiles"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/privilegebroker"
	"github.com/vrooli/vrooli/internal/safeguards/observabilityinstall"
	onboardingapplyprivileges "github.com/vrooli/vrooli/internal/safeguards/onboarding-apply-privileges"
)

const helperProcessEnv = "VROOLI_SETUP_PRIVILEGE_TEST_HELPER"
const helperWaitForCancelEnv = "VROOLI_SETUP_PRIVILEGE_TEST_WAIT_FOR_CANCEL"
const helperOperationMarkerEnv = "VROOLI_SETUP_PRIVILEGE_TEST_OPERATION_MARKER"

type stagedCancelContext struct {
	context.Context
	firstCheck    chan struct{}
	continueFirst chan struct{}
	secondCheck   chan struct{}
	checks        sync.Once
}

func (c *stagedCancelContext) Err() error {
	first := false
	c.checks.Do(func() {
		first = true
		close(c.firstCheck)
	})
	if first {
		<-c.continueFirst
		return nil
	}
	close(c.secondCheck)
	return c.Context.Err()
}

func TestPackageCommandAllowsOnlyFixedManagersAndSafePackageTokens(t *testing.T) {
	tests := []struct {
		manager string
		pkg     string
		want    []string
		wantErr bool
	}{
		{manager: "apt", pkg: "libx11-6", want: []string{"install", "-y", "libx11-6"}},
		{manager: "dnf", pkg: "golang.x86_64", want: []string{"install", "-y", "golang.x86_64"}},
		{manager: "pacman", pkg: "go", want: []string{"-S", "--noconfirm", "go"}},
		{manager: "apk", pkg: "go", want: []string{"add", "go"}},
		{manager: "sh", pkg: "go;id", wantErr: true},
		{manager: "apt-get", pkg: "-oAPT::Get::AllowUnauthenticated=true", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.manager+"/"+tt.pkg, func(t *testing.T) {
			_, args, err := packageCommand(tt.manager, tt.pkg)
			if (err != nil) != tt.wantErr {
				t.Fatalf("packageCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && strings.Join(args, " ") != strings.Join(tt.want, " ") {
				t.Fatalf("packageCommand() args = %v, want %v", args, tt.want)
			}
		})
	}
}

func TestSystemPackageInstallHonorsContextDeadline(t *testing.T) {
	binDir := t.TempDir()
	apt := filepath.Join(binDir, "apt-get")
	if err := os.WriteFile(apt, []byte("#!/bin/sh\nsleep 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err := RunSystemPackageInstall(ctx, "apt", "git", io.Discard, io.Discard)
	if err == nil || !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatalf("package install error=%v context=%v, want command stopped at context deadline", err, ctx.Err())
	}
}

func TestOnboardingGrantPolicyUsesCurrentIdentityAndSelectedRequirements(t *testing.T) {
	tools, safeguards := []string{"git"}, []string{"clock"}
	content, err := onboardingapplyprivileges.PolicyContentForCurrentProcess(tools, safeguards)
	if err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	executable = strings.NewReplacer("\\", "\\\\", " ", "\\ ").Replace(executable)
	wantPrefix := "# Managed by Vrooli -- do not edit manually\n" + hostreqkit.InvokingUser() + " ALL=(root) NOPASSWD: "
	if !strings.HasPrefix(content, wantPrefix) {
		t.Fatalf("generated grant identity = %q, want prefix %q", content, wantPrefix)
	}
	if !strings.Contains(content, executable+" host install git --json --sudo-mode error") || !strings.Contains(content, executable+" host safeguard clock --json --sudo-mode error") {
		t.Fatalf("generated grant omitted selected requirements: %q", content)
	}
	if _, err := onboardingapplyprivileges.PolicyContentForCurrentProcess([]string{"git;id"}, nil); err == nil {
		t.Fatal("unsafe selected requirement was accepted")
	}
}

func TestAptRepositoryRequestIsRestrictedToKnownSourcesAndPaths(t *testing.T) {
	key := []byte("binary-key")
	valid := "deb [signed-by=/usr/share/keyrings/stripe.gpg] https://packages.stripe.dev/stripe-cli-debian-local stable main"
	if err := validateAptRepository("stripe", key, valid); err != nil {
		t.Fatalf("valid repository rejected: %v", err)
	}
	hashicorp := "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg arch=arm64] https://apt.releases.hashicorp.com noble main"
	if err := validateAptRepository("hashicorp", key, hashicorp); err != nil {
		t.Fatalf("valid arm64 repository rejected: %v", err)
	}
	for _, tc := range []struct{ name, repository, source string }{
		{"unknown repository", "attacker", valid},
		{"wrong key path", "stripe", "deb [signed-by=/tmp/key] https://packages.stripe.dev stable main"},
		{"wrong source host", "stripe", "deb [signed-by=/usr/share/keyrings/stripe.gpg] https://evil.example stable main"},
		{"multiple lines", "stripe", valid + "\nmalicious"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateAptRepository(tc.repository, key, tc.source); err == nil {
				t.Fatal("invalid repository request accepted")
			}
		})
	}
}

func TestAppArmorProfileRequestsAreClosedAndReuseWorker(t *testing.T) {
	var calls []string
	input := strings.NewReader("{\"version\":1,\"kind\":\"install-apparmor-profile\",\"profile_id\":\"bubblewrap-userns\"}\n" +
		"{\"version\":1,\"kind\":\"install-apparmor-profile\",\"profile_id\":\"workspace-sandbox-userns\"}\n")
	var output bytes.Buffer
	err := runWorker(input, &output, nil, nil, nil, nil, func(_ context.Context, id string) error {
		calls = append(calls, id)
		return nil
	}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"ok":true`) || !reflect.DeepEqual(calls, []string{"bubblewrap-userns", "workspace-sandbox-userns"}) {
		t.Fatalf("worker calls=%v response=%s", calls, output.String())
	}
}

func TestAppArmorProfileRequestRejectsOpenFieldsAndUnknownIDs(t *testing.T) {
	for _, input := range []string{
		`{"version":1,"kind":"install-apparmor-profile","profile_id":"bubblewrap-userns","path":"/tmp/evil"}`,
		`{"version":1,"kind":"install-apparmor-profile","profile_id":"bubblewrap-userns","manager":"apt"}`,
		`{"version":1,"kind":"install-apparmor-profile","profile_id":"bubblewrap-userns","manager":""}`,
		`{"version":1,"kind":"install-apparmor-profile","profile_id":"unknown"}`,
	} {
		t.Run(input, func(t *testing.T) {
			var output bytes.Buffer
			calls := 0
			err := runWorker(strings.NewReader(input+"\n"), &output, nil, nil, nil, nil, func(context.Context, string) error { calls++; return nil }, nil, nil)
			if err == nil && (calls != 0 || !strings.Contains(output.String(), `"ok":false`)) {
				t.Fatalf("open request accepted: calls=%d response=%s", calls, output.String())
			}
			if strings.Contains(input, `"path"`) && err == nil {
				t.Fatal("unknown path field was not rejected during decoding")
			}
		})
	}
}

func TestPrimaryFingerprintRequiresOneCompletePrimaryKey(t *testing.T) {
	valid := "pub:-:4096:1:KEYID:1::::::scSC:\nfpr:::::::::D55C0D1AC78A8D8126CB631CFC9CA96ACA026560:\nsub:-:4096:1:SUBKEY:1::::::e:\nfpr:::::::::SUBFINGERPRINT:"
	fingerprint, err := primaryFingerprint(valid)
	if err != nil || fingerprint != aptRepositories["hashicorp"].primaryFingerprint {
		t.Fatalf("primaryFingerprint() = %q, %v", fingerprint, err)
	}
	if _, err := primaryFingerprint(valid + "\npub:-:4096:1:SECOND:1::::::scSC:\nfpr:::::::::SECONDPRIMARY:"); err == nil {
		t.Fatal("multiple primary keys were accepted")
	}
	if _, err := primaryFingerprint("pub:-:4096:1:KEYID:1::::::scSC:"); err == nil {
		t.Fatal("primary key without fingerprint was accepted")
	}
}

func TestAptRepositoryRetryRepairsPartialWritesAndIndexFailure(t *testing.T) {
	spec := aptRepositories["stripe"]
	files := map[string]string{}
	failedSourceWrite := false
	failedIndexUpdate := false
	write := func(path string, data []byte) error {
		if path == spec.sourcePath && !failedSourceWrite {
			failedSourceWrite = true
			return errors.New("simulated source write failure")
		}
		files[path] = string(data)
		return nil
	}
	update := func(context.Context) error {
		if !failedIndexUpdate {
			failedIndexUpdate = true
			return errors.New("simulated apt index failure")
		}
		return nil
	}
	key, source := []byte("key-bytes"), "deb [signed-by=/usr/share/keyrings/stripe.gpg] https://packages.stripe.dev/stripe-cli-debian-local stable main"
	if err := applyAptRepository(context.Background(), spec, key, source, write, update); err == nil {
		t.Fatal("partial source-write failure was ignored")
	}
	if _, exists := files[spec.sourcePath]; exists {
		t.Fatal("source file appeared despite injected source-write failure")
	}
	if err := applyAptRepository(context.Background(), spec, key, source, write, update); err == nil {
		t.Fatal("APT index failure was ignored")
	}
	if files[spec.sourcePath] != source+"\n" {
		t.Fatalf("source file after index failure = %q", files[spec.sourcePath])
	}
	if err := applyAptRepository(context.Background(), spec, key, source, write, update); err != nil {
		t.Fatalf("retry did not converge: %v", err)
	}
	if files[spec.keyPath] != string(key) || files[spec.sourcePath] != source+"\n" {
		t.Fatalf("repository files did not converge: %#v", files)
	}
}

func TestWorkerDispatchesOnlyTypedOperationsAndKeepsSessionOpen(t *testing.T) {
	input := strings.Join([]string{
		`{"version":1,"kind":"install-package","manager":"apt","package":"git"}`,
		`{"version":1,"kind":"install-apt-repository","repository":"stripe","key_data":"YmluYXJ5LWtleQ==","source_line":"deb [signed-by=/usr/share/keyrings/stripe.gpg] https://packages.stripe.dev/stripe-cli-debian-local stable main"}`,
		`{"version":1,"kind":"install-privilege-broker","repo_root":"/repo"}`,
		`{"version":1,"kind":"install-onboarding-apply-grant","tools":["git"],"safeguards":["clock"]}`,
		`{"version":1,"kind":"run"}`,
	}, "\n")
	var output bytes.Buffer
	var calls []string
	err := runWorker(strings.NewReader(input), &output,
		func(_ context.Context, manager, pkg string, stdout, stderr io.Writer) error {
			calls = append(calls, manager+":"+pkg)
			_, _ = io.WriteString(stdout, "package output")
			return nil
		},
		func(_ context.Context, repo string, key []byte, source string) error {
			calls = append(calls, "repo:"+repo+":"+string(key)+":"+source)
			return nil
		},
		func(_ context.Context, root string) (privilegebroker.SetupStatus, error) {
			calls = append(calls, "broker:"+root)
			return privilegebroker.SetupStatus{Available: true, Supported: true}, nil
		},
		func(_ context.Context, tools, safeguards []string) error {
			calls = append(calls, "grant:"+strings.Join(tools, ",")+":"+strings.Join(safeguards, ","))
			return nil
		},
		func(_ context.Context, id string) error {
			if _, ok := apparmorprofiles.Lookup(id); !ok {
				return errors.New("unexpected profile")
			}
			return nil
		},
		nil,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(calls, ",") != "apt:git,repo:stripe:binary-key:deb [signed-by=/usr/share/keyrings/stripe.gpg] https://packages.stripe.dev/stripe-cli-debian-local stable main,broker:/repo,grant:git:clock" {
		t.Fatalf("worker calls = %v", calls)
	}
	var responses []response
	decoder := json.NewDecoder(&output)
	for {
		var item response
		if err := decoder.Decode(&item); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}
		responses = append(responses, item)
	}
	if len(responses) != 5 || !responses[0].OK || !responses[1].OK || !responses[2].OK || !responses[3].OK || responses[4].OK || responses[4].Error == "" {
		t.Fatalf("worker responses = %#v", responses)
	}
	if responses[0].Stdout != "package output" || responses[2].BrokerStatus == nil || !responses[2].BrokerStatus.Available {
		t.Fatalf("typed responses lost output/status: %#v", responses)
	}
}

func TestWorkerBoundsCapturedPackageOutput(t *testing.T) {
	var output bytes.Buffer
	err := runWorker(strings.NewReader(`{"version":1,"kind":"install-package","manager":"apt","package":"git"}`), &output,
		func(_ context.Context, _, _ string, stdout, _ io.Writer) error {
			_, _ = io.WriteString(stdout, strings.Repeat("x", maxOutputBytes+1))
			return nil
		}, nil, nil, nil, nil, nil, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	var result response
	if err := json.NewDecoder(&output).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if !result.OK || !result.OutputTruncated || len(result.Stdout) != maxOutputBytes {
		t.Fatalf("bounded response = ok:%v truncated:%v output:%d", result.OK, result.OutputTruncated, len(result.Stdout))
	}
}

func TestWorkerRejectsUnknownJSONFields(t *testing.T) {
	err := runWorker(strings.NewReader(`{"version":1,"kind":"install-onboarding-apply-grant","content":"arbitrary policy","tools":["git"]}`), &bytes.Buffer{},
		func(context.Context, string, string, io.Writer, io.Writer) error { return nil }, nil, nil, nil, nil, nil, nil)
	if err == nil {
		t.Fatalf("expected protocol decoding error, got %v", err)
	}
}

func TestWorkerHostHardeningRequestIsClosedAndTyped(t *testing.T) {
	var called bool
	run := func(payload string) response {
		t.Helper()
		var got response
		out := &bytes.Buffer{}
		err := runWorker(strings.NewReader(payload+"\n"), out, nil, nil, nil, nil, nil,
			func(_ context.Context, policy hostreqkit.HostHardeningPolicy) error {
				called = true
				if policy.OopsPolicy != "log-and-continue" || policy.SoftlockupPolicy != "warn" || policy.HungTaskTimeout != 30 {
					t.Fatalf("unexpected policy: %+v", policy)
				}
				return nil
			}, nil)
		if err != nil {
			t.Fatalf("run worker: %v", err)
		}
		if err := json.Unmarshal(out.Bytes(), &got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		return got
	}

	valid := `{"version":1,"kind":"apply-host-hardening","host_hardening":{"oops_policy":"log-and-continue","softlockup_policy":"warn","hung_task_timeout_secs":30}}`
	if got := run(valid); !got.OK || !called {
		t.Fatalf("valid request = %+v, called=%v", got, called)
	}
	called = false
	if got := run(strings.TrimSuffix(valid, "}") + `,"package":"unexpected"}`); got.OK || called {
		t.Fatalf("request with unrelated field = %+v, called=%v", got, called)
	}
	called = false
	invalid := strings.Replace(valid, `"warn"`, `"unexpected"`, 1)
	if got := run(invalid); got.OK || called {
		t.Fatalf("invalid policy = %+v, called=%v", got, called)
	}
}

func TestWorkerObservabilityRequestIsClosedAndIdentityBound(t *testing.T) {
	oldRootFn := hostreqkit.RunningAsRootFn
	hostreqkit.RunningAsRootFn = func() bool { return true }
	t.Cleanup(func() { hostreqkit.RunningAsRootFn = oldRootFn })
	t.Setenv("SUDO_USER", "operator")
	valid := []string{
		`{"version":1,"kind":"install-observability","observability_target":1,"target_user":"operator"}`,
		`{"version":1,"kind":"install-observability","observability_target":2,"target_user":"operator","retain_vmcores":3}`,
	}
	invalid := []string{
		`{"version":1,"kind":"install-observability","observability_target":0,"target_user":"operator"}`,
		`{"version":1,"kind":"install-observability","observability_target":3,"target_user":"operator"}`,
		`{"version":1,"kind":"install-observability","observability_target":1,"target_user":"operator","retain_vmcores":2}`,
		`{"version":1,"kind":"install-observability","observability_target":2,"target_user":"operator","retain_vmcores":0}`,
		`{"version":1,"kind":"install-observability","observability_target":1,"target_user":"other"}`,
		`{"version":1,"kind":"install-observability","observability_target":1,"target_user":"operator","path":"/etc/shadow"}`,
	}
	var calls []string
	invoke := func(payload string) response {
		t.Helper()
		var out bytes.Buffer
		err := runWorker(strings.NewReader(payload+"\n"), &out, nil, nil, nil, nil, nil, nil,
			func(_ context.Context, target observabilityinstall.Target, user string, retention int) error {
				calls = append(calls, fmt.Sprintf("%d:%s:%d", target, user, retention))
				return nil
			})
		if strings.Contains(payload, `"path"`) {
			if err == nil {
				t.Fatal("unknown path field decoded")
			}
			return response{}
		}
		if err != nil {
			t.Fatalf("run worker: %v", err)
		}
		var got response
		if err := json.Unmarshal(out.Bytes(), &got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		return got
	}
	for _, payload := range valid {
		if got := invoke(payload); !got.OK {
			t.Fatalf("valid request response = %+v", got)
		}
	}
	if !reflect.DeepEqual(calls, []string{"1:operator:0", "2:operator:3"}) {
		t.Fatalf("worker calls = %v", calls)
	}
	calls = nil
	for _, payload := range invalid {
		if got := invoke(payload); got.OK {
			t.Fatalf("invalid request accepted: %s", payload)
		}
	}
	if len(calls) != 0 {
		t.Fatalf("invalid requests reached privileged installer: %v", calls)
	}
}

func TestSetupPrivilegeSessionHelperProcess(t *testing.T) {
	if os.Getenv(helperProcessEnv) != "1" {
		return
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	err := runWorkerContext(ctx, os.Stdin, os.Stdout,
		func(_ context.Context, _, pkg string, stdout, _ io.Writer) error {
			_, _ = io.WriteString(stdout, "installed:"+pkg+"\n")
			return nil
		},
		func(context.Context, string, []byte, string) error { return nil },
		func(_ context.Context, root string) (privilegebroker.SetupStatus, error) {
			if os.Getenv(helperWaitForCancelEnv) == "1" {
				<-ctx.Done()
				return privilegebroker.SetupStatus{}, ctx.Err()
			}
			if marker := os.Getenv(helperOperationMarkerEnv); marker != "" {
				if err := os.WriteFile(marker, []byte("called"), 0o600); err != nil {
					return privilegebroker.SetupStatus{}, err
				}
			}
			return privilegebroker.SetupStatus{Available: true, Supported: root == "/repo"}, nil
		},
		func(_ context.Context, tools, safeguards []string) error {
			if strings.Join(tools, ",") != "git" || strings.Join(safeguards, ",") != "clock" {
				return errors.New("unexpected grant")
			}
			return nil
		},
		func(_ context.Context, id string) error {
			if _, ok := apparmorprofiles.Lookup(id); !ok {
				return errors.New("unexpected profile")
			}
			return nil
		},
		nil,
		func(_ context.Context, target observabilityinstall.Target, targetUser string, retention int) error {
			if targetUser != hostreqkit.InvokingUser() {
				return errors.New("unexpected observability target user")
			}
			return observabilityinstall.ValidateSetupRequest(target, retention)
		},
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSessionCancelsBrokerOperationWithCallerContext(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("setup privilege session is Linux-only")
	}
	t.Setenv(helperWaitForCancelEnv, "1")
	session := NewSession(context.Background(), os.Args[0], io.Discard, io.Discard)
	session.commandFactory = func(ctx context.Context, binary string) *exec.Cmd {
		cmd := exec.CommandContext(ctx, binary, "-test.run=^TestSetupPrivilegeSessionHelperProcess$")
		cmd.Env = append(os.Environ(), helperProcessEnv+"=1")
		return cmd
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := session.InstallPrivilegeBroker(ctx, "/repo")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("InstallPrivilegeBroker error = %v, want caller deadline", err)
	}
}

func TestSessionDoesNotStartWorkerForCanceledContext(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("setup privilege session is Linux-only")
	}
	session := NewSession(context.Background(), os.Args[0], io.Discard, io.Discard)
	starts := 0
	session.commandFactory = func(ctx context.Context, _ string) *exec.Cmd {
		starts++
		return exec.CommandContext(ctx, "true")
	}
	session.mu.Lock()
	base, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx := &stagedCancelContext{
		Context: base, firstCheck: make(chan struct{}),
		continueFirst: make(chan struct{}), secondCheck: make(chan struct{}),
	}
	done := make(chan error, 1)
	go func() {
		_, err := session.InstallPrivilegeBroker(ctx, "/repo")
		done <- err
	}()
	<-ctx.firstCheck
	cancel()
	close(ctx.continueFirst)
	session.mu.Unlock()
	err := <-done
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("InstallPrivilegeBroker error = %v, want canceled context", err)
	}
	<-ctx.secondCheck
	if starts != 0 {
		t.Fatalf("worker starts = %d, want none for canceled context", starts)
	}
}

func TestSessionDoesNotSendRequestWhenContextCancelsDuringWorkerStart(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("setup privilege session is Linux-only")
	}
	marker := filepath.Join(t.TempDir(), "broker-called")
	started, resume := make(chan struct{}), make(chan struct{})
	session := NewSession(context.Background(), os.Args[0], io.Discard, io.Discard)
	session.commandFactory = func(ctx context.Context, binary string) *exec.Cmd {
		close(started)
		<-resume
		cmd := exec.CommandContext(ctx, binary, "-test.run=^TestSetupPrivilegeSessionHelperProcess$")
		cmd.Env = append(os.Environ(), helperProcessEnv+"=1", helperOperationMarkerEnv+"="+marker)
		return cmd
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := session.InstallPrivilegeBroker(ctx, "/repo")
		done <- err
	}()
	<-started
	cancel()
	close(resume)
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("InstallPrivilegeBroker error = %v, want canceled context", err)
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("broker operation marker stat error = %v, want operation not to run", err)
	}
}

func TestSessionKeepsOneTypedWorkerForSelectedOperations(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("setup privilege session is Linux-only")
	}
	t.Setenv("USER", "operator")
	t.Setenv("SUDO_USER", "operator")
	var stdout, stderr strings.Builder
	session := NewSession(context.Background(), os.Args[0], &stdout, &stderr)
	starts := 0
	session.commandFactory = func(ctx context.Context, binary string) *exec.Cmd {
		starts++
		cmd := exec.CommandContext(ctx, binary, "-test.run=^TestSetupPrivilegeSessionHelperProcess$")
		cmd.Env = append(os.Environ(), helperProcessEnv+"=1")
		return cmd
	}
	if err := session.InstallPackage("apt", "git"); err != nil {
		t.Fatal(err)
	}
	source := "deb [signed-by=/usr/share/keyrings/stripe.gpg] https://packages.stripe.dev/stripe-cli-debian-local stable main"
	if err := session.InstallAptRepository("stripe", []byte("binary-key"), source); err != nil {
		t.Fatal(err)
	}
	if err := session.InstallPackage("apt", "golang-go"); err != nil {
		t.Fatal(err)
	}
	status, err := session.InstallPrivilegeBroker(context.Background(), "/repo")
	if err != nil || !status.Available {
		t.Fatalf("broker status=%+v err=%v", status, err)
	}
	if err := session.InstallOnboardingApplyGrant([]string{"git"}, []string{"clock"}); err != nil {
		t.Fatal(err)
	}
	if err := session.InstallAppArmorProfile(string(apparmorprofiles.BubblewrapUserns)); err != nil {
		t.Fatal(err)
	}
	if err := session.InstallAppArmorProfile(string(apparmorprofiles.WorkspaceSandboxUserns)); err != nil {
		t.Fatal(err)
	}
	if err := session.InstallObservability(observabilityinstall.Pstore, 0); err != nil {
		t.Fatal(err)
	}
	if err := session.InstallObservability(observabilityinstall.Kdump, 2); err != nil {
		t.Fatal(err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	if starts != 1 {
		t.Fatalf("started %d privileged workers, want one", starts)
	}
	if got := stdout.String(); !strings.Contains(got, "installed:git") || !strings.Contains(got, "installed:golang-go") {
		t.Fatalf("package output = %q", got)
	}
}
