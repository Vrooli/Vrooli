package projectmeta

import (
	"path/filepath"
	"testing"
)

func TestMaturity_Production(t *testing.T) {
	dir := t.TempDir()
	writeServiceJSON(t, dir, `{"maturity":"production"}`)
	SetStartDirForTesting(dir)
	if got := Maturity(); got != MaturityProduction {
		t.Fatalf("Maturity() = %q, want %q", got, MaturityProduction)
	}
	if IsGreenfield() {
		t.Fatalf("IsGreenfield() = true, want false for production")
	}
}

func TestMaturity_Pilot(t *testing.T) {
	dir := t.TempDir()
	writeServiceJSON(t, dir, `{"maturity":"pilot"}`)
	SetStartDirForTesting(dir)
	if got := Maturity(); got != MaturityPilot {
		t.Fatalf("Maturity() = %q, want %q", got, MaturityPilot)
	}
}

func TestMaturity_FileMissing_DefaultsToGreenfield(t *testing.T) {
	dir := t.TempDir()
	SetStartDirForTesting(dir)
	if got := Maturity(); got != MaturityGreenfield {
		t.Fatalf("Maturity() with no service.json = %q, want %q", got, MaturityGreenfield)
	}
	if !IsGreenfield() {
		t.Fatalf("IsGreenfield() = false, want true")
	}
}

func TestMaturity_AbsentField_DefaultsToGreenfield(t *testing.T) {
	dir := t.TempDir()
	writeServiceJSON(t, dir, `{}`)
	SetStartDirForTesting(dir)
	if got := Maturity(); got != MaturityGreenfield {
		t.Fatalf("Maturity() with absent field = %q, want %q", got, MaturityGreenfield)
	}
}

func TestMaturity_InvalidValue_DefaultsToGreenfield(t *testing.T) {
	dir := t.TempDir()
	writeServiceJSON(t, dir, `{"maturity":"beta"}`)
	SetStartDirForTesting(dir)
	if got := Maturity(); got != MaturityGreenfield {
		t.Fatalf("Maturity() with invalid value = %q, want %q", got, MaturityGreenfield)
	}
}

func TestMaturity_AscendsToFindServiceJSON(t *testing.T) {
	root := t.TempDir()
	writeServiceJSON(t, root, `{"maturity":"sunset"}`)
	deep := filepath.Join(root, "a", "b", "c")
	SetStartDirForTesting(deep)
	if got := Maturity(); got != MaturitySunset {
		t.Fatalf("Maturity() = %q, want %q", got, MaturitySunset)
	}
}
