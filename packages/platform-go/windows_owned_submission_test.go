package platform

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestWindowsCustodyAcknowledgementBindsExactBirthAndJobHandle(t *testing.T) {
	p := windowsOwnedFixture(t)
	root := windowsOwnedRootRecord{PID: 42, Start: "116444736000000010", Incarnation: "windows:42:1000", ImageID: "windows-file:fixture", ImageHandle: 123, JobHandle: 456, TaskInstance: "exact-instance"}
	c := windowsOwnedCustodyRecord{Version: 1, Intent: p.Intent, BootID: p.BootID, Root: root}
	if !c.matches(p, root) {
		t.Fatal("exact custody refused")
	}
	for _, mutate := range []func(*windowsOwnedCustodyRecord){func(c *windowsOwnedCustodyRecord) { c.Root.JobHandle++ }, func(c *windowsOwnedCustodyRecord) { c.Root.TaskInstance = "foreign-instance" }, func(c *windowsOwnedCustodyRecord) { c.Root.ImageHandle++ }, func(c *windowsOwnedCustodyRecord) { c.Root.PID++ }, func(c *windowsOwnedCustodyRecord) { c.Intent.InvocationID = strings.Repeat("b", 32) }, func(c *windowsOwnedCustodyRecord) { c.BootID = "87654321-1234-1234-1234-123456789abc" }} {
		next := c
		mutate(&next)
		if next.matches(p, root) {
			t.Fatal("foreign custody accepted")
		}
	}
	root.JobHandle = 0
	c.Root = root
	if c.matches(p, root) {
		t.Fatal("no retained Job handle accepted")
	}
}
func TestWindowsOwnedBootstrapEnvironmentCannotBeSuppliedByPayload(t *testing.T) {
	for _, key := range []string{"VROOLI_NATIVE_OWNED_PAYLOAD", "vrooli_native_owned_job_handle"} {
		p := windowsOwnedFixture(t)
		p.Spec.Env = append(p.Spec.Env, key+"=forged")
		i, e := windowsOwnedFixtureIntent(p.Spec)
		if e != nil {
			t.Fatal(e)
		}
		p.Intent = i
		if validateWindowsPayload(p) == nil {
			t.Fatal("reserved bootstrap environment accepted", key)
		}
	}
}

func TestWindowsOwnedInterruptedSubmissionNeverReplaysOrInventsCustody(t *testing.T) {
	p := windowsOwnedFixture(t)
	root := windowsOwnedRootRecord{PID: 42, Start: "116444736000000010", Incarnation: "windows:42:1000", ImageID: "windows-file:fixture", ImageHandle: 123, JobHandle: 456, TaskInstance: "exact-instance"}
	claim := windowsOwnedCustodyRecord{Version: 1, Intent: p.Intent, BootID: p.BootID, Root: root}
	cases := []struct {
		name  string
		root  windowsOwnedRootRecord
		claim windowsOwnedCustodyRecord
		want  windowsOwnedSubmissionState
	}{
		{name: "intent flushed before birth", want: windowsSubmissionReserved},
		{name: "suspended birth before root write", want: windowsSubmissionReserved},
		{name: "root flushed before resume", root: root, want: windowsSubmissionPrepared},
		{name: "resumed before custody ack", root: root, want: windowsSubmissionPrepared},
		{name: "lost ack transport after custody flush", root: root, claim: claim, want: windowsSubmissionClaimed},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			state, e := windowsOwnedSubmissionMetadata(p, c.root, c.claim)
			if e != nil || state != c.want {
				t.Fatal(state, e)
			}
			if !errors.Is(refuseWindowsOwnedReplay(p, p), ErrOperationExists) {
				t.Fatal("interrupted persisted launch replayed")
			}
			other := p
			other.Spec.Env = []string{"APPDATA=foreign"}
			if !errors.Is(refuseWindowsOwnedReplay(p, other), ErrOperationIdentity) {
				t.Fatal("changed persisted launch adopted")
			}
		})
	}
	if _, e := windowsOwnedSubmissionMetadata(p, windowsOwnedRootRecord{}, claim); e == nil {
		t.Fatal("custody without committed birth accepted")
	}
	partial := root
	partial.JobHandle = 0
	if _, e := windowsOwnedSubmissionMetadata(p, partial, windowsOwnedCustodyRecord{}); e == nil {
		t.Fatal("incomplete birth accepted")
	}
	forged := claim
	forged.Root.Start = "116444736000000020"
	if _, e := windowsOwnedSubmissionMetadata(p, root, forged); e == nil {
		t.Fatal("foreign birth custody accepted")
	}
}
func TestWindowsOwnedFrozenInputsRejectInvalidUTF8(t *testing.T) {
	p := windowsOwnedFixture(t)
	p.Spec.Env = []string{"APPDATA=\xff"}
	i, e := windowsOwnedFixtureIntent(p.Spec)
	if e != nil {
		t.Fatal(e)
	}
	p.Intent = i
	if validateWindowsPayload(p) == nil {
		t.Fatal("lossy UTF16 conversion could change frozen environment")
	}
}

