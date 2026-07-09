package notes

import (
	"testing"
	"time"

	"template-manager/internal/notes"

	"github.com/stretchr/testify/require"
)

// TestDomainToProto_PopulatedFields pins the field-by-field mapping
// from the internal notes.Note shape to the wire notesv1.Note shape.
// Catches the regression class where a schema field added to the
// proto stays unwired in domainToProto and silently emits the zero
// value to every client.
func TestDomainToProto_PopulatedFields(t *testing.T) {
	created := time.Date(2026, 5, 1, 12, 30, 45, 123_456_789, time.UTC)
	updated := time.Date(2026, 5, 2, 9, 15, 0, 987_654_321, time.UTC)

	got := domainToProto(notes.Note{
		ID:             "abc",
		Title:          "title",
		Body:           "body",
		CreatedAt:      created,
		UpdatedAt:      updated,
		AttachmentKeys: []string{"notes/abc/a.txt"},
	})

	require.NotNil(t, got)
	require.Equal(t, "abc", got.Id)
	require.Equal(t, "title", got.Title)
	require.Equal(t, "body", got.Body)
	require.True(t, created.Equal(got.CreatedAt.AsTime()))
	require.True(t, updated.Equal(got.UpdatedAt.AsTime()))
	require.Equal(t, []string{"notes/abc/a.txt"}, got.AttachmentKeys)
}

// TestDomainToProto_NormalizesTimestampsToUTC pins the timezone
// contract: timestamps emitted on the wire are always in UTC,
// regardless of the timezone the in-memory time.Time carries. The
// sqlite repository reads timestamps as UTC, but a future repository
// backend (or a hand-constructed Note in tests) could carry a localised
// time.Time — the adapter must normalise rather than emit a `+05:30`
// offset that downstream parsers may or may not accept.
func TestDomainToProto_NormalizesTimestampsToUTC(t *testing.T) {
	istLoc := time.FixedZone("IST", int((5*time.Hour + 30*time.Minute).Seconds()))
	createdLocal := time.Date(2026, 5, 1, 12, 0, 0, 0, istLoc)

	got := domainToProto(notes.Note{
		ID:        "tz",
		CreatedAt: createdLocal,
		UpdatedAt: createdLocal,
	})

	require.Equal(t, time.UTC, got.CreatedAt.AsTime().Location())
	require.True(t, got.CreatedAt.AsTime().Equal(createdLocal),
		"round-tripped timestamp must equal source instant")
}

// TestDomainToProto_ZeroTimestamps pins the zero-time edge: a Note
// with zero-valued CreatedAt / UpdatedAt (e.g. an in-memory fixture
// constructed without timestamps) emits the canonical
// "0001-01-01T00:00:00Z" string rather than panicking. The repository
// backfills timestamps in production, so this is the test-fixture path
// — callers must see a deterministic, parseable shape.
func TestDomainToProto_ZeroTimestamps(t *testing.T) {
	got := domainToProto(notes.Note{ID: "zero"})

	require.Equal(t, "zero", got.Id)
	require.True(t, got.CreatedAt.AsTime().IsZero())
	require.True(t, got.UpdatedAt.AsTime().IsZero())
}

// TestDomainToProto_EmptyOptionalFields pins the proto3-default
// behaviour for optional fields: an empty Body becomes "" on the wire
// (not nil, not omitted), matching the proto3 default-string semantics
// that downstream `fromJson` decoders rely on.
func TestDomainToProto_EmptyOptionalFields(t *testing.T) {
	got := domainToProto(notes.Note{ID: "id-only"})

	require.Equal(t, "", got.Title)
	require.Equal(t, "", got.Body)
}
