package notes

import (
	"time"

	notesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/{{SCENARIO_ID}}/v1/notes"

	"{{SCENARIO_ID}}/internal/notes"
)

// domainToProto converts an internal notes.Note into the wire shape
// the notes proto declares. Times serialise as RFC3339 strings — the
// same format the sqlite repository persists, so round-tripping is
// byte-identical.
//
// Lives in the handler package by intent. The conversion is mechanical
// and only used at the transport edge; pulling it into a separate
// adapters package would create a one-import wrapper for no gain.
func domainToProto(n notes.Note) *notesv1.Note {
	return &notesv1.Note{
		Id:        n.ID,
		Title:     n.Title,
		Body:      n.Body,
		CreatedAt: n.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: n.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}
