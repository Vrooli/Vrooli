package main

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrInvalidBuildTransition = errors.New("invalid extension build transition")
	ErrBuildNotFound          = errors.New("extension build not found")
)

type ExtensionBuildStatus string

const (
	ExtensionBuildBuilding ExtensionBuildStatus = "building"
	ExtensionBuildReady    ExtensionBuildStatus = "ready"
	ExtensionBuildFailed   ExtensionBuildStatus = "failed"
	ExtensionBuildCanceled ExtensionBuildStatus = "canceled"
)

func (status ExtensionBuildStatus) Valid() bool {
	switch status {
	case ExtensionBuildBuilding, ExtensionBuildReady, ExtensionBuildFailed, ExtensionBuildCanceled:
		return true
	default:
		return false
	}
}

func (status ExtensionBuildStatus) Active() bool {
	return status == ExtensionBuildBuilding
}

func (status ExtensionBuildStatus) Terminal() bool {
	return status.Valid() && !status.Active()
}

func (status ExtensionBuildStatus) CanTransition(next ExtensionBuildStatus) bool {
	if !status.Valid() || !next.Valid() {
		return false
	}
	if status == next {
		return true
	}
	return status == ExtensionBuildBuilding && next.Terminal()
}

func (build *ExtensionBuild) TransitionStatus(next ExtensionBuildStatus) error {
	if build == nil || !build.Status.CanTransition(next) {
		if build == nil {
			return fmt.Errorf("%w: nil build", ErrInvalidBuildTransition)
		}
		return fmt.Errorf("%w: %s -> %s", ErrInvalidBuildTransition, build.Status, next)
	}
	build.Status = next
	return nil
}

func (build *ExtensionBuild) Finish(next ExtensionBuildStatus, at time.Time) error {
	if !next.Terminal() {
		return fmt.Errorf("%w: %s is not terminal", ErrInvalidBuildTransition, next)
	}
	if err := build.TransitionStatus(next); err != nil {
		return err
	}
	completedAt := at
	build.CompletedAt = &completedAt
	return nil
}

func (build *ExtensionBuild) Fail(reason string, at time.Time) error {
	if err := build.Finish(ExtensionBuildFailed, at); err != nil {
		return err
	}
	build.ErrorLog = append(build.ErrorLog, reason)
	return nil
}
