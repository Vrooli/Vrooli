//go:build !linux

package hostreqkit

import (
	"context"
	"reflect"
)

func PlanLinuxOpenRGBDeviceAccess(context.Context, OpenRGBPermissionRequest) (OpenRGBPermissionPlan, error) {
	return OpenRGBPermissionPlan{}, ErrUnsupportedPlatform
}

func ApplyLinuxOpenRGBDeviceAccess(context.Context, OpenRGBPermissionPlan) (OpenRGBPermissionResult, error) {
	return OpenRGBPermissionResult{}, ErrUnsupportedPlatform
}

func RollbackLinuxOpenRGBDeviceAccess(context.Context, string) (OpenRGBPermissionResult, error) {
	return OpenRGBPermissionResult{}, ErrUnsupportedPlatform
}

func OpenRGBPermissionPersistentTypes() []reflect.Type { return nil }
