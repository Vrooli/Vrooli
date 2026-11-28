package platform

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func windowsSystemBrokerFixture(t *testing.T) (ServiceInstallOptions, nativeSystemBrokerRoots) {
	t.Helper()
	root := t.TempDir()
	image := []byte("approved native CLI image")
	hash := fmt.Sprintf("%x", sha256.Sum256(image))
	return ServiceInstallOptions{Executable: filepath.Join(root, "source", "vrooli.exe"), SystemBroker: &NativeSystemServiceBrokerOptions{ExecutableSHA256: hash, OperatorPrincipal: "windows:S-1-5-21-100-200-300-1001", OperatorUsername: `HOST\operator`, OperatorHome: filepath.Join(root, "operator")}}, nativeSystemBrokerRoots{Install: filepath.Join(root, "Program Files", "Vrooli", "RuntimeSupervisor"), State: filepath.Join(root, "ProgramData", "Vrooli", "RuntimeSupervisor"), Unit: "VrooliRuntimeSupervisor", SystemRoot: filepath.Join(root, "Windows")}
}
func TestWindowsSystemBrokerPlanUsesSealedDescriptorOnly(t *testing.T) {
	o, r := windowsSystemBrokerFixture(t)
	p, e := buildNativeSystemBrokerPlan("windows", o, r)
	if e != nil {
		t.Fatal(e)
	}
	if len(p.Definition.Args) != 2 || p.Definition.Args[0] != NativeSystemSupervisorVerb || p.Definition.Args[1] != p.DescriptorPath {
		t.Fatal("native operator host uses public CLI arguments")
	}
	if p.Definition.Env["HOME"] != o.SystemBroker.OperatorHome || p.Definition.WorkingDirectory != o.SystemBroker.OperatorHome || p.Definition.Env["VROOLI_SOURCE_ROOT"] != "" || p.Definition.Username != o.SystemBroker.OperatorUsername || p.StateRoot != o.SystemBroker.OperatorHome {
		t.Fatal("native boot task lost its actual operator execution context")
	}
	if strings.Contains(p.Definition.Env["PATH"], o.SystemBroker.OperatorHome) {
		t.Fatal("operator PATH entered sealed native boot definition")
	}
	d, e := decodeNativeSystemServiceBrokerDescriptor(p.Descriptor)
	if e != nil {
		t.Fatal(e)
	}
	if d.Version != 2 {
		t.Fatal("actual-operator boot requires versioned replacement descriptor")
	}
	if _, e = validateNativeSystemBrokerDescriptor(d, "windows", p.DescriptorPath, r); e != nil {
		t.Fatal(e)
	}
}
func TestWindowsSystemBrokerRefusesImplicitAuthorityAndUserConfig(t *testing.T) {
	for _, name := range []string{"no-broker", "user-mode", "source-root", "log-override", "system-operator", "builtin-operator", "virtual-operator", "noncanonical-sid"} {
		t.Run(name, func(t *testing.T) {
			o, r := windowsSystemBrokerFixture(t)
			switch name {
			case "no-broker":
				o.SystemBroker = nil
			case "user-mode":
				o.User = true
			case "source-root":
				o.SourceRoot = t.TempDir()
			case "log-override":
				o.LogPath = filepath.Join(o.SystemBroker.OperatorHome, "log")
			case "system-operator":
				o.SystemBroker.OperatorPrincipal = "windows:S-1-5-18"
			case "builtin-operator":
				o.SystemBroker.OperatorPrincipal = "windows:S-1-5-32-544"
			case "virtual-operator":
				o.SystemBroker.OperatorPrincipal = "windows:S-1-5-80-1-2-3-4-5"
			case "noncanonical-sid":
				o.SystemBroker.OperatorPrincipal = "windows:S-1-5-21-0100-200-300-1001"
			}
			effects := 0
			_, e := installNativeSystemBrokerWithOps("windows", o, r, nativeSystemBrokerInstallOps{Stage: func(nativeSystemBrokerPlan) error { effects++; return nil }, Register: func(nativeSystemBrokerPlan) (ServiceInstallResult, error) {
				effects++
				return ServiceInstallResult{}, nil
			}})
			if e == nil || effects != 0 {
				t.Fatalf("unsafe root request reached native effects: %v, effects %d", e, effects)
			}
		})
	}
}
func TestWindowsSystemBrokerRegistersOnlyVerifiedStagedImage(t *testing.T) {
	o, r := windowsSystemBrokerFixture(t)
	for _, good := range []bool{false, true} {
		registered := 0
		var stored bytes.Buffer
		_, e := installNativeSystemBrokerWithOps("windows", o, r, nativeSystemBrokerInstallOps{Stage: func(p nativeSystemBrokerPlan) error {
			image := "tampered CLI"
			if good {
				image = "approved native CLI image"
			}
			return copyNativeSystemBrokerImage(strings.NewReader(image), &stored, p.SHA256)
		}, Register: func(nativeSystemBrokerPlan) (ServiceInstallResult, error) {
			registered++
			return ServiceInstallResult{Active: true}, nil
		}})
		if good && (e != nil || registered != 1) {
			t.Fatalf("approved image not registered: %v", e)
		}
		if !good && (e == nil || registered != 0) {
			t.Fatal("unverified bytes registered as the operator boot task")
		}
	}
}
func TestWindowsSystemBrokerDescriptorRefusesAmbiguityAndRelocation(t *testing.T) {
	for _, raw := range []string{`{"version":1,"version":1}`, `{"version":1,"Options":{"OperatorHome":"a","operatorhome":"b"}}`, `{"version":1} {}`, `{"version":1,"unknown":true}`} {
		if _, e := decodeNativeSystemServiceBrokerDescriptor([]byte(raw)); e == nil {
			t.Fatalf("ambiguous descriptor accepted: %s", raw)
		}
	}
	o, r := windowsSystemBrokerFixture(t)
	p, e := buildNativeSystemBrokerPlan("windows", o, r)
	if e != nil {
		t.Fatal(e)
	}
	d, e := decodeNativeSystemServiceBrokerDescriptor(p.Descriptor)
	if e != nil {
		t.Fatal(e)
	}
	d.ExecutablePath = o.Executable
	if _, e = validateNativeSystemBrokerDescriptor(d, "windows", p.DescriptorPath, r); !errors.Is(e, ErrOperationIdentity) {
		t.Fatal("descriptor selected operator-writable image", e)
	}
}
func TestWindowsSystemBrokerMetadataChangesHaveImmutablePaths(t *testing.T) {
	o, r := windowsSystemBrokerFixture(t)
	first, e := buildNativeSystemBrokerPlan("windows", o, r)
	if e != nil {
		t.Fatal(e)
	}
	o.SystemBroker.OperatorHome = filepath.Join(t.TempDir(), "moved-operator")
	second, e := buildNativeSystemBrokerPlan("windows", o, r)
	if e != nil {
		t.Fatal(e)
	}
	if first.ImagePath != second.ImagePath || first.DescriptorPath == second.DescriptorPath {
		t.Fatal("operator migration replaces immutable image/descriptor")
	}
	var d nativeSystemBrokerDescriptor
	if e = json.Unmarshal(first.Descriptor, &d); e != nil {
		t.Fatal(e)
	}
	d.Options = *o.SystemBroker
	if _, e = validateNativeSystemBrokerDescriptor(d, "windows", first.DescriptorPath, r); !errors.Is(e, ErrOperationIdentity) {
		t.Fatal("different operator metadata reused sealed descriptor", e)
	}
}

func TestWindowsOperatorBootKeepsExistingProfileAndOrdinaryMutableState(t *testing.T) {
	o, r := windowsSystemBrokerFixture(t)
	o.HomeDir = o.SystemBroker.OperatorHome
	p, e := buildNativeSystemBrokerPlan("windows", o, r)
	if e != nil {
		t.Fatal(e)
	}
	if p.StateRoot != o.HomeDir || p.Definition.WorkingDirectory != o.HomeDir || p.Definition.Logs.Stdout != "" || p.Definition.Logs.Stderr != "" {
		t.Fatal("initial setup planned privileged mutable runtime state")
	}
	artifact, e := RenderWindowsTaskXML(p.Definition)
	if e != nil {
		t.Fatal(e)
	}
	task, e := parseWindowsTaskDefinition(artifact.Files[0].Content)
	if e != nil {
		t.Fatal(e)
	}
	if task.Principals[0].UserID != o.SystemBroker.OperatorUsername || task.Settings.Limit != "PT0S" || len(task.Boot) != 1 {
		t.Fatal("operator boot contract lost")
	}
}
