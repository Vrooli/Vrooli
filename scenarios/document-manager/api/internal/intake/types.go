package intake

import "time"

type Document struct {
	ID            string
	ContentSHA256 string
	SourceName    string
	DetectedMIME  string
	PDFType       string
	PDFConfidence float64
	PrivacyClass  string
	CreatedAt     time.Time
}

type IngestInput struct {
	Content      []byte
	SourceName   string
	PrivacyClass string
}

type TypeVerdict struct {
	MIME, PDFType string
	Confidence    float64
}

type Repository interface {
	Create(Document) (Document, error)
	Get(string) (Document, error)
	List(int) ([]Document, error)
	ListSources() ([]string, error)
	FindByHash(string) (Document, error)
}

type FileStore interface {
	Put(key string, data []byte) (string, error)
}

type PDFClassifier interface {
	Classify(path string) (TypeVerdict, error)
}
