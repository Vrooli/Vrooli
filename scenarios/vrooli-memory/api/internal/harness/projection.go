package harness

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	sourcerecall "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/recall"
	sourceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/recall/recall_v1connect"
)

// ErrMalformedManagedBlock is returned before any write when a projection
// target has only one wake marker, reversed markers, or multiple marker pairs.
// Refusing the write protects curated content from an ambiguous splice.
type ErrMalformedManagedBlock struct{ Path string }

func (e ErrMalformedManagedBlock) Error() string {
	return fmt.Sprintf("malformed vrooli-memory wake block in %s", e.Path)
}

type projectionTarget struct {
	Runtime, Path, Section string
	Cap, LineCap           int
}

type ProjectionResult struct {
	Path, Content        string
	SizeBytes, SizeLines int64
	ByteCap, LineCap     int64
	Overflow             bool
	DryRun               bool
}

type Projector struct {
	db      *sql.DB
	wake    sourceconnect.RecallServiceClient
	roots   *filerouting.RoutedRoots
	targets map[string]projectionTarget
}

func NewProjector(db *sql.DB, wake sourceconnect.RecallServiceClient, roots ...*filerouting.RoutedRoots) *Projector {
	home, _ := os.UserHomeDir()
	workspace, _ := os.Getwd()
	if configured := strings.TrimSpace(os.Getenv(workspaceRootEnv)); configured != "" {
		workspace = configured
	}
	workspace, _ = filepath.Abs(workspace)
	targets, err := LoadMemoryProjectionTargets(discoverResourcesDir(), home, workspace)
	if err != nil {
		targets = map[string]projectionTarget{}
	}
	p := &Projector{db: db, wake: wake, targets: targets}
	if len(roots) > 0 {
		p.roots = roots[0]
	}
	return p
}

func (p *Projector) Target(runtime string) (string, bool) {
	t, ok := p.targets[runtime]
	return t.Path, ok
}

// TargetPaths returns the configured projection paths in deterministic order.
// Importers use this set to keep generated output one-directional.
func (p *Projector) TargetPaths() []string {
	paths := make([]string, 0, len(p.targets))
	for _, target := range p.targets {
		paths = append(paths, target.Path)
	}
	sort.Strings(paths)
	return paths
}

// Runtimes returns every configured projection target in stable order.
func (p *Projector) Runtimes() []string {
	out := make([]string, 0, len(p.targets))
	for runtime := range p.targets {
		out = append(out, runtime)
	}
	sort.Strings(out)
	return out
}

func (p *Projector) Project(ctx context.Context, runtime string, dryRun bool) (ProjectionResult, error) {
	t, ok := p.targets[runtime]
	if !ok {
		return ProjectionResult{}, fmt.Errorf("unsupported projection runtime %q", runtime)
	}
	path, err := p.targetPath(ctx, t)
	if err != nil {
		return ProjectionResult{}, err
	}
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return ProjectionResult{}, err
	}
	existingText := string(existing)
	// Before managed wake markers existed, vrooli-memory wrote the entire
	// target file and prefixed it with generatedHeader. That prefix is this
	// service's old generated artifact, not operator-owned memory. Treat it as
	// empty during the first marker-based projection; otherwise migration would
	// preserve stale generated content as if it were curated content forever.
	existingText = migrateLegacyProjection(existingText)
	if t.Section != "" && generatedOnly(existingText) {
		existingText = ""
	}
	curated := curatedBytes(existingText)
	curatedLines := renderedLineCount(curated)
	remaining := t.LineCap - curatedLines
	if remaining < 0 {
		remaining = 0
	}
	if _, err := spliceProjectionBlock(path, existingText, "", t.Section); err != nil {
		return ProjectionResult{}, err
	}
	wakeResponse, err := p.wake.Wake(ctx, connect.NewRequest(&sourcerecall.WakeRequest{LineBudget: int32(remaining), Scope: "agent-memory"}))
	if err != nil {
		return ProjectionResult{}, err
	}
	content := ""
	fits := func(candidate string) (bool, error) {
		projected, err := spliceProjectionBlock(path, existingText, candidate, t.Section)
		if err != nil {
			return false, err
		}
		return len(projected) <= t.Cap && renderedLineCount(projected) <= t.LineCap, nil
	}
	const header = generatedHeader + "# Unified Vrooli Memory\n\n"
	if ok, err := fits(header); err != nil {
		return ProjectionResult{}, err
	} else if ok {
		content = header
	}
	overflow := wakeResponse.Msg.GetOverflow() || curatedLines > t.LineCap || content == ""
	for _, hit := range wakeResponse.Msg.GetHits() {
		chunk, cost := renderedHit(hit.GetText())
		if content == "" || renderedLineCount(content)+cost+curatedLines > t.LineCap || len(content)+len(chunk) > t.Cap {
			overflow = true
			break
		}
		ok, err := fits(content + chunk)
		if err != nil {
			return ProjectionResult{}, err
		}
		if !ok {
			overflow = true
			break
		}
		content += chunk
	}
	return p.finish(ctx, t, path, existingText, content, overflow, dryRun)
}

