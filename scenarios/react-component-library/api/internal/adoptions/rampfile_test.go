package adoptions

import "testing"

func TestParseRampFilePreservesUnmanagedBytes(t *testing.T) {
	raw := "/* owned */\n:root { --color-surface: white; }\n" + tokenRampBegin + "\n  --space-sm: 1rem;\n" + tokenRampEnd + "\n/* trailing */"
	parsed, err := parseRampFile(raw)
	if err != nil {
		t.Fatal(err)
	}
	parsed.managed = "  --space-md: 2rem;"
	got := parsed.render()
	if got != "/* owned */\n:root { --color-surface: white; }\n"+tokenRampBegin+"\n  --space-md: 2rem;\n"+tokenRampEnd+"\n/* trailing */" {
		t.Fatalf("render changed unmanaged bytes: %q", got)
	}
}

func TestParseRampFileRejectsUnpairedMarkers(t *testing.T) {
	if _, err := parseRampFile(tokenRampBegin); err == nil {
		t.Fatal("expected unpaired marker error")
	}
	if _, err := parseRampFile(tokenRampEnd + tokenRampBegin); err == nil {
		t.Fatal("expected out-of-order marker error")
	}
}
