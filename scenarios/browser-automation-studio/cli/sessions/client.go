package sessions

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"browser-automation-studio/cli/internal/appctx"
)

type sessionProfile struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	CreatedAt       string          `json:"created_at"`
	UpdatedAt       string          `json:"updated_at"`
	LastUsedAt      string          `json:"last_used_at"`
	HasStorageState bool            `json:"has_storage_state"`
	BrowserProfile  json.RawMessage `json:"browser_profile,omitempty"`
}

type listResponse struct {
	Profiles []sessionProfile `json:"profiles"`
}

type storageStateResponse struct {
	Cookies []storageStateCookie `json:"cookies"`
	Origins []storageStateOrigin `json:"origins"`
	Stats   storageStateStats    `json:"stats"`
}

type storageStateCookie struct {
	Name        string  `json:"name"`
	Value       string  `json:"value"`
	ValueMasked bool    `json:"valueMasked"`
	Domain      string  `json:"domain"`
	Path        string  `json:"path"`
	Expires     float64 `json:"expires"`
	HttpOnly    bool    `json:"httpOnly"`
	Secure      bool    `json:"secure"`
	SameSite    string  `json:"sameSite"`
}

type storageStateOrigin struct {
	Origin       string                         `json:"origin"`
	LocalStorage []storageStateLocalStorageItem `json:"localStorage"`
}

type storageStateLocalStorageItem struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type storageStateStats struct {
	CookieCount       int `json:"cookieCount"`
	LocalStorageCount int `json:"localStorageCount"`
	OriginCount       int `json:"originCount"`
}

// browserProfile represents the browser profile settings for anti-detection and behavior.
type browserProfile struct {
	Preset        string                    `json:"preset,omitempty"`
	Behavior      *browserProfileBehavior   `json:"behavior,omitempty"`
	AntiDetection *browserProfileAntiDetect `json:"anti_detection,omitempty"`
}

type browserProfileBehavior struct {
	MouseMovementStyle string `json:"mouse_movement_style,omitempty"`
	ScrollStyle        string `json:"scroll_style,omitempty"`
	TypingDelayMin     int    `json:"typing_delay_min,omitempty"`
	TypingDelayMax     int    `json:"typing_delay_max,omitempty"`
	MicroPauseEnabled  bool   `json:"micro_pause_enabled,omitempty"`
}

type browserProfileAntiDetect struct {
	DisableAutomationControlled bool   `json:"disable_automation_controlled,omitempty"`
	DisableWebRTC               bool   `json:"disable_webrtc,omitempty"`
	PatchNavigatorWebdriver     bool   `json:"patch_navigator_webdriver,omitempty"`
	HeadlessDetectionBypass     bool   `json:"headless_detection_bypass,omitempty"`
	AdBlockingMode              string `json:"ad_blocking_mode,omitempty"`
}

// parseBrowserProfile parses the raw JSON browser profile into a structured type.
func parseBrowserProfile(raw json.RawMessage) *browserProfile {
	if len(raw) == 0 {
		return nil
	}
	var bp browserProfile
	if err := json.Unmarshal(raw, &bp); err != nil {
		return nil
	}
	return &bp
}

func listProfiles(ctx *appctx.Context) ([]sessionProfile, []byte, error) {
	body, err := ctx.Core.APIClient.Get(ctx.APIPath("/recordings/sessions"), nil)
	if err != nil {
		return nil, nil, err
	}
	var parsed listResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, body, err
	}
	return parsed.Profiles, body, nil
}

func createProfile(ctx *appctx.Context, name string) (sessionProfile, []byte, error) {
	payload := map[string]string{"name": name}
	body, err := ctx.Core.APIClient.Request("POST", ctx.APIPath("/recordings/sessions"), nil, payload)
	if err != nil {
		return sessionProfile{}, nil, err
	}
	var profile sessionProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		return sessionProfile{}, body, err
	}
	return profile, body, nil
}

func deleteProfile(ctx *appctx.Context, profileID string) error {
	_, err := ctx.Core.APIClient.Request("DELETE", ctx.APIPath("/recordings/sessions/"+profileID), url.Values{}, nil)
	return err
}

func getStorageState(ctx *appctx.Context, profileID string) (*storageStateResponse, []byte, error) {
	body, err := ctx.Core.APIClient.Get(ctx.APIPath("/recordings/sessions/"+profileID+"/storage"), nil)
	if err != nil {
		return nil, nil, err
	}
	var resp storageStateResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, body, err
	}
	return &resp, body, nil
}

func clearStorage(ctx *appctx.Context, profileID string) error {
	_, err := ctx.Core.APIClient.Request("DELETE", ctx.APIPath("/recordings/sessions/"+profileID+"/storage"), url.Values{}, nil)
	return err
}

func renameProfile(ctx *appctx.Context, profileID, newName string) (sessionProfile, []byte, error) {
	payload := map[string]string{"name": newName}
	body, err := ctx.Core.APIClient.Request("PATCH", ctx.APIPath("/recordings/sessions/"+profileID), nil, payload)
	if err != nil {
		return sessionProfile{}, nil, err
	}
	var profile sessionProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		return sessionProfile{}, body, err
	}
	return profile, body, nil
}

// countProfilesWithName returns the number of profiles with the given name.
func countProfilesWithName(ctx *appctx.Context, name string) (int, error) {
	if strings.TrimSpace(name) == "" {
		return 0, nil
	}
	profiles, _, err := listProfiles(ctx)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, p := range profiles {
		if p.Name == name {
			count++
		}
	}
	return count, nil
}

// resolveProfileID resolves a profile identifier (ID or name) to a profile ID.
// Returns the profile and an error if not found or ambiguous.
func resolveProfileID(ctx *appctx.Context, identifier string) (sessionProfile, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return sessionProfile{}, fmt.Errorf("profile identifier is required")
	}

	profiles, _, err := listProfiles(ctx)
	if err != nil {
		return sessionProfile{}, fmt.Errorf("failed to list profiles: %w", err)
	}

	// First, try exact ID match
	for _, p := range profiles {
		if p.ID == identifier {
			return p, nil
		}
	}

	// Then try exact name match
	var matches []sessionProfile
	for _, p := range profiles {
		if p.Name == identifier {
			matches = append(matches, p)
		}
	}

	if len(matches) == 0 {
		return sessionProfile{}, fmt.Errorf("session profile not found: %s", identifier)
	}
	if len(matches) > 1 {
		return sessionProfile{}, fmt.Errorf("ambiguous profile name '%s': %d profiles match. Use profile ID instead", identifier, len(matches))
	}

	return matches[0], nil
}
