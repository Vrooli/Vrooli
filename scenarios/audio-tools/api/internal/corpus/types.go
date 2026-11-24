//go:build audio_lab

// Package corpus is audio-tools' speech-eval corpus store: per-clip
// metadata (reference transcript, tags, duration, blob key) in a SQLite
// `corpus` domain, with the audio BYTES held separately in the blob store
// under the git-ignored runtime data dir. It is the substrate the eval
// harness (internal/eval) replays against. Operator-facing and local-only
// in v1 — not exposed cross-scenario.
package corpus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"
)

// Source records how a clip was captured: free-form dictation, or a
// scripted prompt read aloud. It is a soft enum stored as text.
type Source string

const (
	SourceFreeForm Source = "free_form"
	SourceScripted Source = "scripted"
)

type Split string

const (
	SplitUnspecified Split = ""
	SplitDev         Split = "dev"
	SplitHeldOut     Split = "held_out"
)

type NoiseCondition string

const (
	NoiseUnspecified NoiseCondition = ""
	NoiseClean       NoiseCondition = "clean"
	NoiseNoisy       NoiseCondition = "noisy"
)

// Valid reports whether s is a known source. The empty string is treated
// as free_form by Normalize.
func (s Source) Valid() bool { return s == SourceFreeForm || s == SourceScripted }

// Normalize coerces an empty/unknown source to the free_form default.
func (s Source) Normalize() Source {
	if s.Valid() {
		return s
	}
	return SourceFreeForm
}

// Clip is one corpus item's metadata. The audio bytes are NOT here — they
// live in the blob store under BlobKey.
type Clip struct {
	ID                       string
	ReferenceText            string
	Tags                     []string
	DurationMs               int64
	SampleRateHz             int
	Format                   string
	BlobKey                  string
	Source                   Source
	CorpusRevision           string
	SourceID                 string
	AccentGroup              string
	ManifestTranscriptSHA256 string
	LineageManifestSHA256    string
	LicenseReference         string
	SpeakerPseudonym         string
	Split                    Split
	NoiseCondition           NoiseCondition
	SNRDB                    float64
	ContentSHA256            string
	CreatedAt                time.Time
	TimedReference           *TimedReference  `json:"timed_reference,omitempty"`
	ReferenceReview          *ReferenceReview `json:"reference_review,omitempty"`
}

// ReferenceReview records a human audit of the clip transcript. ReferenceSHA256
// binds the review to the exact stored transcript; legacy/imported rows without
// this record remain unreviewed and cannot support quality qualification.
type ReferenceReview struct {
	Reviewer        string `json:"reviewer"`
	Protocol        string `json:"protocol"`
	ReviewedAt      string `json:"reviewed_at"`
	ReferenceSHA256 string `json:"reference_sha256"`
	ReviewGroupID   string `json:"review_group_id,omitempty"`
}

func (r *ReferenceReview) Validate(reference string) error {
	if r == nil {
		return nil
	}
	if strings.TrimSpace(r.Reviewer) == "" || strings.TrimSpace(r.Protocol) == "" {
		return fmt.Errorf("reference review requires reviewer and protocol")
	}
	if _, err := time.Parse(time.RFC3339Nano, r.ReviewedAt); err != nil {
		return fmt.Errorf("reference review requires reviewed_at in RFC3339 format")
	}
	if r.ReferenceSHA256 == "" || r.ReferenceSHA256 != ReferenceDigest(reference) {
		return fmt.Errorf("reference review digest does not match transcript")
	}
	return nil
}

func ReferenceDigest(reference string) string {
	sum := sha256.Sum256([]byte(reference))
	return hex.EncodeToString(sum[:])
}

// TimedReference stores word boundaries and the provenance needed to decide
// whether they may be used as reviewed timing evidence. It stores metadata
// only; audio remains in the existing blob store.
type TimedReference struct {
	Words      []TimedWord      `json:"words"`
	Provenance TimingProvenance `json:"provenance"`
}

type TimedWord struct {
	Word         string  `json:"word"`
	StartSeconds float64 `json:"start_seconds"`
	EndSeconds   float64 `json:"end_seconds"`
}

type TimingProvenance struct {
	Status          string `json:"status"` // draft, aligner_only, human_reviewed
	SourceReference string `json:"source_reference"`
	AlignmentTool   string `json:"alignment_tool"`
	AlignmentModel  string `json:"alignment_model"`
	Reviewer        string `json:"reviewer,omitempty"`
	ReviewMethod    string `json:"review_method,omitempty"`
	ReviewedAt      string `json:"reviewed_at,omitempty"`
	Notes           string `json:"notes,omitempty"`
}

const (
	MaxTimedReferenceWords = 20_000
	MaxTimedReferenceBytes = 4 << 20
)

func (r *TimedReference) Validate(durationMs int64) error {
	if r == nil {
		return nil
	}
	if len(r.Words) == 0 || len(r.Words) > MaxTimedReferenceWords {
		return fmt.Errorf("timed reference requires 1..%d words", MaxTimedReferenceWords)
	}
	if err := validateTimingProvenance(r.Provenance); err != nil {
		return err
	}
	var priorStart, priorEnd float64
	for i, word := range r.Words {
		if err := validateTimedWord(i, word, durationMs); err != nil {
			return err
		}
		if i > 0 && (word.StartSeconds < priorStart || word.EndSeconds < priorEnd) {
			return fmt.Errorf("timed reference word %d is not monotonically ordered", i)
		}
		priorStart, priorEnd = word.StartSeconds, word.EndSeconds
	}
	return nil
}

