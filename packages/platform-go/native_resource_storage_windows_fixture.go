package platform

import (
	"context"
	"strings"
)

// Query-origin absence is distinct from any downstream descriptor/image error.
// Once the native task exists, all uncertainty remains an enrollment failure.
type windowsNativeResourceStorageEnrollment struct {
	query     func() (WindowsTaskSnapshot, error)
	missing   func(error) bool
	plan      func(WindowsTaskSnapshot) (nativeSystemBrokerDescriptor, nativeSystemBrokerPlan, WindowsTaskSnapshot, error)
	principal func() (string, error)
	retained  func(context.Context) (NativeSystemServiceBrokerOptions, bool, error)
}

func (e windowsNativeResourceStorageEnrollment) read(ctx context.Context) (NativeSystemServiceBrokerOptions, bool, error) {
	var zero NativeSystemServiceBrokerOptions
	before, err := e.query()
	if e.missing(err) {
		if e.retained == nil {
			return zero, false, nil
		}
		options, configured, recoveryErr := e.retained(ctx)
		if recoveryErr != nil {
			return zero, false, recoveryErr
		}
		if _, checkErr := e.query(); !e.missing(checkErr) {
			if checkErr != nil {
				return zero, false, checkErr
			}
			return zero, false, ErrOperationIdentity
		}
		return options, configured, nil
	}
	if err != nil {
		return zero, false, err
	}
	d, p, _, err := e.plan(before)
	if err != nil {
		return zero, false, err
	}
	principal, err := e.principal()
	if err != nil {
		return zero, false, err
	}
	if principal != d.Options.OperatorPrincipal {
		return zero, false, ErrOperationIdentity
	}
	after, err := e.query()
	if err != nil {
		return zero, false, err
	}
	afterD, afterP, _, err := e.plan(after)
	if err != nil {
		return zero, false, err
	}
	if d.Options != afterD.Options || !strings.EqualFold(p.DescriptorPath, afterP.DescriptorPath) || before.XML != after.XML || before.SecurityDescriptor != after.SecurityDescriptor {
		return zero, false, ErrOperationIdentity
	}
	if err = ctx.Err(); err != nil {
		return zero, false, err
	}
	return d.Options, true, nil
}
