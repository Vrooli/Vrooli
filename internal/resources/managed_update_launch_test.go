package resources

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	platform "github.com/vrooli/platform-go"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

func updateLaunchTestRuntime(t *testing.T) ManagedUpdateRuntime {
	t.Helper()
	root := t.TempDir()
	return ManagedUpdateRuntime{
		Root: root, Home: filepath.Join(root, "home"), StateDir: filepath.Join(root, "state"), Launcher: os.Args[0], LauncherSHA256: updateTestDigest("launcher"), Port: 32042,
		Limits: platform.ServiceLimits{MemoryMaxBytes: 512 << 20, TasksMax: 128, CPUWeight: 100},
	}
}

func TestManagedUpdateRoleRenderingIsolatesMutableInputsAndCallerIdentity(t *testing.T) {
	u := updateTestInitial(t)
	m := u.Target.Manifest
	m.ManagedService.Environment = map[string]string{"CHAIN": "${SECOND}", "SECOND": "$RESOURCE_CONFIG_DIR/settings"}
	m.ManagedService.Config = &resourcedeployment.ServiceConfig{Path: "settings", Content: "listen=$RESOURCE_PORT_HTTP\nchain=${CHAIN}"}
	c, err := newManagedUpdateContract(m, u.Target.Artifact)
	if err != nil {
		t.Fatal(err)
	}
	r := updateLaunchTestRuntime(t)
	t.Setenv("AGENT_MANAGER_RUN_ID", "must-not-inherit")
	t.Setenv("CHAIN", "caller-value")
	a, err := renderManagedUpdateInvocation(c, r, u.ID, "candidate", "private-token")
	if err != nil {
		t.Fatal(err)
	}
	b, err := renderManagedUpdateInvocation(c, r, u.ID, "primary", "other-token")
	if err != nil {
		t.Fatal(err)
	}
	av, bv := managedServiceEnvValues(a.Env), managedServiceEnvValues(b.Env)
	for _, key := range []string{"HOME", "RESOURCE_CONFIG_DIR", "RESOURCE_CACHE_DIR", "RESOURCE_LOGS_DIR", "RESOURCE_STATE_DIR"} {
		if av[key] == bv[key] || !strings.Contains(av[key], string(filepath.Separator)+"candidate"+string(filepath.Separator)) {
			t.Fatalf("shared mutable role path %s", key)
		}
	}
	if av["HOME"] == r.Home {
		t.Fatal("resource retained the operator home instead of a private role home")
	}
	if av["AGENT_MANAGER_RUN_ID"] != "" || av["RESOURCE_DATA_DIR"] != "" || av["CHAIN"] != filepath.Join(av["RESOURCE_CONFIG_DIR"], "settings") || !strings.Contains(a.ConfigContent, "listen=32042") {
		t.Fatalf("role inputs were inherited or unresolved")
	}
	if !strings.Contains(strings.Join(a.Args, " "), av["RESOURCE_STATE_DIR"]) || !strings.Contains(strings.Join(a.Args, " "), "127.0.0.1:32042") {
		t.Fatal("credential path or port did not follow role")
	}
}

func TestManagedUpdateRoleRenderingRejectsCyclesAndUnknownInputs(t *testing.T) {
	for name, env := range map[string]map[string]string{"cycle": {"FIRST": "$SECOND", "SECOND": "${FIRST}"}, "self": {"FIRST": "$FIRST"}, "unknown": {"FIRST": "$UNDECLARED"}, "owner": {"HOME": "/elsewhere"}, "nul": {"FIRST": "bad\x00value"}} {
		t.Run(name, func(t *testing.T) {
			u := updateTestInitial(t)
			u.Target.Manifest.ManagedService.Environment = env
			u.Target.Digest = u.Target.digest()
			if _, err := renderManagedUpdateInvocation(u.Target, updateLaunchTestRuntime(t), u.ID, "candidate", "token"); err == nil {
				t.Fatal("unsafe input was accepted")
			}
		})
	}
}

