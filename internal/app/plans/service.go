package plans

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	configpkg "github.com/vrooli/vrooli/internal/config"
)

type Service struct {
	Root      string
	Home      string
	Now       func() time.Time
	MkdirAll  func(string, os.FileMode) error
	ReadFile  func(string) ([]byte, error)
	WriteFile func(string, []byte, os.FileMode) error
	Remove    func(string) error
	Rename    func(string, string) error
}

type indexFile struct {
	Version int          `json:"version"`
	Plans   []PlanRecord `json:"plans"`
}

const indexFilename = "_index.json"

func (s Service) Add(req AddRequest) (AddOutput, error) {
	title := strings.TrimSpace(req.Title)
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return AddOutput{}, fmt.Errorf("plan content is required")
	}
	slug := slugOrDefault(req.Slug, title)
	if slug == "" {
		slug = s.generatedSlug()
	}
	if title == "" {
		title = titleFromSlug(slug)
	}
	now := s.now()
	idx, err := s.loadIndex()
	if err != nil {
		return AddOutput{}, err
	}
	slug = uniqueSlug(slug, idx.Plans)
	id := uniqueID(slug, idx.Plans)
	path := filepath.Join(s.storageDir(), slug+".md")
	record := PlanRecord{
		ID:          id,
		Title:       title,
		Slug:        slug,
		Path:        path,
		CreatedAt:   now,
		UpdatedAt:   now,
		ContentHash: contentHash(content),
	}
	if err := s.mkdirAll(filepath.Dir(path), 0o755); err != nil {
		return AddOutput{}, err
	}
	if err := s.writeFile(path, []byte(ensureTrailingNewline(req.Content)), 0o600); err != nil {
		return AddOutput{}, err
	}
	idx.Plans = append(idx.Plans, record)
	if err := s.saveIndex(idx); err != nil {
		return AddOutput{}, err
	}
	return AddOutput{Success: true, Plan: record}, nil
}

func (s Service) List(req ListRequest) (ListOutput, error) {
	if req.IncludeAll {
		records, err := s.listAll(req.IncludeArchived)
		if err != nil {
			return ListOutput{}, err
		}
		return ListOutput{Success: true, Plans: records}, nil
	}
	idx, err := s.loadIndex()
	if err != nil {
		return ListOutput{}, err
	}
	return ListOutput{Success: true, Plans: filterArchived(idx.Plans, req.IncludeArchived)}, nil
}

func (s Service) Show(req ShowRequest) (ShowOutput, error) {
	record, err := s.find(req.Ref, req.Repo)
	if err != nil {
		return ShowOutput{}, err
	}
	content, err := s.readFile(record.Path)
	if err != nil {
		return ShowOutput{}, fmt.Errorf("read plan: %w", err)
	}
	return ShowOutput{Success: true, Plan: record, Content: string(content)}, nil
}

func (s Service) Path(req ShowRequest) (PathOutput, error) {
	record, err := s.find(req.Ref, req.Repo)
	if err != nil {
		return PathOutput{}, err
	}
	return PathOutput{Success: true, ID: record.ID, Path: record.Path}, nil
}

func (s Service) Archive(req ArchiveRequest) (ArchiveOutput, error) {
	idx, err := s.loadIndex()
	if err != nil {
		return ArchiveOutput{}, err
	}
	pos, err := findPosition(idx.Plans, req.Ref)
	if err != nil {
		return ArchiveOutput{}, err
	}
	now := s.now()
	idx.Plans[pos].Archived = true
	idx.Plans[pos].ArchivedAt = now
	idx.Plans[pos].UpdatedAt = now
	if err := s.saveIndex(idx); err != nil {
		return ArchiveOutput{}, err
	}
	return ArchiveOutput{Success: true, Plan: idx.Plans[pos]}, nil
}

