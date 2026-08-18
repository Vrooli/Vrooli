// Package homeassistant is an attach-only Home Assistant REST strategy. It
// observes and controls already-owned entities; it never installs, starts,
// stops, or repairs a Home Assistant service.
package homeassistant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"device-control/strategy"
)

type Entity struct {
	EntityID   string         `json:"entity_id"`
	State      string         `json:"state"`
	Attributes map[string]any `json:"attributes"`
}

type Strategy struct {
	baseURL string
	token   string
	client  *http.Client
	entity  *Entity
}

func New(baseURL, token string, client *http.Client) *Strategy {
	if client == nil {
		client = http.DefaultClient
	}
	return &Strategy{baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"), token: token, client: client}
}

// NewFromEnv creates the attach-only adapter from operator configuration. The
// token is retained only in the adapter instance and is never included in a
// declaration, device identity, or audit record.
func NewFromEnv() *Strategy {
	baseURL := firstNonEmptyEnv("HOME_ASSISTANT_URL", "HOME_ASSISTANT_BASE_URL")
	token := firstNonEmptyEnv("HOME_ASSISTANT_TOKEN", "HOME_ASSISTANT_ACCESS_TOKEN")
	return New(baseURL, token, nil)
}

func firstNonEmptyEnv(names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	return ""
}

func (s *Strategy) ID() string { return "home-assistant-rest" }

func (s *Strategy) Describe(ctx context.Context) (strategy.Declaration, error) {
	if err := s.ensureConfigured(); err != nil {
		return strategy.UnavailableDeclaration(s.ID(), err.Error(), unavailableCapabilityList(err.Error()), "Configure the Home Assistant URL and long-lived access token."), nil
	}
	if s.entity == nil {
		return strategy.Declaration{StrategyID: s.ID(), Description: "Home Assistant attach-only REST bridge", Status: strategy.StatusAvailable, Capabilities: map[string]strategy.Capability{}}, nil
	}
	return s.entityDeclaration(), nil
}

func (s *Strategy) Enumerate(ctx context.Context) ([]strategy.Device, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, err
	}
	var entities []Entity
	if err := s.request(ctx, http.MethodGet, "/api/states", nil, &entities); err != nil {
		return nil, err
	}
	out := make([]strategy.Device, 0, len(entities))
	for _, entity := range entities {
		if strings.TrimSpace(entity.EntityID) == "" {
			continue
		}
		out = append(out, strategy.Device{ID: "home-assistant:" + entity.EntityID, Serial: entity.EntityID, IdentityKey: entity.EntityID, Name: entity.EntityID, StrategyID: s.ID(), Transport: "rest", Health: strategy.StatusAvailable, ObservedAt: time.Now().UTC()})
	}
	return out, nil
}

func (s *Strategy) ForDevice(entityID string) strategy.Strategy {
	clone := *s
	clone.entity = &Entity{EntityID: strings.TrimSpace(entityID), Attributes: map[string]any{}}
	return &clone
}

func (s *Strategy) entityDeclaration() strategy.Declaration {
	domain := strings.SplitN(s.entity.EntityID, ".", 2)[0]
	capabilities := unavailableCapabilities("Home Assistant entity domain does not expose this modality")
	d := strategy.Declaration{StrategyID: s.ID(), DeviceID: "home-assistant:" + s.entity.EntityID, Transport: "rest", Description: "Home Assistant entity " + s.entity.EntityID, Status: strategy.StatusAvailable, Capabilities: capabilities, Promotable: false, EvidenceClass: "external-rest"}
	switch domain {
	case "light", "switch", "climate", "cover", "lock":
		capabilities[strategy.CapProperty] = strategy.Capability{Name: strategy.CapProperty, Status: strategy.StatusAvailable, StateClass: strategy.StateBearing}
		d.Properties = []strategy.PropertyDescriptor{{Name: "state", ValueType: "string", Writable: true, Enumeration: propertyStates(domain), StateClass: strategy.StateBearing}}
	case "sensor", "binary_sensor":
		capabilities[strategy.CapSensor] = strategy.Capability{Name: strategy.CapSensor, Status: strategy.StatusAvailable, StateClass: strategy.StateBearing}
	case "media_player":
		capabilities[strategy.CapMedia] = strategy.Capability{Name: strategy.CapMedia, Status: strategy.StatusAvailable, StateClass: strategy.EventBearing}
	case "camera":
		capabilities[strategy.CapScreenshot] = strategy.Capability{Name: strategy.CapScreenshot, Status: strategy.StatusAvailable, StateClass: strategy.EventBearing}
	}
	return d
}

func unavailableCapabilities(reason string) map[string]strategy.Capability {
	capabilities := make(map[string]strategy.Capability, 6)
	for _, name := range []string{strategy.CapInput, strategy.CapScreenshot, strategy.CapSemanticTree, strategy.CapProperty, strategy.CapSensor, strategy.CapMedia} {
		capabilities[name] = strategy.Capability{Name: name, Status: strategy.StatusUnavailable, Reason: reason}
	}
	return capabilities
}

func unavailableCapabilityList(reason string) []strategy.Capability {
	capabilities := unavailableCapabilities(reason)
	out := make([]strategy.Capability, 0, len(capabilities))
	for _, capability := range capabilities {
		out = append(out, capability)
	}
	return out
}

