package intake

import "testing"

func TestSniffingUsesContentAndDistinguishesZipFamilies(t *testing.T) { // [REQ:DOC-P0-002]
	cases := map[string]string{"docx": "PK\x03\x04 [Content_Types].xml word/document.xml", "xlsx": "PK\x03\x04 [Content_Types].xml xl/workbook.xml", "pptx": "PK\x03\x04 [Content_Types].xml ppt/presentation.xml", "epub": "PK\x03\x04 application/epub+zip"}
	want := map[string]string{"docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation", "epub": "application/epub+zip"}
	for name, content := range cases {
		if got := sniff([]byte(content)); got != want[name] {
			t.Errorf("%s: got %q want %q", name, got, want[name])
		}
	}
	if got := sniff([]byte("PK\x03\x04 ordinary zip")); got != "application/zip" {
		t.Fatalf("plain zip sniff=%q", got)
	}
}