func (s Service) Import(req ImportRequest) (ImportOutput, error) {
	source := strings.TrimSpace(req.Path)
	if source == "" {
		return ImportOutput{}, fmt.Errorf("path is required")
	}
	content, err := s.readFile(source)
	if err != nil {
		return ImportOutput{}, fmt.Errorf("read source plan: %w", err)
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = titleFromContentOrPath(string(content), source)
	}
	added, err := s.Add(AddRequest{
		Title:   title,
		Slug:    req.Slug,
		Repo:    req.Repo,
		Content: string(content),
	})
	if err != nil {
		return ImportOutput{}, err
	}
	record := added.Plan
	record.SourcePath = source
	if err := s.updateRecord(record); err != nil {
		return ImportOutput{}, err
	}
	deleted := false
	if req.DeleteSource {
		if filepath.Clean(source) == filepath.Clean(record.Path) {
			return ImportOutput{}, fmt.Errorf("refusing to delete imported destination")
		}
		if err := s.remove(source); err != nil {
			return ImportOutput{}, fmt.Errorf("delete source plan: %w", err)
		}
		deleted = true
	}
	return ImportOutput{Success: true, Plan: record, Deleted: deleted}, nil
}

func (s Service) Export(req ExportRequest) (ExportOutput, error) {
	to := strings.TrimSpace(req.To)
	if to == "" {
		return ExportOutput{}, fmt.Errorf("--to is required")
	}
	show, err := s.Show(ShowRequest{Ref: req.Ref, Repo: req.Repo})
	if err != nil {
		return ExportOutput{}, err
	}
	if err := s.mkdirAll(filepath.Dir(to), 0o755); err != nil {
		return ExportOutput{}, err
	}
	if err := s.writeFile(to, []byte(ensureTrailingNewline(show.Content)), 0o644); err != nil {
		return ExportOutput{}, err
	}
	return ExportOutput{Success: true, ID: show.Plan.ID, Path: to}, nil
}

func (s Service) updateRecord(record PlanRecord) error {
	idx, err := s.loadIndex()
	if err != nil {
		return err
	}
	pos, err := findPosition(idx.Plans, record.ID)
	if err != nil {
		return err
	}
	idx.Plans[pos] = record
	return s.saveIndex(idx)
}

func (s Service) find(ref, repo string) (PlanRecord, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return PlanRecord{}, fmt.Errorf("plan id or slug is required")
	}
	idx, err := s.loadIndex()
	if err != nil {
		return PlanRecord{}, err
	}
	pos, err := findPosition(idx.Plans, ref)
	if err != nil {
		return PlanRecord{}, err
	}
	return idx.Plans[pos], nil
}

func (s Service) listAll(includeArchived bool) ([]PlanRecord, error) {
	idx, err := s.loadIndex()
	if err != nil {
		return nil, err
	}
	records := filterArchived(idx.Plans, includeArchived)
	sortRecords(records)
	return records, nil
}

func (s Service) loadIndex() (indexFile, error) {
	path := s.indexPath()
	data, err := s.readFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return indexFile{Version: 1}, nil
		}
		return indexFile{}, fmt.Errorf("read plan index: %w", err)
	}
	var idx indexFile
	if err := json.Unmarshal(data, &idx); err != nil {
		return indexFile{}, fmt.Errorf("decode plan index: %w", err)
	}
	if idx.Version == 0 {
		idx.Version = 1
	}
	sortRecords(idx.Plans)
	return idx, nil
}

func (s Service) saveIndex(idx indexFile) error {
	idx.Version = 1
	sortRecords(idx.Plans)
	if err := s.mkdirAll(s.storageDir(), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	return s.writeFile(s.indexPath(), append(data, '\n'), 0o600)
}

func (s Service) storageDir() string {
	home := strings.TrimSpace(s.Home)
	if home == "" {
		if resolved, err := configpkg.HomeDir(); err == nil {
			home = resolved
		}
	}
	// Plans dir name comes from the runtime_home authority. A contract-load
	// failure is catastrophic; return empty so callers fail loudly rather than
	// writing to a hand-rolled path.
	dir, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyPlans)
	if err != nil {
		return ""
	}
	return dir
}

func (s Service) indexPath() string {
	return filepath.Join(s.storageDir(), indexFilename)
}

