package main

import "testing"

func TestRestoreCapitalization(t *testing.T) {
	if got := restoreCapitalization("HELLO WORLD. THIS IS FINE!"); got != "Hello world. This is fine!" {
		t.Fatalf("got %q", got)
	}
	if got := restoreCapitalization("OpenAI is mixed case"); got != "OpenAI is mixed case" {
		t.Fatalf("mixed-case text changed to %q", got)
	}
}

func TestNormalizePunctuation(t *testing.T) {
	if got := normalizePunctuation("hello，world。"); got != "hello,world." {
		t.Fatalf("got %q", got)
	}
}
