package sketch

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/platform-go"
)

var safeSegment = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

// Repository keeps authored experience persistence behind the design domain.
// Every mutation must name the snapshot on which the caller based its edit.
type Repository interface {
	Path(scenario, page string) (string, error)
	ReadDesignSource(scenario, source string) (*DesignSourceSnapshot, error)
	Load(scenario, page string) (Document, error)
	Read(scenario, page string) (Snapshot, error)
	Save(scenario, page, expectedHash string, doc Document) (Snapshot, error)
	History(scenario, page string) ([]Revision, error)
	Recover(scenario, page string) (Snapshot, error)
}

type Snapshot struct {
	DeclaredRegions []Region
	PagePurpose     string
	PageElements    []string
	Document        Document
	ContentHash     string
	Changed         bool
}

type ConflictError struct{ Expected, Current string }

func (e *ConflictError) Error() string {
	return fmt.Sprintf("sketch revision conflict: expected %s, current %s", e.Expected, e.Current)
}

var ErrExpectedRevision = errors.New("expected content hash is required; read the current sketch before editing")

type Store struct {
	repoRoot     string
	afterPublish func(string) error
}

func NewStore(repoRoot string) *Store { return &Store{repoRoot: filepath.Clean(repoRoot)} }

func (s *Store) Path(scenario, page string) (string, error) {
	if !safeSegment.MatchString(scenario) || !safeSegment.MatchString(page) {
		return "", fmt.Errorf("rejected sketch path: target must remain inside scenarios/<scenario>/experience/pages")
	}
	return filepath.Join(s.repoRoot, "scenarios", scenario, "experience", "pages", page+".json"), nil
}

// openPages pins each directory before descending. Every operation after the
// scenario boundary is confined to its experience root, including history.
func (s *Store) openExperience(scenario string) (*os.Root, error) {
	if !safeSegment.MatchString(scenario) {
		return nil, fmt.Errorf("invalid scenario identity")
	}
	root, err := os.OpenRoot(s.repoRoot)
	if err != nil {
		return nil, err
	}
	for _, part := range []string{"scenarios", scenario, "experience"} {
		info, err := root.Lstat(part)
		if err != nil {
			root.Close()
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			root.Close()
			return nil, fmt.Errorf("rejected sketch path: symlink %s", part)
		}
		child, err := root.OpenRoot(part)
		root.Close()
		if err != nil {
			return nil, err
		}
		root = child
	}
	return root, nil
}
func (s *Store) openPages(scenario, page string) (*os.Root, string, error) {
	if _, err := s.Path(scenario, page); err != nil {
		return nil, "", err
	}
	root, err := s.openExperience(scenario)
	if err != nil {
		return nil, "", err
	}
	for _, part := range []string{"pages", filepath.Join("pages", page+".json")} {
		info, err := root.Lstat(part)
		if err != nil {
			root.Close()
			return nil, "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			root.Close()
			return nil, "", fmt.Errorf("rejected sketch path: symlink %s", part)
		}
	}
	return root, filepath.Join("pages", page+".json"), nil
}

func decodeSnapshot(raw []byte) (Snapshot, error) {
	if err := rejectDuplicateKeys(raw); err != nil {
		return Snapshot{}, err
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return Snapshot{}, err
	}
	if envelope == nil {
		return Snapshot{}, errors.New("page document must be a JSON object")
	}
	var doc Document
	if value, ok := envelope["sketch"]; ok {
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return Snapshot{}, errors.New("sketch must be an object")
		}
		if err := json.Unmarshal(value, &doc); err != nil {
			return Snapshot{}, err
		}
	}
	// Hash content rather than formatting: the same authored object has one
	// identity regardless of indentation or object-key order.
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return Snapshot{}, err
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return Snapshot{}, err
	}
	digest := sha256.Sum256(canonical)
	var authored struct {
		Page     struct{ Purpose string }
		Elements []struct{ ID string }
		Regions  []struct {
			ID, Purpose string
			Elements    []string
		}
	}
	if err := json.Unmarshal(raw, &authored); err != nil {
		return Snapshot{}, err
	}
	declared := make([]Region, 0, len(authored.Regions))
	for _, region := range authored.Regions {
		declared = append(declared, Region{ID: region.ID, Note: region.Purpose, Elements: region.Elements})
	}
	var elements []string
	for _, element := range authored.Elements {
		elements = append(elements, element.ID)
	}
	return Snapshot{PagePurpose: authored.Page.Purpose, PageElements: elements, Document: doc, ContentHash: hex.EncodeToString(digest[:]), DeclaredRegions: declared}, nil
}

