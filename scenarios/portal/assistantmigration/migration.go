// Package assistantmigration inventories the legacy vrooli-assistant corpus
// without importing it into Portal or starting the legacy runtime. The source
// remains untouched until an operator reviews the exported manifest.
package assistantmigration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	maxFiles         = 4096
	maxFileBytes     = 8 << 20
	maxTotalBytes    = 256 << 20
	manifestMode     = 0o600
	legacyTaskDir    = "data/tasks"
	legacyContextDir = "data/contexts"
)

var ErrConflict = errors.New("current-owner handoff conflicts with existing task")

type Record struct {
	ID           string `json:"id"`
	Kind         string `json:"kind"`
	RelativePath string `json:"relative_path"`
	Bytes        int64  `json:"bytes"`
	SHA256       string `json:"sha256"`
}

type Manifest struct {
	Version    int      `json:"version"`
	SourceRoot string   `json:"source_root"`
	Records    []Record `json:"records"`
}

// Issue is the bounded, metadata-only representation of a legacy Assistant
// task. The original markdown remains the source of truth and is never
// rewritten by the migration package.
type Issue struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Scenario       string `json:"scenario"`
	URL            string `json:"url"`
	CapturedAt     string `json:"captured_at"`
	Description    string `json:"description"`
	ScreenshotPath string `json:"screenshot_path"`
	Status         string `json:"status"`
	RelativePath   string `json:"relative_path"`
}

// EvidenceLink records a relationship that must survive migration. Links are
// metadata only: a screenshot path is not read or copied implicitly.
type EvidenceLink struct {
	IssueID      string `json:"issue_id"`
	Kind         string `json:"kind"`
	RelativePath string `json:"relative_path"`
	Target       string `json:"target,omitempty"`
	SHA256       string `json:"sha256,omitempty"`
}

// CaptureRequest is the current-owner handoff emitted by BuildReview. The
// stable key lets a producer retry after a lost response without admitting a
// second task.
type CaptureRequest struct {
	LegacyID      string         `json:"legacy_id"`
	CaptureKey    string         `json:"capture_key"`
	Owner         string         `json:"owner"`
	Issue         Issue          `json:"issue"`
	EvidenceLinks []EvidenceLink `json:"evidence_links"`
}

// ReviewManifest is a reconciled, operator-reviewable migration projection.
// It contains no bearer credentials and does not authorize source deletion.
type ReviewManifest struct {
	Version              int              `json:"version"`
	SourceRoot           string           `json:"source_root"`
	SourceManifestSHA256 string           `json:"source_manifest_sha256"`
	SourceRecords        int              `json:"source_records"`
	TaskRecords          int              `json:"task_records"`
	ContextRecords       int              `json:"context_records"`
	Links                []EvidenceLink   `json:"links"`
	Requests             []CaptureRequest `json:"requests"`
}

// CaptureReceipt is returned by the current owner after it accepts a request.
// A router must preserve CaptureKey idempotency across retries.
type CaptureReceipt struct {
	CaptureKey string `json:"capture_key"`
	Owner      string `json:"owner"`
	TaskID     string `json:"task_id"`
}

// CaptureRouter is implemented by the current reporting owner (for example,
// a Portal-to-owner or review-bundle adapter). The migration package only
// supplies validated requests and never starts the legacy runtime.
type CaptureRouter interface {
	Capture(context.Context, CaptureRequest) (CaptureReceipt, error)
}

// FileRouter is a local current-owner adapter for operator-reviewed imports.
// It writes private, deterministic task handoffs under owner directories and
// is idempotent when the same capture key is replayed after a lost response.
// The destination must be separate from the legacy source.
type FileRouter struct {
	Root       string
	SourceRoot string
}

func NewFileRouter(root string, source ...string) (*FileRouter, error) {
	root = filepath.Clean(root)
	if root == "." || root == string(filepath.Separator) {
		return nil, errors.New("capture destination is required")
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, errors.New("create capture destination")
	}
	router := &FileRouter{Root: root}
	if len(source) > 0 {
		router.SourceRoot = filepath.Clean(source[0])
	}
	return router, nil
}

