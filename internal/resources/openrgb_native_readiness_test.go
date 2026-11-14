package resources

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"testing"

	platform "github.com/vrooli/platform-go"
)

// Exercise the actual observer during the executor's consumed-claim handoff.
// Only native context/listener/payload observation edges are simulated.
func TestOpenRGBClaimedHandoffWaitsForOwnedListenerBeforePayloadProof(t *testing.T) {
	for _, kind := range []string{"absent", "foreign", "wildcard", "changed-context", "owned"} {
		t.Run(kind, func(t *testing.T) {
			f := newNativeCapsuleExecFixture(t)
			endpoint := netip.MustParseAddrPort("127.0.0.1:6742")
			witness := openRGBExecWitness{Policy: f.session.material.Policy, Incarnation: f.session.ref.Native.RootIncarnation, ClaimSHA256: f.session.material.ClaimSHA256, ContextSHA256: strings.Repeat("d", 64)}
			observation := openRGBServingObservation{ref: f.session.ref, witness: witness, endpoint: endpoint}
			listeners := []platform.ProcessTCPListener{}
			switch kind {
			case "foreign":
				listeners = []platform.ProcessTCPListener{{Address: "127.0.0.1", Port: endpoint.Port(), PID: f.session.ref.RootPID + 1}}
			case "wildcard":
				listeners = []platform.ProcessTCPListener{{Address: "0.0.0.0", Port: endpoint.Port(), PID: f.session.ref.RootPID}}
			case "owned":
				listeners = []platform.ProcessTCPListener{{Address: "127.0.0.1", Port: endpoint.Port(), PID: f.session.ref.RootPID}}
			}
			payloadCalls := 0
			payloadMismatch := errors.New("observed launcher has not become the sealed payload")
			c := &Controller{openRGBPayloadEffects: &openRGBPayloadEffects{
				context: func(platform.ServiceRef, string) (string, error) {
					if kind == "changed-context" {
						return strings.Repeat("e", 64), nil
					}
					return witness.ContextSHA256, nil
				},
				listener: func(actual netip.AddrPort, pid int) error {
					return verifyOpenRGBNativeListeners(actual, pid, listeners)
				},
				payload: func(context.Context, ManagedServiceArtifact, openRGBServingObservation) error {
					payloadCalls++
					return payloadMismatch
				},
			}}
			stopped := errors.New("test stops before vendor execution")
			f.owner.materialize = func(m openRGBNativeMaterial, _ managedUpdateInvocation) error {
				if !f.session.claimed {
					t.Fatal("readiness fixture did not reach the consumed-claim window")
				}
				err := c.observeOpenRGBNativePayload(t.Context(), m.Contract.Artifact, observation)
				switch kind {
				case "absent":
					if !errors.Is(err, errOpenRGBNotReady) || payloadCalls != 0 {
						t.Fatalf("valid handoff became terminal/payload observed: %v calls=%d", err, payloadCalls)
					}
				case "owned":
					if !errors.Is(err, payloadMismatch) || payloadCalls != 1 {
						t.Fatalf("owned listener bypassed payload proof: %v calls=%d", err, payloadCalls)
					}
				default:
					if err == nil || errors.Is(err, errOpenRGBNotReady) || payloadCalls != 0 {
						t.Fatalf("unsafe listener/context classified pending: %v calls=%d", err, payloadCalls)
					}
				}
				return stopped
			}
			if err := executeOpenRGBNativeCapsule(t.Context(), f.capsule, f.owner); !errors.Is(err, stopped) {
				t.Fatal(err)
			}
		})
	}
}