func (s *Store) Read(scenario, page string) (Snapshot, error) {
	root, name, err := s.openPages(scenario, page)
	if err != nil {
		return Snapshot{}, err
	}
	defer root.Close()
	raw, err := root.ReadFile(name)
	if err != nil {
		return Snapshot{}, err
	}
	return decodeSnapshot(raw)
}

func (s *Store) Load(scenario, page string) (Document, error) {
	snapshot, err := s.Read(scenario, page)
	return snapshot.Document, err
}

func (s *Store) Save(scenario, page, expectedHash string, doc Document) (Snapshot, error) {
	if _, err := s.Path(scenario, page); err != nil {
		return Snapshot{}, err
	}
	if expectedHash == "" {
		return Snapshot{}, ErrExpectedRevision
	}
	root, name, err := s.openPages(scenario, page)
	if err != nil {
		return Snapshot{}, err
	}
	defer root.Close()
	lock, err := root.OpenFile(filepath.Join("pages", "."+page+".lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return Snapshot{}, err
	}
	defer lock.Close()
	release, err := platform.LockFile(lock, false)
	if err != nil {
		return Snapshot{}, err
	}
	defer release()
	if _, err := s.recoverPending(root, page, name); err != nil {
		return Snapshot{}, err
	}
	raw, err := root.ReadFile(name)
	if err != nil {
		return Snapshot{}, err
	}
	current, err := decodeSnapshot(raw)
	if err != nil {
		return Snapshot{}, err
	}
	if current.ContentHash != expectedHash {
		return Snapshot{}, &ConflictError{Expected: expectedHash, Current: current.ContentHash}
	}
	updated, err := pageWithSketch(raw, doc)
	if err != nil {
		return Snapshot{}, err
	}
	next, err := decodeSnapshot(updated)
	if err != nil {
		return Snapshot{}, err
	}
	if next.ContentHash == current.ContentHash {
		return current, nil
	}
	if err := s.writeRevision(root, page, current.ContentHash, "", raw); err != nil {
		return Snapshot{}, err
	}
	if err := s.markApplied(root, page, current.ContentHash); err != nil {
		return Snapshot{}, err
	}
	if err := s.writeRevision(root, page, next.ContentHash, current.ContentHash, updated); err != nil {
		return Snapshot{}, err
	}
	if err := s.checkpoint("revisions"); err != nil {
		return Snapshot{}, err
	}
	pending := applyManifest{Expected: current.ContentHash, Next: next.ContentHash}
	pendingBytes, err := json.Marshal(pending)
	if err != nil {
		return Snapshot{}, err
	}
	if err := storage.WriteFileAtomicInRoot(root, manifestPath(page), pendingBytes, 0600); err != nil {
		return Snapshot{}, err
	}
	if err := s.checkpoint("manifest"); err != nil {
		return Snapshot{}, err
	}
	if _, err := s.recoverPending(root, page, name); err != nil {
		return Snapshot{}, err
	}

	next.Changed = true
	return next, nil
}

// Preserve extensions recursively on stable identities, while allowing known
// fields and removed array members to be intentionally cleared by the editor.
func preserveSketchExtensions(before, after []byte) ([]byte, error) {
	var old, next map[string]json.RawMessage
	if len(before) > 0 {
		if err := json.Unmarshal(before, &old); err != nil {
			return nil, err
		}
	}
	if err := json.Unmarshal(after, &next); err != nil {
		return nil, err
	}
	merged, err := mergeKnown(old, next, "sketch")
	if err != nil {
		return nil, err
	}
	return json.Marshal(merged)
}

var knownFields = map[string][]string{
	"sketch":     {"viewport", "template", "placements", "unplaced", "notes", "regions", "render", "intent"},
	"template":   {"asset", "version"},
	"placements": {"region", "fills", "state", "note", "intent"},
	"fills":      {"asset", "version", "placeholder", "intent"},
	"unplaced":   {"was", "reason", "region", "placement"},
	"placement":  {"region", "fills", "state", "note", "intent"},
	"notes":      {"scope", "text"},
	"regions":    {"id", "note", "elements", "grid", "origin", "locked"},
	"grid":       {"x", "y", "w", "h"},
}
var identityFields = map[string][]string{"placements": {"region"}, "unplaced": {"was", "region"}, "notes": {"scope", "text"}, "regions": {"id"}}

func mergeKnown(old, next map[string]json.RawMessage, kind string) (map[string]json.RawMessage, error) {
	out := make(map[string]json.RawMessage)
	for k, v := range old {
		out[k] = v
	}
	for _, k := range knownFields[kind] {
		delete(out, k)
	}
	for k, v := range next {
		if _, ok := knownFields[k]; !ok {
			out[k] = v
			continue
		}
		if identity, array := identityFields[k]; array {
			var previous, proposed []map[string]json.RawMessage
			if len(old[k]) > 0 {
				if err := json.Unmarshal(old[k], &previous); err != nil {
					return nil, err
				}
			}
			if err := json.Unmarshal(v, &proposed); err != nil {
				return nil, err
			}
			key := func(item map[string]json.RawMessage) string {
				parts := make([]string, len(identity))
				for i, f := range identity {
					parts[i] = string(item[f])
				}
				return strings.Join(parts, "\x00")
			}
			byID := map[string]map[string]json.RawMessage{}
			for _, item := range previous {
				byID[key(item)] = item
			}
			for i, item := range proposed {
				merged, err := mergeKnown(byID[key(item)], item, k)
				if err != nil {
					return nil, err
				}
				proposed[i] = merged
			}
			raw, err := json.Marshal(proposed)
			if err != nil {
				return nil, err
			}
			out[k] = raw
		} else {
			var previous, proposed map[string]json.RawMessage
			if len(old[k]) > 0 {
				if err := json.Unmarshal(old[k], &previous); err != nil {
					return nil, err
				}
			}
			if err := json.Unmarshal(v, &proposed); err != nil {
				return nil, err
			}
			merged, err := mergeKnown(previous, proposed, k)
			if err != nil {
				return nil, err
			}
			raw, err := json.Marshal(merged)
			if err != nil {
				return nil, err
			}
			out[k] = raw
		}
	}
	return out, nil
}

func replaceTopLevel(raw []byte, key string, value []byte) ([]byte, error) {
	start, end, found, err := topLevelValueRange(raw, key)
	if err != nil {
		return nil, err
	}
	if found {
		out := make([]byte, 0, len(raw)-end+start+len(value))
		out = append(out, raw[:start]...)
		out = append(out, value...)
		out = append(out, raw[end:]...)
		return out, nil
	}
	close := bytes.LastIndexByte(raw, '}')
	if close < 0 {
		return nil, errors.New("page document is not a JSON object")
	}
	prefix := bytes.TrimSpace(raw[:close])
	comma := byte(',')
	if len(prefix) == 1 && prefix[0] == '{' {
		comma = 0
	}
	var insertion bytes.Buffer
	if comma != 0 {
		insertion.WriteByte(comma)
	}
	insertion.WriteString("\n  \"sketch\": ")
	insertion.Write(value)
	insertion.WriteByte('\n')
	out := make([]byte, 0, len(raw)+insertion.Len())
	out = append(out, bytes.TrimRight(raw[:close], " \t\r\n")...)
	out = append(out, insertion.Bytes()...)
	out = append(out, raw[close:]...)
	return out, nil
}

func topLevelValueRange(raw []byte, wanted string) (int, int, bool, error) {
	i := skipSpace(raw, 0)
	if i >= len(raw) || raw[i] != '{' {
		return 0, 0, false, errors.New("page document is not a JSON object")
	}
	i++
	for {
		i = skipSpace(raw, i)
		if i >= len(raw) {
			return 0, 0, false, errors.New("unterminated page document")
		}
		if raw[i] == '}' {
			return 0, 0, false, nil
		}
		keyStart := i
		keyEnd, err := jsonValueEnd(raw, keyStart)
		if err != nil {
			return 0, 0, false, err
		}
		var key string
		if err := json.Unmarshal(raw[keyStart:keyEnd], &key); err != nil {
			return 0, 0, false, fmt.Errorf("parse object key: %w", err)
		}
		i = skipSpace(raw, keyEnd)
		if i >= len(raw) || raw[i] != ':' {
			return 0, 0, false, errors.New("missing colon after object key")
		}
		valueStart := skipSpace(raw, i+1)
		valueEnd, err := jsonValueEnd(raw, valueStart)
		if err != nil {
			return 0, 0, false, err
		}
		if key == wanted {
			return valueStart, valueEnd, true, nil
		}
		i = skipSpace(raw, valueEnd)
		if i < len(raw) && raw[i] == ',' {
			i++
			continue
		}
		if i < len(raw) && raw[i] == '}' {
			return 0, 0, false, nil
		}
		return 0, 0, false, errors.New("invalid page object separator")
	}
}

func jsonValueEnd(raw []byte, start int) (int, error) {
	if start >= len(raw) {
		return 0, errors.New("missing JSON value")
	}
	if raw[start] == '"' {
		for i := start + 1; i < len(raw); i++ {
			if raw[i] == '\\' {
				i++
				continue
			}
			if raw[i] == '"' {
				return i + 1, nil
			}
		}
		return 0, errors.New("unterminated JSON string")
	}
	if raw[start] == '{' || raw[start] == '[' {
		open := raw[start]
		close := byte('}')
		if open == '[' {
			close = ']'
		}
		depth, inString := 0, false
		for i := start; i < len(raw); i++ {
			if inString {
				if raw[i] == '\\' {
					i++
				} else if raw[i] == '"' {
					inString = false
				}
				continue
			}
			if raw[i] == '"' {
				inString = true
			} else if raw[i] == open {
				depth++
			} else if raw[i] == close {
				depth--
				if depth == 0 {
					return i + 1, nil
				}
			}
		}
		return 0, errors.New("unterminated composite JSON value")
	}
	i := start
	for i < len(raw) && !bytes.ContainsRune([]byte(",}\r\n\t "), rune(raw[i])) {
		i++
	}
	if i == start {
		return 0, errors.New("empty JSON value")
	}
	return i, nil
}

func skipSpace(raw []byte, i int) int {
	for i < len(raw) && (raw[i] == ' ' || raw[i] == '\n' || raw[i] == '\r' || raw[i] == '\t') {
		i++
	}
	return i
}

// encoding/json otherwise accepts duplicate keys by silently choosing the last
// value, while a surgical authored-file update may find the first occurrence.
func rejectDuplicateKeys(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var readValue func() error
	readValue = func() error {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delimiter, composite := token.(json.Delim)
		if !composite {
			return nil
		}
		switch delimiter {
		case '{':
			seen := map[string]bool{}
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return err
				}
				key, ok := keyToken.(string)
				if !ok {
					return errors.New("invalid object key")
				}
				if seen[key] {
					return fmt.Errorf("duplicate JSON key %q", key)
				}
				seen[key] = true
				if err := readValue(); err != nil {
					return err
				}
			}
		case '[':
			for decoder.More() {
				if err := readValue(); err != nil {
					return err
				}
			}
		default:
			return errors.New("unexpected JSON delimiter")
		}
		_, err = decoder.Token()
		return err
	}
	if err := readValue(); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return errors.New("trailing JSON data")
	}
	return nil
}

func pageWithSketch(raw []byte, doc Document) ([]byte, error) {
	before, err := decodeSnapshot(raw)
	if err != nil {
		return nil, err
	}
	if err := validateRegionLocks(before.Document, doc); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	encoded, err = preserveSketchExtensions(envelope["sketch"], encoded)
	if err != nil {
		return nil, err
	}
	var formatted bytes.Buffer
	if err := json.Indent(&formatted, encoded, "  ", "  "); err != nil {
		return nil, err
	}
	updated, err := replaceTopLevel(raw, "sketch", formatted.Bytes())
	if err != nil {
		return nil, err
	}
	return updated, nil
}
