// Package desktopcompanion contains the user-session packaging contract for
// the Device Control companion. Bridge onboarding supplies the artifact and
// invokes this renderer; the package never installs a root daemon.
package desktopcompanion

import (
	"encoding/xml"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

var ErrInvalidSpec = errors.New("invalid desktop companion specification")

type Spec struct {
	Label    string
	Program  string
	Config   string
	UserHome string
}

type plist struct {
	XMLName xml.Name  `xml:"plist"`
	Version string    `xml:"version,attr"`
	Dict    plistDict `xml:"dict"`
}
type plistDict struct {
	Keys   []string      `xml:"key"`
	Values []interface{} `xml:"-"`
}

// RenderLaunchAgent returns a deterministic, user-context LaunchAgent. It
// intentionally contains no privileged user, SSH credential, or node URL.
func RenderLaunchAgent(spec Spec) ([]byte, error) {
	if !validLabel(spec.Label) || strings.TrimSpace(spec.Program) == "" || !filepath.IsAbs(spec.Program) || strings.TrimSpace(spec.Config) == "" || !filepath.IsAbs(spec.Config) || strings.TrimSpace(spec.UserHome) == "" || !filepath.IsAbs(spec.UserHome) || filepath.Clean(spec.UserHome) == string(filepath.Separator) {
		return nil, ErrInvalidSpec
	}
	content := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>Label</key><string>%s</string>
<key>ProgramArguments</key><array><string>%s</string><string>--config</string><string>%s</string></array>
<key>RunAtLoad</key><true/><key>KeepAlive</key><true/>
</dict></plist>
`, xmlEscape(spec.Label), xmlEscape(spec.Program), xmlEscape(spec.Config))
	return []byte(content), nil
}

func Path(spec Spec) (string, error) {
	if !validLabel(spec.Label) || strings.TrimSpace(spec.UserHome) == "" || !filepath.IsAbs(spec.UserHome) || filepath.Clean(spec.UserHome) == string(filepath.Separator) {
		return "", ErrInvalidSpec
	}
	return filepath.Join(spec.UserHome, "Library", "LaunchAgents", spec.Label+".plist"), nil
}

func validLabel(label string) bool {
	label = strings.TrimSpace(label)
	if label == "" || label == "." || label == ".." || filepath.Base(label) != label {
		return false
	}
	for _, char := range label {
		if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') && char != '.' && char != '-' && char != '_' {
			return false
		}
	}
	return true
}
func xmlEscape(value string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(value))
	return b.String()
}
