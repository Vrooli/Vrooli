package androidadb

import (
	"context"
	"device-control/strategy"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func parseProcessExits(data, pkg string) []strategy.ProcessExit {
	var exits []strategy.ProcessExit
	labels := map[int]string{0: "unknown", 1: "exit-self", 2: "signaled", 3: "low-memory", 4: "java-crash", 5: "native-crash", 6: "anr", 7: "initialization-failure", 8: "permission-change", 9: "excessive-resource-usage", 10: "user-requested", 11: "user-stopped", 12: "dependency-died", 13: "other"}
	for _, block := range strings.Split(data, "ApplicationExitInfo ")[1:] {
		p := regexp.MustCompile(`(?m)^\s*process=([A-Za-z0-9_.:]+)\s*$`).FindStringSubmatch(block)
		r := regexp.MustCompile(`(?m)^\s*reason=([0-9]+)\b`).FindStringSubmatch(block)
		if len(p) != 2 || len(r) != 2 || !(p[1] == pkg || strings.HasPrefix(p[1], pkg+":")) {
			continue
		}
		code, err := strconv.Atoi(r[1])
		if err != nil || labels[code] == "" {
			continue
		}
		exit := strategy.ProcessExit{Process: p[1], ReasonCode: code, Reason: labels[code]}
		if stamp := regexp.MustCompile(`(?m)^\s*timestamp=([0-9T: /+.,-]{1,64})\s*$`).FindStringSubmatch(block); len(stamp) == 2 {
			exit.Timestamp = strings.TrimSpace(stamp[1])
		}
		exits = append(exits, exit)
		if len(exits) == 3 {
			break
		}
	}
	return exits
}

func (a *Adapter) processState(ctx context.Context, pkg string) (strategy.AppLifecycleResult, error) {
	r := strategy.AppLifecycleResult{Operation: "process-state", Package: pkg, Status: strategy.StatusUnavailable}
	out, err := a.runner.Run(ctx, "adb", a.args("shell", "dumpsys", "activity", "exit-info", pkg)...)
	if err != nil || len(out) > 256*1024 {
		return r, fmt.Errorf("bounded process exit history is unavailable")
	}
	r.ProcessExits = parseProcessExits(string(out), pkg)
	if len(r.ProcessExits) == 0 {
		return r, fmt.Errorf("no recognized exact-package process exits; history may be absent or unsupported")
	}
	r.Status = strategy.StatusAvailable
	r.Outcome = "process_exits_observed"
	r.ObservedAt = time.Now().UTC()
	return r, nil
}
