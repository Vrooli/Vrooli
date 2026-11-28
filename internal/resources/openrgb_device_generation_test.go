package resources

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/lightingruntime"
)

func TestOpenRGBDeviceGenerationRefreshIsOneQuiescedExactReplacement(t *testing.T) {
	for _, mode := range []string{"changed", "unsupported-unchanged", "quiesce-failed", "retire-failed", "second-change"} {
		t.Run(mode, func(t *testing.T) {
			var calls []string
			cause := ErrOpenRGBDeviceGenerationChanged
			if mode == "unsupported-unchanged" {
				cause = errors.New("unsupported selected device")
			}
			expected := lightingruntime.Backend{Endpoint: netip.MustParseAddrPort("127.0.0.1:6742"), Incarnation: "boot:new-root"}
			backend, err := refreshOpenRGBDeviceGeneration(t.Context(), cause, func(context.Context) error {
				calls = append(calls, "quiesce")
				if mode == "quiesce-failed" {
					return errors.New("writer drain failed")
				}
				return nil
			}, func(context.Context) error {
				calls = append(calls, "retire")
				if mode == "retire-failed" {
					return errors.New("native quiescence unknown")
				}
				return nil
			}, func(context.Context) (lightingruntime.Backend, error) {
				calls = append(calls, "start")
				if mode == "second-change" {
					return lightingruntime.Backend{}, ErrOpenRGBDeviceGenerationChanged
				}
				return expected, nil
			})
			if mode == "changed" {
				if err != nil || backend != expected || strings.Join(calls, ",") != "quiesce,retire,start" {
					t.Fatalf("refresh %+v %v %v", backend, err, calls)
				}
			} else {
				if err == nil || backend != (lightingruntime.Backend{}) {
					t.Fatalf("unverified refresh %+v %v", backend, err)
				}
				if mode == "unsupported-unchanged" && len(calls) != 0 {
					t.Fatal("unsupported unchanged device caused restart")
				}
				if mode == "quiesce-failed" && len(calls) != 1 {
					t.Fatal("failed drain advanced replacement")
				}
				if mode == "retire-failed" && len(calls) != 2 {
					t.Fatal("failed retirement advanced replacement")
				}
				if mode == "second-change" && len(calls) != 3 {
					t.Fatal("Ensure entered an automatic restart loop")
				}
			}
		})
	}
}
func TestOpenRGBNativeWitnessRequiresCurrentKernelDeviceGeneration(t *testing.T) {
	x := newInitialExecFixture(t)
	state := ManagedServiceState{InitialLaunch: &ManagedInitialServingLaunch{AttemptID: x.a.ID, NativeRef: x.ref}, InstanceID: x.a.ID, PID: x.ref.RootPID, Incarnation: "boot:" + x.ref.RootStartTime, ArtifactPath: x.a.Contract.Artifact.Path, ArtifactVersion: x.a.Contract.Artifact.Version, ArtifactSHA256: x.a.Contract.Artifact.SHA256}
	check := OpenRGBDevicePolicyCheck{PID: state.PID, Incarnation: state.Incarnation, Artifact: x.a.Contract.Artifact, PolicyID: x.a.Policy.ID, DetectorSHA256: x.a.Policy.DetectorCatalogSHA256, DeviceGenerationSHA256: updateTestDigest("fresh kernel nodes")}
	read := func() (ManagedServiceState, error) { return state, nil }
	for _, generation := range []string{check.DeviceGenerationSHA256, updateTestDigest("replaced kernel nodes"), ""} {
		err := verifyOpenRGBNativePolicy(t.Context(), check, read, func(context.Context, ManagedServiceState) (openRGBExecWitness, error) {
			return openRGBExecWitness{Policy: x.a.Policy, DeviceGenerationSHA256: generation}, nil
		})
		if generation == check.DeviceGenerationSHA256 {
			if err != nil {
				t.Fatal(err)
			}
		} else if !errors.Is(err, ErrOpenRGBDeviceGenerationChanged) {
			t.Fatalf("changed/legacy generation accepted: %q %v", generation, err)
		}
	}
}