func propertyStates(domain string) []string {
	switch domain {
	case "light", "switch":
		return []string{"on", "off"}
	case "lock":
		return []string{"locked", "unlocked"}
	default:
		return nil
	}
}

func (s *Strategy) GetProperty(ctx context.Context, name string) (any, error) {
	entity, err := s.state(ctx)
	if err != nil {
		return nil, err
	}
	if name == "state" {
		return entity.State, nil
	}
	value, ok := entity.Attributes[name]
	if !ok {
		return nil, fmt.Errorf("Home Assistant entity %q has no property %q", entity.EntityID, name)
	}
	return value, nil
}

func (s *Strategy) SetProperty(ctx context.Context, set strategy.PropertySet) error {
	if s.entity == nil {
		return &strategy.UnsupportedCapabilityError{Capability: strategy.CapProperty, Operation: "set-property"}
	}
	if strings.TrimSpace(set.Name) != "state" {
		return fmt.Errorf("Home Assistant entity %q exposes only the state property", s.entity.EntityID)
	}
	domain := strings.SplitN(s.entity.EntityID, ".", 2)[0]
	value := strings.ToLower(strings.TrimSpace(fmt.Sprint(set.Value)))
	service := ""
	data := map[string]any{"entity_id": s.entity.EntityID}
	switch domain {
	case "light", "switch":
		service = "turn_on"
		if value == "off" {
			service = "turn_off"
		}
	case "lock":
		service = "lock"
		if value == "unlocked" {
			service = "unlock"
		}
	case "cover":
		service = map[string]string{"open": "open_cover", "closed": "close_cover", "stopped": "stop_cover"}[value]
	case "climate":
		service = "set_hvac_mode"
		data["hvac_mode"] = set.Value
	default:
		return fmt.Errorf("Home Assistant domain %q does not support property actuation", domain)
	}
	if service == "" {
		return fmt.Errorf("unsupported Home Assistant %s state %q", domain, value)
	}
	return s.request(ctx, http.MethodPost, "/api/services/"+domain+"/"+service, data, nil)
}

func (s *Strategy) ReadSensors(ctx context.Context) ([]strategy.SensorReading, error) {
	entity, err := s.state(ctx)
	if err != nil {
		return nil, err
	}
	unit, _ := entity.Attributes["unit_of_measurement"].(string)
	return []strategy.SensorReading{{Name: entity.EntityID, Value: entity.State, Unit: unit, ObservedAt: time.Now().UTC(), StateClass: strategy.StateBearing}}, nil
}

func (s *Strategy) ControlMedia(ctx context.Context, command strategy.MediaCommand) error {
	if s.entity == nil {
		return &strategy.UnsupportedCapabilityError{Capability: strategy.CapMedia, Operation: "media-control"}
	}
	service := map[string]string{"play": "media_play", "pause": "media_pause", "stop": "media_stop", "next": "media_next", "previous": "media_previous", "volume-up": "volume_up", "volume-down": "volume_down"}[command.Action]
	data := map[string]any{"entity_id": s.entity.EntityID}
	if command.Action == "volume" {
		service = "volume_set"
		data["volume_level"] = command.Value
	}
	if service == "" {
		return fmt.Errorf("unsupported Home Assistant media action %q", command.Action)
	}
	return s.request(ctx, http.MethodPost, "/api/services/media_player/"+service, data, nil)
}

func (s *Strategy) Observe(ctx context.Context) (strategy.Frame, error) {
	if s.entity == nil || !strings.HasPrefix(s.entity.EntityID, "camera.") {
		return strategy.Frame{}, &strategy.UnsupportedCapabilityError{Capability: strategy.CapScreenshot, Operation: "observe"}
	}
	if err := s.ensureConfigured(); err != nil {
		return strategy.Frame{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+"/api/camera_proxy/"+s.entity.EntityID, nil)
	if err != nil {
		return strategy.Frame{}, err
	}
	req.Header.Set("Authorization", "Bearer "+s.token)
	resp, err := s.client.Do(req)
	if err != nil {
		return strategy.Frame{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return strategy.Frame{}, fmt.Errorf("Home Assistant camera returned %s", resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return strategy.Frame{}, err
	}
	return strategy.Frame{Bytes: data, MediaType: resp.Header.Get("Content-Type"), Timestamp: time.Now().UTC()}, nil
}

func (s *Strategy) state(ctx context.Context) (Entity, error) {
	var entity Entity
	err := s.request(ctx, http.MethodGet, "/api/states/"+s.entity.EntityID, nil, &entity)
	return entity, err
}

func (s *Strategy) request(ctx context.Context, method, path string, body any, out any) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}
	var payload *bytes.Reader
	if body == nil {
		payload = bytes.NewReader(nil)
	} else {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		payload = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, s.baseURL+path, payload)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Home Assistant REST %s returned %s", path, resp.Status)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func (s *Strategy) ensureConfigured() error {
	if s.baseURL == "" || strings.TrimSpace(s.token) == "" {
		return &strategy.AvailabilityError{Reason: "attach-only Home Assistant endpoint and token are not configured", NextAction: "Configure the Home Assistant URL and long-lived access token."}
	}
	return nil
}