func (r *FileRouter) Capture(ctx context.Context, request CaptureRequest) (CaptureReceipt, error) {
	if err := ctx.Err(); err != nil {
		return CaptureReceipt{}, err
	}
	if r == nil || r.Root == "" || request.LegacyID == "" || request.CaptureKey == "" || request.Owner == "" {
		return CaptureReceipt{}, errors.New("capture request or destination is invalid")
	}
	owner, err := safeSegment(request.Owner)
	if err != nil {
		return CaptureReceipt{}, err
	}
	id, err := safeSegment(request.LegacyID)
	if err != nil {
		return CaptureReceipt{}, err
	}
	if _, err := safeSegment(request.CaptureKey); err != nil {
		return CaptureReceipt{}, err
	}
	links, err := r.materializeContext(request, owner, id)
	if err != nil {
		return CaptureReceipt{}, err
	}
	payload := struct {
		Version      int            `json:"version"`
		CaptureKey   string         `json:"capture_key"`
		Owner        string         `json:"owner"`
		LegacyID     string         `json:"legacy_id"`
		Issue        Issue          `json:"issue"`
		EvidenceLink []EvidenceLink `json:"evidence_links"`
	}{1, request.CaptureKey, request.Owner, request.LegacyID, request.Issue, links}
	path := filepath.Join(r.Root, owner, "tasks", id+".json")
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return CaptureReceipt{}, errors.New("encode current-owner handoff")
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return CaptureReceipt{}, errors.New("create current-owner task directory")
	}
	if existing, readErr := os.ReadFile(path); readErr == nil {
		if string(existing) != string(data) {
			return CaptureReceipt{}, ErrConflict
		}
		return CaptureReceipt{CaptureKey: request.CaptureKey, Owner: request.Owner, TaskID: filepath.ToSlash(filepath.Join(owner, "tasks", id+".json"))}, nil
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return CaptureReceipt{}, errors.New("read existing current-owner handoff")
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".handoff-*")
	if err != nil {
		return CaptureReceipt{}, errors.New("create current-owner handoff")
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err = tmp.Chmod(0o600); err == nil {
		_, err = tmp.Write(data)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return CaptureReceipt{}, errors.New("write current-owner handoff")
	}
	if err = os.Rename(tmpName, path); err != nil {
		if existing, readErr := os.ReadFile(path); readErr == nil && string(existing) == string(data) {
			return CaptureReceipt{CaptureKey: request.CaptureKey, Owner: request.Owner, TaskID: filepath.ToSlash(filepath.Join(owner, "tasks", id+".json"))}, nil
		}
		return CaptureReceipt{}, errors.New("publish current-owner handoff")
	}
	if err := ctx.Err(); err != nil {
		return CaptureReceipt{}, err
	}
	return CaptureReceipt{CaptureKey: request.CaptureKey, Owner: request.Owner, TaskID: filepath.ToSlash(filepath.Join(owner, "tasks", id+".json"))}, nil
}

func (r *FileRouter) materializeContext(request CaptureRequest, owner, id string) ([]EvidenceLink, error) {
	links := dedupeLinks(request.EvidenceLinks)
	if r.SourceRoot == "" {
		return links, nil
	}
	for i, link := range links {
		if link.Kind != "context" {
			continue
		}
		if link.RelativePath == "" || link.SHA256 == "" {
			return nil, errors.New("context evidence link is incomplete")
		}
		name, err := safeSegment(filepath.Base(filepath.FromSlash(link.RelativePath)))
		if err != nil {
			return nil, err
		}
		source := filepath.Join(r.SourceRoot, filepath.FromSlash(link.RelativePath))
		data, err := os.ReadFile(source)
		if err != nil || len(data) > maxFileBytes {
			return nil, errors.New("context evidence is unavailable")
		}
		digest := sha256.Sum256(data)
		if hex.EncodeToString(digest[:]) != link.SHA256 {
			return nil, errors.New("context evidence checksum mismatch")
		}
		target := filepath.Join(r.Root, owner, "evidence", id, name)
		if err := writeAtomicPrivate(target, data); err != nil {
			return nil, err
		}
		links[i].Target = filepath.ToSlash(filepath.Join(owner, "evidence", id, name))
	}
	return links, nil
}

func writeAtomicPrivate(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return errors.New("create evidence directory")
	}
	if existing, err := os.ReadFile(path); err == nil {
		if string(existing) == string(data) {
			return nil
		}
		return ErrConflict
	} else if !errors.Is(err, os.ErrNotExist) {
		return errors.New("read existing evidence")
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".evidence-*")
	if err != nil {
		return errors.New("create evidence file")
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err = tmp.Chmod(0o600); err == nil {
		_, err = tmp.Write(data)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return errors.New("write evidence file")
	}
	if err = os.Rename(tmpName, path); err != nil {
		if existing, readErr := os.ReadFile(path); readErr == nil && string(existing) == string(data) {
			return nil
		}
		return errors.New("publish evidence file")
	}
	return nil
}

