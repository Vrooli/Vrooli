package resources

import (
	"errors"
	"path/filepath"
	"strings"
)

// OpenRGB's ResourceManager constructor accesses settings before CLI --config
// processing. Keep XDG private everywhere and APPDATA private on native
// profiles. The qualified pre-native Linux contract omitted APPDATA; when
// explicitly supplied it must still name the same private root.
func verifyOpenRGBConstructorEnvironment(goos, configDir string, env map[string]string) error {
	if goos != "linux" && goos != "darwin" && goos != "windows" {
		return errOpenRGBReleasePayloadUnqualified
	}
	if !filepath.IsAbs(configDir) || filepath.Clean(configDir) != configDir || configDir == string(filepath.Separator) || strings.ContainsAny(configDir, "\x00\r\n") {
		return errors.New("OpenRGB constructor requires private canonical configuration root")
	}
	if !openRGBConstructorEnvironmentIsolated(goos, configDir, env) {
		return errors.New("OpenRGB constructors require isolated platform configuration with offscreen Qt")
	}
	return nil
}

// The manifest and resolved invocation share one environment policy. Only
// qualified legacy Linux profiles may omit APPDATA; explicit ambient values
// and missing native-platform isolation remain invalid.
func openRGBConstructorEnvironmentIsolated(goos, configDir string, env map[string]string) bool {
	switch goos {
	case "linux", "darwin", "windows":
	default:
		return false
	}
	if env["XDG_CONFIG_HOME"] != configDir || env["QT_QPA_PLATFORM"] != "offscreen" {
		return false
	}
	appdata, present := env["APPDATA"]
	if present {
		return appdata == configDir
	}
	return goos == "linux"
}
