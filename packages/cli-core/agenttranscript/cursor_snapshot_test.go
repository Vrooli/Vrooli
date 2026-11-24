package agenttranscript

import (
	"encoding/json"
	"reflect"
	"testing"
)

type cursorKVTestRow struct {
	key   string
	value []byte
}

type cursorKVTestRows struct {
	rows   []cursorKVTestRow
	index  int
	closed bool
	err    error
}

func (r *cursorKVTestRows) Next() bool { return r.index < len(r.rows) }

func (r *cursorKVTestRows) Scan(dest ...any) error {
	row := r.rows[r.index]
	r.index++
	*dest[0].(*string) = row.key
	*dest[1].(*[]byte) = append([]byte(nil), row.value...)
	return nil
}

func (r *cursorKVTestRows) Err() error { return r.err }

func (r *cursorKVTestRows) Close() error { r.closed = true; return nil }

func TestDecodeCursorConversationSnapshotSupportsNativeObjectsAndByteSlices(t *testing.T) {
	composer := []byte(`{"composerId":"composer-1","fullConversationHeadersOnly":[{"bubbleId":"bubble-1","type":1}]}`)
	bubble := []byte(`{"text":"hello from Cursor"}`)
	want := []Message{{Role: "user", Text: "hello from Cursor", SessionID: "composer-1"}}

	objectSnapshot, err := DecodeCursorConversationSnapshot([]byte(`{"composer":{"composerId":"composer-1","fullConversationHeadersOnly":[{"bubbleId":"bubble-1","type":1}]},"bubbles":{"bubble-1":{"text":"hello from Cursor"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := ParseCursorConversation(objectSnapshot.Composer, objectSnapshot.Bubbles); !reflect.DeepEqual(got, want) {
		t.Fatalf("object snapshot messages=%#v want %#v", got, want)
	}

	encoded, err := json.Marshal(struct {
		Composer []byte            `json:"composer"`
		Bubbles  map[string][]byte `json:"bubbles"`
	}{Composer: composer, Bubbles: map[string][]byte{"bubble-1": bubble}})
	if err != nil {
		t.Fatal(err)
	}
	byteSnapshot, err := DecodeCursorConversationSnapshot(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if got := ParseCursorConversation(byteSnapshot.Composer, byteSnapshot.Bubbles); !reflect.DeepEqual(got, want) {
		t.Fatalf("byte-slice snapshot messages=%#v want %#v", got, want)
	}
}

func TestDecodeCursorWorkspaceSnapshotDecodesEachComposer(t *testing.T) {
	workspace := []byte(`{"composers":{"composer-1":{"composer":{"composerId":"composer-1","fullConversationHeadersOnly":[{"bubbleId":"bubble-1","type":"assistant"}]},"bubbles":{"bubble-1":{"text":"workspace message"}}}}}`)
	snapshots, err := DecodeCursorWorkspaceSnapshot(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshots) != 1 {
		t.Fatalf("workspace composers=%d want 1", len(snapshots))
	}
	got := ParseCursorConversation(snapshots["composer-1"].Composer, snapshots["composer-1"].Bubbles)
	want := []Message{{Role: "assistant", Text: "workspace message", SessionID: "composer-1"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("workspace messages=%#v want %#v", got, want)
	}
}

func TestCursorWorkspaceSnapshotBuilderGroupsNativeRowsOnce(t *testing.T) {
	builder := NewCursorWorkspaceSnapshotBuilder()
	composer := []byte(`{"composerId":"composer-1"}`)
	bubble := []byte(`{"text":"answer"}`)
	builder.AddKV("composerData:composer-1", composer)
	builder.AddKV("bubbleId:composer-1:bubble:part", bubble)
	builder.AddKV("bubbleId:orphan:bubble", []byte(`{"text":"ignored"}`))
	builder.AddKV("unknown:key", []byte(`{}`))

	workspace := builder.WorkspaceSnapshot()
	if len(workspace.Composers) != 1 {
		t.Fatalf("workspace composers=%d want 1", len(workspace.Composers))
	}
	got := workspace.Composers["composer-1"]
	if string(got.Composer) != string(composer) || string(got.Bubbles["bubble:part"]) != string(bubble) {
		t.Fatalf("grouped workspace snapshot=%#v", got)
	}
	conversation := builder.ConversationSnapshot("composer-1")
	if string(conversation.Composer) != string(composer) || string(conversation.Bubbles["bubble:part"]) != string(bubble) {
		t.Fatalf("grouped conversation snapshot=%#v", conversation)
	}
}

func TestCursorWorkspaceSnapshotBuilderAddsBoundedRowsAndClosesSource(t *testing.T) {
	rows := &cursorKVTestRows{rows: []cursorKVTestRow{{key: "composerData:c1", value: []byte(`{"composerId":"c1"}`)}}}
	builder := NewCursorWorkspaceSnapshotBuilder()
	size, exceeded, err := builder.AddRows(rows, 0, 1024)
	if err != nil || exceeded || !rows.closed {
		t.Fatalf("AddRows() = size %d exceeded %t err %v closed %t", size, exceeded, err, rows.closed)
	}
	if string(builder.ConversationSnapshot("c1").Composer) != `{"composerId":"c1"}` {
		t.Fatalf("AddRows() did not group row: %#v", builder.ConversationSnapshot("c1"))
	}

	limitedRows := &cursorKVTestRows{rows: []cursorKVTestRow{{key: "composerData:c2", value: []byte(`{}`)}}}
	_, exceeded, err = NewCursorWorkspaceSnapshotBuilder().AddRows(limitedRows, 0, 1)
	if err != nil || !exceeded || !limitedRows.closed {
		t.Fatalf("limited AddRows() = exceeded %t err %v closed %t", exceeded, err, limitedRows.closed)
	}
}

func TestDecodeCursorWorkspaceSnapshotPreservesRawMessageTolerance(t *testing.T) {
	workspace := []byte(`{"composers":{"composer-1":{"composer":7,"bubbles":{"bubble-1":false}}}}`)
	snapshots, err := DecodeCursorWorkspaceSnapshot(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if got := ParseCursorConversation(snapshots["composer-1"].Composer, snapshots["composer-1"].Bubbles); len(got) != 0 {
		t.Fatalf("unsupported native shapes produced messages: %#v", got)
	}
}

func TestDecodeCursorConversationSnapshotRejectsMalformedEnvelopeValues(t *testing.T) {
	for _, raw := range []string{
		`not-json`,
		`{"composer":7}`,
		`{"composer":{"composerId":"c1"},"bubbles":{"b1":"%%%"}}`,
	} {
		if _, err := DecodeCursorConversationSnapshot([]byte(raw)); err == nil {
			t.Errorf("DecodeCursorConversationSnapshot(%q) accepted malformed input", raw)
		}
	}
}
