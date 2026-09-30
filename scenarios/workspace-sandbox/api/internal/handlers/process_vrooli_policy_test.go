package handlers_test

import (
	"encoding/json"
	"net/http"
	goruntime "runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"workspace-sandbox/internal/config"
	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/handlers"
	"workspace-sandbox/internal/runtime"
	"workspace-sandbox/internal/testutil/mocks"
	"workspace-sandbox/internal/testutil/mocks/procmocks"
	"workspace-sandbox/internal/types"
)

func TestExec_BlocksDestructiveVrooliMaintenance(t *testing.T) {
	id := uuid.New()
	sb := newProtectedSandboxFixture(id, nil)
	live := newLive(t, protectedService(sb))

	resp, body := live.DoJSON(t, "POST", sandboxesPath(id, "/exec"),
		`{"command": "/usr/bin/env", "args": ["vrooli", "cleanup", "locks"]}`)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("Exec status = %d, want 403; body=%s", resp.StatusCode, body)
	}

	var denial struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &denial); err != nil {
		t.Fatalf("decode denial body: %v", err)
	}
	if denial.Error != runtime.VrooliPolicyDestructiveMaintenanceBlocked {
		t.Errorf("denial.Error = %q, want %q", denial.Error, runtime.VrooliPolicyDestructiveMaintenanceBlocked)
	}
	if denial.Message == "" {
		t.Error("denial.Message should be non-empty")
	}
}

func TestProcessNetworkOverridePreservesExplicitDenial(t *testing.T) {
	if goruntime.GOOS != "linux" {
		t.Skip("bubblewrap launch contract is Linux-specific")
	}
	for _, endpoint := range []string{"/exec", "/processes"} {
		for _, network := range []string{"none", "localhost", "full"} {
			for _, override := range []string{"omitted", "null", "false", "true"} {
				t.Run(endpoint+"/"+network+"/"+override, func(t *testing.T) {
					sb := newProtectedSandboxFixture(uuid.New(), nil)
					sb.MergedDir = t.TempDir()
					sb.ProjectRoot = sb.MergedDir
					starter := procmocks.NewFakeStarter()
					starter.SetLookPath("bwrap", "/usr/bin/bwrap")
					starter.SetDefault(procmocks.CommandBehavior{})
					drv := mocks.NewFakeDriver()
					drv.ContainmentLevelVal = driver.ContainmentRequired
					live := newLive(t, protectedService(sb), withStarter(starter), withDriver(drv), func(h *handlers.Handlers) {
						h.SetProfileSnapshot(map[string]config.IsolationProfile{
							"full": {ID: "full", NetworkAccess: network, HomeOverlayRequirement: types.HomeOverlayNotNeeded},
						})
					})
					body := `{"command":"/bin/true"`
					if override != "omitted" {
						body += `,"allowNetwork":` + override
					}
					resp, raw := live.DoJSON(t, "POST", sandboxesPath(sb.ID, endpoint), body+"}")
					if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
						t.Fatalf("launch: HTTP %d: %s", resp.StatusCode, raw)
					}
					denied := override == "false" || (network == "none" && override != "true")
					launches := 0
					for _, call := range starter.MatchedCalls("/usr/bin/bwrap") {
						if !slices.Contains(call.Args, "/bin/true") {
							continue
						}
						launches++
						if got := slices.Contains(call.Args, "--unshare-net"); got != denied {
							t.Errorf("network namespace isolation = %t, want %t", got, denied)
						}
					}
					if launches != 1 {
						t.Fatalf("launched %d commands, want 1", launches)
					}
					var result struct {
						Containment struct{ Enforcements []string } `json:"containment"`
					}
					if err := json.Unmarshal(raw, &result); err != nil {
						t.Fatal(err)
					}
					if got := slices.Contains(result.Containment.Enforcements, "network-deny"); got != denied {
						t.Errorf("reported network denial = %t, want %t", got, denied)
					}
				})
			}
		}
	}
}

func TestTrackingProcessRefusesExplicitNetworkDenial(t *testing.T) {
	for _, endpoint := range []string{"/exec", "/processes", "/exec-interactive"} {
		t.Run(endpoint, func(t *testing.T) {
			sb := newProtectedSandboxFixture(uuid.New(), nil)
			sb.MergedDir = t.TempDir()
			starter := procmocks.NewFakeStarter()
			starter.SetDefault(procmocks.CommandBehavior{})
			live := newLive(t, protectedService(sb), withStarter(starter), func(h *handlers.Handlers) {
				h.SetProfileSnapshot(map[string]config.IsolationProfile{
					"full": {ID: "full", NetworkAccess: "full", HomeOverlayRequirement: types.HomeOverlayNotNeeded},
				})
			})
			if endpoint == "/exec-interactive" {
				conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(live.URL, "http")+sandboxesPath(sb.ID, endpoint), nil)
				if err != nil {
					t.Fatal(err)
				}
				defer conn.Close()
				_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
				if err := conn.WriteJSON(map[string]any{"command": "/not-a-real-review-command", "allowNetwork": false}); err != nil {
					t.Fatal(err)
				}
				var message handlers.InteractiveMessage
				if err := conn.ReadJSON(&message); err != nil || message.Type != handlers.MsgTypeError || !strings.Contains(message.Data, "explicit network denial requires process containment") {
					t.Fatalf("interactive denial = %+v, error %v", message, err)
				}
				return
			}
			resp, body := live.DoJSON(t, "POST", sandboxesPath(sb.ID, endpoint), `{"command":"/bin/true","allowNetwork":false}`)
			if resp.StatusCode != http.StatusConflict {
				t.Fatalf("denial without containment: HTTP %d: %s", resp.StatusCode, body)
			}
			if calls := starter.MatchedCalls(""); len(calls) != 0 {
				t.Fatalf("refused launch started processes: %+v", calls)
			}
			// Tracking remains available when the caller makes no denial claim.
			resp, body = live.DoJSON(t, "POST", sandboxesPath(sb.ID, endpoint), `{"command":"/bin/true"}`)
			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
				t.Fatalf("ordinary tracking launch: HTTP %d: %s", resp.StatusCode, body)
			}
			if calls := starter.MatchedCalls("/bin/true"); len(calls) != 1 {
				t.Fatalf("ordinary tracking launched %d commands, want 1", len(calls))
			}
		})
	}
}
