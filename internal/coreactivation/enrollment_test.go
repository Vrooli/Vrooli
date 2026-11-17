package coreactivation

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestEnrollmentPersistsExplicitChoiceAndRefusesLostOrAmbiguousState(t *testing.T) {
	e, _, _, _ := fixture(t)
	s := e.Store
	if _, exists, err := s.Enrollment(); err != nil || exists {
		t.Fatalf("fresh enrollment: %t %v", exists, err)
	}
	locked, released := false, false
	qualify := func(context.Context) (func(), error) { locked = true; return func() { released = true }, nil }
	policy, err := s.ConfigureEnrollment(context.Background(), true, qualify)
	if err != nil || !policy.Enabled || !locked || !released {
		t.Fatalf("explicit enrollment not durable: %+v %v", policy, err)
	}
	if loaded, exists, err := s.Enrollment(); err != nil || !exists || loaded != policy {
		t.Fatalf("enrollment not readable: %+v %v", loaded, err)
	}
	if _, err := s.ConfigureEnrollment(context.Background(), false, qualify); err != nil {
		t.Fatal(err)
	}
	loaded, _, err := s.Enrollment()
	if err != nil || loaded.Enabled {
		t.Fatal("explicit disable not retained")
	}
	path := filepath.Join(s.Directory(), "enrollment.json")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Enrollment(); err == nil {
		t.Fatal("missing enrolled policy silently became unenrolled")
	}
}

func TestEnrollmentCannotChangeDuringAnAdmittedUpdate(t *testing.T) {
	e, r, _, _ := fixture(t)
	admit(t, e, r)
	called := false
	if _, err := e.Store.ConfigureEnrollment(context.Background(), false, func(context.Context) (func(), error) { called = true; return func() {}, nil }); err == nil {
		t.Fatal("enrollment changed across active update")
	}
	if called {
		t.Fatal("active update reached enrollment side effects")
	}
}

func TestEnrollmentPauseIntentIsAtomicAndConditional(t *testing.T) {
	e, _, _, _ := fixture(t)
	s := e.Store
	qualify := func(context.Context) (func(), error) { return func() {}, nil }
	initial, err := s.ConfigureEnrollment(context.Background(), true, qualify)
	if err != nil {
		t.Fatal(err)
	}
	paused, changed, err := s.CompareAndConfigureEnrollment(context.Background(), initial, false, true, qualify)
	if err != nil || !changed {
		t.Fatalf("pause: %v %t", err, changed)
	}
	loaded, exists, err := s.Enrollment()
	if err != nil || !exists || loaded != paused || loaded.Enabled || !loaded.RebaselinePaused {
		t.Fatalf("durable intent: %+v %v", loaded, err)
	}
	explicit, err := s.ConfigureEnrollment(context.Background(), false, qualify)
	if err != nil {
		t.Fatal(err)
	}
	if explicit.RebaselinePaused {
		t.Fatal("operator decision retained pause intent")
	}
	_, changed, err = s.CompareAndConfigureEnrollment(context.Background(), paused, true, false, qualify)
	if err != nil || changed {
		t.Fatalf("stale restore won: %v %t", err, changed)
	}
	loaded, _, err = s.Enrollment()
	if err != nil || loaded != explicit {
		t.Fatalf("operator decision overwritten: %+v %v", loaded, err)
	}
	_, changed, err = s.CompareAndConfigureEnrollment(context.Background(), initial, false, true, qualify)
	if err != nil || changed {
		t.Fatalf("stale pause won: %v %t", err, changed)
	}
}

// A process from before the pause field rejects the newer canonical record.
// After an interruption the new owner must recover it; an old installer must
// never silently interpret an unrecognized pause as an ordinary disable.
func TestPausedEnrollmentAcrossOwnerVersionsAndRecovery(t *testing.T) {
	e, _, _, _ := fixture(t)
	s := e.Store
	qualify := func(context.Context) (func(), error) { return func() {}, nil }
	initial, err := s.ConfigureEnrollment(context.Background(), true, qualify)
	if err != nil {
		t.Fatal(err)
	}
	paused, changed, err := s.CompareAndConfigureEnrollment(context.Background(), initial, false, true, qualify)
	if err != nil || !changed {
		t.Fatalf("pause: %v %t", err, changed)
	}
	type legacyEnrollment struct {
		SchemaVersion   int    `json:"schema_version"`
		Home            string `json:"home"`
		Executable      string `json:"executable"`
		UnitName        string `json:"unit_name"`
		Enabled         bool   `json:"enabled"`
		UpdatedUnixNano int64  `json:"updated_unix_nano"`
	}
	readLegacy := func() error {
		data, err := os.ReadFile(filepath.Join(s.Directory(), "enrollment.json"))
		if err != nil {
			return err
		}
		var legacy legacyEnrollment
		return canonicalMetadata(data, &legacy)
	}
	if err := readLegacy(); err == nil {
		t.Fatal("legacy owner accepted an unrecognized pause")
	}
	// Reopen the store, as after the transition process crashes.
	reopened, err := NewStore(initial.Home)
	if err != nil {
		t.Fatal(err)
	}
	loaded, exists, err := reopened.Enrollment()
	if err != nil || !exists || loaded != paused {
		t.Fatalf("reopen paused intent: %+v %v", loaded, err)
	}
	_, changed, err = reopened.CompareAndConfigureEnrollment(context.Background(), loaded, true, false, qualify)
	if err != nil || !changed {
		t.Fatalf("restore: %v %t", err, changed)
	}
	if err := readLegacy(); err != nil {
		t.Fatalf("legacy owner cannot read restored enrollment: %v", err)
	}
}
