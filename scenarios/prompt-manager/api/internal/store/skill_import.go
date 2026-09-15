package store

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var immutableCommit = regexp.MustCompile(`^[0-9a-fA-F]{7,64}$`)
var checksumPattern = regexp.MustCompile(`^sha256:[0-9a-fA-F]{64}$`)

// ImportRequest accepts only caller-pinned local material. Network fetching is
// intentionally outside the store so the bytes can be reviewed and hashed.
type ImportRequest struct {
	SourceDir, SourceURL, Commit, License, Checksum, ImportedBy, UpstreamVersion, ID string
}

// ImportSkill verifies pinned material and writes it to an inactive vendor pack.
func (s *FileSkillStore) ImportSkill(req ImportRequest) (*Skill, error) {
	if strings.TrimSpace(req.SourceURL) == "" || !immutableCommit.MatchString(req.Commit) {
		return nil, errors.New("import requires a source URL and immutable hexadecimal commit")
	}
	if strings.TrimSpace(req.License) == "" || !checksumPattern.MatchString(req.Checksum) {
		return nil, errors.New("import requires a license and sha256 checksum")
	}
	if strings.TrimSpace(req.ImportedBy) == "" || !isValidSkillID(req.ID) {
		return nil, errors.New("import requires importedBy and a valid skill ID")
	}
	contentPath := filepath.Join(req.SourceDir, "SKILL.md")
	contentBytes, err := os.ReadFile(contentPath)
	if err != nil {
		return nil, fmt.Errorf("read pinned skill: %w", err)
	}
	if err := validateSkillFrontmatter(string(contentBytes), req.ID); err != nil {
		return nil, err
	}
	digest := sha256.Sum256(contentBytes)
	actual := "sha256:" + hex.EncodeToString(digest[:])
	if !strings.EqualFold(actual, req.Checksum) {
		return nil, fmt.Errorf("checksum mismatch: expected %s, got %s", req.Checksum, actual)
	}

	skillDir := filepath.Join(s.packsDir(), "vendor", req.ID)
	if _, err := os.Stat(skillDir); err == nil {
		return nil, fmt.Errorf("import target already exists: %s", req.ID)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("check import target: %w", err)
	}
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		return nil, fmt.Errorf("create vendor skill: %w", err)
	}
	importedAt := time.Now().UTC().Format(time.RFC3339)
	skill := &Skill{BaseEntity: BaseEntity{Kind: KindSkill, SchemaVersion: CurrentSchemaVersion}, ID: req.ID, Name: req.ID, Status: StatusDraft, Entry: "SKILL.md", Pack: "vendor", Timestamps: NewTimestamps(), Origin: &SkillOrigin{Kind: OriginImported, SourceURL: req.SourceURL, Commit: req.Commit, License: req.License, Checksum: actual, ImportedBy: req.ImportedBy, ImportedAt: importedAt, UpstreamVersion: req.UpstreamVersion, Review: SkillReview{Verdict: ReviewVerdictPending}}}
	if err := SaveJSON(filepath.Join(skillDir, "skill.json"), skill); err != nil {
		_ = os.RemoveAll(skillDir)
		return nil, fmt.Errorf("write imported metadata: %w", err)
	}
	contentWithOrigin, err := addImportedOriginFrontmatter(string(contentBytes), *skill.Origin)
	if err != nil {
		_ = os.RemoveAll(skillDir)
		return nil, fmt.Errorf("write imported origin: %w", err)
	}
	if err := WriteContent(filepath.Join(skillDir, "SKILL.md"), contentWithOrigin); err != nil {
		_ = os.RemoveAll(skillDir)
		return nil, fmt.Errorf("write imported content: %w", err)
	}
	if err := s.ensureVendorInactive(); err != nil {
		_ = os.RemoveAll(skillDir)
		return nil, err
	}
	return skill, nil
}

func addImportedOriginFrontmatter(content string, origin SkillOrigin) (string, error) {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", errors.New("imported content has no frontmatter")
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return "", errors.New("imported content has unterminated frontmatter")
	}
	header := strings.Join(lines[1:end], "\n")
	if strings.Contains(header, "\n  origin:") || strings.HasPrefix(header, "  origin:") {
		return "", errors.New("imported content already declares an origin block")
	}
	block := []string{}
	if strings.Contains(header, "\nmetadata:") || strings.HasPrefix(header, "metadata:") {
		block = []string{"  origin:"}
	} else {
		block = []string{"metadata:", "  origin:"}
	}
	block = append(block,
		"    kind: imported",
		"    source_url: "+yamlQuote(origin.SourceURL),
		"    commit: "+yamlQuote(origin.Commit),
		"    license: "+yamlQuote(origin.License),
		"    checksum: "+yamlQuote(origin.Checksum),
		"    imported_by: "+yamlQuote(origin.ImportedBy),
		"    imported_at: "+yamlQuote(origin.ImportedAt),
	)
	if origin.UpstreamVersion != "" {
		block = append(block, "    upstream_version: "+yamlQuote(origin.UpstreamVersion))
	}
	block = append(block, "    review:", "      verdict: pending")
	lines = append(lines[:end], append(block, lines[end:]...)...)
	return strings.Join(lines, "\n"), nil
}