func updateExecTestFixture(t *testing.T) (ManagedServiceUpdate, managedUpdateExecOwner, *int) {
	t.Helper()
	u := updateTestInitial(t)
	u.Target.Manifest.ManagedService.CredentialFiles = nil
	u.Target.Digest = u.Target.digest()
	if err := os.WriteFile(u.Target.Artifact.Path, []byte("new-image"), 0o755); err != nil {
		t.Fatal(err)
	}
	r := updateLaunchTestRuntime(t)
	file, err := os.Open(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	r.LauncherSHA256, err = managedUpdateImageDigest(context.Background(), file)
	file.Close()
	if err != nil {
		t.Fatal(err)
	}
	i := ManagedUpdateLaunchIntent{InstanceID: "candidate", OwnershipTokenHash: managedServiceTokenHash("private-token"), ContractDigest: u.Target.Digest, Runtime: r}
	i.NativeIntent, err = platform.NewServiceIntent("resource-update", i.InstanceID, managedUpdateWrapperSpec(u, i))
	if err != nil {
		t.Fatal(err)
	}
	u.Candidate.Intent, u.Phase = &i, UpdateCandidateStarting
	root, err := r.roleRoot(u.ID, i.InstanceID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ownership-token"), []byte("private-token"), 0o600); err != nil {
		t.Fatal(err)
	}
	unit, _ := i.NativeIntent.UnitName()
	ref := platform.ServiceRef{Intent: platform.OperationIntent(i.NativeIntent), Unit: unit, RootPID: os.Getpid(), RootStartTime: "123", NativeInvocationID: strings.Repeat("a", 32), Scope: platform.ScopeRef{Kind: platform.ScopeKindCgroup, Path: "/user.slice/" + platform.ServicesSlice + "/" + unit}}
	calls := new(int)
	owner := managedUpdateExecOwner{
		lookup:    func(context.Context, platform.ServiceIntent) (platform.ServiceRef, error) { return ref, nil },
		selfImage: func(int) (*os.File, error) { return os.Open(os.Args[0]) }, incarnation: func(int) (string, error) { return "boot:123", nil },
		claim: func(context.Context, managedUpdateExecFence, platform.ServiceRef) (managedUpdateExecFence, error) {
			return managedUpdateExecFence{}, nil
		},
		chdir:   func(string) error { return nil },
		replace: func(file *os.File, argv, env []string) error { *calls++; return nil },
	}
	return u, owner, calls
}

func TestManagedUpdateProductionExecConsumesFenceBeforeReplacingVerifiedImage(t *testing.T) {
	u, owner, calls := updateExecTestFixture(t)
	claimed := false
	owner.claim = func(_ context.Context, f managedUpdateExecFence, ref platform.ServiceRef) (managedUpdateExecFence, error) {
		if f.Intent != *u.Candidate.Intent || ref.RootPID != os.Getpid() {
			t.Fatal("claim identity changed")
		}
		claimed = true
		return f, nil
	}
	owner.replace = func(file *os.File, argv, env []string) error {
		if !claimed || argv[0] != u.Target.Artifact.Path || managedServiceEnvValues(env)[managedServiceOwnershipTokenEnv] != "private-token" {
			t.Fatal("exec preceded claim or lost its contract")
		}
		if _, err := file.Seek(0, 0); err != nil {
			t.Fatal(err)
		}
		if digest, err := managedUpdateImageDigest(context.Background(), file); err != nil || digest != u.Target.Artifact.SHA256 {
			t.Fatal("resource bytes changed")
		}
		*calls++
		return nil
	}
	if err := executeManagedUpdateRole(context.Background(), u, "candidate", owner); err != nil {
		t.Fatal(err)
	}
	if *calls != 1 {
		t.Fatal("resource was not executed exactly once")
	}
}

func TestManagedUpdatePrivateMaterializationRejectsRoleDirectoryAliases(t *testing.T) {
	u := updateTestInitial(t)
	u.Target.Manifest.ManagedService.CredentialFiles = nil
	u.Target.Digest = u.Target.digest()
	r := updateLaunchTestRuntime(t)
	invocation, err := renderManagedUpdateInvocation(u.Target, r, u.ID, "candidate", "token")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(invocation.RuntimeRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	other := t.TempDir()
	if err := os.Symlink(other, filepath.Join(invocation.RuntimeRoot, "cache")); err != nil {
		t.Skipf("directory aliases unavailable: %v", err)
	}
	if err := materializeManagedUpdateInvocation(u, u.Target, invocation); err == nil {
		t.Fatal("candidate accepted another runtime's directory alias")
	}
	entries, err := os.ReadDir(other)
	if err != nil || len(entries) != 0 {
		t.Fatal("candidate wrote through rejected alias")
	}
}

func TestManagedUpdatePreparationReusesPrivateTokenAndNativeIntent(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	u := updateTestInitial(t)
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	c := NewController(projectRoot(t), home)
	first, err := c.prepareManagedUpdateLaunch(context.Background(), u, u.Target, "candidate", 32042)
	if runtime.GOOS != "linux" {
		if !errors.Is(err, platform.ErrUnsupported) {
			t.Fatalf("unsupported preparation did not fail closed: %v", err)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	second, err := c.prepareManagedUpdateLaunch(context.Background(), u, u.Target, "candidate", 32042)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("lost preparation reply changed native identity or runtime")
	}
	if _, err := c.prepareManagedUpdateLaunch(context.Background(), u, u.Target, "candidate", 32043); !errors.Is(err, ErrManagedUpdateConflict) {
		t.Fatal("changed role port reused prepared permission")
	}
	fences, err := newManagedUpdateExecStore(u.Resource, u.ID, "candidate")
	if err != nil {
		t.Fatal(err)
	}
	f, err := fences.read()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fences.Revoke(context.Background(), f); err != nil {
		t.Fatal(err)
	}
	if _, err := c.prepareManagedUpdateLaunch(context.Background(), u, u.Target, "candidate", 32042); !errors.Is(err, ErrManagedUpdateExecRevoked) {
		t.Fatal("preparation reset a revoked launch")
	}
}

func TestManagedUpdateProductionExecRejectsUnadmittedOrChangedRoles(t *testing.T) {
	for _, name := range []string{"phase", "instance", "port", "launcher", "native-root", "token", "image", "revoked", "unknown-claim"} {
		t.Run(name, func(t *testing.T) {
			u, owner, calls := updateExecTestFixture(t)
			instance := "candidate"
			switch name {
			case "phase":
				u.Phase = UpdatePrepared
			case "instance":
				instance = "unadmitted"
			case "port":
				u.Candidate.Intent.Runtime.Port = u.Target.Manifest.Ports[0].Host
			case "launcher":
				u.Candidate.Intent.Runtime.LauncherSHA256 = updateTestDigest("different")
			case "native-root":
				owner.incarnation = func(int) (string, error) { return "boot:999", nil }
			case "token":
				u.Candidate.Intent.OwnershipTokenHash = updateTestDigest("different")
			case "image":
				if err := os.WriteFile(u.Target.Artifact.Path, []byte("changed"), 0o755); err != nil {
					t.Fatal(err)
				}
			case "revoked":
				owner.claim = func(context.Context, managedUpdateExecFence, platform.ServiceRef) (managedUpdateExecFence, error) {
					return managedUpdateExecFence{}, ErrManagedUpdateExecRevoked
				}
			case "unknown-claim":
				owner.claim = func(context.Context, managedUpdateExecFence, platform.ServiceRef) (managedUpdateExecFence, error) {
					return managedUpdateExecFence{}, ErrManagedUpdateCommitUnknown
				}
			}
			if err := executeManagedUpdateRole(context.Background(), u, instance, owner); err == nil {
				t.Fatal("unsafe exec accepted")
			}
			if *calls != 0 {
				t.Fatal("resource exec occurred after rejection")
			}
		})
	}
}

func TestManagedUpdateWrapperInvocationDoesNotInheritAgentAuthority(t *testing.T) {
	u, _, _ := updateExecTestFixture(t)
	s := managedUpdateWrapperSpec(u, *u.Candidate.Intent)
	if !reflect.DeepEqual(s.Args, []string{managedUpdateExecCommand, u.Resource, u.RequestID, "candidate"}) || len(s.Env) != 3 {
		t.Fatal("wrapper invocation is not explicit")
	}
	if err := u.Candidate.Intent.NativeIntent.ValidateSpec(s); err != nil {
		t.Fatal(err)
	}
	s.Env = append(s.Env, "AGENT_MANAGER_RUN_ID=caller")
	if err := u.Candidate.Intent.NativeIntent.ValidateSpec(s); err == nil || !errors.Is(err, platform.ErrOperationIdentity) {
		t.Fatal("changed wrapper authority was accepted")
	}
}
