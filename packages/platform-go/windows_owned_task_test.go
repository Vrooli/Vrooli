package platform

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func windowsTaskFixture(t *testing.T) windowsOwnedPayload {
	t.Helper()
	p := windowsOwnedFixture(t)
	p.Spec.Args = []string{"__vrooli-openrgb-native-capsule", filepath.Join(t.TempDir(), "custom data", "launch", "attempt", "capsule.json")}
	i, e := NewServiceIntent("resource-initial", "attempt", OwnedServiceSpec{Path: p.Spec.Path, Dir: p.Spec.Dir, Args: p.Spec.Args, Env: p.Spec.Env, NativeAccount: p.Spec.NativeAccount, Limits: ServiceLimits{MemoryMaxBytes: p.Spec.Limits.MemoryMaxBytes, TasksMax: p.Spec.Limits.TasksMax, CPUWeight: p.Spec.Limits.CPUWeight, Enforcement: p.Spec.Limits.Enforcement}})
	if e != nil {
		t.Fatal(e)
	}
	p.Intent = OperationIntent(i)
	return p
}
func TestWindowsTaskAdmissionPersistsIntentAndIssuedBarrierBeforeOneNativeRun(t *testing.T) {
	p := windowsTaskFixture(t)
	for _, fail := range []string{"reserve", "register", "issued", "run", ""} {
		t.Run("failure="+fail, func(t *testing.T) {
			var observed []string
			reserved, issued := false, false
			effects := windowsOwnedTaskSubmissionEffects{
				Reserve: func(windowsOwnedPayload) error {
					observed = append(observed, "reserve")
					if reserved {
						return ErrOperationExists
					}
					reserved = true
					if fail == "reserve" {
						return errors.New(fail)
					}
					return nil
				},
				Register: func(windowsOwnedPayload) error {
					observed = append(observed, "register")
					if !reserved {
						t.Fatal("native registration before durable intent")
					}
					if fail == "register" {
						return errors.New(fail)
					}
					return nil
				},
				MarkIssued: func(OperationIntent) error {
					observed = append(observed, "issued")
					if fail == "issued" {
						return errors.New(fail)
					}
					issued = true
					return nil
				},
				Run: func(name string) error {
					observed = append(observed, "run")
					if !issued || name != windowsOwnedTaskName(p.Intent) {
						t.Fatal("unbound/unissued Run")
					}
					if fail == "run" {
						return errors.New("lost dispatch acknowledgement")
					}
					return nil
				},
			}
			err := submitWindowsOwnedTask(p, effects)
			order := []string{"reserve", "register", "issued", "run"}
			want := order
			if fail != "" {
				for n, item := range order {
					if item == fail {
						want = order[:n+1]
					}
				}
				if err == nil {
					t.Fatal("failure hidden")
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(observed, want) {
				t.Fatal(observed, want)
			}
			observed = nil
			if err = submitWindowsOwnedTask(p, effects); !errors.Is(err, ErrOperationExists) || !reflect.DeepEqual(observed, []string{"reserve"}) {
				t.Fatal("interrupted submission replayed", observed, err)
			}
		})
	}
}
func TestWindowsTaskRejectsGenericOrSubstitutedLaunchBeforeAnyEffect(t *testing.T) {
	p := windowsTaskFixture(t)
	for _, change := range []func(*windowsOwnedPayload){
		func(p *windowsOwnedPayload) { p.Intent.Owner = "other" },
		func(p *windowsOwnedPayload) { p.Spec.Args[0] = "generic-exec" },
		func(p *windowsOwnedPayload) { p.Spec.Args = append(p.Spec.Args, "extra") },
		func(p *windowsOwnedPayload) {
			p.Spec.Args[1] = filepath.Join(t.TempDir(), "launch", "foreign", "capsule.json")
		},
		func(p *windowsOwnedPayload) {
			p.Spec.Args[1] = filepath.Join(t.TempDir(), "%AMBIENT%", "launch", "attempt", "capsule.json")
		},
		func(p *windowsOwnedPayload) { p.Spec.Dir = filepath.Dir(p.Spec.Dir) },
	} {
		next := p
		next.Spec.Args = append([]string(nil), p.Spec.Args...)
		change(&next)
		called := false
		effect := func(windowsOwnedPayload) error { called = true; return nil }
		if err := submitWindowsOwnedTask(next, windowsOwnedTaskSubmissionEffects{Reserve: effect, Register: effect, MarkIssued: func(OperationIntent) error { called = true; return nil }, Run: func(string) error { called = true; return nil }}); err == nil || called {
			t.Fatal("unmaintained launch reached effects", err)
		}
	}
}
func TestWindowsOwnedTaskHasNoBootRetryOrImplicitLifetimeLimit(t *testing.T) {
	p := windowsTaskFixture(t)
	snapshot := WindowsTaskSnapshot{Name: windowsOwnedTaskName(p.Intent), XML: windowsOwnedTaskXML(p)}
	if e := windowsOwnedTaskDefinitionMatches(p, snapshot); e != nil {
		t.Fatal(e)
	}
	for _, replacement := range [][2]string{
		{"<AllowHardTerminate>true", "<AllowHardTerminate>false"}, {"PT0S", "PT72H"}, {"IgnoreNew", "Parallel"}, {"<Triggers/>", "<Triggers><BootTrigger><Enabled>true</Enabled></BootTrigger></Triggers>"},
		{"<StopIfGoingOnBatteries>false", "<StopIfGoingOnBatteries>true"},
		{"<RunOnlyIfNetworkAvailable>false", "<RunOnlyIfNetworkAvailable>true"},
		{"<RunOnlyIfIdle>false", "<RunOnlyIfIdle>true"},
		{"<StartWhenAvailable>false", "<StartWhenAvailable>true"},
		{"<AllowStartOnDemand>true", "<AllowStartOnDemand>false"},
		{"LeastPrivilege", "HighestAvailable"}, {"S4U", "InteractiveToken"},
		{"</Settings>", "<RestartOnFailure><Interval>PT1M</Interval><Count>1</Count></RestartOnFailure></Settings>"},
		{xmlText(p.Spec.Args[1]), xmlText(p.Spec.Args[1] + ".foreign")},
	} {
		changed := snapshot
		changed.XML = strings.Replace(changed.XML, replacement[0], replacement[1], 1)
		if e := windowsOwnedTaskDefinitionMatches(p, changed); e == nil {
			t.Fatal("unsafe task change accepted", replacement)
		}
	}
}

// Model only: real inherited Job/Scheduler topology requires the explicit
// native qualification below. This exercises the production admission effects
// and distinguishes manager submission from direct caller-child creation.
func TestWindowsOwnedTaskLifetimeModelSurvivesOuterCallerJobClose(t *testing.T) {
	p := windowsTaskFixture(t)
	callerAlive, wrapperAlive, sdkAlive, jobCustody := true, false, false, false
	effects := windowsOwnedTaskSubmissionEffects{
		Reserve: func(windowsOwnedPayload) error { return nil }, Register: func(windowsOwnedPayload) error { return nil }, MarkIssued: func(OperationIntent) error { return nil },
		Run: func(string) error { wrapperAlive = true; jobCustody = true; sdkAlive = true; return nil },
	}
	if e := submitWindowsOwnedTask(p, effects); e != nil {
		t.Fatal(e)
	}
	callerAlive = false
	if callerAlive || !wrapperAlive || !sdkAlive || !jobCustody {
		t.Fatal("manager-owned role inherited caller lifetime")
	}
	// The only role Job holder is the task action wrapper. Losing that holder,
	// or exact task termination, kills the SDK rather than orphaning it.
	wrapperAlive = false
	jobCustody = false
	if !jobCustody {
		sdkAlive = false
	}
	if wrapperAlive || sdkAlive {
		t.Fatal("SDK escaped lost wrapper custody")
	}
}

func TestWindowsOwnedPayloadBoundsSerializedEscapesBeforeAdmission(t *testing.T) {
	for _, value := range []string{strings.Repeat("x", 40000), strings.Repeat("\t", 17000), strings.Repeat("x", 2000)} {
		p := windowsTaskFixture(t)
		p.Spec.Env = []string{"APPDATA=" + value}
		i, e := NewServiceIntent("resource-initial", "attempt", OwnedServiceSpec{Path: p.Spec.Path, Dir: p.Spec.Dir, Args: p.Spec.Args, Env: p.Spec.Env, NativeAccount: p.Spec.NativeAccount, Limits: ServiceLimits{MemoryMaxBytes: p.Spec.Limits.MemoryMaxBytes, TasksMax: p.Spec.Limits.TasksMax, CPUWeight: p.Spec.Limits.CPUWeight, Enforcement: p.Spec.Limits.Enforcement}})
		if e != nil {
			t.Fatal(e)
		}
		p.Intent = OperationIntent(i)
		raw, e := json.Marshal(p)
		if e != nil {
			t.Fatal(e)
		}
		e = validateWindowsPayload(p)
		if len(raw) > windowsOwnedPayloadMaxJSONBytes {
			if e == nil {
				t.Fatal("unrecoverable encoded ledger admitted", len(raw))
			}
		} else if e != nil {
			t.Fatal("recoverable payload refused", e)
		}
	}
}

func TestWindowsOwnedRebootRetirementNeverSignalsOldRootOrNewJob(t *testing.T) {
	oldBoot := "12345678-1234-1234-1234-123456789abc"
	newBoot := "87654321-1234-1234-1234-123456789abc"
	oldRoot := windowsOwnedRootRecord{PID: 42, Start: "old-start", Incarnation: "old-incarnation", JobHandle: 123, TaskInstance: "old-instance"}
	taskRoot, e := windowsOwnedRetirementTaskRoot(oldBoot, newBoot, oldRoot, true)
	if e != nil || taskRoot != (windowsOwnedRootRecord{}) {
		t.Fatal("absent old-boot kernel objects retained stale PID/instance authority", taskRoot, e)
	}
	if _, e = windowsOwnedRetirementTaskRoot(oldBoot, newBoot, oldRoot, false); !errors.Is(e, ErrOperationIdentity) {
		t.Fatal("new boot same-name Job authorized retirement", e)
	}
	taskRoot, e = windowsOwnedRetirementTaskRoot(oldBoot, oldBoot, oldRoot, false)
	if e != nil || taskRoot != oldRoot {
		t.Fatal("sameboot exactroot authority changed", taskRoot, e)
	}
	if _, e = windowsOwnedRetirementTaskRoot(oldBoot, "", oldRoot, true); !errors.Is(e, ErrOperationIdentity) {
		t.Fatal("unknown boot accepted", e)
	}
}
