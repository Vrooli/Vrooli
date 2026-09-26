package sse

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseStream_DataOnly(t *testing.T) {
	stream := "data: hello\n\ndata: world\n\n"
	frames := ParseStream(strings.NewReader(stream))
	if len(frames) != 2 {
		t.Fatalf("got %d frames, want 2", len(frames))
	}
	if frames[0].Event != "" {
		t.Errorf("frame[0].Event = %q, want empty (default message)", frames[0].Event)
	}
	if string(frames[0].Data) != "hello" {
		t.Errorf("frame[0].Data = %q, want hello", string(frames[0].Data))
	}
	if string(frames[1].Data) != "world" {
		t.Errorf("frame[1].Data = %q, want world", string(frames[1].Data))
	}
}

func TestParseStream_NamedEvent(t *testing.T) {
	stream := "event: exit\ndata: {\"code\":0}\n\nevent: end\ndata: \n\n"
	frames := ParseStream(strings.NewReader(stream))
	if len(frames) != 2 {
		t.Fatalf("got %d frames, want 2 (frames=%v)", len(frames), frames)
	}
	if frames[0].Event != "exit" {
		t.Errorf("frame[0].Event = %q, want exit", frames[0].Event)
	}
	if !bytes.Contains(frames[0].Data, []byte(`"code":0`)) {
		t.Errorf("frame[0].Data = %q, want to contain code:0", string(frames[0].Data))
	}
	if frames[1].Event != "end" {
		t.Errorf("frame[1].Event = %q, want end", frames[1].Event)
	}
}

func TestParseStream_MultilineData(t *testing.T) {
	stream := "event: msg\ndata: line1\ndata: line2\n\n"
	frames := ParseStream(strings.NewReader(stream))
	if len(frames) != 1 {
		t.Fatalf("got %d frames, want 1", len(frames))
	}
	if string(frames[0].Data) != "line1\nline2" {
		t.Errorf("got %q, want %q", string(frames[0].Data), "line1\nline2")
	}
}

func TestParseStream_IgnoresComments(t *testing.T) {
	stream := ": this is a comment\ndata: hi\n\n"
	frames := ParseStream(strings.NewReader(stream))
	if len(frames) != 1 {
		t.Fatalf("got %d frames, want 1", len(frames))
	}
	if string(frames[0].Data) != "hi" {
		t.Errorf("got %q, want hi", string(frames[0].Data))
	}
}

func TestParseStream_TrailingFrameWithoutBlankLine(t *testing.T) {
	stream := "event: tail\ndata: x"
	frames := ParseStream(strings.NewReader(stream))
	if len(frames) != 1 {
		t.Fatalf("trailing frame should still dispatch on EOF; got %d", len(frames))
	}
}
