package ingest

import "testing"

func TestChunkTextDefaults(t *testing.T) {
	chunks := ChunkText("", 10, 2, 0)
	if len(chunks) != 0 {
		t.Fatalf("expected no chunks for empty input")
	}

	single := ChunkText("  hello world  ", 0, 0, 0)
	if len(single) != 1 || single[0] != "hello world" {
		t.Fatalf("expected trimmed single chunk, got %v", single)
	}
}

func TestChunkTextOverlapHandling(t *testing.T) {
	chunks := ChunkText("abcdefghij", 4, 10, 0)
	if len(chunks) == 0 {
		t.Fatalf("expected chunks")
	}
}

func TestHashDocumentStable(t *testing.T) {
	one := HashDocument("ns", "hello\r\nworld")
	two := HashDocument("ns", "hello\nworld")
	if one != two {
		t.Fatalf("expected normalized hash to match")
	}
}

func TestRecordIDForChunkStable(t *testing.T) {
	id := RecordIDForChunk("ns", "doc", 1, "chunk")
	if id == "" {
		t.Fatalf("expected record id")
	}
}
