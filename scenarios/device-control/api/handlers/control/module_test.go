package control

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authdomain "device-control/internal/auth"
	internal "device-control/internal/control"
	"device-control/strategy"
	"device-control/strategy/androidtvremote"
	"device-control/strategy/fakes"
	"device-control/strategy/googlecast"
	strategyregistry "device-control/strategy/registry"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	flowsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/flows"
)

type failingRemoteDiscovery struct{}

func (failingRemoteDiscovery) DiscoverMDNS(context.Context) ([]androidtvremote.Device, error) {
	return nil, errors.New("multicast interface unavailable")
}

type castDiscoveryFixture struct{}

func (castDiscoveryFixture) DiscoverCast(context.Context) ([]googlecast.Device, error) {
	return []googlecast.Device{{ID: "cast-id-1", Name: "Living Room", Endpoint: "192.168.1.158:8009", Address: "192.168.1.158", Port: 8009, Service: "_googlecast._tcp", TXT: map[string]string{"id": "cast-id-1", "fn": "Living Room"}, IdentityKey: "cast-id-1"}}, nil
}

type remoteDiscoveryFixture struct{}

func (remoteDiscoveryFixture) DiscoverMDNS(context.Context) ([]androidtvremote.Device, error) {
	return []androidtvremote.Device{{Serial: "bt-1", IdentityKey: "bt-1", Name: "Living Room", Endpoint: "192.168.1.158:6466"}}, nil
}

func TestAcquireRequestUsesDistinctSnakeCaseFields(t *testing.T) {
	var got acquireRequest
	err := json.Unmarshal([]byte(`{"device_id":"phone-1","actor":"operator","ttl_seconds":90}`), &got)
	require.NoError(t, err)
	require.Equal(t, "phone-1", got.DeviceID)
	require.Equal(t, "operator", got.Actor)
	require.Equal(t, 90, got.TTLSeconds)
}

func TestFlowRequestUsesOneSnakeCaseConvention(t *testing.T) {
	var got flowRequest
	err := json.Unmarshal([]byte(`{"flow":{"id":"flow-1","name":"wake","steps":[{"id":"wake","kind":"key","required_capabilities":["input"],"target":"KEYCODE_WAKEUP","timeout_ms":5000,"arguments":{"settle_ms":25}}]},"strategy_id":"android-adb","device_id":"phone-1","actor":"operator"}`), &got)
	require.NoError(t, err)
	require.Equal(t, "flow-1", got.Flow.ID)
	require.Equal(t, "wake", got.Flow.Name)
	require.Len(t, got.Flow.Steps, 1)
	require.Equal(t, "KEYCODE_WAKEUP", got.Flow.Steps[0].Target)
	require.Equal(t, int64(5000), got.Flow.Steps[0].TimeoutMS)
	require.Equal(t, "android-adb", got.StrategyID)
	require.Equal(t, "phone-1", got.DeviceID)
	require.Equal(t, "operator", got.Actor)
}

func TestFlowProtoCarriesExplicitTransport(t *testing.T) {
	got := flowFromProto(&flowsv1.Flow{Id: "flow-1", Transport: "wireless"})
	require.Equal(t, "wireless", got.Transport)
}

func TestRunResultProtoCarriesDisconnectMetadata(t *testing.T) {
	got := runResultProto(internal.RunResult{RunID: "run-1", Disposition: "device_disconnected", Incomplete: true, DisconnectReason: "ADB endpoint disappeared", DisconnectStep: "actuate"})
	require.Equal(t, "run-1", got.RunId)
	require.Equal(t, "device_disconnected", got.Disposition)
	require.True(t, got.Incomplete)
	require.Equal(t, "ADB endpoint disappeared", got.DisconnectReason)
	require.Equal(t, "actuate", got.DisconnectStep)
}

