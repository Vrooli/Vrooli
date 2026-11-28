//go:build windows

package platform

import (
	"context"

	"golang.org/x/sys/windows"
)

func enrolledNativeOperatorResourceStorageOptions(ctx context.Context) (NativeSystemServiceBrokerOptions, bool, error) {
	engine := windowsNativeResourceStorageEnrollment{
		query:   func() (WindowsTaskSnapshot, error) { return WindowsTaskStatus(runtimeSupervisorService) },
		missing: taskNativeErrorMissing, plan: windowsInstalledTaskPlanFromSnapshot, retained: windowsRetainedNativeResourceStorageOptions,
		principal: func() (string, error) {
			u, err := windows.GetCurrentThreadEffectiveToken().GetTokenUser()
			if err != nil {
				return "", err
			}
			return "windows:" + u.User.Sid.String(), nil
		},
	}
	return engine.read(ctx)
}
