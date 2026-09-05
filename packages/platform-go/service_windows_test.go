//go:build windows

package platform

import "testing"

func TestParseSchtasksCSVFixture(t *testing.T) {
	// Fixture shape captured from `schtasks /Query /FO CSV /NH`: the localized
	// table renderer is avoided and the third CSV field is the state column.
	raw := `"\\VrooliAutoheal","N/A","Running"` + "\r\n"
	if got := parseSchtasksState(raw, nil); got != ServiceStateRunning {
		t.Fatalf("state = %q, want running", got)
	}
	ready := `"\\VrooliAutoheal","N/A","Ready"` + "\r\n"
	if got := parseSchtasksState(ready, nil); got != ServiceStateStopped {
		t.Fatalf("ready state = %q, want stopped", got)
	}
}

func TestParseSchtasksNumericStateWithLocalizedFixture(t *testing.T) {
	// Fixture shape captured from `schtasks /Query /FO LIST /V` on a
	// non-English Windows host: the field labels and state text are localized,
	// but the numeric STATE code remains stable.
	raw := "Nom de la tâche : \\VrooliAutoheal\r\nÉTAT : 4 (en cours d’exécution)\r\n"
	if got := parseSchtasksState(raw, nil); got != ServiceStateRunning {
		t.Fatalf("localized running state = %q, want running", got)
	}
}
