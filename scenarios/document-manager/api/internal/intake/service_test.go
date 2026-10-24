package intake_test

import (
	"testing"

	"document-manager/internal/intake"
	"document-manager/internal/intake/mocks"
)

type memoryFiles struct{ writes map[string][]byte }

func (m *memoryFiles) Put(key string, data []byte) (string, error) {
	if m.writes == nil {
		m.writes = map[string][]byte{}
	}
	m.writes[key] = append([]byte(nil), data...)
	return key, nil
}

type fakeClassifier struct{}

func (fakeClassifier) Classify(string) (intake.TypeVerdict, error) {
	return intake.TypeVerdict{PDFType: "text_based", Confidence: 0.99}, nil
}

func TestIngestIdenticalBytesStoresOneDocumentAndBlob(t *testing.T) { // [REQ:DOC-P0-001]
	repo := &mocks.Repository{}
	files := &memoryFiles{}
	svc := intake.NewService(repo, files, fakeClassifier{})
	first, duplicate, err := svc.Ingest(intake.IngestInput{Content: []byte("same bytes"), SourceName: "one.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if duplicate {
		t.Fatal("first ingest marked duplicate")
	}
	second, duplicate, err := svc.Ingest(intake.IngestInput{Content: []byte("same bytes"), SourceName: "renamed.pdf"})
	if err != nil {
		t.Fatal(err)
	}
	if !duplicate {
		t.Fatal("second ingest was not duplicate")
	}
	if first.ContentSHA256 != second.ContentSHA256 {
		t.Fatal("hash changed")
	}
	if len(repo.Documents) != 1 || len(files.writes) != 1 {
		t.Fatalf("documents=%d blobs=%d, want one each", len(repo.Documents), len(files.writes))
	}
}

func TestPDFVerdictIsStoredAtIntake(t *testing.T) { // [REQ:DOC-P0-003] [REQ:DOC-P0-004] [REQ:DOC-P0-012]
	doc, _, err := intake.NewService(&mocks.Repository{}, &memoryFiles{}, fakeClassifier{}).Ingest(intake.IngestInput{Content: []byte("%PDF-1.7 scanned"), SourceName: "misnamed.bin"})
	if err != nil {
		t.Fatal(err)
	}
	if doc.DetectedMIME != "application/pdf" || doc.PDFType != "text_based" || doc.PDFConfidence != 0.99 {
		t.Fatalf("stored verdict = %+v", doc)
	}
}
