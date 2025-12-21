package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strings"

	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/database"
	exportservices "github.com/vrooli/browser-automation-studio/services/export"
)

const replayConfigSettingsKey = "replay_config.v1"

// ReplayConfigRequest captures a persisted replay configuration payload.
type ReplayConfigRequest struct {
	Config map[string]any `json:"config"`
}

// ReplayConfigResponse returns the persisted replay configuration payload.
type ReplayConfigResponse struct {
	Config map[string]any `json:"config"`
}

// GetReplayConfig handles GET /api/v1/replay-config
func (h *Handler) GetReplayConfig(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	config, err := h.loadReplayConfig(ctx)
	if err != nil {
		h.log.WithError(err).Error("Failed to load replay config")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_replay_config"}))
		return
	}

	h.respondSuccess(w, http.StatusOK, ReplayConfigResponse{Config: config})
}

// PutReplayConfig handles PUT /api/v1/replay-config
func (h *Handler) PutReplayConfig(w http.ResponseWriter, r *http.Request) {
	var req ReplayConfigRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		h.log.WithError(err).Error("Failed to decode replay config request")
		h.respondError(w, ErrInvalidRequest)
		return
	}

	if req.Config == nil {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "config"}))
		return
	}

	payload, err := json.Marshal(req.Config)
	if err != nil {
		h.log.WithError(err).Error("Failed to marshal replay config")
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"field": "config"}))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	if err := h.repo.SetSetting(ctx, replayConfigSettingsKey, string(payload)); err != nil {
		h.log.WithError(err).Error("Failed to persist replay config")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "set_replay_config"}))
		return
	}

	h.respondSuccess(w, http.StatusOK, ReplayConfigResponse{Config: req.Config})
}

// DeleteReplayConfig handles DELETE /api/v1/replay-config
func (h *Handler) DeleteReplayConfig(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	if err := h.repo.DeleteSetting(ctx, replayConfigSettingsKey); err != nil {
		h.log.WithError(err).Error("Failed to delete replay config")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "delete_replay_config"}))
		return
	}

	h.respondSuccess(w, http.StatusOK, ReplayConfigResponse{Config: map[string]any{}})
}

func (h *Handler) loadReplayConfig(ctx context.Context) (map[string]any, error) {
	value, err := h.repo.GetSetting(ctx, replayConfigSettingsKey)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return map[string]any{}, nil
		}
		return nil, err
	}

	if value == "" {
		return map[string]any{}, nil
	}

	var config map[string]any
	if err := json.Unmarshal([]byte(value), &config); err != nil {
		return nil, err
	}

	if config == nil {
		return map[string]any{}, nil
	}

	return config, nil
}

func (h *Handler) replayConfigOverrides(ctx context.Context) *executionExportOverrides {
	config, err := h.loadReplayConfig(ctx)
	if err != nil {
		if h.log != nil {
			h.log.WithError(err).Warn("Failed to load replay config for export overrides")
		}
		return nil
	}
	return replayConfigToOverrides(config)
}

func replayConfigToOverrides(config map[string]any) *executionExportOverrides {
	if config == nil {
		return nil
	}

	chromeTheme := firstString(config, "chromeTheme", "replayChromeTheme")
	backgroundTheme := firstString(config, "backgroundTheme", "replayBackgroundTheme")
	cursorTheme := firstString(config, "cursorTheme", "replayCursorTheme")
	cursorInitial := firstString(config, "cursorInitialPosition", "replayCursorInitialPosition")
	clickAnimation := firstString(config, "cursorClickAnimation", "replayCursorClickAnimation")
	cursorScale, hasScale := firstFloat(config, "cursorScale", "replayCursorScale")

	var themePreset *themePresetOverride
	if chromeTheme != "" || backgroundTheme != "" {
		themePreset = &themePresetOverride{
			ChromeTheme:     chromeTheme,
			BackgroundTheme: backgroundTheme,
		}
	}

	var cursorPreset *cursorPresetOverride
	if cursorTheme != "" || cursorInitial != "" || clickAnimation != "" || hasScale {
		cursorPreset = &cursorPresetOverride{
			Theme:           cursorTheme,
			InitialPosition: cursorInitial,
			ClickAnimation:  clickAnimation,
		}
		if hasScale && cursorScale > 0 {
			cursorPreset.Scale = cursorScale
		}
	}

	if themePreset == nil && cursorPreset == nil {
		return nil
	}

	return &executionExportOverrides{
		ThemePreset:  themePreset,
		CursorPreset: cursorPreset,
	}
}

