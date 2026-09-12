package conversationsearch

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

type ContentClass int

const (
	ContentClassUnspecified ContentClass = iota
	ContentClassProse
	ContentClassQuotedProse
	ContentClassToolCall
	ContentClassToolResult
	ContentClassInjectedContext
	ContentClassEvidenceOnlyDuplicate
)

// DefaultSearchContentClass identifies the conversational corpus used when a
// caller has not explicitly opted into noisy forensic classes.
func DefaultSearchContentClass(class ContentClass) bool {
	return class == ContentClassProse || class == ContentClassQuotedProse
}

const (
	DefaultRecipeVersion = "conversation-search-v3"
	DefaultMaxChunkBytes = 1024
	DefaultOverlapBytes  = 128
)

// SourceMessage is the canonical, engine-neutral input to normalization. It
// contains attribution but never a raw transcript path or attachment body.
type SourceMessage struct {
	RunID             string
	EventID           string
	MessageID         string
	Sequence          int64
	Role              string
	OccurredAt        time.Time
	Content           string
	ProviderEventType string
	EvidenceOnly      bool
	EvidenceForEvent  string
	Deleted           bool
	Disallowed        bool
	Harness           string
	SourceSessionID   string
	ProviderOrigin    string
	Importer          string
	ProjectScope      string
	CWDScope          string
	Runner            string
	Model             string
	Profile           string
	RunStatus         string
	RunLabel          string
	Tags              []string
	Workloads         []string
}

type NormalizerConfig struct {
	RecipeVersion string
	MaxChunkBytes int
	OverlapBytes  int
	Now           func() time.Time
}

type Normalizer struct {
	recipeVersion string
	maxChunkBytes int
	overlapBytes  int
	now           func() time.Time
}

func NewNormalizer(config NormalizerConfig) (*Normalizer, error) {
	if config.RecipeVersion == "" {
		config.RecipeVersion = DefaultRecipeVersion
	}
	if config.MaxChunkBytes == 0 {
		config.MaxChunkBytes = DefaultMaxChunkBytes
	}
	if config.OverlapBytes == 0 {
		config.OverlapBytes = DefaultOverlapBytes
	}
	if config.MaxChunkBytes < 64 {
		return nil, errors.New("maximum chunk bytes must be at least 64")
	}
	if config.OverlapBytes < 0 || config.OverlapBytes >= config.MaxChunkBytes/2 {
		return nil, errors.New("overlap bytes must be non-negative and less than half the chunk limit")
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	return &Normalizer{recipeVersion: config.RecipeVersion, maxChunkBytes: config.MaxChunkBytes, overlapBytes: config.OverlapBytes, now: config.Now}, nil
}

func MustNormalizer(config NormalizerConfig) *Normalizer {
	normalizer, err := NewNormalizer(config)
	if err != nil {
		panic(err)
	}
	return normalizer
}

// Normalize produces zero or more stable chunks. Empty, deleted, and
// disallowed messages produce no documents, so neither lexical nor semantic
// stores can rediscover them.
func (n *Normalizer) Normalize(source SourceMessage) ([]Document, error) {
	if source.RunID == "" || source.EventID == "" || source.Role == "" || source.OccurredAt.IsZero() {
		return nil, errors.New("run, event, role, and occurred time are required")
	}
	if source.Sequence < 0 {
		return nil, errors.New("event sequence must be non-negative")
	}
	if source.Deleted || source.Disallowed {
		return nil, nil
	}
	content := strings.TrimSpace(strings.ReplaceAll(source.Content, "\r\n", "\n"))
	content = strings.ReplaceAll(content, "\r", "\n")
	if content == "" {
		return nil, nil
	}
	messageID := strings.TrimSpace(source.MessageID)
	if messageID == "" {
		messageID = source.EventID
	}
	class := classifyContent(source, content)
	spans := splitChunkSpans(content, n.maxChunkBytes, n.overlapBytes)
	if len(spans) == 0 {
		return nil, nil
	}
	sourceHash := digest(source.RunID, source.EventID, messageID, source.Role, content, source.ProviderEventType, source.ProviderOrigin, source.EvidenceForEvent, n.recipeVersion)
	documents := make([]Document, 0, len(spans))
	indexedAt := n.now().UTC()
	for index, span := range spans {
		body := content[span.start:span.end]
		documentID := digest(source.RunID, source.EventID, messageID, n.recipeVersion, fmt.Sprint(index), fmt.Sprint(span.start), fmt.Sprint(span.end))
		documents = append(documents, Document{
			DocumentID: documentID, SourceRunID: source.RunID, SourceEventID: source.EventID, SourceMessageID: messageID,
			ChunkIndex: index, ChunkTotal: len(spans), StartByte: span.start, EndByte: span.end,
			EventSequence: source.Sequence, Role: strings.ToLower(strings.TrimSpace(source.Role)), OccurredAt: source.OccurredAt.UTC(),
			Content: body, ContentClass: class, SourceHash: sourceHash, ContentHash: digest(body), RecipeVersion: n.recipeVersion,
			Harness: source.Harness, SourceSessionID: source.SourceSessionID, ProviderOrigin: source.ProviderOrigin, Importer: source.Importer,
			ProjectScope: source.ProjectScope, CWDScope: source.CWDScope, Runner: source.Runner, Model: source.Model,
			Profile: source.Profile, RunStatus: source.RunStatus, RunLabel: source.RunLabel,
			Tags: compactStrings(source.Tags), Workloads: compactStrings(source.Workloads),
			EvidenceRef: stableEvidenceRef(source.RunID, source.EventID), Visible: true, IndexedAt: indexedAt,
		})
	}
	return documents, nil
}

func classifyContent(source SourceMessage, content string) ContentClass {
	if source.EvidenceOnly || source.EvidenceForEvent != "" {
		return ContentClassEvidenceOnlyDuplicate
	}
	role := strings.ToLower(strings.TrimSpace(source.Role))
	eventType := strings.ToLower(source.ProviderEventType)
	if role == "tool" || role == "tool_result" || strings.Contains(eventType, "tool_result") {
		return ContentClassToolResult
	}
	if role == "tool_call" || strings.Contains(eventType, "tool_call") || strings.Contains(eventType, "function_call") {
		return ContentClassToolCall
	}
	if role == "system" || isInjectedContext(content) {
		return ContentClassInjectedContext
	}
	if looksQuoted(content) {
		return ContentClassQuotedProse
	}
	return ContentClassProse
}

func isInjectedContext(content string) bool {
	lower := strings.ToLower(strings.TrimSpace(content))
	markers := []string{"<system-reminder", "<instructions>", "# agents.md instructions", "<environment_context>", "<skills_instructions>"}
	for _, marker := range markers {
		if strings.HasPrefix(lower, marker) || strings.Contains(lower, "\n"+marker) {
			return true
		}
	}
	return false
}

func looksQuoted(content string) bool {
	lines := strings.Split(content, "\n")
	nonEmpty, quoted := 0, 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		nonEmpty++
		if strings.HasPrefix(trimmed, ">") {
			quoted++
		}
	}
	return nonEmpty >= 2 && quoted*2 >= nonEmpty
}

