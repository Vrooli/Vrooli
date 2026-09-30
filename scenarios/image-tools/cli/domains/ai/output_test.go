package ai

import "testing"

func TestPathForContentCorrectsAMismatchedExtension(t *testing.T) {
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 2048 2048"/>`)
	png := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
	jpg := []byte("\xff\xd8\xff\xe0\x00\x10JFIF")
	cases := []struct {
		out  string
		data []byte
		want string
	}{
		{"/out/lyre.png", svg, "/out/lyre.svg"},
		{"/out/lyre.img", svg, "/out/lyre.svg"},
		{"/out/lyre", svg, "/out/lyre.svg"},
		{"/out/lyre.svg", svg, "/out/lyre.svg"},
		{"/out/lyre.png", png, "/out/lyre.png"},
		{"/out/lyre.jpeg", jpg, "/out/lyre.jpeg"},
		{"/out/lyre.png", jpg, "/out/lyre.jpg"},
		{"/out/notes.dat", []byte("plain text"), "/out/notes.dat"},
	}
	for _, tc := range cases {
		if got := pathForContent(tc.out, tc.data); got != tc.want {
			t.Errorf("pathForContent(%q) = %q, want %q", tc.out, got, tc.want)
		}
	}
}