func (s Service) repoRoot(override string) string {
	if trimmed := strings.TrimSpace(override); trimmed != "" {
		if abs, err := filepath.Abs(trimmed); err == nil {
			return filepath.Clean(abs)
		}
		return filepath.Clean(trimmed)
	}
	if strings.TrimSpace(s.Root) != "" {
		return filepath.Clean(s.Root)
	}
	if cwd, err := os.Getwd(); err == nil {
		return filepath.Clean(cwd)
	}
	return "unknown"
}

func (s Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s Service) mkdirAll(path string, mode os.FileMode) error {
	if s.MkdirAll != nil {
		return s.MkdirAll(path, mode)
	}
	return os.MkdirAll(path, mode)
}

func (s Service) readFile(path string) ([]byte, error) {
	if s.ReadFile != nil {
		return s.ReadFile(path)
	}
	return os.ReadFile(path)
}

func (s Service) writeFile(path string, data []byte, mode os.FileMode) error {
	if s.WriteFile != nil {
		return s.WriteFile(path, data, mode)
	}
	return os.WriteFile(path, data, mode)
}

func (s Service) remove(path string) error {
	if s.Remove != nil {
		return s.Remove(path)
	}
	return os.Remove(path)
}

func (s Service) generatedSlug() string {
	return randomWord(planAdjectives) + "-" + randomWord(planNouns)
}

func slugOrDefault(slug, title string) string {
	if cleaned := slugify(slug); cleaned != "" {
		return cleaned
	}
	return slugify(title)
}

var slugCleaner = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(value string) string {
	lower := strings.ToLower(strings.TrimSpace(value))
	lower = slugCleaner.ReplaceAllString(lower, "-")
	return strings.Trim(lower, "-")
}

func uniqueSlug(base string, records []PlanRecord) string {
	used := map[string]bool{}
	for _, record := range records {
		used[record.Slug] = true
	}
	if !used[base] {
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if !used[candidate] {
			return candidate
		}
	}
}

func uniqueID(base string, records []PlanRecord) string {
	used := map[string]bool{}
	for _, record := range records {
		used[record.ID] = true
	}
	if !used[base] {
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if !used[candidate] {
			return candidate
		}
	}
}

func findPosition(records []PlanRecord, ref string) (int, error) {
	ref = strings.TrimSpace(ref)
	for i, record := range records {
		if record.ID == ref || record.Slug == ref {
			return i, nil
		}
	}
	return -1, fmt.Errorf("plan %q not found", ref)
}

func filterArchived(records []PlanRecord, includeArchived bool) []PlanRecord {
	filtered := make([]PlanRecord, 0, len(records))
	for _, record := range records {
		if record.Archived && !includeArchived {
			continue
		}
		filtered = append(filtered, record)
	}
	sortRecords(filtered)
	return filtered
}

func sortRecords(records []PlanRecord) {
	slices.SortFunc(records, func(a, b PlanRecord) int {
		if !a.CreatedAt.Equal(b.CreatedAt) {
			if a.CreatedAt.After(b.CreatedAt) {
				return -1
			}
			return 1
		}
		return strings.Compare(a.ID, b.ID)
	})
}

func contentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func ensureTrailingNewline(value string) string {
	if strings.HasSuffix(value, "\n") {
		return value
	}
	return value + "\n"
}

func titleFromContentOrPath(content, path string) string {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			return strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
		}
		if trimmed != "" {
			break
		}
	}
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return strings.ReplaceAll(base, "-", " ")
}

func titleFromSlug(slug string) string {
	words := strings.Split(slug, "-")
	for i, word := range words {
		if word == "" {
			continue
		}
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}
	return strings.Join(words, " ")
}

func randomWord(words []string) string {
	if len(words) == 0 {
		return "plan"
	}
	max := big.NewInt(int64(len(words)))
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return words[time.Now().UTC().Nanosecond()%len(words)]
	}
	return words[n.Int64()]
}

var planAdjectives = []string{
	"bright",
	"calm",
	"clear",
	"crisp",
	"direct",
	"fresh",
	"plain",
	"steady",
	"swift",
	"tidy",
}

var planNouns = []string{
	"anchor",
	"bridge",
	"field",
	"frame",
	"harbor",
	"ledger",
	"signal",
	"thread",
	"trail",
	"window",
}
