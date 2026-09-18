package session

import (
	"strings"
	"testing"

	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"
)

// `session list` output has to be usable as input to the commands its own
// retrieval hints name. A shortened id is not: `session get` rejects it. The
// row therefore carries the full id.
func TestSessionRowsCarryFullIDForReuse(t *testing.T) {
	const id = "83f2b8ad-a8e3-4f0f-bfcb-f869bb180dd8"
	rows := sessionRows([]*sessionsv1.Session{{
		Id: id, Shell: "/bin/bash", Backend: "persistent", Cols: 80, Rows: 24,
		Origin: sessionsv1.SessionOrigin_SESSION_ORIGIN_UI,
	}})
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	if !strings.Contains(rows[0], id) {
		t.Fatalf("row %q does not contain the full session id %q", rows[0], id)
	}
}

// The label is the only human-meaningful name a session has. Omitting it made
// a fully labelled workspace list as a column of identical shells.
func TestSessionRowsShowDisplayLabel(t *testing.T) {
	rows := sessionRows([]*sessionsv1.Session{{
		Id: "83f2b8ad-a8e3-4f0f-bfcb-f869bb180dd8", Shell: "/bin/bash",
		Backend: "persistent", Cols: 80, Rows: 24,
		Origin: sessionsv1.SessionOrigin_SESSION_ORIGIN_UI, DisplayLabel: "Route solver",
	}})
	if !strings.Contains(rows[0], "Route solver") {
		t.Fatalf("row %q omits the session label", rows[0])
	}
}

// An unlabelled session must still render, without an empty label field.
func TestSessionRowsOmitEmptyLabel(t *testing.T) {
	rows := sessionRows([]*sessionsv1.Session{{
		Id: "83f2b8ad-a8e3-4f0f-bfcb-f869bb180dd8", Shell: "/bin/bash",
		Backend: "persistent", Cols: 80, Rows: 24,
		Origin: sessionsv1.SessionOrigin_SESSION_ORIGIN_UI,
	}})
	if strings.Contains(rows[0], "label=") {
		t.Fatalf("row %q renders an empty label field", rows[0])
	}
}