func yamlQuote(value string) string {
	return `"` + strings.ReplaceAll(strings.ReplaceAll(value, `\`, `\\`), `"`, `\"`) + `"`
}

// ReviewImportedSkill records an independent human verdict without changing
// immutable origin fields. A passed skill becomes active; a rejected one stays
// out of every active projection.
func (s *FileSkillStore) ReviewImportedSkill(id, reviewer, verdict string) error {
	skill, err := s.loadSkill("vendor", id)
	if err != nil {
		return err
	}
	if skill.Origin == nil || skill.Origin.Kind != OriginImported {
		return errors.New("skill is not imported")
	}
	if reviewer == "" || reviewer == skill.Origin.ImportedBy {
		return errors.New("reviewer must be distinct from importer")
	}
	if verdict != ReviewVerdictPassed && verdict != ReviewVerdictRejected {
		return errors.New("verdict must be passed or rejected")
	}
	skill.Origin.Review = SkillReview{Verdict: verdict, Reviewer: reviewer, ReviewedAt: time.Now().UTC().Format(time.RFC3339)}
	if verdict == ReviewVerdictPassed {
		skill.Status = StatusActive
		if err := s.activateVendor(); err != nil {
			return err
		}
	} else {
		skill.Status = StatusArchived
	}
	return SaveJSON(filepath.Join(s.packsDir(), "vendor", id, "skill.json"), skill)
}

// ImportedSkillOverlayPath is the only supported mutation lane for vendored
// content; the immutable imported SKILL.md is never edited in place.
func (s *FileSkillStore) ImportedSkillOverlayPath(id string) string {
	return filepath.Join(s.packsDir(), "vendor", id, "overlays")
}

// WriteImportedSkillOverlay stores a patch beside, rather than over, the
// pinned upstream bytes. The filename is intentionally constrained so an
// overlay cannot escape its skill's overlay directory.
func (s *FileSkillStore) WriteImportedSkillOverlay(id, filename, patch string) (string, error) {
	if _, err := s.loadSkill("vendor", id); err != nil {
		return "", err
	}
	if filename == "" || filepath.Base(filename) != filename || strings.Contains(filename, string(filepath.Separator)) {
		return "", errors.New("overlay filename must be a single non-empty file name")
	}
	dir := s.ImportedSkillOverlayPath(id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(patch), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

// ImportedSkillStaleness compares the recorded upstream version with a caller
// supplied current version. The source fetcher remains outside the store.
func (s *FileSkillStore) ImportedSkillStaleness(id, currentVersion string) (recorded string, stale bool, err error) {
	skill, err := s.loadSkill("vendor", id)
	if err != nil {
		return "", false, err
	}
	if skill.Origin == nil || skill.Origin.Kind != OriginImported {
		return "", false, errors.New("skill is not imported")
	}
	if strings.TrimSpace(skill.Origin.UpstreamVersion) == "" || strings.TrimSpace(currentVersion) == "" {
		return skill.Origin.UpstreamVersion, false, nil
	}
	return skill.Origin.UpstreamVersion, skill.Origin.UpstreamVersion != currentVersion, nil
}

func (s *FileSkillStore) ensureVendorInactive() error {
	order, err := s.getPackOrder()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if order == nil {
		order = &PackOrder{}
	}
	for _, pack := range order.ActivePacks {
		if pack == "vendor" {
			return fmt.Errorf("vendor pack unexpectedly active")
		}
	}
	for _, pack := range order.InactivePacks {
		if pack == "vendor" {
			return nil
		}
	}
	order.InactivePacks = append(order.InactivePacks, "vendor")
	return SaveJSON(filepath.Join(s.skillsDir(), "_pack-order.json"), order)
}

func (s *FileSkillStore) activateVendor() error {
	order, err := s.getPackOrder()
	if err != nil {
		return err
	}
	filtered := order.InactivePacks[:0]
	for _, pack := range order.InactivePacks {
		if pack == "vendor" {
		} else {
			filtered = append(filtered, pack)
		}
	}
	order.InactivePacks = filtered
	active := false
	for _, pack := range order.ActivePacks {
		if pack == "vendor" {
			active = true
		}
	}
	if !active {
		order.ActivePacks = append([]string{"vendor"}, order.ActivePacks...)
	}
	return SaveJSON(filepath.Join(s.skillsDir(), "_pack-order.json"), order)
}

func validateSkillFrontmatter(content, expectedID string) error {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "---\n") {
		return errors.New("imported SKILL.md requires YAML frontmatter")
	}
	end := strings.Index(trimmed[4:], "\n---")
	if end < 0 {
		return errors.New("imported SKILL.md has unterminated frontmatter")
	}
	header := trimmed[4 : 4+end]
	if !strings.Contains(header, "name:") || !strings.Contains(header, "description:") {
		return errors.New("frontmatter requires name and description")
	}
	for _, line := range strings.Split(header, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "name:") {
			name := strings.Trim(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "name:")), "\"'")
			if name != expectedID {
				return fmt.Errorf("frontmatter name must match skill ID %q", expectedID)
			}
		}
	}
	return nil
}