func Inventory(root string) (Manifest, error) {
	root = filepath.Clean(root)
	if root == "." || root == string(filepath.Separator) {
		return Manifest{}, errors.New("legacy assistant root is required")
	}
	manifest := Manifest{Version: 1, SourceRoot: root, Records: make([]Record, 0)}
	var total int64
	for _, entry := range []struct{ dir, kind string }{{legacyTaskDir, "task"}, {legacyContextDir, "context"}} {
		path := filepath.Join(root, entry.dir)
		files, err := os.ReadDir(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return Manifest{}, fmt.Errorf("read assistant %s: %w", entry.kind, err)
		}
		for _, file := range files {
			if file.IsDir() || strings.HasPrefix(file.Name(), ".") {
				continue
			}
			if len(manifest.Records) >= maxFiles {
				return Manifest{}, errors.New("assistant corpus exceeds file bound")
			}
			full := filepath.Join(path, file.Name())
			info, err := os.Lstat(full)
			if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o022 != 0 || info.Mode().Perm()&0o400 == 0 {
				return Manifest{}, fmt.Errorf("assistant file is not a stable regular file: %s", file.Name())
			}
			if info.Size() > maxFileBytes || total+info.Size() > maxTotalBytes {
				return Manifest{}, errors.New("assistant corpus exceeds byte bound")
			}
			digest, err := digestFile(full, info.Size())
			if err != nil {
				return Manifest{}, err
			}
			id := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
			if id == "" {
				return Manifest{}, errors.New("assistant record has empty id")
			}
			manifest.Records = append(manifest.Records, Record{ID: id, Kind: entry.kind, RelativePath: filepath.ToSlash(filepath.Join(entry.dir, file.Name())), Bytes: info.Size(), SHA256: digest})
			total += info.Size()
		}
	}
	sort.Slice(manifest.Records, func(i, j int) bool { return manifest.Records[i].RelativePath < manifest.Records[j].RelativePath })
	return manifest, nil
}

func Export(root, destination string) (Manifest, error) {
	manifest, err := Inventory(root)
	if err != nil {
		return Manifest{}, err
	}
	if destination == "" || filepath.Clean(destination) == filepath.Clean(root) {
		return Manifest{}, errors.New("export destination must be separate from source")
	}
	if err := os.MkdirAll(destination, 0o700); err != nil {
		return Manifest{}, errors.New("create migration destination")
	}
	for _, record := range manifest.Records {
		source := filepath.Join(root, filepath.FromSlash(record.RelativePath))
		target := filepath.Join(destination, filepath.FromSlash(record.RelativePath))
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return Manifest{}, errors.New("create migration record directory")
		}
		data, err := os.ReadFile(source)
		if err != nil {
			return Manifest{}, errors.New("read migration record")
		}
		if err := os.WriteFile(target, data, manifestMode); err != nil {
			return Manifest{}, errors.New("write migration record")
		}
	}
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return Manifest{}, errors.New("encode migration manifest")
	}
	if err := os.WriteFile(filepath.Join(destination, "manifest.json"), encoded, manifestMode); err != nil {
		return Manifest{}, errors.New("write migration manifest")
	}
	return manifest, nil
}

func Reconcile(root string, manifest Manifest) error {
	actual, err := Inventory(root)
	if err != nil {
		return err
	}
	if actual.Version != manifest.Version || len(actual.Records) != len(manifest.Records) {
		return errors.New("assistant corpus changed since inventory")
	}
	for i := range actual.Records {
		if actual.Records[i] != manifest.Records[i] {
			return errors.New("assistant corpus checksum or identity mismatch")
		}
	}
	return nil
}

