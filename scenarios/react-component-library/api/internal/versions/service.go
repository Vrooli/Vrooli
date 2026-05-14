package versions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

// defaultListLimit caps List rows when callers pass 0.
const defaultListLimit = 100

// versionHeaderRe extracts the `@version` field from a component file
// header. Component files conventionally place `@version` either in a
// JSDoc block (`* @version 1.2.3`) or a line comment (`// @version
// 1.2.3`). The regex is permissive on the prefix and picks the first
// whitespace-delimited token after `@version`; downstream callers strip
// quote/punctuation noise.
var versionHeaderRe = regexp.MustCompile(`@version\s+([^\s]+)`)

// ParseVersionHeader returns the `@version` value found in content, or
// "" when none is present. Exported for the recorder adapter so it can
// derive a version label from the post-save body without re-importing
// the components header parser.
func ParseVersionHeader(content string) string {
	m := versionHeaderRe.FindStringSubmatch(content)
	if len(m) < 2 {
		return ""
	}
	return strings.Trim(m[1], `,;"'`)
}

// AdoptionResolver resolves an `adoption:<id>` reference passed to
// DiffVersions into raw file content on disk. Production wires the
// handler-level adapter over the adoptions service + scenarios root;
// tests inject a fake. Returning ("", nil) is allowed for missing
// adopted files — the diff just renders as deletions.
type AdoptionResolver interface {
	ResolveAdoption(ctx context.Context, adoptionID string) (content string, err error)
}

// Service is the application surface for the versions domain.
type Service interface {
	Record(ctx context.Context, in RecordInput) (recorded bool, v Version, err error)
	List(ctx context.Context, q ListQuery) ([]Version, error)
	Get(ctx context.Context, componentID, version string) (Version, error)
	Diff(ctx context.Context, in DiffInput) (DiffResult, error)
}

// DiffInput selects the two sides to compare. Either `From` or `To`
// (or both) may carry the `adoption:<id>` prefix; otherwise they are
// `@version` values matched against the version history.
type DiffInput struct {
	ComponentID string
	From        string
	To          string
}

// DiffOp tags an aligned diff cell.
type DiffOp string

const (
	DiffOpEqual  DiffOp = "equal"
	DiffOpRemove DiffOp = "remove"
	DiffOpAdd    DiffOp = "add"
	DiffOpEmpty  DiffOp = "empty"
)

// DiffCell is one side of a row in the side-by-side diff.
type DiffCell struct {
	LineNumber int
	Text       string
	Op         DiffOp
}

// DiffRow pairs a left and a right cell.
type DiffRow struct {
	Left  DiffCell
	Right DiffCell
}

// DiffResult is the structured server-computed diff. The handler
// translates it to the proto wire shape.
type DiffResult struct {
	Rows      []DiffRow
	Additions int
	Removals  int
	FromLabel string
	ToLabel   string
}

type service struct {
	repo       Repository
	adoptions  AdoptionResolver // optional; nil disables adoption diff
}

// NewService constructs a Service backed by repo. The optional
// AdoptionResolver enables `adoption:<id>` diff sides.
func NewService(repo Repository, resolver AdoptionResolver) Service {
	return &service{repo: repo, adoptions: resolver}
}

var _ Service = (*service)(nil)

// Record inserts a new version row if either the content sha or the
// `@version` header has changed since the last recorded row. Returns
// recorded=false (and the existing latest row) on a no-op.
func (s *service) Record(ctx context.Context, in RecordInput) (bool, Version, error) {
	if strings.TrimSpace(in.ComponentID) == "" {
		return false, Version{}, ErrInvalidVersion{Field: "component_id", Reason: "required"}
	}
	if in.Content == "" {
		return false, Version{}, ErrInvalidVersion{Field: "content", Reason: "required"}
	}
	sha := digest(in.Content)
	parsed := ParseVersionHeader(in.Content)

	latest, err := s.repo.Latest(ctx, in.ComponentID)
	if err != nil {
		return false, Version{}, err
	}
	if latest.ID != "" && latest.ContentSHA256 == sha && latest.Version == parsed {
		return false, latest, nil
	}

	changelog := in.ChangelogMD
	if strings.TrimSpace(changelog) == "" {
		changelog = "auto-recorded on save"
	}
	inserted, err := s.repo.Insert(ctx, Version{
		ComponentID:   in.ComponentID,
		Version:       parsed,
		Content:       in.Content,
		ContentSHA256: sha,
		ChangelogMD:   changelog,
	})
	if err != nil {
		return false, Version{}, err
	}
	return true, inserted, nil
}

