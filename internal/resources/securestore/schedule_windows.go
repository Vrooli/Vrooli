//go:build windows

package securestore

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const nativeScheduleProvider = "windows-task-scheduler-user"
const nativeScheduleSupported = true
const credentialCopyTask = "VrooliCredentialStoreCopy"

func installNativeCopySchedule(executable string, interval time.Duration, enabled bool) error {
	if !enabled {
		output, err := exec.Command("schtasks", "/Delete", "/TN", credentialCopyTask, "/F").CombinedOutput()
		if err != nil && !strings.Contains(strings.ToLower(string(output)), "does not exist") {
			return fmt.Errorf("remove credential-store copy task: %w: %s", err, output)
		}
		return nil
	}
	minutes := int64(interval / time.Minute)
	if minutes < 1 {
		minutes = 1
	}
	command := `"` + strings.ReplaceAll(executable, `"`, `\"`) + `" credentials store copy scheduled --format json`
	output, err := exec.Command("schtasks", "/Create", "/TN", credentialCopyTask, "/SC", "MINUTE", "/MO", strconv.FormatInt(minutes, 10), "/TR", command, "/F").CombinedOutput()
	if err != nil {
		return fmt.Errorf("enable credential-store copy task: %w: %s", err, output)
	}
	return nil
}
