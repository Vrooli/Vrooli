//go:build !linux

package nativeisolation

func checkObservationMarkers(ProbeSpec) error { return ErrRefused }
