package control

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"device-control/strategy"
)

// LaunchAppRequest opens one application by common name ("netflix") or by an
// exact Android package. The call itself is the operator's explicit launch
// request, so lifecycle confirmation is supplied here rather than by the caller.
type LaunchAppRequest struct {
	Device string `json:"device"`
	App    string `json:"app"`
	Actor  string `json:"actor"`
	// Wake brings the screen up first (default for the CLI verb).
	Wake bool `json:"wake,omitempty"`
}

type LaunchAppResult struct {
	DeviceID          string   `json:"device_id"`
	App               string   `json:"app"`
	Package           string   `json:"package,omitempty"`
	Status            string   `json:"status"` // foreground_verified | launched_unverified | not_installed | failed
	ForegroundPackage string   `json:"foreground_package,omitempty"`
	Detail            string   `json:"detail,omitempty"`
	NextActions       []string `json:"next_actions,omitempty"`
}

const (
	launchVerifyAttempts = 6
	launchVerifyInterval = 700 * time.Millisecond
)

// knownApps maps a lowercase spoken name to candidate Android packages, TV
// package first. The first installed candidate wins, so one name works on a TV
// and on a phone.
var knownApps = map[string][]string{
	"netflix":       {"com.netflix.ninja", "com.netflix.mediaclient"},
	"youtube":       {"com.google.android.youtube.tv", "com.google.android.youtube"},
	"youtube tv":    {"com.google.android.youtube.tvunplugged"},
	"prime video":   {"com.amazon.amazonvideo.livingroom", "com.amazon.avod.thirdpartyclient"},
	"disney+":       {"com.disney.disneyplus"},
	"disney plus":   {"com.disney.disneyplus"},
	"hulu":          {"com.hulu.livingroomplus", "com.hulu.plus"},
	"max":           {"com.wbd.stream"},
	"plex":          {"com.plexapp.android"},
	"spotify":       {"com.spotify.tv.android", "com.spotify.music"},
	"twitch":        {"tv.twitch.android.app"},
	"apple tv":      {"com.apple.atve.androidtv.appletv"},
	"settings":      {"com.android.tv.settings", "com.android.settings"},
	"chrome":        {"com.android.chrome"},
	"play store":    {"com.android.vending"},
	"google photos": {"com.google.android.apps.photos"},
}

