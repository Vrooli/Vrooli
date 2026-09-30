package settings

import (
	"context"
	"log"
	"time"

	"flow-verifier/internal/settings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	settingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/settings"
)

// Deps wires the seams the settings Connect handler needs.
type Deps struct {
	Service *settings.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) GetSettings(ctx context.Context, _ *connect.Request[settingsv1.GetSettingsRequest]) (*connect.Response[settingsv1.GetSettingsResponse], error) {
	got, err := h.deps.Service.Get(ctx)
	if err != nil {
		h.deps.Logger.Printf("settings.GetSettings: %v", err)
		return nil, settings.ToConnectError(err)
	}
	return connect.NewResponse(&settingsv1.GetSettingsResponse{Settings: domainToProto(got)}), nil
}

func (h *connectHandler) UpdateSettings(ctx context.Context, req *connect.Request[settingsv1.UpdateSettingsRequest]) (*connect.Response[settingsv1.UpdateSettingsResponse], error) {
	if req.Msg == nil || req.Msg.Settings == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errEmptyBody)
	}
	patch := buildPatch(req.Msg.Settings, req.Msg.UpdateMask)
	updated, err := h.deps.Service.Upsert(ctx, patch)
	if err != nil {
		connectErr := settings.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("settings.UpdateSettings: %v", err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&settingsv1.UpdateSettingsResponse{Settings: domainToProto(updated)}), nil
}

// errEmptyBody is the sentinel returned when the client sends an
// UpdateSettings request without a Settings message. Modelled as a
// package value so the test can assert on it via errors.Is/As if
// needed.
var errEmptyBody = &validationStub{Field: "settings", Message: "required"}

// validationStub adapts the empty-body case to settings.ValidationError
// shape so the connect error envelope carries the same code and message
// as other invalid_argument failures. Kept local to the connect handler;
// the SQLite layer never sees it.
type validationStub struct{ Field, Message string }

func (e *validationStub) Error() string { return "settings: " + e.Field + ": " + e.Message }

// buildPatch translates the proto request into the internal settings.Patch.
// FieldMask determines which fields are merged; empty mask means "every
// non-default field on the request is included," matching the template's
// permissive default.
func buildPatch(s *settingsv1.Settings, mask *fieldmaskpb.FieldMask) settings.Patch {
	include := func(string) bool { return true }
	if mask != nil && len(mask.Paths) > 0 {
		set := make(map[string]struct{}, len(mask.Paths))
		for _, p := range mask.Paths {
			set[p] = struct{}{}
		}
		include = func(field string) bool { _, ok := set[field]; return ok }
	}
	var p settings.Patch
	if include("theme") {
		t := protoToTheme(s.Theme)
		p.Theme = &t
	}
	if include("font_scale") {
		f := protoToFontScale(s.FontScale)
		p.FontScale = &f
	}
	if include("reduced_motion") {
		v := s.ReducedMotion
		p.ReducedMotion = &v
	}
	if include("rtl") {
		v := s.Rtl
		p.RTL = &v
	}
	if include("default_root") {
		v := s.DefaultRoot
		p.DefaultRoot = &v
	}
	if include("density") {
		d := protoToDensity(s.Density)
		p.Density = &d
	}
	if include("sidebar_width") {
		v := int(s.SidebarWidth)
		p.SidebarWidth = &v
	}
	if include("inventory_filters") {
		f := protoToInventoryFilters(s.InventoryFilters)
		p.InventoryFilters = &f
	}
	return p
}

func domainToProto(s settings.Settings) *settingsv1.Settings {
	return &settingsv1.Settings{
		PrincipalId:      s.PrincipalID,
		Theme:            themeToProto(s.Theme),
		FontScale:        fontScaleToProto(s.FontScale),
		ReducedMotion:    s.ReducedMotion,
		Rtl:              s.RTL,
		DefaultRoot:      s.DefaultRoot,
		Density:          densityToProto(s.Density),
		SidebarWidth:     int32(s.SidebarWidth),
		InventoryFilters: inventoryFiltersToProto(s.InventoryFilters),
		UpdatedAt:        s.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func themeToProto(t settings.Theme) settingsv1.Theme {
	switch t {
	case settings.ThemeLight:
		return settingsv1.Theme_THEME_LIGHT
	case settings.ThemeDark:
		return settingsv1.Theme_THEME_DARK
	case settings.ThemeSystem:
		return settingsv1.Theme_THEME_SYSTEM
	}
	return settingsv1.Theme_THEME_UNSPECIFIED
}

func protoToTheme(t settingsv1.Theme) settings.Theme {
	switch t {
	case settingsv1.Theme_THEME_LIGHT:
		return settings.ThemeLight
	case settingsv1.Theme_THEME_DARK:
		return settings.ThemeDark
	case settingsv1.Theme_THEME_SYSTEM:
		return settings.ThemeSystem
	}
	return ""
}

func fontScaleToProto(f settings.FontScale) settingsv1.FontScale {
	switch f {
	case settings.FontScaleSm:
		return settingsv1.FontScale_FONT_SCALE_SM
	case settings.FontScaleMd:
		return settingsv1.FontScale_FONT_SCALE_MD
	case settings.FontScaleLg:
		return settingsv1.FontScale_FONT_SCALE_LG
	}
	return settingsv1.FontScale_FONT_SCALE_UNSPECIFIED
}

func protoToFontScale(f settingsv1.FontScale) settings.FontScale {
	switch f {
	case settingsv1.FontScale_FONT_SCALE_SM:
		return settings.FontScaleSm
	case settingsv1.FontScale_FONT_SCALE_MD:
		return settings.FontScaleMd
	case settingsv1.FontScale_FONT_SCALE_LG:
		return settings.FontScaleLg
	}
	return ""
}

func densityToProto(d settings.Density) settingsv1.Density {
	switch d {
	case settings.DensityComfortable:
		return settingsv1.Density_DENSITY_COMFORTABLE
	case settings.DensityCompact:
		return settingsv1.Density_DENSITY_COMPACT
	}
	return settingsv1.Density_DENSITY_UNSPECIFIED
}

func protoToDensity(d settingsv1.Density) settings.Density {
	switch d {
	case settingsv1.Density_DENSITY_COMFORTABLE:
		return settings.DensityComfortable
	case settingsv1.Density_DENSITY_COMPACT:
		return settings.DensityCompact
	}
	return ""
}

func inventoryFiltersToProto(f settings.InventoryFilters) *settingsv1.InventoryFilters {
	status := append([]string(nil), f.Status...)
	return &settingsv1.InventoryFilters{
		Search:   f.Search,
		Language: f.Language,
		Status:   status,
		Sort: &settingsv1.InventorySortOrder{
			Key: f.Sort.Key,
			Dir: f.Sort.Dir,
		},
	}
}

func protoToInventoryFilters(f *settingsv1.InventoryFilters) settings.InventoryFilters {
	if f == nil {
		return settings.InventoryFilters{}
	}
	var sort settings.InventorySortOrder
	if f.Sort != nil {
		sort = settings.InventorySortOrder{Key: f.Sort.Key, Dir: f.Sort.Dir}
	}
	status := append([]string(nil), f.Status...)
	return settings.InventoryFilters{
		Search:   f.Search,
		Language: f.Language,
		Status:   status,
		Sort:     sort,
	}
}
