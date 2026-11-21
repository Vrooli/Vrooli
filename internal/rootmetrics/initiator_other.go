//go:build !linux

package rootmetrics

func observeInitiator() string { return "unknown" }
