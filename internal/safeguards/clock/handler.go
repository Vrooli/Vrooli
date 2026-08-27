package clock

import (
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const toleranceSeconds = 300 // 5 minutes

var timeSources = []string{
	"https://www.google.com",
	"https://www.cloudflare.com",
	"https://httpbin.org/get",
}

var syncMethods = []struct {
	Name    string
	Command string
	Args    []string
}{
	{"hwclock", "hwclock", []string{"-s"}},
	{"ntpdate", "ntpdate", []string{"-s", "time.nist.gov"}},
	{"timedatectl", "timedatectl", []string{"set-ntp", "true"}},
}

// HTTPHeadFn is swappable for testing.
var HTTPHeadFn = func(url string) (*http.Response, error) {
	client := &http.Client{Timeout: tuning.ServiceHealthTimeout}
	return client.Head(url)
}

// NowFn is swappable for testing.
var NowFn = time.Now

type handler struct {
	manifest hostreqkit.SafeguardManifest
}

func NewHandler(manifest hostreqkit.SafeguardManifest) hostreqkit.Handler {
	return handler{manifest: manifest}
}

func (h handler) Name() string           { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind { return hostreqspec.KindSafeguard }

func (h handler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.BaseStatus(requirement)
	status.SupportClass = hostreqkit.SupportSupported

	if requirement.Manual {
		status.SupportClass = hostreqkit.SupportManualOnly
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		return status
	}

	accurate, drift := checkClockAccuracy()
	if accurate {
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		if drift == 0 {
			status.Notes = append(status.Notes, "could not verify clock against remote time sources; assuming accurate")
		} else {
			status.Notes = append(status.Notes, fmt.Sprintf("system clock is accurate (drift: %.0fs)", math.Abs(drift)))
		}
		return status
	}

	status.Notes = append(status.Notes, fmt.Sprintf("system clock drift is %.0fs (tolerance: %ds)", math.Abs(drift), toleranceSeconds))

	if host.OS != "linux" {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "automatic clock sync is only supported on Linux")
		return status
	}

	return status
}

func (h handler) Apply(host hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	switch status.SupportClass {
	case hostreqkit.SupportUnsupported:
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		return status, nil
	case hostreqkit.SupportNotApplicable:
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		return status, nil
	case hostreqkit.SupportManualOnly:
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		status.Notes = append(status.Notes, "manual safeguard action required by manifest declaration")
		return status, nil
	}

	if status.Applied {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		return status, nil
	}

	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes, "dry-run: would attempt clock synchronization via hwclock/ntpdate/timedatectl")
		return status, nil
	}

	for _, method := range syncMethods {
		if _, err := hostreqkit.LookPathFn(method.Command); err != nil {
			continue
		}
		if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, method.Command, method.Args, opts); err != nil {
			status.Notes = append(status.Notes, fmt.Sprintf("%s failed: %s", method.Name, err))
			continue
		}
		accurate, _ := checkClockAccuracy()
		if accurate {
			status.Applied = true
			status.ExecutionState = hostreqkit.ExecutionApplied
			status.Notes = append(status.Notes, "clock synchronized via "+method.Name)
			return status, nil
		}
		status.Notes = append(status.Notes, method.Name+" ran but clock is still inaccurate")
	}

	status.ExecutionState = hostreqkit.ExecutionFailed
	status.Notes = append(status.Notes, "all clock synchronization methods failed; manual intervention required")
	return status, nil
}

func checkClockAccuracy() (bool, float64) {
	for _, source := range timeSources {
		resp, err := HTTPHeadFn(source)
		if err != nil {
			continue
		}
		resp.Body.Close()
		dateStr := resp.Header.Get("Date")
		if dateStr == "" {
			continue
		}
		remoteTime, err := http.ParseTime(dateStr)
		if err != nil {
			continue
		}
		drift := NowFn().Sub(remoteTime).Seconds()
		if math.Abs(drift) <= toleranceSeconds {
			return true, drift
		}
		return false, drift
	}
	return true, 0
}
