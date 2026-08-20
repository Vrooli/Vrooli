package vroolicli

import (
	"context"
	"errors"
	"strings"
	"testing"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// exitStubRunner models exec.Cmd.Output's real contract: stdout is returned
// alongside a non-zero-exit error. The package's other stub deliberately drops
// it, which would hide the typed body `host volume` reports on a refusal.
type exitStubRunner struct {
	output []byte
	err    error
	args   []string
}

var _ Runner = (*exitStubRunner)(nil)

func (r *exitStubRunner) Run(_ context.Context, _ string, args ...string) ([]byte, error) {
	r.args = append([]string(nil), args...)
	return r.output, r.err
}

func (r *exitStubRunner) RunCombined(ctx context.Context, name string, args ...string) ([]byte, error) {
	return r.Run(ctx, name, args...)
}

func TestHostVolumeBuildsADeterministicCommand(t *testing.T) {
	runner := &exitStubRunner{output: []byte(`{"action":"repair","status":"changed","changed":true}`)}
	client := New(WithRunner(runner))

	resp, err := client.HostVolume(context.Background(), VolumeRequest{
		Action: VolumeRepair, Device: "/dev/sda1", Filesystem: "ntfs3",
		UUID: "E26A883E6A881189", Serial: "WD-1", AcknowledgeDataLoss: true, DryRun: true,
	})
	if err != nil {
		t.Fatalf("HostVolume: %v", err)
	}
	if !VolumeChanged(resp) || !VolumeSatisfied(resp) {
		t.Fatalf("response = %+v", resp)
	}

	got := strings.Join(runner.args, " ")
	want := "--no-stale-check host volume repair --device /dev/sda1 --json --filesystem ntfs3 --uuid E26A883E6A881189 --serial WD-1 --acknowledge-data-loss --dry-run"
	if got != want {
		t.Fatalf("args =\n  %s\nwant\n  %s", got, want)
	}
}

// The CLI exits non-zero for a refusal on purpose. The typed reason must
// survive that rather than being lost to the exit status.
func TestHostVolumeKeepsTheRefusalBodyDespiteNonZeroExit(t *testing.T) {
	runner := &exitStubRunner{
		output: []byte(`{"action":"repair","status":"refused","refusal_reason":"repair requires the volume to be unmounted","operator_command":"sudo umount /media/user/Elements"}`),
		err:    errors.New("exit status 1"),
	}
	client := New(WithRunner(runner))

	resp, err := client.HostVolume(context.Background(), VolumeRequest{Action: VolumeRepair, Device: "/dev/sda1", AcknowledgeDataLoss: true})
	if err != nil {
		t.Fatalf("a refusal must not surface as a transport error: %v", err)
	}
	if resp.GetStatus() != "refused" {
		t.Fatalf("status = %q", resp.GetStatus())
	}
	if !strings.Contains(resp.GetRefusalReason(), "unmounted") {
		t.Fatalf("refusal reason = %q", resp.GetRefusalReason())
	}
	if resp.GetOperatorCommand() == "" {
		t.Fatal("operator command was lost")
	}
	if VolumeSatisfied(resp) {
		t.Fatal("a refusal must not count as satisfied")
	}
}

// Genuinely getting no answer is an error, not a silent empty result.
func TestHostVolumeSurfacesATransportFailure(t *testing.T) {
	runner := &exitStubRunner{err: errors.New("executable file not found")}
	client := New(WithRunner(runner))

	if _, err := client.HostVolume(context.Background(), VolumeRequest{Action: VolumeInspect, Device: "/dev/sda1"}); err == nil {
		t.Fatal("expected an error when the CLI produced no output")
	}
}

func TestHostVolumeRequiresActionAndDevice(t *testing.T) {
	client := New(WithRunner(&exitStubRunner{}))
	if _, err := client.HostVolume(context.Background(), VolumeRequest{Action: VolumeInspect}); err == nil {
		t.Fatal("expected an error without a device")
	}
	if _, err := client.HostVolume(context.Background(), VolumeRequest{Device: "/dev/sda1"}); err == nil {
		t.Fatal("expected an error without an action")
	}
}

func TestVolumeSatisfiedVocabulary(t *testing.T) {
	satisfied := map[string]bool{
		"verified": true, "changed": true, "already_satisfied": true,
		"refused": false, "unsupported": false, "failed": false, "": false,
	}
	for status, want := range satisfied {
		resp := &cliv1.VolumeRemediationResponse{Status: status}
		if got := VolumeSatisfied(resp); got != want {
			t.Fatalf("VolumeSatisfied(%q) = %v, want %v", status, got, want)
		}
	}
	if VolumeSatisfied(nil) {
		t.Fatal("a nil response must not count as satisfied")
	}
	if VolumeChanged(nil) {
		t.Fatal("a nil response must not count as changed")
	}
}
