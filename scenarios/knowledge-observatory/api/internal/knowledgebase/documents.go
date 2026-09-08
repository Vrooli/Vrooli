// Package knowledgebase owns bounded, read-only source evidence for agent workflows.
package knowledgebase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"io/fs"
	"os"
	"path"
	"regexp"
	"strings"
	"unicode/utf8"

	kov1 "github.com/vrooli/vrooli/packages/proto/gen/go/knowledge-observatory/v1"
	"google.golang.org/protobuf/types/known/structpb"
	"knowledge-observatory/internal/doccontract"
)

const MaxBytes = 1 << 20

var (
	ErrInvalid  = errors.New("invalid document request")
	ErrRevision = errors.New("source revision changed")
)

type Service struct{ RepoRoot string }

func Metadata(value map[string]any) *structpb.Struct {
	body, _ := json.Marshal(value)
	out := &structpb.Struct{}
	_ = out.UnmarshalJSON(body)
	return out
}

// PortablePath accepts a repository-relative path independent of the serving OS.
// Colon and backslash reject Windows drive/UNC forms even on a Unix server.
func PortablePath(value string) (string, error) {
	if value == "" || strings.ContainsAny(value, "\\:\x00") || strings.HasPrefix(value, "/") {
		return "", ErrInvalid
	}
	for _, part := range strings.Split(value, "/") {
		if part == ".." {
			return "", ErrInvalid
		}
	}
	return path.Clean(value), nil
}

func prose(p string) bool {
	switch strings.ToLower(path.Ext(p)) {
	case ".md", ".mdx", ".markdown", ".txt", ".html", ".htm", ".json":
		return true
	}
	return false
}

// References can originate in code and configuration as well as prose.
func referenceSource(p string) bool {
	if prose(p) {
		return true
	}
	switch strings.ToLower(path.Ext(p)) {
	case ".go", ".ts", ".tsx", ".js", ".jsx", ".py", ".sh", ".yaml", ".yml", ".toml":
		return true
	}
	return false
}

func read(root *os.Root, p string) ([]byte, error) {
	f, err := root.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > MaxBytes {
		return nil, fmt.Errorf("%w: unsupported or oversized source", ErrInvalid)
	}
	body, err := io.ReadAll(io.LimitReader(f, MaxBytes+1))
	if len(body) > MaxBytes {
		return nil, ErrInvalid
	}
	if !utf8.Valid(body) {
		return nil, fmt.Errorf("%w: source is not UTF-8", ErrInvalid)
	}
	return body, err
}

func (s *Service) Inspect(ctx context.Context, req *kov1.InspectDocumentRequest) (*kov1.InspectDocumentResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p, err := PortablePath(req.Path)
	if err != nil || !prose(p) || req.Offset < 0 || req.Limit < 0 || req.Limit > 12000 {
		return nil, ErrInvalid
	}
	root, err := os.OpenRoot(s.RepoRoot)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	body, err := read(root, p)
	if req.AllowMissing && errors.Is(err, fs.ErrNotExist) {
		// Lstat distinguishes an absent path from a dangling symlink.
		if _, statErr := root.Lstat(p); errors.Is(statErr, fs.ErrNotExist) {
			return &kov1.InspectDocumentResponse{Path: p, Missing: true}, nil
		}
	}
	if err != nil {
		return nil, err
	}
	hash := doccontract.FileSHA256(body)
	if req.ExpectedSha256 != "" && req.ExpectedSha256 != hash {
		return nil, ErrRevision
	}
	chars := []rune(string(body))
	start := int(req.Offset)
	if start > len(chars) {
		return nil, ErrInvalid
	}
	limit := int(req.Limit)
	if limit == 0 {
		limit = 4000
	}
	end := start + limit
	if end > len(chars) {
		end = len(chars)
	}
	refs, truncated := references(root, p, string(body))
	return &kov1.InspectDocumentResponse{Path: p, Sha256: hash, Content: string(chars[start:end]), Offset: int32(start), NextOffset: int32(end), Truncated: end < len(chars), SizeBytes: int64(len(body)), Metadata: Metadata(doccontract.ReadKnowledgeMetadata(s.RepoRoot, p)), References: refs, ReferencesTruncated: truncated}, nil
}

var (
	markdownLink = regexp.MustCompile(`\[[^\]\n]*\]\(<?([^\s)>]+)>?(?:\s+[^)]*)?\)`)
	typedPath    = regexp.MustCompile(`\bpath:([^\s\x60<>]+)`)
)

var maintenanceReferencePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?m)^\s{0,3}\[[^]\n]+\]:\s*<?([^\s>]+)>?`),
	regexp.MustCompile(`(?i)(?:href|src)\s*=\s*["']([^"']+)["']`),
	regexp.MustCompile(`\[(?:DOC|CODE):\s*([^]\s]+)\]`),
	regexp.MustCompile(`(?m)//\s*DOC:\s*([^\s]+)`),
}

