package workspaces

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
)

// timestampLayout is fixed-width UTC so lexical ordering matches time order
// and the value matches the JavaScript ISO shape already stored by the studio.
const timestampLayout = "2006-01-02T15:04:05.000Z"

const (
	defaultListLimit     = 100
	maxListLimit         = 500
	defaultRevisionLimit = 50
	maxRevisionLimit     = 200
)

// queryer is the subset shared by *sql.DB and *sql.Tx. Reads run through the
// pool; aggregate reads inside a save run through the transaction that owns
// the write.
type queryer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// SQLiteRepository is the SQLite-backed workspace aggregate store.
type SQLiteRepository struct {
	db       Database
	now      func() time.Time
	recorder Recorder
}

var _ Repository = (*SQLiteRepository)(nil)

// NewSQLiteRepository returns the workspace repository over the injected SQL
// surface. Production passes RoutedDB so request contexts can select an
// isolated test pool; ordinary *sql.DB pools remain convenient in tests.
// Callers that need activity wiring chain WithRecorder.
func NewSQLiteRepository(db Database) *SQLiteRepository {
	return &SQLiteRepository{db: db, now: time.Now}
}

// WithRecorder attaches the activity seam. A nil recorder leaves activity
// logging disabled without changing any mutation behaviour.
func (r *SQLiteRepository) WithRecorder(recorder Recorder) *SQLiteRepository {
	r.recorder = recorder
	return r
}

func (r *SQLiteRepository) record(ctx context.Context, action, subjectID string, before, after *workspacev1.Workspace) error {
	if r.recorder == nil {
		return nil
	}
	beforeJSON, err := marshalWorkspace(before)
	if err != nil {
		return err
	}
	afterJSON, err := marshalWorkspace(after)
	if err != nil {
		return err
	}
	if err := r.recorder.Record(ctx, action, subjectID, beforeJSON, afterJSON); err != nil {
		// The mutation is already committed. Reporting this as a failed write
		// would be a lie that invites a duplicate retry, so the message states
		// the committed state explicitly and the handler logs it as internal.
		return fmt.Errorf("record %s for workspace %s after commit: %w", action, subjectID, err)
	}
	return nil
}

// marshalWorkspace is the one snapshot encoding. Snake_case proto names keep
// revision snapshots readable and stable if this package's Go types change.
func marshalWorkspace(workspace *workspacev1.Workspace) ([]byte, error) {
	if workspace == nil {
		return nil, nil
	}
	encoded, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(workspace)
	if err != nil {
		return nil, fmt.Errorf("encode workspace snapshot: %w", err)
	}
	return encoded, nil
}

func (r *SQLiteRepository) timestamp() string {
	now := r.now
	if now == nil {
		now = time.Now
	}
	return now().UTC().Format(timestampLayout)
}

// nowTimestamp is the migration path's clock: the importer has no repository
// instance and must not depend on one just to stamp a receipt.
func nowTimestamp() string {
	return time.Now().UTC().Format(timestampLayout)
}

func listLimit(limit int32) int32 {
	switch {
	case limit <= 0:
		return defaultListLimit
	case limit > maxListLimit:
		return maxListLimit
	default:
		return limit
	}
}

func revisionLimit(limit int32) int32 {
	switch {
	case limit <= 0:
		return defaultRevisionLimit
	case limit > maxRevisionLimit:
		return maxRevisionLimit
	default:
		return limit
	}
}

func tagsJSON(tags []string) string {
	if len(tags) == 0 {
		return "[]"
	}
	encoded, err := json.Marshal(tags)
	if err != nil {
		// A []string cannot fail to marshal; returning [] keeps the column
		// valid rather than writing an empty string.
		return "[]"
	}
	return string(encoded)
}

func parseTags(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return []string{}, nil
	}
	var tags []string
	if err := json.Unmarshal([]byte(raw), &tags); err != nil {
		return nil, fmt.Errorf("decode tags_json: %w", err)
	}
	if tags == nil {
		tags = []string{}
	}
	return tags, nil
}

// escapeLike makes a user's substring a literal. Without this, a query of
// "100%" would match every name and "_" would match any character.
func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	value = strings.ReplaceAll(value, `_`, `\_`)
	return value
}

func cloneWorkspace(workspace *workspacev1.Workspace) *workspacev1.Workspace {
	if workspace == nil {
		return nil
	}
	return proto.Clone(workspace).(*workspacev1.Workspace)
}

func clonePages(pages []*workspacev1.WorkspacePage) []*workspacev1.WorkspacePage {
	out := make([]*workspacev1.WorkspacePage, 0, len(pages))
	for _, page := range pages {
		if page == nil {
			out = append(out, nil)
			continue
		}
		out = append(out, proto.Clone(page).(*workspacev1.WorkspacePage))
	}
	return out
}

// reidentifyPages deep-copies the source composition with fresh page, display
// and state ids. A fork must not share nested identity with its origin: ids are
// URLs and foreign keys, and a captured state on the fork must not overwrite
// the origin's evidence.
func reidentifyPages(pages []*workspacev1.WorkspacePage) []*workspacev1.WorkspacePage {
	out := make([]*workspacev1.WorkspacePage, 0, len(pages))
	for _, sourcePage := range pages {
		if sourcePage == nil {
			out = append(out, nil)
			continue
		}
		page := proto.Clone(sourcePage).(*workspacev1.WorkspacePage)
		page.Id = "" // assigned by normalizeAggregate
		for _, display := range page.Displays {
			if display == nil {
				continue
			}
			display.Id = ""
			for _, state := range display.States {
				if state == nil {
					continue
				}
				state.Id = ""
			}
		}
		out = append(out, page)
	}
	return out
}
