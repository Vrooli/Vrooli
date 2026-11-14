package androidadb

import (
	"context"
	"device-control/strategy"
	"fmt"
	"regexp"
	"strings"
	"time"
)

func parseApplicationStatus(data string) map[string]string {
	lines := strings.Split(strings.TrimSpace(data), "\n")
	if len(lines) == 0 || len(lines) > 32 {
		return nil
	}
	rowPattern := regexp.MustCompile(`^Row:\s*[0-9]+\s+key=([a-z][a-z0-9_.]{0,63}), value=([A-Za-z0-9_./:$-]{1,512})\s*$`)
	result := map[string]string{}
	for _, line := range lines {
		row := rowPattern.FindStringSubmatch(line)
		if row == nil {
			return nil
		}
		if _, duplicate := result[row[1]]; duplicate {
			return nil
		}
		result[row[1]] = row[2]
	}
	return result
}

func (a *Adapter) applicationStatus(ctx context.Context, pkg string) (strategy.AppLifecycleResult, error) {
	r := strategy.AppLifecycleResult{Operation: "app-status", Package: pkg, Status: strategy.StatusUnavailable}
	user, err := a.currentAndroidUser(ctx)
	if err != nil {
		return r, err
	}
	r.AndroidUser = user
	out, err := a.runner.Run(ctx, "adb", a.args("shell", "content", "query", "--user", user, "--uri", "content://"+pkg+".vrooli.status/state")...)
	if err != nil || len(out) > 64*1024 {
		return r, fmt.Errorf("bounded application status provider is unavailable")
	}
	r.ApplicationStatus = parseApplicationStatus(string(out))
	if len(r.ApplicationStatus) == 0 {
		return r, fmt.Errorf("application status provider returned no recognized metadata rows")
	}
	r.Status = strategy.StatusAvailable
	r.Outcome = "application_status_observed"
	r.ObservedAt = time.Now().UTC()
	return r, nil
}
