package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	MaxURLBytes     = 2048
	MaxContentBytes = 4 << 20
	MaxPassageBytes = 64 << 10
)

type Receipt struct {
	ReceiptID           string
	ObservationID       string
	ProducerExecutionID string
	URL                 string
	FinalURL            string
	RedirectURLs        []string
	RetrievedAt         time.Time
	ContentHash         string
	ArtifactID          string
	ExtractionRevision  string
	Retention           string
	FailureCode         string
	ETag                string
	LastModified        string
}

type Passage struct {
	PassageID string
	ReceiptID string
	StartByte int
	EndByte   int
	Content   string
	Hash      string
}

type Assessment struct {
	AssessmentID   string
	ClaimID        string
	Disposition    string
	PolicyRevision string
	Reason         string
	EvidenceJSON   string
}

type NewObservation struct {
	URL                 string
	FinalURL            string
	RedirectURLs        []string
	RetrievedAt         time.Time
	ProducerExecutionID string
	Content             []byte
	ExtractionRevision  string
	Retention           string
	FailureCode         string
	ETag                string
	LastModified        string
}

func (o NewObservation) Validate() error {
	if len(o.URL) == 0 || len(o.URL) > MaxURLBytes {
		return fmt.Errorf("url must contain 1..%d bytes", MaxURLBytes)
	}
	u, err := url.Parse(o.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil {
		return fmt.Errorf("url must be an http(s) URL without userinfo")
	}
	if o.FinalURL != "" {
		final, finalErr := url.Parse(o.FinalURL)
		if finalErr != nil || (final.Scheme != "http" && final.Scheme != "https") || final.Hostname() == "" || final.User != nil || len(o.FinalURL) > MaxURLBytes {
			return fmt.Errorf("final url must be a bounded http(s) URL without userinfo")
		}
	}
	if len(o.RedirectURLs) > 10 {
		return fmt.Errorf("redirect chain exceeds ten hops")
	}
	for _, redirect := range o.RedirectURLs {
		parsed, parseErr := url.Parse(redirect)
		if parseErr != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" || parsed.User != nil || len(redirect) > MaxURLBytes {
			return fmt.Errorf("redirect url must be a bounded http(s) URL without userinfo")
		}
	}
	if len(o.Content) > MaxContentBytes {
		return fmt.Errorf("content exceeds %d bytes", MaxContentBytes)
	}
	if !utf8.Valid(o.Content) {
		return fmt.Errorf("content must be valid UTF-8")
	}
	if strings.TrimSpace(o.ExtractionRevision) == "" || len(o.ExtractionRevision) > 128 {
		return fmt.Errorf("extraction revision is required and bounded")
	}
	return nil
}

func ContentHash(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func validateSpan(content []byte, start, end int) error {
	if start < 0 || end < start || end > len(content) || end-start > MaxPassageBytes {
		return fmt.Errorf("passage span is outside content bounds")
	}
	if !utf8.Valid(content) || !utf8.Valid(content[start:end]) {
		return fmt.Errorf("passage span must contain valid UTF-8")
	}
	if start > 0 && start < len(content) && (content[start]&0xc0) == 0x80 {
		return fmt.Errorf("passage start is inside a UTF-8 character")
	}
	if end > 0 && end < len(content) && (content[end]&0xc0) == 0x80 {
		return fmt.Errorf("passage end is inside a UTF-8 character")
	}
	return nil
}
