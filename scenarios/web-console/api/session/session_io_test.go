package session

import (
	"bytes"
	"testing"
	"unicode/utf8"
)

func TestSplitCompleteUTF8_MatchesStdlib(t *testing.T) {
	cases := [][]byte{
		[]byte("plain text"),
		{0xe2},
		{0xe2, 0x82},
		{0xe2, 0x82, 0xac},
		{0xf0, 0x9f, 0x8c},
		{0xf0, 0x9f, 0x8c, 0x8d},
		{0x80, 0x81}, // orphaned continuation bytes pass through
	}
	for _, input := range cases {
		wantAt := len(input)
		for i := len(input) - 1; i >= 0 && i >= max(0, len(input)-4); i-- {
			if utf8.RuneStart(input[i]) && !utf8.FullRune(input[i:]) {
				wantAt = i
				break
			}
		}
		got, remainder := splitCompleteUTF8(input)
		if !bytes.Equal(got, input[:wantAt]) || !bytes.Equal(remainder, input[wantAt:]) {
			t.Fatalf("splitCompleteUTF8(%#v) = (%#v, %#v), want (%#v, %#v)", input, got, remainder, input[:wantAt], input[wantAt:])
		}
	}
}
