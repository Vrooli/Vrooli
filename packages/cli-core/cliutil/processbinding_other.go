//go:build !linux

package cliutil

import "errors"

type ProcessBinding struct {
	PID    int    `json:"pid"`
	UID    int    `json:"uid"`
	Start  string `json:"start"`
	Boot   string `json:"boot"`
	Cgroup string `json:"cgroup"`
}

func ObserveProcessBinding(int) (ProcessBinding, error) {
	return ProcessBinding{}, errors.New("live process grants are unsupported on this host")
}
func ProcessDescendsFrom(int, ProcessBinding) bool { return false }

func ProcessParent(int) int { return 0 }
