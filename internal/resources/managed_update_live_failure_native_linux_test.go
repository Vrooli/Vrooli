//go:build linux

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/config"
)

// This explicit live fault drill stops only the newly dispatched primary of
// the selected operation, through the exact native owner. It neither stops an
// original service nor manufactures a recovery receipt. The ordinary owner must
// recover; the restored serving process is deliberately left running.
func TestManagedUpdateCurrentHostLivePrimaryFailureDrill(t *testing.T) {
	request := os.Getenv("VROOLI_MANAGED_UPDATE_LIVE_FAILURE_REQUEST")
	if request == "" {
		t.Skip("explicit selected live operation is required")
	}
	sourcePID, err := strconv.Atoi(os.Getenv("VROOLI_MANAGED_UPDATE_LIVE_FAILURE_SOURCE_PID"))
	if err != nil || sourcePID <= 1 {
		t.Fatal("exact expected original PID is required")
	}
	sourceSHA := os.Getenv("VROOLI_MANAGED_UPDATE_LIVE_FAILURE_SOURCE_SHA")
	if !updateDigest(sourceSHA) || !managedUpdateKey.MatchString(request) {
		t.Fatal("exact expected original checksum and request are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 140*time.Second)
	defer cancel()
	store, err := newManagedUpdateStore("cloudflared")
	if err != nil {
		t.Fatal(err)
	}
	out := "/tmp/outage-cloudflared-live-primary-failure-drill.json"
	var selected ManagedServiceUpdate
	var stopped *platform.ServiceRef
	var injectedAt time.Time
	for ctx.Err() == nil {
		u, e := store.Lookup(request)
		if e == nil {
			selected = u
			if u.Original.PID != sourcePID || u.Original.ArtifactSHA256 != sourceSHA || u.Target.Artifact.SHA256 == sourceSHA {
				t.Fatal("live operation differs from the selected original and replacement")
			}
			if u.Phase.Terminal() {
				break
			}
			if stopped == nil && u.Phase == UpdatePrimaryStarting && u.PrimaryStopVerified && u.Primary.Intent != nil {
				fences, e := newManagedUpdateExecStore(u.Resource, u.ID, u.Primary.Intent.InstanceID)
				if e != nil {
					t.Fatal(e)
				}
				fence, e := fences.read()
				if e == nil && fence.State == "claimed" && fence.Claim != nil && fence.ClaimedAt != nil && time.Since(*fence.ClaimedAt) < 1500*time.Millisecond {
					image, e := platform.OpenProcessExecutable(fence.Claim.RootPID)
					if e == nil {
						digest, hashErr := managedUpdateImageDigest(ctx, image)
						image.Close()
						if hashErr == nil && digest == u.Target.Artifact.SHA256 {
							_, rollbackErr := observeManagedUpdateLiveDrillShadow(ctx, u.RollbackProbe)
							_, candidateErr := observeManagedUpdateLiveDrillShadow(ctx, u.Candidate)
							if rollbackErr == nil && candidateErr == nil {
								// Require early startup, before a healthy primary can be
								// published and its serving shadows cleaned up.
								connections := uint64(0)
								client := &http.Client{Timeout: 100 * time.Millisecond, Transport: &http.Transport{Proxy: nil, DisableKeepAlives: true}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
								response, readErr := client.Get("http://127.0.0.1:" + strconv.Itoa(u.Primary.Intent.Runtime.Port) + "/ready")
								if readErr == nil {
									var body struct {
										ReadyConnections uint64 `json:"readyConnections"`
									}
									readErr = json.NewDecoder(response.Body).Decode(&body)
									response.Body.Close()
									connections = body.ReadyConnections
								}
								latest, readStateErr := store.Lookup(request)
								if readStateErr == nil && latest.Phase == UpdatePrimaryStarting && latest.ID == u.ID && connections <= 1 && time.Since(*fence.ClaimedAt) < 1500*time.Millisecond {
									before := map[string]any{"request": request, "operation": u.ID, "phase": latest.Phase, "revision": latest.Revision, "primary_ref": fence.Claim, "primary_image_sha256": digest, "primary_connections_before_fault": connections, "both_serving_shadows_fresh": true, "injected": false}
									data, _ := json.MarshalIndent(before, "", "  ")
									if err := config.WriteOwnedFileDurable(out, append(data, '\n'), 0o600); err != nil {
										t.Fatal(err)
									}
									latest, e = store.Lookup(request)
									if e != nil || latest.Phase != UpdatePrimaryStarting || latest.ID != u.ID || time.Since(*fence.ClaimedAt) >= 1500*time.Millisecond {
										continue
									}
									stopped = fence.Claim
									injectedAt = time.Now().UTC()
									stopCtx, stopCancel := context.WithTimeout(ctx, 500*time.Millisecond)
									receipt, e := platform.StopOwnedService(stopCtx, *fence.Claim)
									stopCancel()
									t.Logf("early primary fault stop quiescent=%t error=%v; ordinary owner must establish recovery", receipt.NativeQuiescent, e)
									t.Logf("stopped the exact early primary at revision%d with both serving shadows verified", latest.Revision)
								}
								client.CloseIdleConnections()
							}
						}
					}
				}
			}
		}
		select {
		case <-ctx.Done():
		case <-time.After(10 * time.Millisecond):
		}
	}
	if stopped == nil {
		t.Fatalf("no safe early-primary window was observed; no connector was signalled (last phase %s)", selected.Phase)
	}
	result := map[string]any{"request": request, "operation": selected.ID, "injected": true, "injected_at": injectedAt, "stopped_primary_ref": stopped, "phase": selected.Phase, "revision": selected.Revision, "restored": selected.Restored, "candidate_quiescent": selected.CandidateQuiescent, "rollback_probe_quiescent": selected.RollbackProbeQuiescent, "error": selected.Error, "authenticated_remote_proved": false}
	data, _ := json.MarshalIndent(result, "", "  ")
	if err := config.WriteOwnedFileDurable(out, append(data, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if selected.Phase != UpdateFailedRestored || selected.Restored.State == nil || selected.Restored.State.ArtifactSHA256 != sourceSHA || !selected.CandidateQuiescent || !selected.RollbackProbeQuiescent {
		t.Fatal(fmt.Sprintf("live failure did not produce a fully cleaned restored primary: %s", selected.Phase))
	}
}

// The rollout owner has just verified each frozen image before admitting the
// original stop. This fault gate independently rechecks the exact native owner,
// ownership token and current connected readiness without hashing both large
// images again and missing the early startup window.
func observeManagedUpdateLiveDrillShadow(ctx context.Context, p ManagedUpdateProcess) (bool, error) {
	if p.NativeRef == nil || p.State == nil || p.Intent == nil {
		return false, fmt.Errorf("serving shadow has no exact owner")
	}
	native, err := platform.InspectOwnedService(ctx, *p.NativeRef)
	if err != nil {
		return false, err
	}
	if native.State != platform.OperationRunning || !native.RootAlive || !native.Populated {
		return false, fmt.Errorf("serving shadow native owner is not running")
	}
	if err := verifyManagedServiceOwnership(*p.State); err != nil {
		return false, err
	}
	transport := &http.Transport{Proxy: nil, DisableKeepAlives: true}
	defer transport.CloseIdleConnections()
	client := &http.Client{Timeout: time.Second, Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:"+strconv.Itoa(p.Intent.Runtime.Port)+"/ready", nil)
	if err != nil {
		return false, err
	}
	response, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer response.Body.Close()
	var body struct {
		ReadyConnections uint64 `json:"readyConnections"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		return false, err
	}
	if response.StatusCode != http.StatusOK || body.ReadyConnections < 4 {
		return false, fmt.Errorf("serving shadow does not have four connected edges")
	}
	return true, nil
}
