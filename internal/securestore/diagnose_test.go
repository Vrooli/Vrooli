package securestore

import (
	"errors"
	"testing"
)

type diagnosticStore struct {
	getErr    error
	putErr    error
	deleteErr error
	puts      int
}

func (s *diagnosticStore) Put(string, string, string) error {
	s.puts++
	return s.putErr
}

func (s *diagnosticStore) Get(string, string) (string, error) {
	return "", s.getErr
}

func (s *diagnosticStore) Delete(string, string) error { return s.deleteErr }
func (*diagnosticStore) AdapterName() string           { return "test-backend" }

func TestDiagnoseSeparatesReadableFromWritable(t *testing.T) {
	store := &diagnosticStore{getErr: ErrNotFound, putErr: ErrUnavailable}
	diagnosis := diagnoseStore(store, true)

	if !diagnosis.Available || diagnosis.Condition != "available" {
		t.Fatalf("read diagnosis = %+v, want available", diagnosis)
	}
	if diagnosis.Writable || diagnosis.WriteCondition != "unavailable" {
		t.Fatalf("write diagnosis = %+v, want unavailable", diagnosis)
	}
	if diagnosis.WriteExplanation == "" || diagnosis.WriteFix == "" {
		t.Fatalf("write diagnosis = %+v, want explanation and fix", diagnosis)
	}
	if store.puts != 1 {
		t.Fatalf("write probes = %d, want one", store.puts)
	}
}

func TestDiagnoseDoesNotWriteWhenBackendIsAbsent(t *testing.T) {
	store := &diagnosticStore{getErr: ErrAbsent}
	diagnosis := diagnoseStore(store, true)

	if diagnosis.Available || diagnosis.Condition != "absent" {
		t.Fatalf("read diagnosis = %+v, want absent", diagnosis)
	}
	if diagnosis.WriteCondition != "absent" || diagnosis.Writable {
		t.Fatalf("write diagnosis = %+v, want absent", diagnosis)
	}
	if store.puts != 0 {
		t.Fatalf("write probes = %d, want none for absent backend", store.puts)
	}
}

func TestDiagnoseMapsUnexpectedWriteFailureToUnavailable(t *testing.T) {
	store := &diagnosticStore{getErr: ErrNotFound, putErr: errors.New("permission denied")}
	diagnosis := diagnoseStore(store, true)
	if diagnosis.WriteCondition != "unavailable" {
		t.Fatalf("write condition = %q, want unavailable", diagnosis.WriteCondition)
	}
}
