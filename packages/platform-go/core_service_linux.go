//go:build linux

package platform

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var queryCoreSupervisor = queryCoreSupervisorSystemd
var queryCoreSupervisorParent = func(ctx context.Context) (string, error) {
	bounded, cancel := context.WithTimeout(ctx, operationCommandWait)
	defer cancel()
	return queryOwnedSliceSystemd(bounded, "app.slice")
}

type coreServiceOutput struct {
	data     bytes.Buffer
	overflow bool
}

func (b *coreServiceOutput) Write(p []byte) (int, error) {
	n := len(p)
	remaining := (64 << 10) - b.data.Len()
	if len(p) > remaining {
		b.overflow = true
		p = p[:remaining]
	}
	_, _ = b.data.Write(p)
	return n, nil
}

func queryCoreSupervisorSystemd(ctx context.Context) (CoreServiceState, error) {
	ctx, cancel := context.WithTimeout(ctx, operationCommandWait)
	defer cancel()
	unit, _ := CoreUnitByID(CoreUnitRuntimeSupervisor)
	cmd := exec.CommandContext(ctx, "systemctl", "--user", "show", "--property=LoadState,ActiveState,SubState,ControlGroup,InvocationID,MainPID,Job,FragmentPath,DropInPaths,KillMode,NeedDaemonReload", "--", unit.Systemd)
	cmd.Env = userManagerEnv(nil)
	cmd.WaitDelay = time.Second
	var output coreServiceOutput
	cmd.Stdout = &output
	if err := cmd.Run(); err != nil || output.overflow {
		return CoreServiceState{}, fmt.Errorf("core supervisor native observation unavailable or exceeds its bound")
	}
	values := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(output.data.String()), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return CoreServiceState{}, ErrOperationIdentity
		}
		if _, exists := values[key]; exists {
			return CoreServiceState{}, ErrOperationIdentity
		}
		values[key] = value
	}
	for _, key := range []string{"LoadState", "ActiveState", "SubState", "ControlGroup", "InvocationID", "MainPID", "Job", "FragmentPath", "DropInPaths", "KillMode", "NeedDaemonReload"} {
		if _, ok := values[key]; !ok {
			return CoreServiceState{}, ErrOperationIdentity
		}
	}
	pid, err := strconv.Atoi(values["MainPID"])
	if err != nil || pid < 0 {
		return CoreServiceState{}, ErrOperationIdentity
	}
	job := values["Job"]
	return CoreServiceState{Unit: unit.Systemd, LoadState: values["LoadState"], ActiveState: values["ActiveState"], SubState: values["SubState"], ControlGroup: values["ControlGroup"], InvocationID: values["InvocationID"], FragmentPath: values["FragmentPath"], DropInPaths: values["DropInPaths"], KillMode: values["KillMode"], NeedDaemonReload: values["NeedDaemonReload"], PID: pid, PendingJob: job != "" && job != "0" && !strings.HasPrefix(job, "0 ")}, nil
}

func inspectCoreSupervisor(ctx context.Context) (CoreServiceState, error) {
	if err := ctx.Err(); err != nil {
		return CoreServiceState{}, err
	}
	return queryCoreSupervisor(ctx)
}

func pinCoreSupervisor(ctx context.Context, expected CoreServiceState) (*CoreSupervisorFence, error) {
	before, err := inspectCoreSupervisor(ctx)
	if err != nil {
		return nil, err
	}
	if before.LoadState != "loaded" {
		return nil, ErrOperationIdentity
	}
	if before != expected {
		return nil, ErrOperationIdentity
	}
	var pinned *pinnedServiceScope
	group := before.ControlGroup
	if group == "" {
		parent, e := queryCoreSupervisorParent(ctx)
		if e != nil {
			return nil, e
		}
		group = path.Join(parent, before.Unit)
	}
	{
		if !path.IsAbs(group) || path.Clean(group) != group || path.Base(group) != before.Unit || !strings.HasPrefix(group, "/user.slice/") {
			return nil, ErrOperationIdentity
		}
		pinned, _, _, err = openPinnedServiceScope(ScopeRef{Kind: ScopeKindCgroup, Path: group})
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		if err != nil && before.ControlGroup != "" {
			return nil, err
		}
	}
	close := func() error {
		if pinned != nil {
			return pinned.dir.Close()
		}
		return nil
	}
	after, err := inspectCoreSupervisor(ctx)
	if err != nil || before.InvocationID != after.InvocationID || before.ControlGroup != after.ControlGroup || before.PID != after.PID {
		_ = close()
		return nil, ErrOperationIdentity
	}
	return &CoreSupervisorFence{close: close, stopped: func(ctx context.Context) error {
		for {
			if err := ctx.Err(); err != nil {
				return err
			}
			state, err := inspectCoreSupervisor(ctx)
			if err != nil {
				return err
			}
			if state.PendingJob || state.LoadState != "loaded" || state.ActiveState != "inactive" || state.PID != 0 || state.InvocationID != "" && state.InvocationID != before.InvocationID {
				return fmt.Errorf("core supervisor native stop remains unresolved")
			}
			if state.ControlGroup != "" && state.ControlGroup != before.ControlGroup {
				return ErrOperationIdentity
			}
			if pinned == nil {
				if state.ControlGroup != "" {
					return ErrOperationIdentity
				}
				_, e := os.Stat(filepath.Join(operationCgroupRoot, strings.TrimPrefix(group, "/")))
				if errors.Is(e, os.ErrNotExist) {
					return nil
				}
				if e != nil {
					return e
				}
				return ErrOperationIdentity
			}
			empty, err := pinned.empty()
			if err != nil {
				return err
			}
			if empty {
				held, e := pinned.dir.Stat()
				if e != nil {
					return e
				}
				current, e := os.Stat(filepath.Join(operationCgroupRoot, strings.TrimPrefix(group, "/")))
				if errors.Is(e, os.ErrNotExist) {
					return nil
				}
				if e != nil {
					return e
				}
				if !os.SameFile(held, current) {
					return ErrOperationIdentity
				}
				return nil
			}
			timer := time.NewTimer(operationPoll)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
	}}, nil
}
