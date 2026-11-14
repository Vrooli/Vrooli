package platform

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeEnrolledStorageRootDistinguishesAbsenceDefaultAndCustom(t *testing.T) {
	home := t.TempDir()
	custom := t.TempDir()
	for _, tc := range []struct {
		name, root string
		configured bool
	}{{"absent", "", false}, {"default", "", true}, {"custom", custom, true}} {
		t.Run(tc.name, func(t *testing.T) {
			root, configured, err := enrolledNativeOperatorResourceStorageRootWith(context.Background(), home, func(context.Context) (NativeSystemServiceBrokerOptions, bool, error) {
				return NativeSystemServiceBrokerOptions{OperatorHome: home, GlobalResourceStorageRoot: tc.root}, tc.configured, nil
			})
			if err != nil || root != tc.root || configured != tc.configured {
				t.Fatalf("enrollment lost: %q %t %v", root, configured, err)
			}
		})
	}
}

func TestNativeEnrolledStorageRootRejectsForeignMalformedAndQueryFailures(t *testing.T) {
	home := t.TempDir()
	denied := errors.New("native task access denied")
	for _, tc := range []struct {
		name, owner, root string
		err               error
	}{{"foreign", t.TempDir(), "", nil}, {"relative", home, "relative", nil}, {"alias", home, filepath.Join(home, "nested") + string(filepath.Separator) + "..", nil}, {"newline", home, home + "\ninjection", nil}, {"query", home, "", denied}} {
		t.Run(tc.name, func(t *testing.T) {
			_, configured, err := enrolledNativeOperatorResourceStorageRootWith(context.Background(), home, func(context.Context) (NativeSystemServiceBrokerOptions, bool, error) {
				return NativeSystemServiceBrokerOptions{OperatorHome: tc.owner, GlobalResourceStorageRoot: tc.root}, true, tc.err
			})
			if err == nil || configured {
				t.Fatalf("uncertain enrollment accepted: %t %v", configured, err)
			}
			if tc.err != nil && !errors.Is(err, tc.err) {
				t.Fatalf("query failure lost: %v", err)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := enrolledNativeOperatorResourceStorageRootWith(ctx, home, func(context.Context) (NativeSystemServiceBrokerOptions, bool, error) {
		t.Fatal("canceled read performed")
		return NativeSystemServiceBrokerOptions{}, false, nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestNativeBootStorageRootFrozenInDescriptorAndDefinition(t *testing.T) {
	o, r := windowsSystemBrokerFixture(t)
	root := t.TempDir()
	o.SystemBroker.GlobalResourceStorageRoot = root
	p, err := buildNativeSystemBrokerPlan("windows", o, r)
	if err != nil {
		t.Fatal(err)
	}
	d, err := decodeNativeSystemServiceBrokerDescriptor(p.Descriptor)
	if err != nil {
		t.Fatal(err)
	}
	if d.Options.GlobalResourceStorageRoot != root || p.Definition.Env["VROOLI_RESOURCE_STORAGE_ROOT"] != root {
		t.Fatal("boot root not frozen")
	}
	o.SystemBroker.GlobalResourceStorageRoot = "relative"
	if _, err = buildNativeSystemBrokerPlan("windows", o, r); err == nil {
		t.Fatal("relative boot root admitted")
	}
}

func TestNativeBootStorageRootOverridesAmbientOnlyForManager(t *testing.T) {
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", t.TempDir())
	enrolled := t.TempDir()
	if err := applyNativeOperatorResourceStorageRoot(enrolled); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("VROOLI_RESOURCE_STORAGE_ROOT") != enrolled {
		t.Fatal("machine ambient root survived frozen enrollment")
	}
	if err := applyNativeOperatorResourceStorageRoot(""); err != nil {
		t.Fatal(err)
	}
	if _, ok := os.LookupEnv("VROOLI_RESOURCE_STORAGE_ROOT"); ok {
		t.Fatal("enrolled default retained ambient override")
	}
}

func TestWindowsStorageEnrollmentNeverTreatsExistingTaskMissingDescriptorAsAbsent(t *testing.T) {
	missing := errors.New("native missing code")
	snapshot := WindowsTaskSnapshot{Name: "fixed", XML: "protected", SecurityDescriptor: "protected"}
	queried, validated := 0, 0
	e := windowsNativeResourceStorageEnrollment{query: func() (WindowsTaskSnapshot, error) { queried++; return snapshot, nil }, missing: func(err error) bool { return errors.Is(err, missing) }, plan: func(s WindowsTaskSnapshot) (nativeSystemBrokerDescriptor, nativeSystemBrokerPlan, WindowsTaskSnapshot, error) {
		validated++
		return nativeSystemBrokerDescriptor{}, nativeSystemBrokerPlan{}, s, missing
	}, principal: func() (string, error) { t.Fatal("invalid descriptor entered principal observation"); return "", nil }}
	if _, configured, err := e.read(context.Background()); !errors.Is(err, missing) || configured || queried != 1 || validated != 1 {
		t.Fatalf("existing task lost enrollment: %t %v query=%d validate=%d", configured, err, queried, validated)
	}
	e.query = func() (WindowsTaskSnapshot, error) { return WindowsTaskSnapshot{}, missing }
	validated = 0
	if _, configured, err := e.read(context.Background()); err != nil || configured || validated != 0 {
		t.Fatalf("missing task not distinguished: %t %v", configured, err)
	}
	denied := errors.New("task access denied")
	e.query = func() (WindowsTaskSnapshot, error) { return WindowsTaskSnapshot{}, denied }
	if _, configured, err := e.read(context.Background()); !errors.Is(err, denied) || configured {
		t.Fatalf("denied task treated as absent: %t %v", configured, err)
	}
}

func TestWindowsStorageEnrollmentRejectsChangedTaskAndForeignOperator(t *testing.T) {
	o := NativeSystemServiceBrokerOptions{OperatorHome: t.TempDir(), OperatorPrincipal: "windows:operator", GlobalResourceStorageRoot: t.TempDir()}
	calls := 0
	e := windowsNativeResourceStorageEnrollment{query: func() (WindowsTaskSnapshot, error) {
		calls++
		xml := "before"
		if calls == 2 {
			xml = "changed"
		}
		return WindowsTaskSnapshot{XML: xml}, nil
	}, missing: func(error) bool { return false }, plan: func(s WindowsTaskSnapshot) (nativeSystemBrokerDescriptor, nativeSystemBrokerPlan, WindowsTaskSnapshot, error) {
		return nativeSystemBrokerDescriptor{Options: o}, nativeSystemBrokerPlan{DescriptorPath: "/fixed"}, s, nil
	}, principal: func() (string, error) { return o.OperatorPrincipal, nil }}
	if _, configured, err := e.read(context.Background()); err == nil || configured {
		t.Fatal("changed task accepted")
	}
	calls = 0
	e.principal = func() (string, error) { return "windows:foreign", nil }
	if _, configured, err := e.read(context.Background()); err == nil || configured || calls != 1 {
		t.Fatal("foreign operator read admitted")
	}
}

func TestRetainedNamespaceUnambiguousVersionsAndNoForeignAuthority(t *testing.T) {
	a := NativeSystemServiceBrokerOptions{OperatorPrincipal: "operator", OperatorHome: t.TempDir(), GlobalResourceStorageRoot: t.TempDir(), ExecutableSHA256: "old"}
	same := a
	same.ExecutableSHA256 = "new"
	different := a
	different.GlobalResourceStorageRoot = t.TempDir()
	foreign := a
	foreign.OperatorPrincipal = "other"
	for _, tc := range []struct {
		name    string
		records []NativeSystemServiceBrokerOptions
		want    bool
	}{{"none", nil, false}, {"retained", []NativeSystemServiceBrokerOptions{a}, true}, {"versions", []NativeSystemServiceBrokerOptions{a, same}, true}, {"ambiguous roots", []NativeSystemServiceBrokerOptions{a, different}, false}, {"foreign", []NativeSystemServiceBrokerOptions{a, foreign}, false}} {
		t.Run(tc.name, func(t *testing.T) {
			var paths []string
			for range tc.records {
				paths = append(paths, "fixed")
			}
			i := 0
			got, configured, err := retainedNativeResourceStorageOptionsWith(context.Background(), "operator", paths, func(string) (NativeSystemServiceBrokerOptions, error) { o := tc.records[i]; i++; return o, nil })
			if tc.want {
				if err != nil || !configured || got.GlobalResourceStorageRoot != a.GlobalResourceStorageRoot {
					t.Fatalf("retained root lost: %+v %t %v", got, configured, err)
				}
			} else if len(tc.records) > 0 && (err == nil || configured) {
				t.Fatal("ambiguous authority accepted")
			} else if len(tc.records) == 0 && (err != nil || configured) {
				t.Fatal("true absence confused")
			}
		})
	}
	denied := errors.New("retained descriptor access denied")
	if _, configured, err := retainedNativeResourceStorageOptionsWith(context.Background(), "operator", []string{"fixed"}, func(string) (NativeSystemServiceBrokerOptions, error) {
		return NativeSystemServiceBrokerOptions{}, denied
	}); !errors.Is(err, denied) || configured {
		t.Fatal("uncertain retained evidence became absence")
	}
}

func TestWindowsTaskRemovalKeepsRetainedNamespaceAndRejectsReappearance(t *testing.T) {
	missing := errors.New("native task missing")
	o := NativeSystemServiceBrokerOptions{OperatorHome: t.TempDir(), OperatorPrincipal: "operator", GlobalResourceStorageRoot: t.TempDir()}
	queries := 0
	e := windowsNativeResourceStorageEnrollment{query: func() (WindowsTaskSnapshot, error) { queries++; return WindowsTaskSnapshot{}, missing }, missing: func(err error) bool { return errors.Is(err, missing) }, retained: func(ctx context.Context) (NativeSystemServiceBrokerOptions, bool, error) {
		return retainedNativeResourceStorageOptionsWith(ctx, "operator", []string{"fixed"}, func(string) (NativeSystemServiceBrokerOptions, error) { return o, nil })
	}}
	actual, configured, err := e.read(context.Background())
	if err != nil || !configured || actual != o || queries != 2 {
		t.Fatalf("retained task ownership lost: %+v %t %v query%d", actual, configured, err, queries)
	}
	queries = 0
	e.query = func() (WindowsTaskSnapshot, error) {
		queries++
		if queries == 1 {
			return WindowsTaskSnapshot{}, missing
		}
		return WindowsTaskSnapshot{XML: "reappeared"}, nil
	}
	if _, configured, err = e.read(context.Background()); err == nil || configured {
		t.Fatal("task appeared during retained ownership read")
	}
}

func TestKnownNativeBrokerUnpublishedResidueCannotBecomeAuthority(t *testing.T) {
	nonce := strings.Repeat("a", 32)
	for _, name := range []string{"vrooli.exe.stage-" + nonce, "broker-" + strings.Repeat("b", 64) + ".json.stage-" + nonce} {
		if !knownNativeBrokerUnpublishedStage(name) {
			t.Fatalf("exact uncommitted publisher residue refused: %s", name)
		}
	}
	for _, name := range []string{"foreign.exe.stage-" + nonce, "broker-malformed.json.stage-" + nonce, "vrooli.exe.stage-" + strings.Repeat("A", 32), "vrooli.exe.stage-" + nonce + ".json", "vrooli.exe.stage-short", "vrooli.exe.stage-" + strings.Repeat("x", 32)} {
		if knownNativeBrokerUnpublishedStage(name) {
			t.Fatalf("unknown authority hidden as residue: %s", name)
		}
	}
}
