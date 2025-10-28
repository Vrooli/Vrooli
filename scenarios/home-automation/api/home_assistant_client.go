package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type HomeAssistantClient struct {
	baseURL      string
	token        string
	tokenType    string
	refreshToken string
	tokenExpiry  time.Time
	httpClient   *http.Client
	mu           sync.Mutex
}

func NewHomeAssistantClient(baseURL, token, tokenType, refreshToken string, expiresAt *time.Time) *HomeAssistantClient {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		return nil
	}

	client := &http.Client{Timeout: 10 * time.Second}

	return &HomeAssistantClient{
		baseURL:      strings.TrimRight(trimmed, "/"),
		token:        strings.TrimSpace(token),
		tokenType:    strings.TrimSpace(tokenType),
		refreshToken: strings.TrimSpace(refreshToken),
		httpClient:   client,
		tokenExpiry: func() time.Time {
			if expiresAt == nil {
				return time.Time{}
			}
			return expiresAt.UTC()
		}(),
	}
}

func (c *HomeAssistantClient) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	if c == nil {
		return nil, fmt.Errorf("home assistant client not configured")
	}
	if err := c.ensureAccessToken(ctx); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		if len(data) > 0 {
			return nil, fmt.Errorf("home assistant error: %s", strings.TrimSpace(string(data)))
		}
		return nil, fmt.Errorf("home assistant error: %s", resp.Status)
	}

	return resp, nil
}

func (c *HomeAssistantClient) ensureAccessToken(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("home assistant client not configured")
	}

	if !strings.EqualFold(strings.TrimSpace(c.tokenType), "refresh") {
		if strings.TrimSpace(c.token) == "" {
			return fmt.Errorf("home assistant client not configured")
		}
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.refreshToken == "" {
		return fmt.Errorf("home assistant refresh token not configured")
	}

	if strings.TrimSpace(c.token) != "" {
		if c.tokenExpiry.IsZero() || time.Now().Add(30*time.Second).Before(c.tokenExpiry) {
			return nil
		}
	}

	accessToken, expiresIn, err := requestHomeAssistantAccessToken(ctx, c.baseURL, c.refreshToken)
	if err != nil {
		return err
	}

	c.token = accessToken
	if expiresIn > 0 {
		c.tokenExpiry = time.Now().Add(time.Duration(expiresIn) * time.Second)
	} else {
		c.tokenExpiry = time.Time{}
	}

	return nil
}

type haState struct {
	EntityID    string                 `json:"entity_id"`
	State       string                 `json:"state"`
	Attributes  map[string]interface{} `json:"attributes"`
	LastChanged string                 `json:"last_changed"`
	LastUpdated string                 `json:"last_updated"`
}

func (c *HomeAssistantClient) ListDevices(ctx context.Context) ([]DeviceStatus, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/states", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var states []haState
	if err := json.NewDecoder(resp.Body).Decode(&states); err != nil {
		return nil, err
	}

	devices := make([]DeviceStatus, 0, len(states))
	for _, st := range states {
		if !isSupportedDomain(st.EntityID) {
			continue
		}
		devices = append(devices, mapHAStateToDeviceStatus(st))
	}

	sort.Slice(devices, func(i, j int) bool {
		return strings.ToLower(devices[i].Name) < strings.ToLower(devices[j].Name)
	})

	return devices, nil
}

func (c *HomeAssistantClient) GetDevice(ctx context.Context, entityID string) (*DeviceStatus, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/states/"+entityID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var state haState
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		return nil, err
	}

	device := mapHAStateToDeviceStatus(state)
	return &device, nil
}

func (c *HomeAssistantClient) ControlDevice(ctx context.Context, req DeviceControlRequest) (*HACommandResult, error) {
	domain := domainFromEntity(req.DeviceID)
	if domain == "" {
		return nil, fmt.Errorf("invalid device domain")
	}

	if req.Action == "refresh" {
		status, err := c.GetDevice(ctx, req.DeviceID)
		if err != nil {
			return nil, err
		}
		return &HACommandResult{
			State:   status.State,
			Message: fmt.Sprintf("Device %s refreshed via Home Assistant", req.DeviceID),
		}, nil
	}

	servicePath, payload, err := buildServiceCall(domain, req)
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	if _, err := c.doRequest(ctx, http.MethodPost, "/api/services/"+servicePath, bytes.NewReader(body)); err != nil {
		return nil, err
	}

	status, err := c.GetDevice(ctx, req.DeviceID)
	if err != nil {
		return nil, err
	}

	return &HACommandResult{
		State:   status.State,
		Message: fmt.Sprintf("Device %s %s via Home Assistant", req.DeviceID, req.Action),
	}, nil
}

