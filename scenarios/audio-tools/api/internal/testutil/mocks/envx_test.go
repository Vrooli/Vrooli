package mocks_test

import (
	"testing"

	"audio-tools/internal/envx"
	"audio-tools/internal/testutil/mocks"
)

func TestFakeEnv_SatisfiesReader(t *testing.T) {
	var _ envx.Reader = mocks.NewFakeEnv(nil)
}

func TestFakeEnv_GetReturnsSeededValueAndRecordsReads(t *testing.T) {
	e := mocks.NewFakeEnv(map[string]string{"FOO": "bar"})
	if got := e.Get("FOO"); got != "bar" {
		t.Errorf("Get(FOO) = %q, want %q", got, "bar")
	}
	if got := e.Get("MISSING"); got != "" {
		t.Errorf("Get(MISSING) = %q, want empty", got)
	}
	reads := e.Reads()
	if len(reads) != 2 || reads[0] != "FOO" || reads[1] != "MISSING" {
		t.Errorf("Reads() = %v, want [FOO MISSING]", reads)
	}
}

func TestFakeEnv_SetOverwrites(t *testing.T) {
	e := mocks.NewFakeEnv(nil)
	e.Set("A", "1")
	e.Set("A", "2")
	if got := e.Get("A"); got != "2" {
		t.Errorf("Get(A) = %q, want %q", got, "2")
	}
}