func applyReplayConfigToSpec(spec *exportservices.ReplayMovieSpec, config map[string]any) {
	if spec == nil || config == nil {
		return
	}

	if speed := firstString(config, "cursorSpeedProfile", "cursor_speed_profile"); speed != "" {
		spec.CursorMotion.SpeedProfile = speed
	}
	if pathStyle := firstString(config, "cursorPathStyle", "cursor_path_style"); pathStyle != "" {
		spec.CursorMotion.PathStyle = pathStyle
	}

	if watermark := mapWatermark(config["watermark"]); watermark != nil {
		spec.Watermark = watermark
	}
	if intro := mapIntroCard(config["introCard"]); intro != nil {
		spec.IntroCard = intro
	}
	if outro := mapOutroCard(config["outroCard"]); outro != nil {
		spec.OutroCard = outro
	}
}

func firstString(config map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := config[key]; ok {
			if str, ok := value.(string); ok {
				if trimmed := strings.TrimSpace(str); trimmed != "" {
					return trimmed
				}
			}
		}
	}
	return ""
}

func firstFloat(config map[string]any, keys ...string) (float64, bool) {
	for _, key := range keys {
		if value, ok := config[key]; ok {
			switch v := value.(type) {
			case float64:
				if isFiniteFloat(v) {
					return v, true
				}
			case float32:
				f := float64(v)
				if isFiniteFloat(f) {
					return f, true
				}
			case int:
				return float64(v), true
			case int64:
				return float64(v), true
			case int32:
				return float64(v), true
			}
		}
	}
	return 0, false
}

func isFiniteFloat(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func mapWatermark(value any) *exportservices.ExportWatermark {
	obj, ok := value.(map[string]any)
	if !ok || obj == nil {
		return nil
	}
	settings := &exportservices.ExportWatermark{
		Enabled:  toBool(obj["enabled"]),
		AssetID:  firstString(obj, "assetId", "asset_id"),
		Position: firstString(obj, "position"),
		Size:     toInt(obj["size"]),
		Opacity:  toInt(obj["opacity"]),
		Margin:   toInt(obj["margin"]),
	}
	if !settings.Enabled && settings.AssetID == "" && settings.Position == "" && settings.Size == 0 &&
		settings.Opacity == 0 && settings.Margin == 0 {
		return nil
	}
	return settings
}

func mapIntroCard(value any) *exportservices.ExportIntroCard {
	obj, ok := value.(map[string]any)
	if !ok || obj == nil {
		return nil
	}
	settings := &exportservices.ExportIntroCard{
		Enabled:           toBool(obj["enabled"]),
		Title:             firstString(obj, "title"),
		Subtitle:          firstString(obj, "subtitle"),
		LogoAssetID:       firstString(obj, "logoAssetId", "logo_asset_id"),
		BackgroundAssetID: firstString(obj, "backgroundAssetId", "background_asset_id"),
		BackgroundColor:   firstString(obj, "backgroundColor", "background_color"),
		TextColor:         firstString(obj, "textColor", "text_color"),
		DurationMs:        toInt(obj["duration"]),
	}
	if settings.DurationMs == 0 {
		settings.DurationMs = toInt(obj["duration_ms"])
	}
	if !settings.Enabled && settings.Title == "" && settings.Subtitle == "" && settings.LogoAssetID == "" &&
		settings.BackgroundAssetID == "" && settings.BackgroundColor == "" && settings.TextColor == "" &&
		settings.DurationMs == 0 {
		return nil
	}
	return settings
}

func mapOutroCard(value any) *exportservices.ExportOutroCard {
	obj, ok := value.(map[string]any)
	if !ok || obj == nil {
		return nil
	}
	settings := &exportservices.ExportOutroCard{
		Enabled:           toBool(obj["enabled"]),
		Title:             firstString(obj, "title"),
		CtaText:           firstString(obj, "ctaText", "cta_text"),
		CtaURL:            firstString(obj, "ctaUrl", "cta_url"),
		LogoAssetID:       firstString(obj, "logoAssetId", "logo_asset_id"),
		BackgroundAssetID: firstString(obj, "backgroundAssetId", "background_asset_id"),
		BackgroundColor:   firstString(obj, "backgroundColor", "background_color"),
		TextColor:         firstString(obj, "textColor", "text_color"),
		DurationMs:        toInt(obj["duration"]),
	}
	if settings.DurationMs == 0 {
		settings.DurationMs = toInt(obj["duration_ms"])
	}
	if !settings.Enabled && settings.Title == "" && settings.CtaText == "" && settings.CtaURL == "" &&
		settings.LogoAssetID == "" && settings.BackgroundAssetID == "" && settings.BackgroundColor == "" &&
		settings.TextColor == "" && settings.DurationMs == 0 {
		return nil
	}
	return settings
}

func toBool(value any) bool {
	if v, ok := value.(bool); ok {
		return v
	}
	return false
}

func toInt(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float32:
		if isFiniteFloat(float64(v)) {
			return int(v)
		}
	case float64:
		if isFiniteFloat(v) {
			return int(v)
		}
	}
	return 0
}
