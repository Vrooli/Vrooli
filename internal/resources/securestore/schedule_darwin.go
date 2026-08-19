//go:build darwin

package securestore

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const nativeScheduleProvider = "launchd-user"
const nativeScheduleSupported = true
const credentialCopyLaunchLabel = "com.vrooli.credential-store-copy"

func installNativeCopySchedule(executable string, interval time.Duration, enabled bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home for credential-store copy schedule: %w", err)
	}
	path := filepath.Join(home, "Library", "LaunchAgents", credentialCopyLaunchLabel+".plist")
	domain := fmt.Sprintf("gui/%d", os.Getuid())
	target := domain + "/" + credentialCopyLaunchLabel
	if !enabled {
		_ = exec.Command("launchctl", "bootout", target).Run()
		_ = os.Remove(path)
		return nil
	}
	seconds := int64(interval / time.Second)
	if seconds < 60 {
		seconds = 60
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create credential-store copy launch-agent directory: %w", err)
	}
	escape := func(value string) string {
		return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;", "'", "&apos;").Replace(value)
	}
	content := fmt.Sprintf("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<!DOCTYPE plist PUBLIC \"-//Apple//DTD PLIST 1.0//EN\" \"http://www.apple.com/DTDs/PropertyList-1.0.dtd\">\n<plist version=\"1.0\"><dict>\n  <key>Label</key><string>%s</string>\n  <key>ProgramArguments</key><array><string>%s</string><string>credentials</string><string>store</string><string>copy</string><string>scheduled</string><string>--format</string><string>json</string></array>\n  <key>RunAtLoad</key><true/>\n  <key>StartInterval</key><integer>%d</integer>\n</dict></plist>\n", credentialCopyLaunchLabel, escape(executable), seconds)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("write credential-store copy launch agent: %w", err)
	}
	_ = exec.Command("launchctl", "bootout", target).Run()
	if output, err := exec.Command("launchctl", "bootstrap", domain, path).CombinedOutput(); err != nil {
		return fmt.Errorf("enable credential-store copy launch agent: %w: %s", err, output)
	}
	return nil
}
