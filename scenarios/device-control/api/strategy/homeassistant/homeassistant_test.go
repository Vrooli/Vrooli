package homeassistant

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"device-control/strategy"
	"github.com/stretchr/testify/require"
)

func TestAttachOnlyRESTFixtureEnumeratesTypedEntities(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/states":
			_ = json.NewEncoder(w).Encode([]Entity{{EntityID: "sensor.living_room", State: "21.5", Attributes: map[string]any{"unit_of_measurement": "°C"}}, {EntityID: "media_player.tv", State: "paused", Attributes: map[string]any{}}})
		case "/api/states/sensor.living_room":
			_ = json.NewEncoder(w).Encode(Entity{EntityID: "sensor.living_room", State: "21.5", Attributes: map[string]any{"unit_of_measurement": "°C"}})
		default:
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()
	s := New(server.URL, "fixture-token", server.Client())
	devices, err := s.Enumerate(context.Background())
	require.NoError(t, err)
	require.Len(t, devices, 2)
	sensor := s.ForDevice("sensor.living_room")
	declaration, err := sensor.Describe(context.Background())
	require.NoError(t, err)
	require.Equal(t, strategy.StatusAvailable, declaration.Capabilities[strategy.CapSensor].Status)
	require.Equal(t, strategy.StatusUnavailable, declaration.Capabilities[strategy.CapInput].Status)
	readings, err := sensor.(strategy.SensorReader).ReadSensors(context.Background())
	require.NoError(t, err)
	require.Equal(t, "21.5", readings[0].Value)
}

func TestNewFromEnvReadsAttachConfiguration(t *testing.T) {
	t.Setenv("HOME_ASSISTANT_URL", "http://ha.example:8123/")
	t.Setenv("HOME_ASSISTANT_TOKEN", "fixture-token")
	s := NewFromEnv()
	require.Equal(t, "http://ha.example:8123", s.baseURL)
	require.Equal(t, "fixture-token", s.token)
	declaration, err := s.Describe(context.Background())
	require.NoError(t, err)
	require.Equal(t, strategy.StatusAvailable, declaration.Status)
	require.NotContains(t, declaration.Description, "fixture-token")
}

func TestAttachOperationsRequireTheLongLivedToken(t *testing.T) {
	s := New("http://ha.example:8123", "", nil)
	declaration, err := s.Describe(context.Background())
	require.NoError(t, err)
	require.Equal(t, strategy.StatusUnavailable, declaration.Status)
	_, err = s.Enumerate(context.Background())
	var availability *strategy.AvailabilityError
	require.ErrorAs(t, err, &availability)
	require.Contains(t, availability.Reason, "token")
}

func TestEntityDeclarationNamesUnavailableModalities(t *testing.T) {
	tests := []struct {
		entity      string
		available   string
		unavailable []string
	}{
		{entity: "light.kitchen", available: strategy.CapProperty, unavailable: []string{strategy.CapInput, strategy.CapScreenshot, strategy.CapSensor, strategy.CapMedia}},
		{entity: "sensor.temperature", available: strategy.CapSensor, unavailable: []string{strategy.CapInput, strategy.CapScreenshot, strategy.CapProperty, strategy.CapMedia}},
		{entity: "media_player.tv", available: strategy.CapMedia, unavailable: []string{strategy.CapInput, strategy.CapScreenshot, strategy.CapProperty, strategy.CapSensor}},
		{entity: "camera.front_door", available: strategy.CapScreenshot, unavailable: []string{strategy.CapInput, strategy.CapProperty, strategy.CapSensor, strategy.CapMedia}},
	}
	for _, test := range tests {
		t.Run(test.entity, func(t *testing.T) {
			declaration, err := New("http://ha.example", "token", nil).ForDevice(test.entity).Describe(context.Background())
			require.NoError(t, err)
			require.Equal(t, strategy.StatusAvailable, declaration.Capabilities[test.available].Status)
			for _, name := range test.unavailable {
				capability, ok := declaration.Capabilities[name]
				require.True(t, ok, "capability %s must be explicit", name)
				require.Equal(t, strategy.StatusUnavailable, capability.Status)
				require.NotEmpty(t, capability.Reason)
			}
		})
	}
}

func TestMediaVolumeUsesTypedHomeAssistantService(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	media := New(server.URL, "fixture-token", server.Client()).ForDevice("media_player.tv").(strategy.MediaController)
	require.NoError(t, media.ControlMedia(context.Background(), strategy.MediaCommand{Action: "volume", Value: 0.5}))
	require.Equal(t, "/api/services/media_player/volume_set", gotPath)
	require.Equal(t, "media_player.tv", gotBody["entity_id"])
	require.Equal(t, 0.5, gotBody["volume_level"])
}
