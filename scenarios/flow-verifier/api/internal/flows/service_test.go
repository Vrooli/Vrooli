package flows

import (
	"strings"
	"testing"
)

func TestListReturnsEmptyOnEmptyRoot(t *testing.T) {
	dir := t.TempDir()
	got, err := List(dir, "")
	if err != nil {
		t.Fatalf("List on empty dir should not error, got: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty Summary slice, got %d entries", len(got))
	}
}

func TestListFlowIDFilterMissReportsError(t *testing.T) {
	dir := t.TempDir()
	_, err := List(dir, "does.not.exist")
	if err == nil {
		t.Fatal("expected error when flowID filter matches no flow")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "does.not.exist") {
		t.Fatalf("error should name the missing flow id, got: %v", err)
	}
}

func TestExplainOnEmptyRootReturnsError(t *testing.T) {
	dir := t.TempDir()
	_, err := Explain(dir, "any")
	if err == nil {
		t.Fatal("expected error when explaining a flow that does not exist")
	}
}
