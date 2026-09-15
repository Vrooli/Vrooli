package peerrecord

import (
	"os"
	"testing"
	"time"
)

func TestWriteReadRoundTripUsesTypedJSONContract(t *testing.T) {
	home := t.TempDir()
	want := Record{
		Scenario:      "demo",
		Instance:      "instance-1",
		Tier:          1,
		OwnerPID:      os.Getpid(),
		StartedAt:     time.Unix(123, 456).UTC(),
		Ports:         map[string]int{"api": 17573},
		AuthTokenPath: "/tmp/demo-token",
	}
	if err := Write(home, want); err != nil {
		t.Fatal(err)
	}

	got, err := Read(home, want.Scenario)
	if err != nil {
		t.Fatal(err)
	}
	if got.SchemaVersion != SchemaVersion || got.Scenario != want.Scenario || got.Instance != want.Instance || got.OwnerPID != want.OwnerPID || !got.StartedAt.Equal(want.StartedAt) {
		t.Fatalf("round-trip record = %#v, want %#v", got, want)
	}
	if got.Ports["api"] != want.Ports["api"] || got.AuthTokenPath != want.AuthTokenPath {
		t.Fatalf("round-trip payload = %#v, want %#v", got, want)
	}
}

func TestReadRejectsTrailingJSONValue(t *testing.T) {
	home := t.TempDir()
	if err := Write(home, Record{Scenario: "demo", OwnerPID: os.Getpid()}); err != nil {
		t.Fatal(err)
	}
	path := Path(home, "demo")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, []byte("{}")...)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(home, "demo"); err == nil {
		t.Fatal("expected trailing JSON to be rejected")
	}
}
