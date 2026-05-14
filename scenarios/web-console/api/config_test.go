package main

import (
	"testing"

	"web-console/internal/ptyfake"
)

// [REQ:P1-001a] Session Policy Controls - max sessions enforcement
func TestMaxSessions_Enforcement(t *testing.T) {
	sm := NewSessionManagerWithFactory(ptyfake.NewFactory())
	sm.cfg.MaxSessions = 2

	s1, err := sm.Create("", 0, 0, "", nil)
	if err != nil {
		t.Fatalf("first session: %v", err)
	}
	defer func() { _ = sm.Delete(s1.ID) }()

	s2, err := sm.Create("", 0, 0, "", nil)
	if err != nil {
		t.Fatalf("second session: %v", err)
	}
	defer func() { _ = sm.Delete(s2.ID) }()

	_, err = sm.Create("", 0, 0, "", nil)
	if err == nil {
		t.Error("third session should be rejected when MaxSessions=2")
	}
}
