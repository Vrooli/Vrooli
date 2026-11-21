package resources

import (
	"testing"
)

func TestOpenRGBELFVersionStringsAreBounded(t *testing.T) {
	table := []byte("\x00GLIBC_2.39\x00GLIBCXX_3.4.32\x00")
	for _, tc := range []struct {
		offset uint32
		want   string
	}{{1, "GLIBC_2.39"}, {12, "GLIBCXX_3.4.32"}} {
		got, err := openRGBELFString(table, tc.offset)
		if err != nil || got != tc.want {
			t.Fatalf("version=%q error=%v", got, err)
		}
	}
	if _, err := openRGBELFString(table, 999); err == nil {
		t.Fatal("out-of-table metadata accepted")
	}
	if _, err := openRGBELFString([]byte("unterminated"), 0); err == nil {
		t.Fatal("unterminated metadata accepted")
	}
}