// ParseIssue parses one legacy task markdown file without following paths from
// its contents. It accepts the stable format emitted by vrooli-assistant and
// rejects malformed records before they can become an owner task.
func ParseIssue(root string, record Record) (Issue, error) {
	if record.Kind != "task" || record.RelativePath == "" {
		return Issue{}, errors.New("migration record is not a task")
	}
	data, err := readRecord(root, record)
	if err != nil {
		return Issue{}, err
	}
	issue := Issue{
		ID:             record.ID,
		Title:          sectionHeading(string(data), "# Issue:"),
		Scenario:       fieldLine(string(data), "**Scenario**:"),
		URL:            fieldLine(string(data), "**URL**:"),
		CapturedAt:     fieldLine(string(data), "**Captured**:"),
		Description:    sectionBody(string(data), "## Description"),
		ScreenshotPath: contextField(string(data), "Screenshot"),
		Status:         contextField(string(data), "Status"),
		RelativePath:   record.RelativePath,
	}
	if issue.ID == "" || issue.Title == "" || issue.Description == "" {
		return Issue{}, errors.New("legacy task is missing identity, title, or description")
	}
	if contextID := contextField(string(data), "Issue ID"); contextID != "" && canonicalIssueID(contextID) != canonicalIssueID(issue.ID) {
		return Issue{}, errors.New("legacy task issue identity mismatch")
	}
	if issue.CapturedAt != "" {
		if _, err := time.Parse(time.RFC3339, issue.CapturedAt); err != nil {
			return Issue{}, errors.New("legacy task has invalid capture time")
		}
	}
	return issue, nil
}

// BuildReview reconciles the source manifest and constructs deterministic
// current-owner requests. It is safe to run repeatedly: unchanged source
// records produce identical capture keys and ordered output.
func BuildReview(root string, manifest Manifest) (ReviewManifest, error) {
	if err := Reconcile(root, manifest); err != nil {
		return ReviewManifest{}, err
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return ReviewManifest{}, errors.New("encode source migration manifest")
	}
	manifestDigest := sha256.Sum256(manifestBytes)
	review := ReviewManifest{
		Version:              1,
		SourceRoot:           filepath.Clean(root),
		SourceManifestSHA256: hex.EncodeToString(manifestDigest[:]),
		SourceRecords:        len(manifest.Records),
		Links:                make([]EvidenceLink, 0),
		Requests:             make([]CaptureRequest, 0),
	}
	contextByIssue := make(map[string][]EvidenceLink)
	for _, record := range manifest.Records {
		switch record.Kind {
		case "context":
			review.ContextRecords++
			issueID, screenshot, parseErr := parseContextMetadata(root, record)
			if parseErr != nil {
				return ReviewManifest{}, fmt.Errorf("parse %s: %w", record.RelativePath, parseErr)
			}
			if issueID != "" {
				link := EvidenceLink{IssueID: canonicalIssueID(issueID), Kind: "context", RelativePath: record.RelativePath, Target: screenshot, SHA256: record.SHA256}
				contextByIssue[canonicalIssueID(issueID)] = append(contextByIssue[canonicalIssueID(issueID)], link)
				review.Links = append(review.Links, link)
			}
		case "task":
			review.TaskRecords++
			issue, err := ParseIssue(root, record)
			if err != nil {
				return ReviewManifest{}, fmt.Errorf("parse %s: %w", record.RelativePath, err)
			}
			owner := currentOwner(issue.Scenario)
			links := []EvidenceLink{{IssueID: issue.ID, Kind: "legacy-record", RelativePath: issue.RelativePath, SHA256: record.SHA256}}
			if issue.ScreenshotPath != "" {
				links = append(links, EvidenceLink{IssueID: issue.ID, Kind: "screenshot", RelativePath: issue.RelativePath, Target: issue.ScreenshotPath})
			}
			links = append(links, contextByIssue[canonicalIssueID(issue.ID)]...)
			keyHash := sha256.Sum256([]byte(issue.ID + "\x00" + record.SHA256 + "\x00" + owner))
			request := CaptureRequest{LegacyID: issue.ID, CaptureKey: hex.EncodeToString(keyHash[:]), Owner: owner, Issue: issue, EvidenceLinks: links}
			review.Requests = append(review.Requests, request)
		}
	}
	// Context files can appear before or after their task in the manifest. Add
	// all links after parsing so the request projection is order-independent.
	for i := range review.Requests {
		review.Requests[i].EvidenceLinks = append(review.Requests[i].EvidenceLinks, contextByIssue[canonicalIssueID(review.Requests[i].LegacyID)]...)
		review.Requests[i].EvidenceLinks = dedupeLinks(review.Requests[i].EvidenceLinks)
	}
	sort.Slice(review.Links, func(i, j int) bool { return linkKey(review.Links[i]) < linkKey(review.Links[j]) })
	sort.Slice(review.Requests, func(i, j int) bool { return review.Requests[i].LegacyID < review.Requests[j].LegacyID })
	return review, nil
}