func TestUnknownDeviceIsBadRequestAndPreservesMessage(t *testing.T) {
	service := internal.New(strategyregistry.New(fakes.New("phone-1", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot)))
	router := mux.NewRouter()
	Module(service).Mount(router)

	body := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/acquire", strings.NewReader(`{"device_id":"missing-phone","actor":"operator"}`))
	router.ServeHTTP(body, req)

	require.Equal(t, http.StatusBadRequest, body.Code)
	var response map[string]any
	require.NoError(t, json.Unmarshal(body.Body.Bytes(), &response))
	require.Equal(t, "unknown_device", response["code"])
	require.Contains(t, response["message"], "missing-phone")
}

func TestRetainedArtifactIsReadableWithoutExposingItsPath(t *testing.T) {
	service := internal.New(strategyregistry.New(fakes.New("phone-1", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot)))
	result, err := service.Run(context.Background(), internal.Flow{Steps: []internal.Step{{ID: "capture", Kind: "observe"}}}, "phone-1", "operator")
	require.NoError(t, err)
	require.Len(t, result.Evidence, 1)

	router := mux.NewRouter()
	Module(service).Mount(router)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/evidence/"+result.Evidence[0].ID, nil)
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "image/png", response.Header().Get("Content-Type"))
	require.NotEmpty(t, response.Body.Bytes())
}

func TestAndroidConformancePlanAndUnavailableRunAreExplicit(t *testing.T) {
	service := internal.New(strategyregistry.New(fakes.New("phone-1", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot)))
	router := mux.NewRouter()
	Module(service).Mount(router)

	planResponse := httptest.NewRecorder()
	router.ServeHTTP(planResponse, httptest.NewRequest(http.MethodGet, "/api/v1/conformance/android", nil))
	require.Equal(t, http.StatusOK, planResponse.Code)
	var plan struct {
		ID       string `json:"id"`
		Chapters []any  `json:"chapters"`
	}
	require.NoError(t, json.Unmarshal(planResponse.Body.Bytes(), &plan))
	require.Equal(t, "android-device-capability-self-test-v1", plan.ID)
	require.Len(t, plan.Chapters, 5)

	runResponse := httptest.NewRecorder()
	router.ServeHTTP(runResponse, httptest.NewRequest(http.MethodPost, "/api/v1/conformance/android/run", strings.NewReader(`{"device_id":"phone-1","actor":"operator"}`)))
	require.Equal(t, http.StatusOK, runResponse.Code)
	var result struct {
		Disposition string `json:"disposition"`
		Reason      string `json:"reason"`
		Verdict     struct {
			Disposition int    `json:"disposition"`
			Detail      string `json:"detail"`
		} `json:"verdict"`
	}
	require.NoError(t, json.Unmarshal(runResponse.Body.Bytes(), &result))
	require.Equal(t, "unavailable", result.Disposition)
	require.Contains(t, result.Reason, "not present in device-control inventory")
	require.NotEmpty(t, result.Verdict.Disposition)
	require.Contains(t, result.Verdict.Detail, "device_id=phone-1")
}

