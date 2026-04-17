package phases

import "testing"

func TestDeriveStatus(t *testing.T) {
	t.Run("returns failed when execution failed", func(t *testing.T) {
		if got := deriveStatus(nil, assertErr{}, ""); got != "failed" {
			t.Fatalf("deriveStatus() = %q, want failed", got)
		}
	})

	t.Run("returns failed when failure class is set", func(t *testing.T) {
		if got := deriveStatus(nil, nil, FailureClassSystem); got != "failed" {
			t.Fatalf("deriveStatus() = %q, want failed", got)
		}
	})

	t.Run("returns skipped only when all meaningful observations are skips", func(t *testing.T) {
		obs := []Observation{
			NewSectionObservation("🔍", "Checks"),
			NewSkipObservation("python not detected"),
			NewSkipObservation("bats suite not found"),
		}
		if got := deriveStatus(obs, nil, ""); got != "skipped" {
			t.Fatalf("deriveStatus() = %q, want skipped", got)
		}
	})

	t.Run("returns passed when skips are mixed with passing observations", func(t *testing.T) {
		obs := []Observation{
			NewSectionObservation("🔍", "Checks"),
			NewSuccessObservation("go tests passed"),
			NewSkipObservation("python not detected"),
		}
		if got := deriveStatus(obs, nil, ""); got != "passed" {
			t.Fatalf("deriveStatus() = %q, want passed", got)
		}
	})
}

type assertErr struct{}

func (assertErr) Error() string { return "boom" }