func TestWindowsCustodyRequiresSuccessfulFlushBeforeReady(t *testing.T) {
	if windowsCustodyRegistryWriteAccess != 0x0001|0x0002 {
		t.Fatal("custody write/flush handle must request QUERY_VALUE|SET_VALUE")
	}
	for _, fail := range []string{"pending", "flush", "ready", ""} {
		t.Run(fail, func(t *testing.T) {
			events := []string{}
			pending, ready := "", ""
			err := commitWindowsOwnedCustody("exact custody", windowsCustodyCommitEffects{
				WritePending: func(raw string) error {
					events = append(events, "pending")
					if fail == "pending" {
						return errors.New("write failed")
					}
					pending = raw
					return nil
				},
				Flush: func() error {
					events = append(events, "flush")
					if windowsCustodyReadyMatches(pending, ready) {
						t.Fatal("visible pending value became completed ACK before flush")
					}
					if fail == "flush" {
						return errors.New("flush failed")
					}
					return nil
				},
				WriteReady: func(marker string) error {
					events = append(events, "ready")
					if fail == "ready" {
						return errors.New("ready write failed")
					}
					ready = marker
					return nil
				},
			})
			if fail == "" {
				if err != nil || !windowsCustodyReadyMatches(pending, ready) {
					t.Fatal(events, err)
				}
			} else if err == nil || windowsCustodyReadyMatches(pending, ready) {
				t.Fatal("failed barrier became ready", events, err)
			}
			if fail == "pending" && len(events) != 1 {
				t.Fatal(events)
			}
			if fail == "flush" && len(events) != 2 {
				t.Fatal(events)
			}
			if windowsCustodyReadyMatches("different custody", ready) {
				t.Fatal("ready marker matched changed pending bytes")
			}
		})
	}
}
func TestWindowsCustodyPausedFlushCannotBeAcknowledgedByVisibility(t *testing.T) {
	pendingVisible := make(chan string, 1)
	releaseFlush := make(chan struct{})
	readyPublished := make(chan string, 1)
	done := make(chan error, 1)
	go func() {
		done <- commitWindowsOwnedCustody("pending custody", windowsCustodyCommitEffects{
			WritePending: func(raw string) error { pendingVisible <- raw; return nil },
			Flush:        func() error { <-releaseFlush; return nil },
			WriteReady:   func(marker string) error { readyPublished <- marker; return nil },
		})
	}()
	pending := <-pendingVisible
	select {
	case <-readyPublished:
		t.Fatal("ready published while durable flush paused")
	default:
	}
	if windowsCustodyReadyMatches(pending, "") {
		t.Fatal("Start could accept visible pending ACK")
	}
	close(releaseFlush)
	if e := <-done; e != nil {
		t.Fatal(e)
	}
	if !windowsCustodyReadyMatches(pending, <-readyPublished) {
		t.Fatal("completed durable ACK not discoverable")
	}
}
func TestWindowsPrePayloadInterruptionIsRecoverableOnlyWithNativeAbsence(t *testing.T) {
	for _, stage := range []string{"key created before Payload", "Payload write failed without value"} {
		t.Run(stage, func(t *testing.T) {
			if !windowsOwnedUnpublishedSlotRemovable(0, 0, true) {
				t.Fatal("empty protected pre-publication slot stranded")
			}
		})
	}
	for _, c := range []struct {
		values, subkeys uint32
		absent          bool
	}{{1, 0, true}, {0, 1, true}, {0, 0, false}, {1, 0, false}} {
		if windowsOwnedUnpublishedSlotRemovable(c.values, c.subkeys, c.absent) {
			t.Fatal("value-bearing/foreign/native-retained slot authorized cleanup", c)
		}
	}
}
func TestWindowsChildFrozenInputRejectsLossyUTF16Handoff(t *testing.T) {
	for _, c := range []struct {
		name, path, dir string
		argv, env       []string
	}{
		{name: "argv", argv: []string{"OpenRGB.exe", "bad\xff"}, env: []string{}},
		{name: "environment value", argv: []string{"OpenRGB.exe"}, env: []string{"APPDATA=bad\xff"}},
		{name: "environment key", argv: []string{"OpenRGB.exe"}, env: []string{"bad\xff=value"}},
		{name: "image path", path: "bad\xff", argv: []string{"OpenRGB.exe"}},
		{name: "working directory", dir: "bad\xff", argv: []string{"OpenRGB.exe"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			if validateWindowsFrozenText(c.path, c.dir, c.argv, c.env) == nil {
				t.Fatal("lossy child handoff accepted")
			}
		})
	}
	args := []string{"OpenRGB.exe", "Привет", "灯光", "😀"}
	env := []string{"APPDATA=配置"}
	if e := validateWindowsFrozenText(`C:\灯光\OpenRGB.exe`, `C:\用户`, args, env); e != nil {
		t.Fatal(e)
	}
	for _, value := range append(args, env...) {
		if string(utf16.Decode(utf16.Encode([]rune(value)))) != value {
			t.Fatal("valid Unicode changed", value)
		}
	}
}

func TestWindowsOwnedObserverAdmissionAlwaysProducesCanonicalReceipt(t *testing.T) {
	for _, observer := range []string{"", "sid:S-1-5-21-100-200-300-1001", "sid:S-1-5-21-100-200-300-1002"} {
		t.Run(observer, func(t *testing.T) {
			p := windowsOwnedFixture(t)
			p.Spec.ObserverPrincipal = observer
			fresh, e := windowsOwnedFixtureIntent(p.Spec)
			if e != nil {
				t.Fatal(e)
			}
			p.Intent = fresh
			e = validateWindowsPayload(p)
			if observer != "" && observer != p.Spec.NativeAccount {
				if !errors.Is(e, ErrOperationIdentity) {
					t.Fatal("foreign observer admitted before native effects", e)
				}
				return
			}
			if e != nil {
				t.Fatal(e)
			}
			if e = windowsRef(p).ValidateIdentity(); e != nil {
				t.Fatal("accepted invocation cannot be inspected/stopped/released", e)
			}
		})
	}
}
