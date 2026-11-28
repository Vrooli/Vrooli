package observabilityinstall

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/daemonreload"
	"github.com/vrooli/vrooli/internal/hostreqkit"
)

type setupPrivilegeStub struct {
	target    Target
	retention int
	calls     int
}

func (*setupPrivilegeStub) InstallPackage(string, string) error { return nil }
func (*setupPrivilegeStub) InstallAptRepository(string, []byte, string) error {
	return nil
}
func (*setupPrivilegeStub) InstallOnboardingApplyGrant([]string, []string) error { return nil }
func (*setupPrivilegeStub) InstallAppArmorProfile(string) error                  { return nil }
func (s *setupPrivilegeStub) InstallObservability(target Target, retention int) error {
	s.target, s.retention = target, retention
	s.calls++
	return nil
}

func TestSetupSessionReceivesOneClosedObservabilityOperation(t *testing.T) {
	session := &setupPrivilegeStub{}
	err := ApplyForSetup(Kdump, "operator", Content{Collector: "must not cross worker boundary"}, 7,
		hostreqkit.EnsureOptions{SetupPrivilegeSession: session})
	if err != nil {
		t.Fatal(err)
	}
	if session.calls != 1 || session.target != Kdump || session.retention != 7 {
		t.Fatalf("setup session operation = %+v", session)
	}
}

func TestSetupRequestRejectsOpenTargetsAndInvalidRetention(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	for _, tc := range []struct {
		target    Target
		retention int
		wantErr   bool
	}{
		{target: Pstore},
		{target: Pstore, retention: 1, wantErr: true},
		{target: Kdump, retention: 1},
		{target: Kdump, retention: 0, wantErr: true},
		{target: Kdump, retention: maxInt, wantErr: true},
		{target: Target(255), wantErr: true},
	} {
		err := ValidateSetupRequest(tc.target, tc.retention)
		if (err != nil) != tc.wantErr {
			t.Fatalf("ValidateSetupRequest(%d, %d) = %v, wantErr %v", tc.target, tc.retention, err, tc.wantErr)
		}
	}
}

func TestFilesAreClosedToKnownSafeguards(t *testing.T) {
	for target, want := range map[Target]targetFiles{
		Pstore: {
			collector: PstoreCollectorPath, export: PstoreExportDir,
			service: PstoreServicePath, timer: PstoreTimerPath,
			serviceName: PstoreServiceName, timerName: PstoreTimerName,
		},
		Kdump: {
			collector: KdumpCollectorPath, export: KdumpExportDir,
			service: KdumpServicePath, timer: KdumpTimerPath,
			serviceName: KdumpServiceName, timerName: KdumpTimerName,
		},
	} {
		got, err := files(target)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("files(%d) = %#v, %v; want %#v", target, got, err, want)
		}
	}
	if _, err := files(Target(255)); err == nil {
		t.Fatal("unknown target must be rejected")
	}
}

func TestApplySuccessOrder(t *testing.T) {
	oldProbe, oldRun, oldFacts, oldReload := hostreqkit.CombinedOutputFn, hostreqkit.RunCommandFn, hostreqkit.ElevationFactsFn, reloadFn
	t.Cleanup(func() {
		hostreqkit.CombinedOutputFn, hostreqkit.RunCommandFn, hostreqkit.ElevationFactsFn, reloadFn = oldProbe, oldRun, oldFacts, oldReload
	})
	var got []string
	hostreqkit.CombinedOutputFn = func(string, ...string) ([]byte, error) { return nil, errors.New("missing") }
	hostreqkit.ElevationFactsFn = func() hostreqkit.ElevationFacts { return hostreqkit.ElevationFacts{Elevated: true} }
	hostreqkit.RunCommandFn = func(name string, args []string, _ hostreqkit.EnsureOptions) error {
		entry := name + " " + strings.Join(args, " ")
		if name == "install" && len(args) >= 3 {
			entry = name + " " + args[1] + " " + args[len(args)-1]
		}
		got = append(got, entry)
		return nil
	}
	reloadFn = func(context.Context, string, hostreqkit.EnsureOptions) (daemonreload.Result, error) {
		got = append(got, "daemon-reload")
		return daemonreload.Result{Reloaded: true}, nil
	}
	err := Apply(Pstore, "operator", Content{Collector: "collector", Service: "service", Timer: "timer"}, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"groupadd --system " + Group,
		"usermod -aG " + Group + " operator",
		"mkdir -p " + LibexecDir,
		"mkdir -p " + ExportRoot,
		"mkdir -p " + ExportRoot + "/pstore",
		"chown root:" + Group + " " + ExportRoot + " " + ExportRoot + "/pstore",
		"chmod 750 " + ExportRoot + " " + ExportRoot + "/pstore",
		"install 755 " + PstoreCollectorPath,
		"install 644 " + PstoreServicePath,
		"install 644 " + PstoreTimerPath,
		"daemon-reload",
		"systemctl enable --now " + PstoreTimerName,
		"systemctl start " + PstoreServiceName,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("installer operations = %v, want %v", got, want)
	}
}

func TestApplyAttributesFirstFailedStage(t *testing.T) {
	oldProbe, oldRun, oldFacts := hostreqkit.CombinedOutputFn, hostreqkit.RunCommandFn, hostreqkit.ElevationFactsFn
	t.Cleanup(func() {
		hostreqkit.CombinedOutputFn, hostreqkit.RunCommandFn, hostreqkit.ElevationFactsFn = oldProbe, oldRun, oldFacts
	})
	hostreqkit.CombinedOutputFn = func(string, ...string) ([]byte, error) { return nil, errors.New("missing") }
	hostreqkit.ElevationFactsFn = func() hostreqkit.ElevationFacts {
		return hostreqkit.ElevationFacts{Elevated: true}
	}
	hostreqkit.RunCommandFn = func(name string, args []string, _ hostreqkit.EnsureOptions) error {
		if name != "groupadd" || len(args) != 2 || args[1] != Group {
			t.Fatalf("first mutation = %s %v", name, args)
		}
		return errors.New("denied")
	}
	err := Apply(Pstore, "operator", Content{}, hostreqkit.EnsureOptions{})
	if err == nil || !strings.Contains(err.Error(), "create observability group failed: denied") {
		t.Fatalf("Apply error = %v, want stage attribution", err)
	}
}
