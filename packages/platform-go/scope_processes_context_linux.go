//go:build linux

package platform

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var queryScopeState = func(ctx context.Context, unit string) (load, active string, err error) {
	cmd := exec.CommandContext(ctx, "systemctl", "--user", "show", "--property=LoadState", "--property=ActiveState", "--", unit)
	cmd.Env = userManagerEnv(nil)
	cmd.WaitDelay = time.Second
	out, runErr := cmd.CombinedOutput()
	for _, line := range strings.Split(string(out), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch key {
		case "LoadState":
			load = value
		case "ActiveState":
			active = value
		}
	}
	if load == "" || active == "" {
		if runErr == nil {
			runErr = errors.New("manager omitted unit state")
		}
		return "", "", runErr
	}
	if runErr != nil && !(load == "not-found" && active == "inactive") {
		return load, active, runErr
	}
	return load, active, nil
}

func scopeProcessesContext(ctx context.Context, ref ScopeRef) ([]int, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	pids, err := scopeProcesses(ref)
	if err == nil {
		return pids, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	dir, e := readableCgroup(ref)
	if e != nil {
		return nil, e
	}
	if _, e = os.Stat(dir); e == nil {
		return nil, fmt.Errorf("scope directory exists but membership is unavailable: %s", ref.Path)
	} else if !errors.Is(e, os.ErrNotExist) {
		return nil, e
	}
	// A gone observation is accepted only for the exact previously acquired,
	// owned scope. Do not infer unit identity from arbitrary missing paths.
	if _, e = freezableCgroup(ref); e != nil {
		return nil, e
	}
	unit := ref.Name
	if !strings.HasSuffix(unit, scopeSuffix) {
		unit += scopeSuffix
	}
	if ref.Name == "" || filepath.Base(dir) != unit {
		return nil, errors.New("gone scope name/path identity mismatch")
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	load, active, e := queryScopeState(ctx, unit)
	if e != nil {
		return nil, fmt.Errorf("scope gone state unavailable: %w", e)
	}
	if active != "inactive" || (load != "not-found" && load != "loaded") {
		return nil, fmt.Errorf("scope cgroup absent but unit state is load=%s active=%s", load, active)
	}
	return nil, nil
}