type chunkSpan struct{ start, end int }

func splitChunkSpans(content string, maximum, overlap int) []chunkSpan {
	var spans []chunkSpan
	for start := 0; start < len(content); {
		end := len(content)
		if end-start > maximum {
			end = runeBoundaryAtOrBefore(content, start+maximum)
			if candidate := preferredBreak(content, start, end); candidate > start {
				end = candidate
			}
		}
		spanStart, spanEnd := trimSpan(content, start, end)
		if spanEnd > spanStart {
			spans = append(spans, chunkSpan{start: spanStart, end: spanEnd})
		}
		if end >= len(content) {
			break
		}
		next := runeBoundaryAtOrBefore(content, end-overlap)
		if next <= start {
			next = end
		}
		start = next
	}
	return spans
}

func preferredBreak(content string, start, end int) int {
	segment := content[start:end]
	minimum := len(segment) / 2
	best := -1
	for _, separator := range []string{"\n\n", "\n", ". ", "! ", "? ", "; ", ", ", " "} {
		if index := strings.LastIndex(segment, separator); index >= minimum && index+len(separator) > best {
			best = index + len(separator)
		}
	}
	if best <= 0 {
		return end
	}
	return runeBoundaryAtOrBefore(content, start+best)
}

func runeBoundaryAtOrBefore(content string, index int) int {
	if index >= len(content) {
		return len(content)
	}
	if index < 0 {
		return 0
	}
	for index > 0 && !utf8.RuneStart(content[index]) {
		index--
	}
	return index
}

func trimSpan(content string, start, end int) (int, int) {
	segment := content[start:end]
	left := strings.TrimLeftFunc(segment, unicode.IsSpace)
	start += len(segment) - len(left)
	left = strings.TrimRightFunc(left, unicode.IsSpace)
	return start, start + len(left)
}

func compactStrings(values []string) []string {
	output := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		output = append(output, value)
	}
	return output
}

func stableEvidenceRef(runID, eventID string) string {
	return "agent-manager://runs/" + runID + "/events/" + eventID
}

func digest(parts ...string) string {
	hash := sha256.New()
	for _, part := range parts {
		_, _ = hash.Write([]byte(part))
		_, _ = hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}
