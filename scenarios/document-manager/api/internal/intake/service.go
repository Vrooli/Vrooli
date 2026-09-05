package intake

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/api-core/storage"
)

type Service struct {
	Repo       Repository
	Files      FileStore
	Classifier PDFClassifier
}

func NewService(repo Repository, files FileStore, classifier PDFClassifier) Service {
	return Service{Repo: repo, Files: files, Classifier: classifier}
}

func (s Service) Ingest(input IngestInput) (Document, bool, error) {
	if len(input.Content) == 0 {
		return Document{}, false, fmt.Errorf("content is required")
	}
	sum := sha256.Sum256(input.Content)
	hash := hex.EncodeToString(sum[:])
	if existing, err := s.Repo.FindByHash(hash); err == nil {
		return existing, true, nil
	} else if _, ok := err.(ErrNotFound); !ok {
		return Document{}, false, err
	}
	mime := sniff(input.Content)
	privacyClass := input.PrivacyClass
	if privacyClass == "" {
		privacyClass = "internal"
	}
	doc := Document{ContentSHA256: hash, SourceName: input.SourceName, DetectedMIME: mime, PrivacyClass: privacyClass, CreatedAt: time.Now().UTC()}
	if mime == "application/pdf" && s.Classifier != nil {
		path, err := s.Files.Put(hash+".pdf", input.Content)
		if err != nil {
			return Document{}, false, err
		}
		verdict, err := s.Classifier.Classify(path)
		if err != nil {
			return Document{}, false, err
		}
		doc.PDFType, doc.PDFConfidence = verdict.PDFType, verdict.Confidence
	} else {
		if _, err := s.Files.Put(hash, input.Content); err != nil {
			return Document{}, false, err
		}
	}
	created, err := s.Repo.Create(doc)
	return created, false, err
}

func (s Service) Get(id string) (Document, error)    { return s.Repo.Get(id) }
func (s Service) List(limit int) ([]Document, error) { return s.Repo.List(limit) }
func (s Service) Sources() ([]string, error)         { return s.Repo.ListSources() }

func sniff(data []byte) string {
	if len(data) >= 4 && string(data[:4]) == "%PDF" {
		return "application/pdf"
	}
	if len(data) >= 5 && string(data[:5]) == "{\\rtf" {
		return "application/rtf"
	}
	if len(data) >= 2 && data[0] == 'P' && data[1] == 'K' {
		text := string(data)
		switch {
		case strings.Contains(text, "word/document.xml"):
			return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
		case strings.Contains(text, "xl/workbook.xml"):
			return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		case strings.Contains(text, "ppt/presentation.xml"):
			return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
		case strings.Contains(text, "application/epub+zip"):
			return "application/epub+zip"
		case strings.Contains(text, "application/vnd.oasis.opendocument.text"):
			return "application/vnd.oasis.opendocument.text"
		default:
			return "application/zip"
		}
	}
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "<!DOCTYPE html") || strings.Contains(strings.ToLower(trimmed[:min(len(trimmed), 200)]), "<html") {
		return "text/html"
	}
	if strings.HasPrefix(trimmed, "<?xml") || strings.HasPrefix(trimmed, "<") {
		return "application/xml"
	}
	return http.DetectContentType(data)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type CommandPDFClassifier struct{}

func (CommandPDFClassifier) Classify(path string) (TypeVerdict, error) {
	out, err := exec.Command("resource-doc-parse", "classify", path).Output() // #nosec G204 -- executable and subcommand are fixed; path is the scenario-owned staged document.
	if err != nil {
		return TypeVerdict{}, fmt.Errorf("doc-parse classify: %w", err)
	}
	var v struct {
		PDFType    string  `json:"pdf_type"`
		Confidence float64 `json:"confidence"`
	}
	if err := json.Unmarshal(out, &v); err != nil {
		return TypeVerdict{}, err
	}
	return TypeVerdict{MIME: "application/pdf", PDFType: v.PDFType, Confidence: v.Confidence}, nil
}

type RoutedFileStore struct {
	Roots interface {
		Pick(context.Context, storage.Class) (string, error)
	}
	RootPath func(context.Context, string) (string, error)
}

func (s RoutedFileStore) Put(key string, data []byte) (string, error) {
	var path string
	var err error
	if s.RootPath != nil {
		path, err = s.RootPath(context.Background(), filepath.Base(key))
	} else {
		root, pickErr := s.Roots.Pick(context.Background(), storage.ClassData)
		err = pickErr
		path = filepath.Join(root, "documents", filepath.Base(key))
	}
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", err
	}
	return path, nil
}
