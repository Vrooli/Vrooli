package hub

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Attachment notes: while viewing an attachment the operator can pin a note to
// it ("Hair is too glossy" on character.glb). They travel with the answer
// (docs/concepts/DECISION_PACKETS.md, "Notes on attachments").

const (
	maxAttachmentNotes    = 24
	maxAttachmentNoteRune = 2_000
)

// AskAttachmentNote is an operator note pinned to one attachment and, for a
// folder, optionally to one file in it.
type AskAttachmentNote struct {
	// Attachment is the attachment's name.
	Attachment string `json:"attachment"`
	// Path is a folder entry's relative path; empty for a file or a whole folder.
	Path string `json:"path,omitempty"`
	Note string `json:"note"`
}

// target is how the note names what it is about: "name" or "name/path".
func (n AskAttachmentNote) target() string {
	if n.Path == "" {
		return n.Attachment
	}
	return n.Attachment + "/" + n.Path
}

// normalizeAttachmentNotes validates the operator's notes against the ask's
// attachments and returns them in the order given, without empty ones. A note
// naming an attachment (or a folder entry) the ask does not have is refused.
func normalizeAttachmentNotes(raw []AskAttachmentNote, attachments []AskAttachment) ([]AskAttachmentNote, error) {
	if len(raw) > maxAttachmentNotes {
		return nil, fmt.Errorf("%w: at most %d attachment notes are allowed, got %d", ErrInvalidArgument, maxAttachmentNotes, len(raw))
	}
	byName := make(map[string]AskAttachment, len(attachments))
	for _, attachment := range attachments {
		byName[attachment.Name] = attachment
	}
	var out []AskAttachmentNote
	seen := make(map[string]bool, len(raw))
	for _, note := range raw {
		note.Attachment, note.Path, note.Note = strings.TrimSpace(note.Attachment), strings.TrimSpace(note.Path), strings.TrimSpace(note.Note)
		attachment, ok := byName[note.Attachment]
		if !ok {
			return nil, fmt.Errorf("%w: attachment note names %q, which this ask does not have", ErrInvalidArgument, note.Attachment)
		}
		if err := checkAttachmentNotePath(note, attachment); err != nil {
			return nil, err
		}
		if err := checkText("attachment note", note.Note, 0, maxAttachmentNoteRune); err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidArgument, err.Error())
		}
		if note.Note == "" {
			continue
		}
		if seen[note.target()] {
			return nil, fmt.Errorf("%w: more than one note for %q", ErrInvalidArgument, note.target())
		}
		seen[note.target()] = true
		out = append(out, note)
	}
	return out, nil
}

func checkAttachmentNotePath(note AskAttachmentNote, attachment AskAttachment) error {
	if note.Path == "" {
		return nil
	}
	if attachment.Kind != AttachmentFolder {
		return fmt.Errorf("%w: attachment %q is a file, so a note on it takes no path", ErrInvalidArgument, attachment.Name)
	}
	for _, entry := range attachment.Entries {
		if entry.Path == note.Path {
			return nil
		}
	}
	return fmt.Errorf("%w: attachment note names %q, which is not a file of folder %q", ErrInvalidArgument, note.Path, attachment.Name)
}

func encodeAttachmentNotes(notes []AskAttachmentNote) string {
	if len(notes) == 0 {
		return ""
	}
	encoded, _ := json.Marshal(notes)
	return string(encoded)
}

func decodeAttachmentNotes(encoded string) []AskAttachmentNote {
	if strings.TrimSpace(encoded) == "" {
		return nil
	}
	var notes []AskAttachmentNote
	if json.Unmarshal([]byte(encoded), &notes) != nil {
		return nil
	}
	return notes
}

// AttachmentNotePayload renders notes as plain JSON-able values for event
// payloads (see ResponsePayload).
func AttachmentNotePayload(notes []AskAttachmentNote) []any {
	out := make([]any, 0, len(notes))
	for _, note := range notes {
		out = append(out, map[string]any{"attachment": note.Attachment, "path": note.Path, "note": note.Note})
	}
	return out
}
