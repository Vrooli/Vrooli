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
