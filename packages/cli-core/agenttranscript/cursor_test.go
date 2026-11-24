package agenttranscript

import (
	"reflect"
	"testing"
)

func TestParseCursorConversationUsesNativeHeaderOrderAndBubbleIDs(t *testing.T) {
	composer := []byte(`{"composerId":"composer-native-42","fullConversationHeadersOnly":[{"bubbleId":"h1","type":1},{"bubbleId":"a1","type":2},{"bubbleId":"tool","type":3},{"bubbleId":"missing","type":1}]}`)
	bubbles := map[string][]byte{
		"h1":   []byte(`{"type":1,"text":"question"}`),
		"a1":   []byte(`{"type":2,"text":"answer"}`),
		"tool": []byte(`{"text":"tool record is not a human or assistant bubble"}`),
	}
	want := []Message{
		{Role: "user", Text: "question", SessionID: "composer-native-42"},
		{Role: "assistant", Text: "answer", SessionID: "composer-native-42"},
	}
	if got := ParseCursorConversation(composer, bubbles); !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseCursorConversation() = %#v, want %#v", got, want)
	}
	if got := CursorConversationSessionID(composer); got != "composer-native-42" {
		t.Fatalf("CursorConversationSessionID() = %q", got)
	}
}

func TestParseCursorConversationRejectsMalformedOrUnidentifiedComposer(t *testing.T) {
	if got := ParseCursorConversation([]byte(`{"composerId":"composer-1","fullConversationHeadersOnly":[`), nil); len(got) != 0 {
		t.Fatalf("malformed Cursor document produced messages: %#v", got)
	}
	if got := ParseCursorConversation([]byte(`{"fullConversationHeadersOnly":[{"bubbleId":"h","type":1}]}`), map[string][]byte{"h": []byte(`{"text":"no identity"}`)}); len(got) != 0 {
		t.Fatalf("unidentified Cursor document produced messages: %#v", got)
	}
}

func TestParseCursorConversationForIDReturnsIdentityAndMessagesFromOneParse(t *testing.T) {
	composer := []byte(`{"composerId":" composer-1 ","fullConversationHeadersOnly":[{"bubbleId":"bubble-1","type":2}]}`)
	bubbles := map[string][]byte{"bubble-1": []byte(`{"text":"assistant reply"}`)}
	gotID, gotMessages, ok := ParseCursorConversationForID(composer, bubbles, "composer-1")
	if !ok {
		t.Fatal("valid composer document was not parsed")
	}
	if gotID != "composer-1" {
		t.Fatalf("session ID=%q want normalized composer-1", gotID)
	}
	want := []Message{{Role: "assistant", Text: "assistant reply", SessionID: " composer-1 "}}
	if !reflect.DeepEqual(gotMessages, want) {
		t.Fatalf("messages=%#v want %#v", gotMessages, want)
	}
	if messages := ParseCursorConversation(composer, bubbles); !reflect.DeepEqual(messages, want) {
		t.Fatalf("compatibility parser messages=%#v want %#v", messages, want)
	}
	if _, _, ok := ParseCursorConversationForID(composer, bubbles, "other-composer"); ok {
		t.Fatal("mismatched expected composer ID was accepted")
	}
}