func references(root *os.Root, from, body string) ([]*kov1.DocumentReference, bool) {
	refs := []*kov1.DocumentReference{}
	seen := map[string]bool{}
	matches := markdownLink.FindAllStringSubmatch(body, 257)
	matches = append(matches, typedPath.FindAllStringSubmatch(body, 257)...)
	// Definitions and examples may be illustrative; inspect the original source.
	for _, pattern := range maintenanceReferencePatterns {
		matches = append(matches, pattern.FindAllStringSubmatch(body, 257)...)
	}
	for _, m := range matches {
		target := html.UnescapeString(strings.TrimRight(m[1], ".,;"))
		if seen[target] {
			continue
		}
		seen[target] = true
		if len(refs) >= 128 {
			return refs, true
		}
		kind := "local"
		dest := ""
		exists := false
		if strings.HasPrefix(target, "#") {
			kind = "anchor"
		} else if strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") {
			kind = "external"
		} else {
			raw := strings.Split(strings.Split(target, "#")[0], "?")[0]
			if strings.HasPrefix(m[0], "path:") || strings.HasPrefix(raw, "/") {
				dest = path.Clean(strings.TrimPrefix(raw, "/"))
			} else if strings.Contains(m[0], "DOC:") || strings.Contains(m[0], "CODE:") {
				parts := strings.Split(from, "/")
				base := "."
				if len(parts) > 2 && parts[0] == "scenarios" {
					base = path.Join(parts[0], parts[1])
				}
				dest = path.Join(base, raw)
			} else {
				dest = path.Join(path.Dir(from), raw)
			}
			if _, err := PortablePath(dest); err != nil {
				kind = "outside_repository"
				dest = ""
			} else {
				_, err = root.Stat(dest)
				exists = err == nil
			}
		}
		refs = append(refs, &kov1.DocumentReference{Target: target, Path: dest, Exists: exists, Kind: kind})
	}
	return refs, len(matches) >= 257
}

var skipped = map[string]bool{".git": true, "node_modules": true, "vendor": true, ".vrooli": true, "gen": true, "dist": true, "build": true, ".venv": true}

func (s *Service) Review(ctx context.Context, req *kov1.ReviewDocumentsRequest) (*kov1.ReviewDocumentsResponse, error) {
	if len(req.Paths) == 0 || len(req.Paths) > 8 || req.MaxFiles < 0 || req.MaxFiles > 1000 {
		return nil, ErrInvalid
	}
	base, err := PortablePath(req.BasePath)
	if err != nil {
		return nil, err
	}
	capFiles := int(req.MaxFiles)
	if capFiles == 0 {
		capFiles = 300
	}
	root, err := os.OpenRoot(s.RepoRoot)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	info, err := root.Stat(base)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, ErrInvalid
	}
	out := &kov1.ReviewDocumentsResponse{BasePath: base, Gaps: []string{"Semantic agreement and authority require editorial review; reference scan covers inline Markdown links, link definitions, HTML href/src, DOC/CODE markers and path: references in supported text files; dynamic references, arbitrary manifest paths and heading validity require owner checks."}}
	selected := map[string]*kov1.InspectDocumentResponse{}
	for _, p := range req.Paths {
		doc, err := s.Inspect(ctx, &kov1.InspectDocumentRequest{Path: p, Limit: 2000})
		if err != nil {
			return nil, err
		}
		if selected[doc.Path] != nil {
			continue
		}
		selected[doc.Path] = doc
		out.Documents = append(out.Documents, doc)
		if doc.ReferencesTruncated {
			out.Truncated = true
			out.Gaps = append(out.Gaps, "Selected source references truncated: "+doc.Path)
		}
	}
	add := func(p, kind, related, detail string) {
		if len(out.Observations) >= 100 {
			out.Truncated = true
			return
		}
		out.Observations = append(out.Observations, &kov1.DocumentObservation{Path: p, Kind: kind, RelatedPath: related, Detail: detail})
	}
	visited := 0
	err = fs.WalkDir(root.FS(), base, func(p string, d fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		visited++
		if visited > 10000 {
			out.Truncated = true
			return fs.SkipAll
		}
		if walkErr != nil {
			out.Truncated = true
			if len(out.Gaps) < 12 {
				out.Gaps = append(out.Gaps, "Unreadable path: "+p)
			}
			return nil
		}
		if d.IsDir() {
			if p != base && skipped[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}
		if !referenceSource(p) {
			return nil
		}
		if int(out.FilesChecked) >= capFiles {
			out.Truncated = true
			return fs.SkipAll
		}
		out.FilesChecked++
		body, err := read(root, p)
		if err != nil {
			out.Truncated = true
			if len(out.Gaps) < 12 {
				out.Gaps = append(out.Gaps, "Unreadable/oversized source: "+p)
			}
			return nil
		}
		hash := doccontract.FileSHA256(body)
		for target, doc := range selected {
			if p != target && hash == doc.Sha256 {
				add(target, "exact_duplicate", p, "Identical file bytes; ownership and incoming references still require review.")
			}
		}
		refs, more := references(root, p, string(body))
		if more {
			out.Truncated = true
		}
		for _, ref := range refs {
			if _, ok := selected[ref.Path]; ok && p != ref.Path {
				add(ref.Path, "incoming_reference", p, ref.Target)
			}
			if selected[p] != nil && ref.Kind == "local" && !ref.Exists {
				add(p, "missing_reference", ref.Path, ref.Target)
			}
		}
		if selected[p] != nil {
			for _, cue := range []string{"/home/", "/Users/", "C:\\", "systemctl", "/proc/", "apt-get", "brew install", "minimouse"} {
				if strings.Contains(string(body), cue) {
					add(p, "portability_review", "", "Scope this OS/machine cue; occurrence alone is not a defect: "+cue)
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
