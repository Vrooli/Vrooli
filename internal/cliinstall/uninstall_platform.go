package cliinstall

import (
	"encoding/hex"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vrooli/vrooli/internal/repocontractmeta"
	"github.com/vrooli/vrooli/internal/shell"
)

const (
	uninstallPlatformParameterA = 32
	uninstallPlatformParameterB = 36
)

func isPlanID(id string) bool {
	id = strings.TrimSpace(id)
	if len(id) == uninstallPlatformParameterA {
		_, err := hex.DecodeString(id)
		return err == nil
	}
	if len(id) != uninstallPlatformParameterB {
		return false
	}
	for index, r := range id {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			if r != '-' {
				return false
			}
			continue
		}
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}

func mustInstallRecordPath(home string) string {
	path, err := InstallRecordPath(home)
	if err != nil {
		return filepath.Join(filepath.Clean(home), repocontractmeta.ProjectConfigDir, "state", "install-record.json")
	}
	return path
}

func stopRecordedService(entry InstallEntry) error {
	manager := strings.ToLower(strings.TrimSpace(entry.ServiceManager))
	name := strings.TrimSpace(entry.ServiceName)
	switch {
	case manager == "systemd" || (manager == "" && runtime.GOOS == "linux"):
		if name == "" {
			name = filepath.Base(entry.Path)
		}
		args := []string{"disable", "--now", name}
		if strings.EqualFold(entry.ServiceDomain, "system") {
			return runNativeServiceCommand("systemctl", args...)
		}
		return runNativeServiceCommand("systemctl", append([]string{"--user"}, args...)...)
	case manager == "launchd" || (manager == "" && runtime.GOOS == "darwin"):
		if name == "" {
			name = strings.TrimSuffix(filepath.Base(entry.Path), filepath.Ext(entry.Path))
		}
		domain := strings.TrimSpace(entry.ServiceDomain)
		if domain == "" {
			domain = launchdDomainForPath(entry.Path)
		}
		return runNativeServiceCommand("launchctl", "bootout", domain+"/"+name)
	default:
		return nil
	}
}

func launchdDomainForPath(path string) string {
	clean := filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
	if clean == "/Library/LaunchDaemons" || strings.HasPrefix(clean, "/Library/LaunchDaemons/") {
		return "system"
	}
	return "gui/" + currentUserID()
}

func currentUserID() string {
	if uid := strings.TrimSpace(os.Getenv("UID")); uid != "" {
		return uid
	}
	if current, err := user.Current(); err == nil && strings.TrimSpace(current.Uid) != "" {
		return strings.TrimSpace(current.Uid)
	}
	return "0"
}

func runNativeServiceCommand(name string, args ...string) error {
	cmd := shell.NewCommand(name, args...)
	if output, err := cmd.CombinedOutput(); err != nil {

		if strings.Contains(strings.ToLower(string(output)), "not found") || strings.Contains(strings.ToLower(string(output)), "no such process") {
			return nil
		}
		return fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}
