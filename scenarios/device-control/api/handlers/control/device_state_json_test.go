package control

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	internal "device-control/internal/control"
	"device-control/strategy"
	"device-control/strategy/fakes"
	strategyregistry "device-control/strategy/registry"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

type stateJSONReader struct {
	*fakes.Strategy
	state strategy.DeviceState
}

func (s *stateJSONReader) Enumerate(context.Context) ([]strategy.Device, error) {
	return []strategy.Device{{
		ID: "state-device", Name: "State Device", StrategyID: s.ID(),
		Transport: "state-json", Health: strategy.StatusAvailable,
		IdentityKey: "state-device-identity", ObservedAt: time.Now().UTC(),
	}}, nil
}

func (s *stateJSONReader) ReadState(context.Context) (strategy.DeviceState, error) {
	return s.state, nil
}

func TestDeviceStateEndpointSerializesDreamFields(t *testing.T) {
	reader := &stateJSONReader{
		Strategy: fakes.New("state-json", strategy.StatusAvailable),
		state: strategy.DeviceState{
			DreamState:     "active",
			DreamComponent: "com.example.dream/.DreamService",
			Unavailable:    map[string]string{"screen_state": "screen probe unavailable"},
		},
	}
	service := internal.New(strategyregistry.New(reader))
	require.Len(t, service.Devices(context.Background()), 1)

	router := mux.NewRouter()
	Module(service).Mount(router)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/devices/state-device/state", nil))

	require.Equal(t, http.StatusOK, response.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	require.Equal(t, "active", body["dream_state"])
	require.Equal(t, "com.example.dream/.DreamService", body["dream_component"])
	require.NotContains(t, body, "auto_rotate_known")
	require.NotContains(t, body, "rotation_settings")
	_, unavailablePresent := body["unavailable"]
	require.True(t, unavailablePresent)
}
