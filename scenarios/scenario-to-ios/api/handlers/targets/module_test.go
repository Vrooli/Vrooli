package targets

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	"scenario-to-ios/internal/targets"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestModule_UsesVersionedOperatorRoute(t *testing.T) {
	m := Module(targets.Prober{GOOS: "linux"})
	router := mux.NewRouter()
	m.Mount(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/ios/targets", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, response.Body.String(), "ios:simulator:linux")

	legacyRequest := httptest.NewRequest(http.MethodGet, "/ios/targets", nil)
	legacyResponse := httptest.NewRecorder()
	router.ServeHTTP(legacyResponse, legacyRequest)
	require.Equal(t, http.StatusNotFound, legacyResponse.Code)
}

// stubBridgeSource supplies bridge targets without contacting a fleet.
type stubBridgeSource struct {
	targets []deliveryramp.Target
	err     error
}

func (s stubBridgeSource) Discover(context.Context) ([]deliveryramp.Target, error) {
	return s.targets, s.err
}

func availableMacTarget() deliveryramp.Target {
	return deliveryramp.Target{
		ID: "bridge:mac-1", Label: "minimouse", Platform: "ios", OS: "darwin", Architecture: "amd64",
		DeviceKind: "emulator", Mode: "remote", Available: true, NodeID: "mac-1",
		Capabilities: []string{"xcodebuild", "simctl", "ios-simulator"},
		Transport:    deliveryramp.Transport{Kind: deliveryramp.TransportBridge, ID: "mac-1", Available: true},
		Health:       deliveryramp.TargetHealth{Status: "healthy"},
	}
}

// Without a bridge source a Linux host can only ever answer "no registered
// macOS bridge node", so the inventory must consult the sources it was given.
func TestModule_IncludesBridgeDiscoveredIOSTargets(t *testing.T) {
	m := Module(targets.Prober{GOOS: "linux"}, stubBridgeSource{targets: []deliveryramp.Target{availableMacTarget()}})
	router := mux.NewRouter()
	m.Mount(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/ios/targets", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	body := response.Body.String()
	require.Contains(t, body, "bridge:mac-1")
	// The terminal Linux row must survive alongside the reachable bridge row.
	require.Contains(t, body, "ios:simulator:linux")
}

// A Linux host reports its native simulator as terminally unsupported whether
// or not a bridge node exists; that row is a fact of the world, not a gap.
func TestModule_KeepsLinuxSimulatorUnsupportedAlongsideBridge(t *testing.T) {
	m := Module(targets.Prober{GOOS: "linux"}, stubBridgeSource{targets: []deliveryramp.Target{availableMacTarget()}})
	router := mux.NewRouter()
	m.Mount(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/ios/targets", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	var payload struct {
		Targets []struct {
			ID        string `json:"id"`
			Available bool   `json:"available"`
			Health    struct {
				Status string `json:"status"`
			} `json:"health"`
		} `json:"targets"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &payload))
	for _, target := range payload.Targets {
		if target.ID == "ios:simulator:linux" {
			require.False(t, target.Available)
			require.Equal(t, "unsupported", target.Health.Status)
		}
	}
}
