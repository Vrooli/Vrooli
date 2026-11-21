package androidadb

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"device-control/strategy"
)

func (a *Adapter) Actuate(ctx context.Context, event strategy.Actuation) error {
	if event.Pointer != nil {
		x, y, err := a.pointerPixels(ctx, event.Pointer)
		if err != nil {
			return err
		}
		verb := "tap"
		args := []string{"shell", "input", verb, strconv.Itoa(x), strconv.Itoa(y)}
		switch event.Pointer.Kind {
		case "long-press":
			args = []string{"shell", "input", "swipe", strconv.Itoa(x), strconv.Itoa(y), strconv.Itoa(x), strconv.Itoa(y), strconv.Itoa(maxInt(event.Pointer.DurationMS, 800))}
		case "drag", "fling", "swipe":
			endX, endY, parseErr := parsePointerEnd(event.Pointer.Button)
			if parseErr != nil {
				return parseErr
			}
			finalEndX, finalEndY := int(endX), int(endY)
			if event.Pointer.Normalized {
				finalEndX, finalEndY, parseErr = a.normalizedPixels(ctx, endX, endY)
				if parseErr != nil {
					return parseErr
				}
			}
			args = []string{"shell", "input", "swipe", strconv.Itoa(x), strconv.Itoa(y), strconv.Itoa(finalEndX), strconv.Itoa(finalEndY), strconv.Itoa(maxInt(event.Pointer.DurationMS, 300))}
		}
		if event.Pointer.Kind == "double-tap" {
			if _, err = a.runner.Run(ctx, "adb", a.args(args...)...); err != nil {
				return err
			}
			_, err = a.runner.Run(ctx, "adb", a.args(args...)...)
			return err
		}
		_, err = a.runner.Run(ctx, "adb", a.args(args...)...)
		return err
	}
	if event.Key != nil {
		_, err := a.runner.Run(ctx, "adb", a.args("shell", "input", "keyevent", event.Key.Key)...)
		return err
	}
	if event.Text != "" {
		// adb input text treats %s as a space and interprets shell metacharacters;
		// quote them into the input-text vocabulary before passing one argument.
		text := strings.ReplaceAll(event.Text, "%", "%%")
		text = strings.ReplaceAll(text, " ", "%s")
		_, err := a.runner.Run(ctx, "adb", a.args("shell", "input", "text", text)...)
		return err
	}
	if event.Action != "" {
		var args []string
		switch event.Action {
		case "swipe":
			args = append(args, "shell", "input", "swipe")
			parts := strings.Split(event.Value, ",")
			if len(parts) < 4 || len(parts) > 5 {
				return fmt.Errorf("swipe requires start x,y, end x,y, and optional duration")
			}
			args = append(args, parts...)
		case "scroll-to":
			startX, startY, err := a.normalizedPixels(ctx, .5, .8)
			if err != nil {
				return err
			}
			endX, endY, err := a.normalizedPixels(ctx, .5, .2)
			if err != nil {
				return err
			}
			args = append(args, "shell", "input", "swipe", strconv.Itoa(startX), strconv.Itoa(startY), strconv.Itoa(endX), strconv.Itoa(endY), "400")
		case "pinch":
			return &strategy.AvailabilityError{Reason: "multi-touch is unavailable on this ADB adapter", NextAction: "Use a strategy that declares multi-touch."}
		case "install":
			// Reinstall in place so update_migration preserves application data and
			// the same verb is safe when a prior chapter already installed the APK.
			args = append(args, "install", "-r", event.Value)
		case "launch":
			output, err := a.launchPackage(ctx, event.Package)
			if event.Output != nil {
				*event.Output = output
			}
			return err
		case "stop":
			args = append(args, "shell", "am", "force-stop", event.Package)
		case "uninstall":
			args = append(args, "shell", "pm", "uninstall", event.Package)
		case "clear-data":
			args = append(args, "shell", "pm", "clear", event.Package)
		case "grant-permission":
			args = append(args, "shell", "pm", "grant", event.Package, event.Permission)
		case "revoke-permission":
			args = append(args, "shell", "pm", "revoke", event.Package, event.Permission)
		case "rotate":
			// Refuse mutation if exact recovery cannot be captured. Normalized
			// orientation alone cannot distinguish an absent setting from zero.
			baseline, err := a.ReadState(ctx)
			if err != nil || len(baseline.RotationSettings) != 2 {
				return fmt.Errorf("rotation requires exact raw settings for recovery")
			}
			rotations := map[string]string{"portrait": "0", "landscape": "1", "180": "2", "reverse-portrait": "2", "270": "3", "reverse-landscape": "3"}
			rotation, ok := rotations[strings.ToLower(strings.TrimSpace(event.Value))]
			if !ok {
				return fmt.Errorf("rotation must be portrait, landscape, 180, or 270")
			}
			args = append(args, "shell", "settings", "put", "system", "accelerometer_rotation", "0")
			_, err = a.runner.Run(ctx, "adb", a.args(args...)...)
			if err != nil {
				return err
			}
			_, err = a.runner.Run(ctx, "adb", a.args("shell", "settings", "put", "system", "user_rotation", rotation)...)
			return err
		case "network":
			state := strings.ToLower(strings.TrimSpace(event.Value))
			if state != "offline" && state != "online" {
				return fmt.Errorf("network action requires offline or online")
			}
			enabled := "enable"
			if state == "offline" {
				enabled = "disable"
			}
			_, err := a.runner.Run(ctx, "adb", a.args("shell", "svc", "wifi", enabled)...)
			if err != nil {
				// Some Android 13 images reject svc wifi while still exposing the
				// connectivity shell command. Preserve the exact primary failure
				// only if the named fallback also fails.
				_, fallbackErr := a.runner.Run(ctx, "adb", a.args("shell", "cmd", "connectivity", "airplane-mode", enabled)...)
				if fallbackErr != nil {
					return fmt.Errorf("network control unavailable: adb svc wifi: %v; adb cmd connectivity airplane-mode: %w", err, fallbackErr)
				}
				return nil
			}
			_, err = a.runner.Run(ctx, "adb", a.args("shell", "svc", "data", enabled)...)
			if err != nil {
				_, fallbackErr := a.runner.Run(ctx, "adb", a.args("shell", "cmd", "connectivity", "airplane-mode", enabled)...)
				if fallbackErr != nil {
					return fmt.Errorf("network data control unavailable: adb svc data: %v; adb cmd connectivity airplane-mode: %w", err, fallbackErr)
				}
				err = nil
			}
			return err
		case "bluetooth":
			state := strings.ToLower(strings.TrimSpace(event.Value))
			if state != "online" && state != "offline" && state != "enable" && state != "disable" {
				return fmt.Errorf("bluetooth action requires online, offline, enable, or disable")
			}
			enabled := "enable"
			if state == "offline" || state == "disable" {
				enabled = "disable"
			}
			_, err := a.runner.Run(ctx, "adb", a.args("shell", "svc", "bluetooth", enabled)...)
			if err != nil {
				_, fallbackErr := a.runner.Run(ctx, "adb", a.args("shell", "cmd", "bluetooth_manager", enabled)...)
				if fallbackErr != nil {
					return fmt.Errorf("bluetooth unavailable: adb svc bluetooth: %v; adb cmd bluetooth_manager: %w", err, fallbackErr)
				}
			}
			return nil
		case "airplane-mode":
			state := strings.ToLower(strings.TrimSpace(event.Value))
			if state != "online" && state != "offline" && state != "enable" && state != "disable" {
				return fmt.Errorf("airplane-mode action requires online, offline, enable, or disable")
			}
			value := "1"
			if state == "online" || state == "disable" {
				value = "0"
			}
			if _, err := a.runner.Run(ctx, "adb", a.args("shell", "settings", "put", "global", "airplane_mode_on", value)...); err != nil {
				return err
			}
			broadcastState := "false"
			if value == "1" {
				broadcastState = "true"
			}
			_, err := a.runner.Run(ctx, "adb", a.args("shell", "am", "broadcast", "-a", "android.intent.action.AIRPLANE_MODE", "--ez", "state", broadcastState)...)
			return err
		case "screen":
			switch strings.ToLower(strings.TrimSpace(event.Value)) {
			case "wake":
				return a.SetScreenState(ctx, "on")
			case "sleep":
				return a.SetScreenState(ctx, "off")
			case "unlock":
				_, err := a.runner.Run(ctx, "adb", a.args("shell", "input", "keyevent", "KEYCODE_WAKEUP")...)
				if err != nil {
					return err
				}
				_, err = a.runner.Run(ctx, "adb", a.args("shell", "input", "keyevent", "KEYCODE_MENU")...)
				return err
			default:
				return fmt.Errorf("screen action requires wake, sleep, or unlock")
			}
		case "deep-link":
			if strings.TrimSpace(event.Value) == "" {
				return fmt.Errorf("deep-link action requires a URI")
			}
			args = append(args, "shell", "am", "start", "-a", "android.intent.action.VIEW", "-d", event.Value)
			if event.Package != "" {
				args = append(args, "-p", event.Package)
			}
		case "share":
			if strings.TrimSpace(event.Value) == "" {
				return fmt.Errorf("share action requires text")
			}
			args = append(args, "shell", "am", "start", "-a", "android.intent.action.SEND", "-c", "android.intent.category.DEFAULT", "-t", "text/plain")
			if event.Package != "" {
				args = append(args, "-p", event.Package)
			}
			// Keep the package restriction before the string extra. Android's
			// ActivityTaskManager shell parser can otherwise consume the first
			// word of a multi-word extra as the package value.
			args = append(args, "--es", "android.intent.extra.TEXT", shellQuote(event.Value))
		case "semantic-target":
			return fmt.Errorf("semantic-target is resolver-scoped; execute it through the flow resolver")
		case "semantic-assert":
			tree, err := a.runner.Run(ctx, "adb", a.args("exec-out", "uiautomator", "dump", "/dev/stdout")...)
			if err != nil {
				return err
			}
			values, resolveErr := findSemanticNode(tree, event.Value)
			if resolveErr != nil {
				return resolveErr
			}
			if event.Action == "semantic-assert" {
				if strings.TrimSpace(event.Expected) == "" {
					return fmt.Errorf("semantic assertion for %q requires expected text", event.Value)
				}
				for _, key := range []string{"text", "content-desc", "hint-text"} {
					if values[key] == event.Expected {
						return nil
					}
				}
				return fmt.Errorf("accessibility target %q did not contain expected text %q", event.Value, event.Expected)
			}
			x, y, boundsErr := semanticBounds(values, event.Value)
			if boundsErr != nil {
				return boundsErr
			}
			_, err = a.runner.Run(ctx, "adb", a.args("shell", "input", "tap", strconv.Itoa(x), strconv.Itoa(y))...)
			return err
		case "package-state":
			if strings.TrimSpace(event.Package) == "" {
				return fmt.Errorf("package-state action requires a package")
			}
			state, err := a.runner.Run(ctx, "adb", a.args("shell", "pm", "list", "packages", event.Package)...)
			if err != nil {
				return err
			}
			present := strings.Contains(string(state), "package:"+event.Package)
			expected := strings.ToLower(strings.TrimSpace(event.Value))
			switch expected {
			case "present":
				if !present {
					return fmt.Errorf("package %q is not installed", event.Package)
				}
			case "absent":
				if present {
					return fmt.Errorf("package %q is still installed", event.Package)
				}
			default:
				return fmt.Errorf("package-state action requires expected present or absent")
			}
			return nil
		case "device-logs":
			if event.Value == "crash" {
				if !applicationIdentityPattern.MatchString(event.Package) {
					return fmt.Errorf("crash logs require one exact package identity")
				}
				out, err := a.runner.Run(ctx, "adb", a.args("logcat", "-b", "crash", "-d", "-v", "brief", "-t", "500")...)
				if err != nil || len(out) > 1024*1024 {
					return fmt.Errorf("bounded crash buffer is unavailable")
				}
				filtered := packageJavaCrashes(string(out), event.Package)
				if len(filtered) == 0 {
					return fmt.Errorf("no matching Java crash in the bounded recent buffer; other crash types and older exits remain unknown")
				}
				if event.Output != nil {
					*event.Output = filtered
				}
				return nil
			}
			if event.Package != "" {
				pid, pidErr := a.runner.Run(ctx, "adb", a.args("shell", "pidof", event.Package)...)
				if pidErr != nil || strings.TrimSpace(string(pid)) == "" {
					return fmt.Errorf("device logs: package %q is not running", event.Package)
				}
				args = append(args, "logcat", "-d", "-v", "epoch", "--pid", strings.TrimSpace(string(pid)))
			} else {
				args = append(args, "logcat", "-d", "-v", "epoch")
			}
		case "framework-logs":
			if !validExactAndroidComponent(event.Component) {
				return fmt.Errorf("framework-logs requires an exact package/class component")
			}
			if event.LogTag != "ActivityManager" && event.LogTag != "AccessibilityManagerService" {
				return fmt.Errorf("framework-logs supports only ActivityManager or AccessibilityManagerService tags")
			}
			priority := event.LogPriority
			if priority == "" {
				priority = "W"
			}
			if priority != "W" && priority != "I" {
				return fmt.Errorf("framework-logs priority must be W or I")
			}
			probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			bounded, ok := a.runner.(boundedRunner)
			if !ok {
				return fmt.Errorf("bounded framework log output is unsupported by this runner")
			}
			// Android logcat treats the tag priority as a minimum: I includes
			// Info and higher. W remains the default and the only non-opt-in mode.
			out, err := bounded.RunBounded(probeCtx, "adb", 256*1024, a.args("logcat", "-d", "-v", "epoch", "-t", "250", "-s", event.LogTag+":"+priority)...)
			if err != nil {
				return fmt.Errorf("bounded framework log read failed: %w", err)
			}
			matched := filterExactComponentLogLines(string(out), event.Component)
			if event.Output != nil {
				*event.Output = summarizeFrameworkLogMatches(event.Component, event.LogTag, priority, matched)
			}
			return nil
		case "logcat-start":
			args = append(args, "logcat", "-c")
		case "logcat-stop":
			args = append(args, "logcat", "-d", "-v", "epoch")
		case "clock-sample":
			args = append(args, "shell", "date", "+%s.%N")
		case "screenshot":
			args = append(args, "exec-out", "screencap", "-p")
		case "clipboard-read":
			args = append(args, "shell", "cmd", "clipboard", "get")
		case "clipboard-write":
			if event.Value == "" {
				return fmt.Errorf("clipboard-write requires text")
			}
			args = append(args, "shell", "cmd", "clipboard", "set", event.Value)
		case "screenrecord":
			return fmt.Errorf("screenrecord is session-scoped; use recording-start and recording-stop")
		default:
			return fmt.Errorf("unsupported adb action %q", event.Action)
		}
		out, err := a.runner.Run(ctx, "adb", a.args(args...)...)
		if err != nil && len(out) > 0 {
			return fmt.Errorf("adb %s: %w: %s", event.Action, err, strings.TrimSpace(string(out)))
		}
		if err == nil && event.Action == "screenshot" {
			var decodeErr error
			out, _, _, decodeErr = validateCapturedImage(out)
			if decodeErr != nil {
				return fmt.Errorf("decode adb screenshot: %w", decodeErr)
			}
		}
		if err == nil && event.Output != nil && (event.Action == "device-logs" || event.Action == "logcat-stop" || event.Action == "clock-sample" || event.Action == "screenshot" || event.Action == "clipboard-read") {
			*event.Output = out
		}
		return err
	}
	return fmt.Errorf("actuation must contain pointer or key event")
}