func (s *service) List(ctx context.Context, q ListQuery) ([]Version, error) {
	if q.Limit <= 0 {
		q.Limit = defaultListLimit
	}
	return s.repo.List(ctx, q)
}

func (s *service) Get(ctx context.Context, componentID, version string) (Version, error) {
	return s.repo.Get(ctx, componentID, version)
}

func (s *service) Diff(ctx context.Context, in DiffInput) (DiffResult, error) {
	if strings.TrimSpace(in.ComponentID) == "" {
		return DiffResult{}, ErrInvalidVersion{Field: "component_id", Reason: "required"}
	}
	if strings.TrimSpace(in.From) == "" || strings.TrimSpace(in.To) == "" {
		return DiffResult{}, ErrInvalidVersion{Field: "from/to", Reason: "required"}
	}
	left, err := s.resolveSide(ctx, in.ComponentID, in.From)
	if err != nil {
		return DiffResult{}, err
	}
	right, err := s.resolveSide(ctx, in.ComponentID, in.To)
	if err != nil {
		return DiffResult{}, err
	}
	rows, adds, rems := alignDiff(left, right)
	return DiffResult{
		Rows:      rows,
		Additions: adds,
		Removals:  rems,
		FromLabel: in.From,
		ToLabel:   in.To,
	}, nil
}

func (s *service) resolveSide(ctx context.Context, componentID, ref string) (string, error) {
	if rest, ok := strings.CutPrefix(ref, "adoption:"); ok {
		if s.adoptions == nil {
			return "", fmt.Errorf("versions: adoption diff requested but no AdoptionResolver wired")
		}
		return s.adoptions.ResolveAdoption(ctx, rest)
	}
	v, err := s.repo.Get(ctx, componentID, ref)
	if err != nil {
		return "", err
	}
	return v.Content, nil
}

// alignDiff is a classic LCS line-diff that emits aligned rows.
// O(n*m) over line counts — fine for component files capped at a few
// thousand lines. Public DiffResult exposes additions/removals counts
// so the UI can render a header without re-walking.
func alignDiff(leftSrc, rightSrc string) ([]DiffRow, int, int) {
	left := splitLines(leftSrc)
	right := splitLines(rightSrc)
	lcs := lcsTable(left, right)

	rows := make([]DiffRow, 0, len(left)+len(right))
	var adds, rems int

	var emit func(i, j int)
	emit = func(i, j int) {
		if i > 0 && j > 0 && left[i-1] == right[j-1] {
			emit(i-1, j-1)
			rows = append(rows, DiffRow{
				Left:  DiffCell{LineNumber: i, Text: left[i-1], Op: DiffOpEqual},
				Right: DiffCell{LineNumber: j, Text: right[j-1], Op: DiffOpEqual},
			})
			return
		}
		if j > 0 && (i == 0 || lcs[i][j-1] >= lcs[i-1][j]) {
			emit(i, j-1)
			rows = append(rows, DiffRow{
				Left:  DiffCell{Op: DiffOpEmpty},
				Right: DiffCell{LineNumber: j, Text: right[j-1], Op: DiffOpAdd},
			})
			adds++
			return
		}
		if i > 0 && (j == 0 || lcs[i][j-1] < lcs[i-1][j]) {
			emit(i-1, j)
			rows = append(rows, DiffRow{
				Left:  DiffCell{LineNumber: i, Text: left[i-1], Op: DiffOpRemove},
				Right: DiffCell{Op: DiffOpEmpty},
			})
			rems++
			return
		}
	}
	emit(len(left), len(right))
	return rows, adds, rems
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	// Strip a single trailing newline so a file ending in \n doesn't
	// produce an extra empty cell at the bottom of the diff.
	trimmed := strings.TrimSuffix(s, "\n")
	return strings.Split(trimmed, "\n")
}

func lcsTable(a, b []string) [][]int {
	n, m := len(a), len(b)
	t := make([][]int, n+1)
	for i := range t {
		t[i] = make([]int, m+1)
	}
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			if a[i-1] == b[j-1] {
				t[i][j] = t[i-1][j-1] + 1
			} else if t[i-1][j] >= t[i][j-1] {
				t[i][j] = t[i-1][j]
			} else {
				t[i][j] = t[i][j-1]
			}
		}
	}
	return t
}

func digest(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