// Route sends each unique capture key to the current owner once. Duplicate
// requests in a handoff are collapsed before calling the owner, while the
// owner remains responsible for durable idempotency across process retries.
func Route(ctx context.Context, review ReviewManifest, router CaptureRouter) ([]CaptureReceipt, error) {
	if router == nil {
		return nil, errors.New("current-owner capture router is required")
	}
	seen := make(map[string]string, len(review.Requests))
	receipts := make([]CaptureReceipt, 0, len(review.Requests))
	for _, request := range review.Requests {
		if request.LegacyID == "" || request.CaptureKey == "" || request.Owner == "" {
			return nil, errors.New("capture request is missing identity")
		}
		encoded, err := json.Marshal(request)
		if err != nil {
			return nil, errors.New("encode capture request identity")
		}
		identity := string(encoded)
		if previous, ok := seen[request.CaptureKey]; ok {
			if previous != identity {
				return nil, ErrConflict
			}
			continue
		}
		receipt, err := router.Capture(ctx, request)
		if err != nil {
			return nil, err
		}
		if receipt.CaptureKey != request.CaptureKey || receipt.Owner != request.Owner || receipt.TaskID == "" {
			return nil, errors.New("current owner returned an invalid capture receipt")
		}
		seen[request.CaptureKey] = identity
		receipts = append(receipts, receipt)
	}
	return receipts, nil
}

func readRecord(root string, record Record) ([]byte, error) {
	path := filepath.Join(root, filepath.FromSlash(record.RelativePath))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.New("read migration record")
	}
	if int64(len(data)) != record.Bytes || int64(len(data)) > maxFileBytes {
		return nil, errors.New("migration record changed size")
	}
	digest := sha256.Sum256(data)
	if hex.EncodeToString(digest[:]) != record.SHA256 {
		return nil, errors.New("migration record checksum mismatch")
	}
	return data, nil
}

func sectionHeading(text, prefix string) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return ""
}

func fieldLine(text, prefix string) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), prefix) {
			return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), prefix))
		}
	}
	return ""
}

func contextField(text, label string) string {
	if value := fieldLine(text, "- "+label+":"); value != "" {
		return value
	}
	return fieldLine(text, label+":")
}

func sectionBody(text, heading string) string {
	lines := strings.Split(text, "\n")
	inside := false
	var body []string
	for _, line := range lines {
		if strings.TrimSpace(line) == heading {
			inside = true
			continue
		}
		if inside && strings.HasPrefix(strings.TrimSpace(line), "## ") {
			break
		}
		if inside {
			body = append(body, line)
		}
	}
	return strings.TrimSpace(strings.Join(body, "\n"))
}

func parseContextMetadata(root string, record Record) (string, string, error) {
	data, err := readRecord(root, record)
	if err != nil {
		return "", "", err
	}
	return contextField(string(data), "Issue ID"), contextField(string(data), "Screenshot"), nil
}

func currentOwner(scenario string) string {
	scenario = strings.TrimSpace(strings.ToLower(scenario))
	if scenario == "" || scenario == "manual" {
		return "scenario-qa"
	}
	return scenario
}

func canonicalIssueID(id string) string {
	return strings.TrimPrefix(strings.TrimSpace(id), "issue-")
}

func safeSegment(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || value == "." || value == ".." || strings.ContainsAny(value, `/\\`) {
		return "", errors.New("capture identity contains an unsafe path segment")
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || strings.ContainsRune("._-", r) {
			continue
		}
		return "", errors.New("capture identity contains an unsafe path segment")
	}
	return value, nil
}

func linkKey(link EvidenceLink) string {
	return link.IssueID + "\x00" + link.Kind + "\x00" + link.RelativePath + "\x00" + link.Target
}

func dedupeLinks(links []EvidenceLink) []EvidenceLink {
	seen := make(map[string]struct{}, len(links))
	out := make([]EvidenceLink, 0, len(links))
	for _, link := range links {
		key := linkKey(link)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, link)
	}
	sort.Slice(out, func(i, j int) bool { return linkKey(out[i]) < linkKey(out[j]) })
	return out
}

func digestFile(path string, size int64) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", errors.New("open assistant record")
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.CopyN(hash, file, size); err != nil && !errors.Is(err, io.EOF) {
		return "", errors.New("hash assistant record")
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