func buildServiceCall(domain string, req DeviceControlRequest) (string, map[string]interface{}, error) {
	payload := map[string]interface{}{
		"entity_id": req.DeviceID,
	}

	switch req.Action {
	case "turn_on", "turn_off", "toggle":
		service := req.Action
		switch domain {
		case "lock":
			if req.Action == "turn_on" {
				service = "unlock"
			} else if req.Action == "turn_off" {
				service = "lock"
			}
		case "cover":
			if req.Action == "turn_on" {
				service = "open_cover"
			} else if req.Action == "turn_off" {
				service = "close_cover"
			}
		case "climate":
			if req.Action == "turn_on" || req.Action == "turn_off" {
				return "", nil, fmt.Errorf("use set_mode for climate domain")
			}
		}

		if req.Action == "toggle" && domain == "lock" {
			service = "toggle"
		}

		return domain + "/" + service, mergeParams(payload, req.Parameters), nil
	case "set_brightness":
		payload = mergeParams(payload, req.Parameters)
		if _, ok := payload["brightness"]; !ok {
			if pct, ok := payload["brightness_pct"]; ok {
				payload["brightness_pct"] = pct
			}
		}
		return domain + "/turn_on", payload, nil
	case "set_temperature":
		return "climate/set_temperature", mergeParams(payload, req.Parameters), nil
	case "set_mode":
		if domain == "climate" {
			mode := valueOrDefault(req.Parameters, "mode")
			if mode == nil {
				return "", nil, fmt.Errorf("mode parameter required for climate devices")
			}
			payload = mergeParams(payload, map[string]interface{}{"hvac_mode": mode})
			return "climate/set_hvac_mode", payload, nil
		}
		if domain == "fan" {
			mode := valueOrDefault(req.Parameters, "mode")
			if mode == nil {
				return "", nil, fmt.Errorf("mode parameter required for fan devices")
			}
			payload = mergeParams(payload, map[string]interface{}{"preset_mode": mode})
			return "fan/set_preset_mode", payload, nil
		}
		return "", nil, fmt.Errorf("set_mode not supported for domain %s", domain)
	case "activate":
		return domain + "/turn_on", mergeParams(payload, req.Parameters), nil
	case "refresh":
		return "", nil, fmt.Errorf("refresh handled via status lookup")
	}

	return "", nil, fmt.Errorf("unsupported action: %s", req.Action)
}

func mergeParams(base map[string]interface{}, params map[string]interface{}) map[string]interface{} {
	if params == nil {
		return base
	}
	for k, v := range params {
		base[k] = v
	}
	return base
}

func valueOrDefault(params map[string]interface{}, key string) interface{} {
	if params == nil {
		return nil
	}
	if val, ok := params[key]; ok {
		return val
	}
	return nil
}

func isSupportedDomain(entityID string) bool {
	domain := domainFromEntity(entityID)
	switch domain {
	case "light", "switch", "sensor", "climate", "lock", "cover", "fan", "humidifier", "binary_sensor":
		return true
	default:
		return false
	}
}

func domainFromEntity(entityID string) string {
	parts := strings.SplitN(entityID, ".", 2)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func mapHAStateToDeviceStatus(st haState) DeviceStatus {
	domain := domainFromEntity(st.EntityID)
	name := st.EntityID
	if friendly, ok := st.Attributes["friendly_name"].(string); ok && friendly != "" {
		name = friendly
	}

	available := st.State != "unavailable"
	state := map[string]interface{}{
		"raw_state": st.State,
	}

	switch domain {
	case "light", "switch", "fan":
		state["on"] = st.State == "on"
		if brightness, ok := asFloat(st.Attributes["brightness"]); ok {
			state["brightness"] = brightness
		}
		if pct, ok := asFloat(st.Attributes["brightness_pct"]); ok {
			state["brightness_pct"] = pct
		}
		if speed, ok := st.Attributes["speed"]; ok {
			state["speed"] = speed
		}
	case "sensor", "binary_sensor":
		state["value"] = st.State
		if unit, ok := st.Attributes["unit_of_measurement"].(string); ok {
			state["unit"] = unit
		}
	case "climate":
		state["mode"] = st.State
		if temp, ok := asFloat(st.Attributes["current_temperature"]); ok {
			state["temperature"] = temp
		}
		if target, ok := asFloat(st.Attributes["temperature"]); ok {
			state["target_temperature"] = target
		}
		if hvacAction, ok := st.Attributes["hvac_action"].(string); ok {
			state["hvac_action"] = hvacAction
		}
	case "lock":
		state["locked"] = st.State == "locked"
	case "cover":
		if position, ok := asFloat(st.Attributes["current_position"]); ok {
			state["position"] = position
		}
		state["open"] = st.State == "open"
	case "humidifier":
		state["on"] = st.State != "off"
		if humidity, ok := asFloat(st.Attributes["humidity"]); ok {
			state["humidity"] = humidity
		}
	}

	var lastUpdated string
	if ts, err := time.Parse(time.RFC3339Nano, st.LastChanged); err == nil {
		lastUpdated = ts.Format(time.RFC3339)
	} else if ts, err := time.Parse(time.RFC3339Nano, st.LastUpdated); err == nil {
		lastUpdated = ts.Format(time.RFC3339)
	} else {
		lastUpdated = time.Now().Format(time.RFC3339)
	}

	return DeviceStatus{
		DeviceID:    st.EntityID,
		Name:        name,
		Type:        domain,
		State:       state,
		Available:   available,
		LastUpdated: lastUpdated,
	}
}

func asFloat(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		if f, err := v.Float64(); err == nil {
			return f, true
		}
	case string:
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			return parsed, true
		}
	}
	return 0, false
}
