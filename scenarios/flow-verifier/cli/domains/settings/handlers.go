package settings

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	settingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/settings"
	settingsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/settings/settings_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client settingsconnect.SettingsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{core: core, client: settingsconnect.NewSettingsServiceClient(httpClient, baseURL)}
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	resp, err := h.client.GetSettings(context.Background(), connect.NewRequest(&settingsv1.GetSettingsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get settings", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{"Current preferences"},
		ResultsHeading: "Settings",
		Results:        formatSettings(resp.Msg.Settings),
	})
}

func (h *handlers) set(ctx cliapp.RunContext) error {
	pairs := ctx.Positionals("pair")
	if len(pairs) == 0 {
		return errors.New("settings set requires one or more <key>=<value> pairs")
	}
	settings, mask, err := buildPatch(pairs)
	if err != nil {
		return err
	}
	resp, err := h.client.UpdateSettings(context.Background(), connect.NewRequest(&settingsv1.UpdateSettingsRequest{
		Settings:   settings,
		UpdateMask: mask,
	}))
	if err != nil {
		return cliapp.WrapAPIError("update settings", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{"Settings updated."},
		Changes: formatSettings(resp.Msg.Settings),
	})
}

// buildPatch translates a list of "key=value" pairs into a (Settings, FieldMask).
// The mask carries only the keys named on the CLI; the Settings message
// holds the parsed enum/scalar values for those keys.
func buildPatch(pairs []string) (*settingsv1.Settings, *fieldmaskpb.FieldMask, error) {
	out := &settingsv1.Settings{}
	mask := &fieldmaskpb.FieldMask{}
	for _, pair := range pairs {
		i := strings.IndexByte(pair, '=')
		if i < 0 {
			return nil, nil, fmt.Errorf("invalid assignment %q (expected key=value)", pair)
		}
		key, value := pair[:i], pair[i+1:]
		field, err := applyKV(out, key, value)
		if err != nil {
			return nil, nil, err
		}
		mask.Paths = append(mask.Paths, field)
	}
	return out, mask, nil
}

func applyKV(s *settingsv1.Settings, key, value string) (string, error) {
	switch key {
	case "theme":
		t, err := parseTheme(value)
		if err != nil {
			return "", err
		}
		s.Theme = t
		return "theme", nil
	case "fontScale":
		f, err := parseFontScale(value)
		if err != nil {
			return "", err
		}
		s.FontScale = f
		return "font_scale", nil
	case "density":
		d, err := parseDensity(value)
		if err != nil {
			return "", err
		}
		s.Density = d
		return "density", nil
	case "reducedMotion":
		b, err := parseBool(value)
		if err != nil {
			return "", fmt.Errorf("reducedMotion: %v", err)
		}
		s.ReducedMotion = b
		return "reduced_motion", nil
	case "rtl":
		b, err := parseBool(value)
		if err != nil {
			return "", fmt.Errorf("rtl: %v", err)
		}
		s.Rtl = b
		return "rtl", nil
	case "defaultRoot":
		s.DefaultRoot = value
		return "default_root", nil
	case "sidebarWidth":
		n, err := strconv.Atoi(value)
		if err != nil {
			return "", fmt.Errorf("sidebarWidth: must be an integer; got %q", value)
		}
		s.SidebarWidth = int32(n)
		return "sidebar_width", nil
	}
	return "", fmt.Errorf("unknown setting %q; allowed keys: %s", key, strings.Join(allowedKeys(), ", "))
}

func allowedKeys() []string {
	keys := []string{"defaultRoot", "density", "fontScale", "reducedMotion", "rtl", "sidebarWidth", "theme"}
	sort.Strings(keys)
	return keys
}

func parseTheme(v string) (settingsv1.Theme, error) {
	switch v {
	case "light":
		return settingsv1.Theme_THEME_LIGHT, nil
	case "dark":
		return settingsv1.Theme_THEME_DARK, nil
	case "system":
		return settingsv1.Theme_THEME_SYSTEM, nil
	}
	return 0, fmt.Errorf("theme: must be one of light|dark|system; got %q", v)
}

func parseFontScale(v string) (settingsv1.FontScale, error) {
	switch v {
	case "sm":
		return settingsv1.FontScale_FONT_SCALE_SM, nil
	case "md":
		return settingsv1.FontScale_FONT_SCALE_MD, nil
	case "lg":
		return settingsv1.FontScale_FONT_SCALE_LG, nil
	}
	return 0, fmt.Errorf("fontScale: must be one of sm|md|lg; got %q", v)
}

func parseDensity(v string) (settingsv1.Density, error) {
	switch v {
	case "comfortable":
		return settingsv1.Density_DENSITY_COMFORTABLE, nil
	case "compact":
		return settingsv1.Density_DENSITY_COMPACT, nil
	}
	return 0, fmt.Errorf("density: must be one of comfortable|compact; got %q", v)
}

func parseBool(v string) (bool, error) {
	switch v {
	case "true", "1", "yes":
		return true, nil
	case "false", "0", "no":
		return false, nil
	}
	return false, fmt.Errorf("must be one of true|false; got %q", v)
}

func formatSettings(s *settingsv1.Settings) []string {
	if s == nil {
		return []string{"(no settings)"}
	}
	return []string{
		fmt.Sprintf("theme        = %s", themeText(s.Theme)),
		fmt.Sprintf("fontScale    = %s", fontScaleText(s.FontScale)),
		fmt.Sprintf("density      = %s", densityText(s.Density)),
		fmt.Sprintf("reducedMotion= %t", s.ReducedMotion),
		fmt.Sprintf("rtl          = %t", s.Rtl),
		fmt.Sprintf("defaultRoot  = %s", s.DefaultRoot),
		fmt.Sprintf("sidebarWidth = %d", s.SidebarWidth),
	}
}

func themeText(t settingsv1.Theme) string {
	switch t {
	case settingsv1.Theme_THEME_LIGHT:
		return "light"
	case settingsv1.Theme_THEME_DARK:
		return "dark"
	case settingsv1.Theme_THEME_SYSTEM:
		return "system"
	}
	return "unspecified"
}

func fontScaleText(f settingsv1.FontScale) string {
	switch f {
	case settingsv1.FontScale_FONT_SCALE_SM:
		return "sm"
	case settingsv1.FontScale_FONT_SCALE_MD:
		return "md"
	case settingsv1.FontScale_FONT_SCALE_LG:
		return "lg"
	}
	return "unspecified"
}

func densityText(d settingsv1.Density) string {
	switch d {
	case settingsv1.Density_DENSITY_COMFORTABLE:
		return "comfortable"
	case settingsv1.Density_DENSITY_COMPACT:
		return "compact"
	}
	return "unspecified"
}