func validateTimingProvenance(p TimingProvenance) error {
	if p.Status != "draft" && p.Status != "aligner_only" && p.Status != "human_reviewed" {
		return fmt.Errorf("timed reference provenance status must be draft, aligner_only, or human_reviewed")
	}
	if strings.TrimSpace(p.SourceReference) == "" {
		return fmt.Errorf("timed reference provenance requires source_reference")
	}
	alignmentTool := strings.TrimSpace(p.AlignmentTool)
	alignmentModel := strings.TrimSpace(p.AlignmentModel)
	if p.Status == "aligner_only" {
		if alignmentTool == "" || alignmentModel == "" {
			return fmt.Errorf("timed reference provenance requires alignment_tool and alignment_model")
		}
	}
	if p.Status == "human_reviewed" {
		if err := validateHumanReviewedProvenance(p, alignmentTool, alignmentModel); err != nil {
			return err
		}
	}
	return nil
}

func validateHumanReviewedProvenance(p TimingProvenance, alignmentTool, alignmentModel string) error {
	if (alignmentTool == "") != (alignmentModel == "") {
		return fmt.Errorf("human_reviewed timed reference requires both alignment_tool and alignment_model when either is supplied")
	}
	if strings.TrimSpace(p.Reviewer) == "" || strings.TrimSpace(p.ReviewMethod) == "" {
		return fmt.Errorf("human_reviewed timed reference requires reviewer and review_method")
	}
	if _, err := time.Parse(time.RFC3339Nano, p.ReviewedAt); err != nil {
		return fmt.Errorf("human_reviewed timed reference requires reviewed_at in RFC3339 format")
	}
	return nil
}

func validateTimedWord(index int, word TimedWord, durationMs int64) error {
	if strings.TrimSpace(word.Word) == "" || math.IsNaN(word.StartSeconds) || math.IsInf(word.StartSeconds, 0) || math.IsNaN(word.EndSeconds) || math.IsInf(word.EndSeconds, 0) || word.StartSeconds < 0 || word.EndSeconds < word.StartSeconds {
		return fmt.Errorf("timed reference word %d has invalid text or interval", index)
	}
	if durationMs > 0 && word.EndSeconds > float64(durationMs)/1000+0.05 {
		return fmt.Errorf("timed reference word %d exceeds clip duration", index)
	}
	return nil
}

// ValidateTranscript ensures the timed word sequence scores the same lexical
// reference as the clip transcript. Punctuation and case are ignored; word
// additions, omissions, and substitutions are rejected so timing evidence
// cannot silently describe a different transcript.
func (r *TimedReference) ValidateTranscript(transcript string) error {
	if r == nil {
		return nil
	}
	transcriptWords := lexicalWords(transcript)
	if len(transcriptWords) != len(r.Words) {
		return fmt.Errorf("timed reference has %d words but transcript has %d", len(r.Words), len(transcriptWords))
	}
	for i, word := range r.Words {
		if got := lexicalWords(word.Word); len(got) != 1 || got[0] != transcriptWords[i] {
			return fmt.Errorf("timed reference word %d does not match transcript", i)
		}
	}
	return nil
}

func lexicalWords(text string) []string {
	var words []string
	var current strings.Builder
	flush := func() {
		if current.Len() > 0 {
			words = append(words, current.String())
			current.Reset()
		}
	}
	for _, r := range strings.ToLower(text) {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			current.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return words
}

// ListFilter narrows a List query. TagContains matches clips whose tags
// JSON contains the substring (a cheap LIKE; tags are short).
type ListFilter struct {
	TagContains string
	Limit       int
	Offset      int
}

// ErrClipNotFound is returned when a clip id has no row.
type ErrClipNotFound struct{ ID string }

func (e ErrClipNotFound) Error() string { return fmt.Sprintf("corpus: clip %q not found", e.ID) }

// Repository is the corpus metadata persistence seam. Production is the
// SQLite impl in sqlite.go; tests substitute fakes.
type Repository interface {
	Create(ctx context.Context, c Clip) (Clip, error)
	Get(ctx context.Context, id string) (Clip, error)
	List(ctx context.Context, filter ListFilter) ([]Clip, error)
	ListBySourceID(ctx context.Context, sourceID string) ([]Clip, error)
	CountCreatedInWindow(ctx context.Context, from, to time.Time) (int64, error)
	Delete(ctx context.Context, id string) error
	ReviewReference(ctx context.Context, id, expectedText, referenceText string, review ReferenceReview) (Clip, error)
	ReviewReferenceGroup(ctx context.Context, members []ReviewMember, sourceID, revision, speaker, accent, manifestTranscriptSHA256, lineageManifestSHA256, referenceText string, review ReferenceReview) ([]Clip, error)
	BackfillSourceLineage(ctx context.Context, entries []LineageEntry, dryRun bool) (int, error)
}

type LineageEntry struct {
	ClipID, CorpusRevision, ContentSHA256, ManifestTranscriptSHA256                  string
	SourceID, SpeakerPseudonym, AccentGroup, Split, LicenseReference, ManifestSHA256 string
}

// ReviewMember binds one reviewed audio variant to the transcript revision the
// reviewer opened. All members are checked before any transcript is changed.
type ReviewMember struct {
	ID                      string
	ExpectedReferenceSHA256 string
}

// ErrStaleReference means a review form was opened against a transcript that
// has since changed. Callers must reload the clip before another submission.
type ErrStaleReference struct{ ID string }

func (e ErrStaleReference) Error() string {
	return fmt.Sprintf("corpus: clip %q reference changed since review began", e.ID)
}
