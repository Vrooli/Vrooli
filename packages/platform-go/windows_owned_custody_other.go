//go:build !windows

package platform

func EstablishWindowsOwnedServiceCustody() error { return ErrUnsupported }