func TestDiscoveryReturnsReachableTransportsWhenAnotherBrowseFails(t *testing.T) {
	remote := androidtvremote.New(androidtvremote.WithDiscovery(failingRemoteDiscovery{}))
	cast := googlecast.New(googlecast.WithDiscovery(castDiscoveryFixture{}))
	service := internal.New(strategyregistry.New(remote, cast))
	router := mux.NewRouter()
	Module(service).Mount(router)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/devices/discover", nil))

	require.Equal(t, http.StatusOK, response.Code)
	var body struct {
		Services []map[string]any `json:"services"`
		Health   string           `json:"health"`
		Reason   string           `json:"reason"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	require.Len(t, body.Services, 1)
	require.Equal(t, "google-cast", body.Services[0]["strategy_id"])
	require.Equal(t, float64(8009), body.Services[0]["port"])
	require.Equal(t, "192.168.1.158", body.Services[0]["address"])
	require.Equal(t, map[string]any{"id": "cast-id-1", "fn": "Living Room"}, body.Services[0]["txt"])
	require.Equal(t, "degraded", body.Health)
	require.Contains(t, body.Reason, "android-tv-remote")
}

func TestDiscoveryUsesCanonicalDeviceIDForPairingTarget(t *testing.T) {
	remote := androidtvremote.New(androidtvremote.WithDiscovery(remoteDiscoveryFixture{}))
	service := internal.New(strategyregistry.New(remote))

	found, err := service.DiscoverLAN(context.Background())
	require.NoError(t, err)
	require.Len(t, found, 1)
	require.Equal(t, "android-tv:bt-1", found[0].ID)
}

func TestStrategyListDerivesTiersFromCapabilities(t *testing.T) {
	service := internal.New(strategyregistry.New(googlecast.New(googlecast.WithDiscovery(castDiscoveryFixture{}))))
	router := mux.NewRouter()
	Module(service).Mount(router)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/strategies", nil))
	require.Equal(t, http.StatusOK, response.Code)
	var body struct {
		Strategies []struct {
			ID    string   `json:"id"`
			Tiers []string `json:"tiers"`
		} `json:"strategies"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	for _, item := range body.Strategies {
		if item.ID != "google-cast" {
			continue
		}
		require.ElementsMatch(t, []string{"floor", "property", "sensor", "media"}, item.Tiers)
		return
	}
	t.Fatal("google-cast strategy was not returned")
}

func TestAuthenticationProfileLifecycle(t *testing.T) {
	service := internal.New(strategyregistry.New(fakes.New("phone-1", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot)))
	router := mux.NewRouter()
	Module(service).Mount(router)
	create := httptest.NewRecorder()
	router.ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/api/v1/auth/profiles", strings.NewReader(`{"profile":{"id":"profile-1","device_id":"phone-1","method":"pin","credential_identity":"device-control/phone-1/profile-1","credential_field":"unlock","verification":"fresh_lock_state_unlocked"},"actor":"test"}`)))
	require.Equal(t, http.StatusCreated, create.Code)
	require.NotContains(t, create.Body.String(), "value")
	get := httptest.NewRecorder()
	router.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/api/v1/auth/profiles/profile-1", nil))
	require.Equal(t, http.StatusOK, get.Code)
	var view struct {
		Profile  authdomain.Profile        `json:"profile"`
		Provider authdomain.ProviderStatus `json:"provider"`
	}
	require.NoError(t, json.Unmarshal(get.Body.Bytes(), &view))
	require.Equal(t, "device-control/phone-1/profile-1", view.Profile.CredentialIdentity)
	require.False(t, view.Provider.Configured)
	update := httptest.NewRecorder()
	router.ServeHTTP(update, httptest.NewRequest(http.MethodPut, "/api/v1/auth/profiles/profile-1", strings.NewReader(`{"profile":{"method":"numeric_passcode"},"actor":"test"}`)))
	require.Equal(t, http.StatusOK, update.Code)
	require.Contains(t, update.Body.String(), "numeric_passcode")
	testResponse := httptest.NewRecorder()
	router.ServeHTTP(testResponse, httptest.NewRequest(http.MethodPost, "/api/v1/auth/profiles/profile-1/test", nil))
	require.Equal(t, http.StatusOK, testResponse.Code)
	revoke := httptest.NewRecorder()
	router.ServeHTTP(revoke, httptest.NewRequest(http.MethodDelete, "/api/v1/auth/profiles/profile-1", nil))
	require.Equal(t, http.StatusOK, revoke.Code)
}

func TestUnlockErrorIsRedactedAtTheTransportBoundary(t *testing.T) {
	err := errors.New("runner output contained runtime-only-fixture")
	require.NotContains(t, safeUnlockError(err), "runtime-only-fixture")
	require.Equal(t, "device unlock transaction failed; inspect the typed result and next action", safeUnlockError(err))
	require.Equal(t, "unlock requires an active device lease", safeUnlockError(errors.New("unlock requires an active device lease")))
}
