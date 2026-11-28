package ai

import "testing"

// TestOutputIsRasterFollowsTheDeclaredOutputPort pins which operations the
// image safety review applies to. Reviewing a caption's JSON as an image failed
// every caption job with "not a decodable raster", so the scope is the declared
// output port, not a list of exempt operation names.
func TestOutputIsRasterFollowsTheDeclaredOutputPort(t *testing.T) {
	cases := map[string]bool{
		"text_to_image":    true,
		"depth_map":        true,
		"segment":          true, // mask: PNG bytes
		"face_swap":        true,
		"caption":          false, // text
		"object_detection": false, // data
		"tagging":          false,
		"embedding":        false,
		"scene_geometry":   false,
		"image_to_object":  false, // mesh
		"not-an-operation": true,  // undeclared: fail closed, keep the review on
	}
	for operation, want := range cases {
		if got := outputIsRaster(operation); got != want {
			t.Errorf("outputIsRaster(%q) = %v, want %v", operation, got, want)
		}
	}
}

// TestOutputFormatNamesJSONOutputsAsJSON: a caption, detection list or embedding is
// a JSON document and must not be stored under an image name and type.
func TestOutputFormatNamesJSONOutputsAsJSON(t *testing.T) {
	mime, ext := outputFormat([]byte("{\n  \"caption\": \"a guinea pig\"\n}\n"), nil)
	if mime != "application/json" || ext != ".json" {
		t.Fatalf("json output labelled %s %s", mime, ext)
	}
	if mime, ext := outputFormat([]byte("{not json"), nil); ext == ".json" {
		t.Fatalf("malformed text labelled %s %s", mime, ext)
	}
	if _, ext := outputFormat([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, nil); ext != ".png" {
		t.Fatalf("png labelled %s", ext)
	}
}
