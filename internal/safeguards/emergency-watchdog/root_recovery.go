package emergencywatchdog

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/artifactlock"
	"github.com/vrooli/vrooli/internal/coreactivation"
	"github.com/vrooli/vrooli/internal/hostreqkit"
)

// AcquirePublicationLock serializes the recovery image and its scheduler with
// root admission. The lock proves exclusion, not operator authorization.
func AcquirePublicationLock(ctx context.Context, home string) (func(), error) {
	p := resolvePaths(home)
	return artifactlock.AcquireWithPathContext(ctx, "watchdog-binary:"+p.Binary, func(string) (string, error) { return p.Binary + ".lock", nil })
}

type recoveryStateOutput struct {
	data     bytes.Buffer
	overflow bool
}

func (o *recoveryStateOutput) Write(p []byte) (int, error) {
	n := len(p)
	remaining := (16 << 10) - o.data.Len()
	if len(p) > remaining {
		o.overflow = true
		p = p[:remaining]
	}
	_, _ = o.data.Write(p)
	return n, nil
}

func recoveryNativeProperties(ctx context.Context, unit string) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	name, args := hostreqkit.InvokingUserCommand("systemctl", "--user", "show", "--property=LoadState,ActiveState,FragmentPath,DropInPaths,NeedDaemonReload,UnitFileState", "--", unit)
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.WaitDelay = time.Second
	var output recoveryStateOutput
	cmd.Stdout = &output
	if err := cmd.Run(); err != nil || output.overflow {
		return nil, fmt.Errorf("root recovery native scheduler observation unavailable")
	}
	values := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(output.data.String()), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, coreactivation.ErrConflict
		}
		if _, exists := values[key]; exists {
			return nil, coreactivation.ErrConflict
		}
		values[key] = value
	}
	for _, key := range []string{"LoadState", "ActiveState", "FragmentPath", "DropInPaths", "NeedDaemonReload", "UnitFileState"} {
		if _, ok := values[key]; !ok {
			return nil, coreactivation.ErrConflict
		}
	}
	return values, nil
}

// QualifyRootRecovery requires an installed, canary-verified recovery image,
// exact native definitions and an active enabled timer. Call under the shared
// publication lock and retain it until root admission commits its worker SHA.
func QualifyRootRecovery(ctx context.Context, home, workerSHA, workerBuild string) error {
	p := resolvePaths(home)
	canonical, err := coreactivation.InstalledWatchdogPath(home)
	if err != nil || p.Binary != canonical {
		return fmt.Errorf("watchdog installation differs from canonical runtime authority")
	}
	receipt, err := readWatchdogUpdateReceipt(p)
	if err != nil || receipt == nil || receipt.Phase != "verified" || receipt.CandidateSHA != workerSHA || receipt.CandidateStamp != workerBuild {
		return fmt.Errorf("root recovery executable lacks a verified publication receipt")
	}
	sha, err := coreactivation.ImageDigest(ctx, p.Binary)
	if err != nil || sha != workerSHA {
		return coreactivation.ErrConflict
	}
	artifact, err := rootRecoveryArtifact(p)
	if err != nil {
		return err
	}
	timer, _ := platform.CoreUnitByID(platform.CoreUnitRootUpdateRecoveryTimer)
	for _, file := range artifact.Files {
		path := filepath.Join(filepath.Dir(p.ServiceUnit), file.Name)
		if !hostreqkit.FileContentMatches(path, file.Content) {
			return fmt.Errorf("root recovery native definition is missing or stale")
		}
		properties, err := recoveryNativeProperties(ctx, file.Name)
		if err != nil {
			return err
		}
		if properties["LoadState"] != "loaded" || properties["FragmentPath"] != path || properties["DropInPaths"] != "" || properties["NeedDaemonReload"] != "no" {
			return fmt.Errorf("root recovery loaded definition does not match its frozen owner")
		}
		if file.Name == timer.Systemd && (properties["ActiveState"] != "active" || properties["UnitFileState"] != "enabled") {
			return fmt.Errorf("root recovery timer is not enabled and active")
		}
	}
	return nil
}
