//go:build linux

package securestore

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

const nativeScheduleProvider = "systemd-user"
const nativeScheduleSupported = true
const credentialCopyTimer = "vrooli-credential-store-copy.timer"

func installNativeCopySchedule(executable string, interval time.Duration, enabled bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home for credential-store copy schedule: %w", err)
	}
	unitDir := filepath.Join(home, ".config", "systemd", "user")
	servicePath := filepath.Join(unitDir, "vrooli-credential-store-copy.service")
	timerPath := filepath.Join(unitDir, credentialCopyTimer)
	if !enabled {
		_ = exec.Command("systemctl", "--user", "disable", "--now", credentialCopyTimer).Run()
		_ = os.Remove(servicePath)
		_ = os.Remove(timerPath)
		_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
		return nil
	}
	if interval <= 0 {
		return fmt.Errorf("credential-store copy schedule interval must be positive")
	}
	if err := os.MkdirAll(unitDir, 0o700); err != nil {
		return fmt.Errorf("create credential-store copy systemd directory: %w", err)
	}
	service := "[Unit]\nDescription=Vrooli encrypted credential-store copy\n\n[Service]\nType=oneshot\nExecStart=" + strconv.Quote(executable) + " credentials store copy scheduled --format json\n"
	timer := fmt.Sprintf("[Unit]\nDescription=Refresh Vrooli encrypted credential-store copy\n\n[Timer]\nOnBootSec=5m\nOnUnitActiveSec=%s\nPersistent=true\nUnit=vrooli-credential-store-copy.service\n\n[Install]\nWantedBy=timers.target\n", interval)
	if err := os.WriteFile(servicePath, []byte(service), 0o600); err != nil {
		return fmt.Errorf("write credential-store copy service: %w", err)
	}
	if err := os.WriteFile(timerPath, []byte(timer), 0o600); err != nil {
		return fmt.Errorf("write credential-store copy timer: %w", err)
	}
	if output, err := exec.Command("systemctl", "--user", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("reload credential-store copy timer: %w: %s", err, output)
	}
	if output, err := exec.Command("systemctl", "--user", "enable", "--now", credentialCopyTimer).CombinedOutput(); err != nil {
		return fmt.Errorf("enable credential-store copy timer: %w: %s", err, output)
	}
	return nil
}
