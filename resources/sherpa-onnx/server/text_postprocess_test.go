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
	tests := []struct {
		name, input, want string
	}{
		{name: "ASCII comma", input: "hello,world.", want: "hello, world."},
		{name: "Chinese punctuation in Latin text", input: "hello，world。", want: "hello, world."},
		{name: "English transcript defect", input: "The quick brown,fox jumps.", want: "The quick brown, fox jumps."},
		{name: "punctuation cluster", input: "Really?!Next;Then:Now.", want: "Really?! Next; Then: Now."},
		{name: "numeric grouping and decimal", input: "1,000.25 costs 2,500.00.", want: "1,000.25 costs 2,500.00."},
		{name: "abbreviations and domains", input: "U.S. e.g.example.com", want: "U.S. e.g.example.com"},
		{name: "CJK remains unspaced", input: "你好，世界。", want: "你好,世界."},
		{name: "mixed scripts", input: "你好，world。 café,naïve", want: "你好,world. café, naïve"},
		{name: "existing whitespace", input: "hello, world.", want: "hello, world."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizePunctuation(tt.input); got != tt.want {
				t.Fatalf("normalizePunctuation(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
