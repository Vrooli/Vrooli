//go:build !darwin && !linux

package platform

func darwinSystemBrokerPublishExclusive(string, string) error { return ErrUnsupported }