// KnownAppNames lists the spoken names LaunchApp resolves, for help text.
func KnownAppNames() []string {
	names := make([]string, 0, len(knownApps))
	for name := range knownApps {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func appCandidates(app string) ([]string, error) {
	app = strings.ToLower(strings.TrimSpace(app))
	if app == "" {
		return nil, fmt.Errorf("app is required: a name such as %q or an exact package", "netflix")
	}
	if androidPackageName.MatchString(app) {
		return []string{app}, nil
	}
	if candidates, ok := knownApps[app]; ok {
		return candidates, nil
	}
	return nil, fmt.Errorf("unknown app name %q; pass the exact Android package (for example com.netflix.ninja) or one of: %s", app, strings.Join(KnownAppNames(), ", "))
}

// LaunchApp resolves the app, verifies it is installed, launches it under a
// lease, and reports whether the foreground package actually changed. A command
// acknowledgement alone is reported as launched_unverified.
func (s *Service) LaunchApp(ctx context.Context, request LaunchAppRequest) (result LaunchAppResult, err error) {
	candidates, err := appCandidates(request.App)
	if err != nil {
		return LaunchAppResult{App: request.App, Status: "failed", Detail: err.Error()}, err
	}
	device, err := s.resolveExactOrUniqueDevice(ctx, request.Device)
	if err != nil {
		return LaunchAppResult{App: request.App, Status: "failed", Detail: err.Error()}, err
	}
	result = LaunchAppResult{DeviceID: device.ID, App: request.App, Status: "failed"}
	actor := firstNonEmpty(strings.TrimSpace(request.Actor), "app-launch")
	var prepared []Chapter
	if request.Wake {
		prepared = s.prepareScreen(ctx, device, CaptureRequest{Device: device.ID, Actor: actor})
	}
	lease, err := s.AcquireContext(ctx, device.ID, actor, 2*time.Minute)
	if err != nil {
		result.Detail = err.Error()
		return result, err
	}
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if _, releaseErr := s.ReleaseContext(cleanupCtx, lease.ID); releaseErr != nil && err == nil {
			err = fmt.Errorf("app launch finished but lease release failed: %w", releaseErr)
		}
	}()
	// Probe the adapter once; package-state, launch and foreground-state all use it.
	target, err := s.resolveLifecycleTarget(ctx, device.ID, lease.LeaseToken, "launch")
	if err != nil {
		result.Detail = err.Error()
		result.NextActions = s.launchGapNextActions(ctx, device)
		return result, nil
	}
	lifecycle := func(operation, pkg string) (AppLifecycleResponse, error) {
		return s.runLifecycle(ctx, target, AppLifecycleOperation{Actor: actor, LeaseToken: lease.LeaseToken, Confirmed: true, AppLifecycleRequest: strategy.AppLifecycleRequest{Operation: operation, Package: pkg}})
	}
	for _, pkg := range candidates {
		state, stateErr := lifecycle("package-state", pkg)
		if stateErr != nil {
			result.Detail = stateErr.Error()
			result.NextActions = s.launchGapNextActions(ctx, device)
			return result, nil
		}
		if state.Result.Installed {
			result.Package = pkg
			break
		}
	}
	if result.Package == "" {
		result.Status = "not_installed"
		result.Detail = fmt.Sprintf("none of %s is installed on %s", strings.Join(candidates, ", "), device.Name)
		result.NextActions = []string{"Install the app on the device, or pass the exact package with: device-control device open " + device.ID + " --app <package>"}
		return result, nil
	}
	if _, launchErr := lifecycle("launch", result.Package); launchErr != nil {
		result.Detail = launchErr.Error()
		return result, nil
	}
	result.Status = "launched_unverified"
	for attempt := 0; attempt < launchVerifyAttempts; attempt++ {
		select {
		case <-ctx.Done():
			result.Detail = ctx.Err().Error()
			return result, nil
		case <-time.After(launchVerifyInterval):
		}
		foreground, fgErr := lifecycle("foreground-state", result.Package)
		if fgErr != nil {
			result.Detail = fgErr.Error()
			continue
		}
		result.ForegroundPackage = observedForegroundPackage(foreground.Result)
		if result.ForegroundPackage == result.Package {
			result.Status, result.Detail = "foreground_verified", ""
			return result, nil
		}
	}
	if result.Detail == "" {
		result.Detail = fmt.Sprintf("launch was acknowledged but the foreground package is %q", result.ForegroundPackage)
		if result.ForegroundPackage == "" {
			result.Detail += "; the screen may be locked or off"
			result.NextActions = []string{fmt.Sprintf("device-control device state %s  (shows lock and screen state)", device.ID)}
			if locked := captureFailureNextActions(device, prepared, s.unlockSetupAction(ctx, device)); len(locked) > 0 {
				result.NextActions = locked
			}
		}
	}
	return result, nil
}

// launchGapNextActions points a device with no app-lifecycle transport at the
// same ADB path a capture gap uses.
func (s *Service) launchGapNextActions(ctx context.Context, device Device) []string {
	return s.captureGapNextActions(ctx, device, "screenshot")
}

// observedForegroundPackage names the package the device reports in the
// foreground. foreground-state reports independent signals (resumed activity and
// focused window) rather than one package; the package is trusted only when
// every signal that is present agrees, and is "" when they disagree or none exists.
func observedForegroundPackage(result strategy.AppLifecycleResult) string {
	if result.ForegroundPackage != "" {
		return result.ForegroundPackage
	}
	packageOf := func(component string) string { return strings.SplitN(component, "/", 2)[0] }
	resumed, focused := packageOf(result.ResumedActivity), packageOf(result.FocusedWindowActivity)
	switch {
	case resumed != "" && focused != "" && resumed != focused:
		return ""
	case resumed != "":
		return resumed
	default:
		return focused
	}
}
