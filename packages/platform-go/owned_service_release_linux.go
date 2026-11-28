//go:build linux

package platform

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var serviceBootIDPath = "/proc/sys/kernel/random/boot_id"

func ownedServiceReleased(ctx context.Context, ref ServiceRef, boot string) (bool, error) {
	return ownedWorkloadReleased(ctx, OperationRef(ref), boot)
}

func ownedWorkloadReleased(ctx context.Context, ref OperationRef, boot string) (bool, error) {
	state, err := queryOwnedUnit(ctx, ref.Unit)
	if err != nil {
		return false, err
	}
	if state.LoadState != "not-found" {
		if err := validateOwnedUnitRef(ref, state); err != nil {
			return false, err
		}
		return false, nil
	}
	if _, err := os.Stat(filepath.Join(operationCgroupRoot, strings.TrimPrefix(ref.Scope.Path, "/"))); !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	if boot != "" {
		data, err := os.ReadFile(serviceBootIDPath)
		if err != nil {
			return false, err
		}
		current := strings.TrimSpace(string(data))
		if !serviceBootIDPattern.MatchString(current) {
			return false, fmt.Errorf("platform: current host boot identity is invalid")
		}
		if current != boot {
			if err := ctx.Err(); err != nil {
				return false, err
			}
			return true, nil
		}
	}
	stamp, alive, err := linuxProcessStartTime(ref.RootPID)
	if err != nil {
		return false, err
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	return !alive || stamp != ref.RootStartTime, nil
}

func ownedServiceIntentAbsent(ctx context.Context, intent ServiceIntent) (bool, error) {
	return ownedWorkloadIntentAbsent(ctx, OperationIntent(intent))
}

func ownedWorkloadIntentAbsent(ctx context.Context, intent OperationIntent) (bool, error) {
	unit, err := intent.UnitName()
	if err != nil {
		return false, err
	}
	state, err := queryOwnedUnit(ctx, unit)
	if err != nil {
		return false, err
	}
	if state.LoadState != "not-found" {
		return false, nil
	}
	querySlice, slice := queryOperationsSlice, OperationsSlice
	if intent.Kind == ownedServiceKind {
		querySlice, slice = queryServicesSlice, ServicesSlice
	}
	base, err := querySlice(ctx)
	if err != nil {
		return false, err
	}
	if !validOwnedSlicePath(base, slice) {
		return false, ErrOperationIdentity
	}
	_, err = os.Stat(filepath.Join(operationCgroupRoot, strings.TrimPrefix(filepath.Join(base, unit), "/")))
	if err == nil {
		return false, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	return true, ctx.Err()
}
