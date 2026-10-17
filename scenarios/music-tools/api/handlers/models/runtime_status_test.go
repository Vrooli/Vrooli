package models

import (
	"testing"

	domain "music-tools/internal/models"
)

func TestUnprobedDeclaredModelDoesNotClaimMissingInstallation(t *testing.T) {
	actual := wire(domain.Model{ID: "stabilityai/stable-audio-3-medium", Installed: false, Licence: domain.LicenceRestricted})
	if actual.Status != "runtime_unverified" || actual.LicenseLane != "restricted" {
		t.Fatalf("unprobed declaration falsely classified: %+v", actual)
	}
}