func renderedHit(text string) (string, int) {
	chunk := "- " + strings.TrimSpace(text) + "\n\n"
	return chunk, strings.Count(chunk, "\n")
}

func renderedLineCount(content string) int {
	if content == "" {
		return 0
	}
	lines := strings.Count(content, "\n")
	if !strings.HasSuffix(content, "\n") {
		lines++
	}
	return lines
}

func curatedBytes(existing string) string {
	startCount := strings.Count(existing, wakeStart)
	endCount := strings.Count(existing, wakeEnd)
	if startCount == 0 && endCount == 0 {
		return existing
	}
	if startCount != 1 || endCount != 1 {
		return existing
	}
	start := strings.Index(existing, wakeStart)
	end := strings.Index(existing, wakeEnd)
	if end < start {
		return existing
	}
	return existing[:start] + existing[end+len(wakeEnd):]
}

func migrateLegacyProjection(existing string) string {
	if !strings.HasPrefix(existing, legacyGeneratedHeader) {
		return existing
	}
	start := strings.Index(existing, wakeStart)
	if start < 0 {
		return ""
	}
	end := strings.Index(existing[start+len(wakeStart):], wakeEnd)
	if end < 0 {
		// Leave an incomplete file for the normal malformed-block guard rather
		// than deleting bytes that cannot be classified safely.
		return existing
	}
	return existing[start+len(wakeStart)+end+len(wakeEnd):]
}

func (p *Projector) finish(ctx context.Context, t projectionTarget, path, existing, content string, overflow, dryRun bool) (ProjectionResult, error) {
	projected, err := spliceProjectionBlock(path, string(existing), content, t.Section)
	if err != nil {
		return ProjectionResult{}, err
	}
	projectedLines := renderedLineCount(projected)
	if len(projected) > t.Cap || projectedLines > t.LineCap {
		overflow = true
	}
	r := ProjectionResult{Path: path, Content: projected, SizeBytes: int64(len(projected)), SizeLines: int64(projectedLines), ByteCap: int64(t.Cap), LineCap: int64(t.LineCap), Overflow: overflow, DryRun: dryRun}
	if dryRun {
		return r, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return ProjectionResult{}, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".vrooli-memory-projection-*")
	if err != nil {
		return ProjectionResult{}, err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err = tmp.WriteString(projected); err != nil {
		tmp.Close()
		return ProjectionResult{}, err
	}
	if err = tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return ProjectionResult{}, err
	}
	if err = tmp.Close(); err != nil {
		return ProjectionResult{}, err
	}
	if err = os.Rename(name, path); err != nil {
		return ProjectionResult{}, err
	}
	if p.roots != nil {
		p.roots.RecordWrite(ctx)
	}
	sum := sha256.Sum256([]byte(projected))
	_, err = p.db.ExecContext(ctx, `INSERT INTO harness_projections(runtime,target_path,content_hash,size_bytes,size_lines,updated_at) VALUES(?,?,?,?,?,?) ON CONFLICT(runtime) DO UPDATE SET target_path=excluded.target_path,content_hash=excluded.content_hash,size_bytes=excluded.size_bytes,size_lines=excluded.size_lines,updated_at=excluded.updated_at`, t.Runtime, path, hex.EncodeToString(sum[:]), len(projected), projectedLines, time.Now().UTC().Format(time.RFC3339Nano))
	return r, err
}

