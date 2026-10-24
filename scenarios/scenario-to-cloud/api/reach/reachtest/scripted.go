// Package reachtest is the shared in-memory reach used by handler and probe
// tests: it answers typed commands by their argv and records every call, so a
// test can assert exactly which verbs and observation programs a code path
// issued without any transport.
package reachtest

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"scenario-to-cloud/identity"
	"scenario-to-cloud/reach"
)

// Answer is one scripted reply.
type Answer struct {
	Result reach.Result
	Err    error
}

// Scripted answers commands by exact argv (space-joined) or by prefix.
// Unknown commands return exit 0 with empty output unless Strict is set.
type Scripted struct {
	mu       sync.Mutex
	Answers  map[string]Answer
	Prefixes map[string]Answer
	Strict   bool
	Calls    []reach.Command
	Targets  []identity.TargetRef
	Sessions []reach.SessionSpec
	Session  reach.Session
	Caps     reach.Capabilities
}

var _ reach.Reach = (*Scripted)(nil)

// Key renders a command the way Answers keys it.
func Key(cmd reach.Command) string { return strings.Join(cmd.Argv(), " ") }

// Exec records the call and returns the scripted answer.
func (s *Scripted) Exec(_ context.Context, target identity.TargetRef, cmd reach.Command) (reach.Result, error) {
	if err := reach.ValidateCommand(cmd); err != nil {
		return reach.Result{}, err
	}
	s.mu.Lock()
	s.Calls = append(s.Calls, cmd)
	s.Targets = append(s.Targets, target)
	s.mu.Unlock()
	key := Key(cmd)
	if a, ok := s.Answers[key]; ok {
		return a.Result, a.Err
	}
	// Keep existing semantic fixtures reusable while the transport migrates:
	// typed observations still record the owner verb, but their old fixture
	// answer is looked up by the closed compatibility mapping.
	if cmd.Observation != nil {
		for program, kind := range reach.ObservationKinds {
			if kind != cmd.Observation.Kind {
				continue
			}
			legacy := strings.Join(append([]string{program}, cmd.Observation.Args...), " ")
			if a, ok := s.Answers[legacy]; ok {
				return a.Result, a.Err
			}
			for prefix, a := range s.Prefixes {
				if strings.HasPrefix(legacy, prefix) {
					return a.Result, a.Err
				}
			}
			break
		}
	}
	for prefix, a := range s.Prefixes {
		if strings.HasPrefix(key, prefix) {
			return a.Result, a.Err
		}
	}
	if s.Strict {
		return reach.Result{}, fmt.Errorf("reachtest: no answer scripted for %q", key)
	}
	return reach.Result{ExitCode: 0}, nil
}

// Deliver records nothing and succeeds; delivery is covered by adapter tests.
func (s *Scripted) Deliver(_ context.Context, _ identity.TargetRef, d reach.Delivery) (reach.DeliveryReceipt, error) {
	receipt := reach.DeliveryReceipt{Transport: "test"}
	for _, f := range d.Files {
		receipt.Files = append(receipt.Files, reach.DeliveredFile{Role: f.Role, RemotePath: f.RemotePath, SHA256: f.SHA256})
	}
	return receipt, nil
}

// Negotiate returns the scripted capabilities.
func (s *Scripted) Negotiate(context.Context, identity.TargetRef) (reach.Capabilities, error) {
	return s.Caps, nil
}

// OpenSession records the spec and hands back the scripted session.
func (s *Scripted) OpenSession(_ context.Context, _ identity.TargetRef, spec reach.SessionSpec) (reach.Session, error) {
	s.mu.Lock()
	s.Sessions = append(s.Sessions, spec)
	s.mu.Unlock()
	if s.Session == nil {
		return nil, &reach.Error{Kind: reach.KindProtocolUnsupported, Detail: "no session scripted"}
	}
	return s.Session, nil
}

// CallKeys returns every recorded call as its argv key.
func (s *Scripted) CallKeys() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, 0, len(s.Calls))
	for _, c := range s.Calls {
		out = append(out, Key(c))
	}
	return out
}

// Called reports whether any recorded call has the given argv prefix.
func (s *Scripted) Called(prefix string) bool {
	for _, k := range s.CallKeys() {
		if strings.HasPrefix(k, prefix) {
			return true
		}
	}
	return false
}
