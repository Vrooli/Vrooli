package assets

import "testing"

func TestResolveMimeSniffsContentWhenTheExtensionIsUnknown(t *testing.T) {
	cases := map[string]struct {
		content []byte
		want    string
	}{
		"svg root":        {[]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1 1"/>`), "image/svg+xml"},
		"svg with prolog": {[]byte("<?xml version=\"1.0\"?>\n<svg xmlns=\"http://www.w3.org/2000/svg\"/>"), "image/svg+xml"},
		"png":             {[]byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"), "image/png"},
		"jpeg":            {[]byte("\xff\xd8\xff\xe0\x00\x10JFIF"), "image/jpeg"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := resolveMime("", "candidate.bin", tc.content)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("mime = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolveMimeRejectsContentThatIsNotAnImage(t *testing.T) {
	if _, err := resolveMime("", "notes.bin", []byte("plain text, not an image")); err == nil {
		t.Fatal("want an error for non-image content with no known extension")
	}
}

func TestResolveMimeKeepsTheSuppliedTypeAndTheExtension(t *testing.T) {
	if got, _ := resolveMime("image/webp", "x.bin", []byte("<svg/>")); got != "image/webp" {
		t.Fatalf("supplied mime = %q, want image/webp", got)
	}
	if got, _ := resolveMime("", "x.png", []byte("<svg/>")); got != "image/png" {
		t.Fatalf("extension mime = %q, want image/png", got)
	}
}