func maxInt(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func parsePointerEnd(value string) (float64, float64, error) {
	parts := strings.Split(value, ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("pointer gesture requires end coordinates")
	}
	x, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, 0, err
	}
	y, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return 0, 0, err
	}
	return x, y, nil
}

func (a *Adapter) pointerPixels(ctx context.Context, event *strategy.PointerEvent) (int, int, error) {
	if !event.Normalized {
		return int(event.X), int(event.Y), nil
	}
	return a.normalizedPixels(ctx, event.X, event.Y)
}

func (a *Adapter) normalizedPixels(ctx context.Context, x, y float64) (int, int, error) {
	if x < 0 || x > 1 || y < 0 || y > 1 {
		return 0, 0, fmt.Errorf("normalized pointer coordinates must be between 0 and 1")
	}
	out, err := a.runner.Run(ctx, "adb", a.args("shell", "wm", "size")...)
	if err != nil {
		return 0, 0, fmt.Errorf("read display metrics: %w", err)
	}
	match := regexp.MustCompile(`(\d+)x(\d+)`).FindStringSubmatch(string(out))
	if len(match) != 3 {
		return 0, 0, fmt.Errorf("display metrics did not contain a size")
	}
	width, _ := strconv.Atoi(match[1])
	height, _ := strconv.Atoi(match[2])
	return int(x * float64(width)), int(y * float64(height)), nil
}