// spliceWakeBlock replaces only the generated region. The projection never
// reads generated content back as journal input, and never owns the bytes
// outside these markers. A missing block is installed without discarding the
// existing file so runtimes can migrate safely from whole-file projection.
func spliceWakeBlock(path, existing, generated string) (string, error) {
	startCount := strings.Count(existing, wakeStart)
	endCount := strings.Count(existing, wakeEnd)
	if startCount == 0 && endCount == 0 {
		separator := ""
		if existing != "" && !strings.HasSuffix(existing, "\n") {
			separator = "\n"
		}
		return existing + separator + wakeStart + "\n" + generated + wakeEnd + "\n", nil
	}
	if startCount != 1 || endCount != 1 {
		return "", ErrMalformedManagedBlock{Path: path}
	}
	start := strings.Index(existing, wakeStart)
	end := strings.Index(existing, wakeEnd)
	if end < start {
		return "", ErrMalformedManagedBlock{Path: path}
	}
	return existing[:start] + wakeStart + "\n" + generated + wakeEnd + existing[end+len(wakeEnd):], nil
}

func spliceProjectionBlock(path, existing, generated, section string) (string, error) {
	if section == "" {
		return spliceWakeBlock(path, existing, generated)
	}
	start, end, found := sectionBounds(existing, section)
	if !found {
		if strings.TrimSpace(existing) != "" {
			return "", fmt.Errorf("memory section %q not found in %s", section, path)
		}
		existing = "## " + section + "\n\n"
		start, end, _ = sectionBounds(existing, section)
	}
	body, err := spliceWakeBlock(path+"#"+section, existing[start:end], generated)
	if err != nil {
		return "", err
	}
	return existing[:start] + body + existing[end:], nil
}

func sectionBounds(text, section string) (int, int, bool) {
	offset := 0
	start := -1
	for _, line := range strings.SplitAfter(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if start < 0 && strings.EqualFold(strings.TrimSpace(strings.TrimLeft(trimmed, "# ")), section) {
			start = offset + len(line)
			offset += len(line)
			continue
		}
		if start >= 0 && strings.HasPrefix(trimmed, "#") {
			// The generated wake header is itself a Markdown heading. It is
			// content inside the native section, not the next section boundary.
			// Ignore headings while a managed block is open so refreshes remain
			// idempotent after the first projection.
			sectionBody := text[start:offset]
			if strings.LastIndex(sectionBody, wakeStart) > strings.LastIndex(sectionBody, wakeEnd) {
				offset += len(line)
				continue
			}
			return start, offset, true
		}
		offset += len(line)
	}
	if start < 0 {
		return 0, 0, false
	}
	return start, len(text), true
}

func (p *Projector) targetPath(ctx context.Context, t projectionTarget) (string, error) {
	if !database.IsTestMode(ctx) || p.roots == nil {
		return t.Path, nil
	}
	root, err := p.roots.Pick(ctx, storage.ClassData)
	if err != nil {
		return "", fmt.Errorf("resolve routed projection root: %w", err)
	}
	return filepath.Join(root, "harness-projections", t.Runtime, filepath.Base(t.Path)), nil
}
