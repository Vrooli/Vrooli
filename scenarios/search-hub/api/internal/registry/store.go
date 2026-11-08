package registry

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/schedule"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"

	"google.golang.org/protobuf/encoding/protojson"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
)

// SQLExecutor is the narrow database surface the store depends on. Declared at
// the consumer per seam-discovery: both *sql.DB (used by store tests via
// testutil/db.NewSQLite) and *database.RoutedDB (used in production by
// main.go) satisfy it, so production wiring participates in per-request
// routing without forcing the test fixture to wrap its handle.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// CredentialAuthority is the existing Vrooli secure-store boundary. The
// registry stores only non-secret witness metadata in SQLite; the encrypted
// token set remains owned by the credential authority.
type CredentialAuthority interface {
	Resolve(credentialauthority.Identity, string) (string, error)
	Put(credentialauthority.Identity, string, string) error
}

// Store is the persistence seam the registry handler depends on. Production
// wires the SQLite-backed implementation; handler unit tests wire a fake.
type Store interface {
	// Upsert validates and persists d (upsert keyed by provider_id). Returns
	// created=true when a new leaf was inserted, false when an existing leaf
	// was updated. Returns ErrInvalidDescriptor on validation failure.
	//
	// presentedToken is the control token the caller cached from a prior
	// registration (empty on a first registration or after the provider lost its
	// in-memory copy on restart). On INSERT the store MINTS a fresh token and
	// ignores presentedToken; on UPDATE it requires presentedToken to be empty or
	// to match the stored token (else ErrTokenMismatch). The authoritative token
	// for the provider is always returned so the caller can (re)cache it.
	Upsert(ctx context.Context, d *registryv1.ProviderDescriptor, presentedToken string) (created bool, controlToken string, err error)

	// List returns descriptors matching filter, ordered by provider_id.
	List(ctx context.Context, filter ListFilter) ([]*registryv1.ProviderDescriptor, error)

	// Get returns the descriptor for id or ErrProviderNotFound.
	Get(ctx context.Context, id string) (*registryv1.ProviderDescriptor, error)

	// Token returns the credential-authority token for id (empty string when the
	// provider has no minted-token witness yet). Returns ErrProviderNotFound when no
	// such provider is registered. search-hub uses it to present the token when
	// it calls a provider's token-gated verbs (override/reindex/config-write).
	Token(ctx context.Context, id string) (string, error)

	// Delete removes the leaf with the given provider_id. Returns removed=true
	// when a row was deleted, false when no such leaf existed (idempotent).
	Delete(ctx context.Context, id string) (removed bool, err error)
}

// sqliteStore is the production Store impl. Unexported so callers depend on
// the Store interface and substitute the fake without reaching inside.
type sqliteStore struct {
	db            SQLExecutor
	clock         schedule.Clock
	credentials   CredentialAuthority
	credentialID  credentialauthority.Identity
	credentialErr error
}

const (
	// #nosec G101 -- this identifies a credential entry; it is not a credential value.
	controlTokenIdentity = "vrooli/search-hub"
	controlTokenField    = "provider-control-token-set"
)

var controlTokenSetMu sync.Mutex

// NewSQLiteStore constructs the production Store. db is the connection pool
// opened in main.go (*database.RoutedDB in production, *sql.DB in unit tests);
// clk supplies created_at/updated_at timestamps so tests can advance time
// deterministically.
func NewSQLiteStore(db SQLExecutor, clk schedule.Clock) *sqliteStore {
	credentials, err := credentialauthority.Default()
	return NewSQLiteStoreWithCredentialAuthority(db, clk, credentials, err)
}

// NewSQLiteStoreWithCredentialAuthority makes the credential boundary
// injectable for tests while production uses NewSQLiteStore and the host's
// canonical credential authority.
func NewSQLiteStoreWithCredentialAuthority(db SQLExecutor, clk schedule.Clock, credentials CredentialAuthority, authorityErr error) *sqliteStore {
	id, idErr := credentialauthority.ParseIdentity(controlTokenIdentity)
	if authorityErr == nil {
		authorityErr = idErr
	}
	return &sqliteStore{db: db, clock: clk, credentials: credentials, credentialID: id, credentialErr: authorityErr}
}

// Compile-time guarantee.
var _ Store = (*sqliteStore)(nil)

const providerTimeFormat = time.RFC3339Nano

// marshalOpts keeps blobs compact and deterministic (EmitUnpopulated false so
// zero fields don't bloat the row; proto field names for readability when a
// human inspects the column).
var marshalOpts = protojson.MarshalOptions{UseProtoNames: true}

func (s *sqliteStore) Upsert(ctx context.Context, d *registryv1.ProviderDescriptor, presentedToken string) (bool, string, error) {
	controlTokenSetMu.Lock()
	defer controlTokenSetMu.Unlock()
	if err := s.migrateLegacyTokensLocked(ctx); err != nil {
		return false, "", err
	}
	Normalize(d)
	if err := Validate(d); err != nil {
		return false, "", err
	}

	now := s.clock.Now().UTC().Format(providerTimeFormat)
	if strings.TrimSpace(d.DeclaredAt) == "" {
		d.DeclaredAt = now
	}
	blob, err := marshalOpts.Marshal(d)
	if err != nil {
		return false, "", fmt.Errorf("marshal descriptor: %w", err)
	}

	// Determine insert vs update so we can report `created` honestly. The raw
	// token set is stored in the credential authority; SQLite retains only a
	// per-provider minted witness and an empty legacy column.
	var providerExists int
	err = s.db.QueryRowContext(ctx, `SELECT 1 FROM providers WHERE provider_id = ?`, d.ProviderId).Scan(&providerExists)
	var storedToken string
	if err == nil {
		storedToken, err = s.tokenForExistingProvider(ctx, d.ProviderId)
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, "", fmt.Errorf("probe provider control-token state: %w", err)
	}
	switch {
	case errors.Is(err, sql.ErrNoRows):
		token, tokenErr := s.tokenForNewProvider(ctx, d.ProviderId)
		if tokenErr != nil {
			return false, "", tokenErr
		}
		_, insErr := s.db.ExecContext(ctx, `
INSERT INTO providers (provider_id, provider_group, bucket, type, state, scope, descriptor, control_token, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			d.ProviderId, d.ProviderGroup, int32(d.Bucket), d.Type, int32(d.State), int32(d.Scope),
			string(blob), "", now, now)
		if insErr != nil {
			return false, "", fmt.Errorf("insert provider: %w", insErr)
		}
		return true, token, nil
	case err != nil:
		return false, "", fmt.Errorf("probe provider: %w", err)
	default:
		// Ownership proof: a non-empty presented token must match the stored one.
		// An empty presented token is allowed (the provider re-registers on every
		// boot but holds the token only in memory) and simply receives the echo.
		if presentedToken != "" && presentedToken != storedToken {
			return false, "", ErrTokenMismatch{ProviderID: d.ProviderId}
		}
		if storedToken == "" {
			storedToken = newControlToken()
			if err := s.storeToken(ctx, d.ProviderId, storedToken); err != nil {
				return false, "", err
			}
		}
		_, updErr := s.db.ExecContext(ctx, `
UPDATE providers
SET provider_group = ?, bucket = ?, type = ?, state = ?, scope = ?, descriptor = ?, control_token = ?, updated_at = ?
WHERE provider_id = ?`,
			d.ProviderGroup, int32(d.Bucket), d.Type, int32(d.State), int32(d.Scope),
			string(blob), "", now, d.ProviderId)
		if updErr != nil {
			return false, "", fmt.Errorf("update provider: %w", updErr)
		}
		return false, storedToken, nil
	}
}

// Token returns the stored control token for id (empty when the provider has
// none yet), or ErrProviderNotFound when no such provider is registered.
func (s *sqliteStore) Token(ctx context.Context, id string) (string, error) {
	controlTokenSetMu.Lock()
	defer controlTokenSetMu.Unlock()
	if err := s.migrateLegacyTokensLocked(ctx); err != nil {
		return "", err
	}
	var token string
	err := s.db.QueryRowContext(ctx, `SELECT control_token FROM providers WHERE provider_id = ?`, id).Scan(&token)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrProviderNotFound{ProviderID: id}
	}
	if err != nil {
		return "", fmt.Errorf("get provider token witness: %w", err)
	}
	set, configured, err := s.loadTokenSet()
	if err != nil {
		return "", err
	}
	if value := set[id]; value != "" {
		return value, nil
	}
	if configured || token == "" {
		var witnessed int
		if witnessErr := s.db.QueryRowContext(ctx, `SELECT 1 FROM provider_control_token_witness WHERE provider_id = ?`, id).Scan(&witnessed); witnessErr == nil {
			return "", fmt.Errorf("control token for provider %q is missing from the credential authority", id)
		} else if !errors.Is(witnessErr, sql.ErrNoRows) {
			return "", fmt.Errorf("read provider control-token witness: %w", witnessErr)
		}
		return "", nil
	}
	return "", fmt.Errorf("control token for provider %q is not available in the credential authority", id)
}

// MigrateLegacyControlTokens copies tokens from the old SQLite column into the
// credential authority before clearing the column. The authority write always
// happens first; if it fails, SQLite remains untouched and remains the only
// recoverable copy. The operation is idempotent after interruption.
func (s *sqliteStore) MigrateLegacyControlTokens(ctx context.Context) error {
	controlTokenSetMu.Lock()
	defer controlTokenSetMu.Unlock()
	return s.migrateLegacyTokensLocked(ctx)
}

func (s *sqliteStore) migrateLegacyTokensLocked(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `SELECT provider_id, control_token FROM providers ORDER BY provider_id`)
	if err != nil {
		return fmt.Errorf("read legacy provider control tokens: %w", err)
	}
	type legacyToken struct{ id, token string }
	var legacy []legacyToken
	for rows.Next() {
		var item legacyToken
		if err := rows.Scan(&item.id, &item.token); err != nil {
			rows.Close()
			return fmt.Errorf("scan legacy provider control token: %w", err)
		}
		if item.token != "" {
			legacy = append(legacy, item)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate legacy provider control tokens: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close legacy provider control tokens: %w", err)
	}

	witnesses, err := s.witnessedProviderIDs(ctx)
	if err != nil {
		return err
	}
	if len(legacy) == 0 && len(witnesses) == 0 {
		return nil
	}
	legacyByID := make(map[string]string, len(legacy))
	for _, item := range legacy {
		legacyByID[item.id] = item.token
	}
	set, configured, err := s.loadTokenSet()
	if err != nil {
		return err
	}
	for _, id := range witnesses {
		if set[id] != "" || legacyByID[id] != "" {
			continue
		}
		if !configured {
			return errors.New("provider control-token credential is missing while registry witnesses exist; restore the credential authority before Search Hub can serve providers")
		}
		return fmt.Errorf("provider control-token credential is missing for witnessed provider %q", id)
	}
	if len(legacy) == 0 {
		return nil
	}

	changed := false
	for _, item := range legacy {
		if existing := set[item.id]; existing != "" {
			if existing != item.token {
				return fmt.Errorf("legacy provider control token conflicts with credential authority for %q", item.id)
			}
			continue
		}
		set[item.id] = item.token
		changed = true
	}
	if changed || !configured {
		if err := s.putTokenSet(set); err != nil {
			return fmt.Errorf("persist migrated provider control tokens: %w", err)
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin provider control-token migration: %w", err)
	}
	defer tx.Rollback()
	for _, item := range legacy {
		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO provider_control_token_witness (provider_id, minted_at) VALUES (?, ?)`, item.id, s.clock.Now().UTC().Format(providerTimeFormat)); err != nil {
			return fmt.Errorf("record migrated control-token witness for %q: %w", item.id, err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE providers SET control_token = '' WHERE provider_id = ? AND control_token = ?`, item.id, item.token); err != nil {
			return fmt.Errorf("clear migrated provider control token for %q: %w", item.id, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit provider control-token migration: %w", err)
	}
	return nil
}

func (s *sqliteStore) loadTokenSet() (map[string]string, bool, error) {
	if s.credentialErr != nil {
		return nil, false, fmt.Errorf("search-hub credential authority unavailable: %w", s.credentialErr)
	}
	if s.credentials == nil {
		return nil, false, errors.New("search-hub credential authority unavailable")
	}
	raw, err := s.credentials.Resolve(s.credentialID, controlTokenField)
	if errors.Is(err, credentialauthority.ErrUnconfigured) {
		return map[string]string{}, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("resolve provider control-token credential: %w", err)
	}
	set := make(map[string]string)
	if err := json.Unmarshal([]byte(raw), &set); err != nil {
		return nil, false, errors.New("provider control-token credential is malformed")
	}
	for id, token := range set {
		if strings.TrimSpace(id) == "" || token == "" {
			return nil, false, errors.New("provider control-token credential contains an invalid entry")
		}
	}
	return set, true, nil
}

func (s *sqliteStore) putTokenSet(set map[string]string) error {
	if s.credentialErr != nil {
		return fmt.Errorf("search-hub credential authority unavailable: %w", s.credentialErr)
	}
	if s.credentials == nil {
		return errors.New("search-hub credential authority unavailable")
	}
	blob, err := json.Marshal(set)
	if err != nil {
		return fmt.Errorf("encode provider control-token credential: %w", err)
	}
	if err := s.credentials.Put(s.credentialID, controlTokenField, string(blob)); err != nil {
		return fmt.Errorf("write provider control-token credential: %w", err)
	}
	return nil
}

func (s *sqliteStore) witnessedProviderIDs(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT provider_id FROM provider_control_token_witness ORDER BY provider_id`)
	if err != nil {
		return nil, fmt.Errorf("read provider control-token witnesses: %w", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan provider control-token witness: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate provider control-token witnesses: %w", err)
	}
	return ids, nil
}

func (s *sqliteStore) tokenForExistingProvider(ctx context.Context, id string) (string, error) {
	set, _, err := s.loadTokenSet()
	if err != nil {
		return "", err
	}
	if token := set[id]; token != "" {
		if _, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO provider_control_token_witness (provider_id, minted_at) VALUES (?, ?)`, id, s.clock.Now().UTC().Format(providerTimeFormat)); err != nil {
			return "", fmt.Errorf("record provider control-token witness: %w", err)
		}
		return token, nil
	}
	var witnessed int
	err = s.db.QueryRowContext(ctx, `SELECT 1 FROM provider_control_token_witness WHERE provider_id = ?`, id).Scan(&witnessed)
	if err == nil {
		return "", fmt.Errorf("provider control-token credential is missing for witnessed provider %q", id)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("read provider control-token witness: %w", err)
	}
	return "", nil
}

func (s *sqliteStore) storeToken(ctx context.Context, id, token string) error {
	set, _, err := s.loadTokenSet()
	if err != nil {
		return err
	}
	if current := set[id]; current != "" {
		if current != token {
			return fmt.Errorf("refusing to replace provider control-token credential for %q", id)
		}
	} else {
		set[id] = token
		if err := s.putTokenSet(set); err != nil {
			return err
		}
	}
	if _, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO provider_control_token_witness (provider_id, minted_at) VALUES (?, ?)`, id, s.clock.Now().UTC().Format(providerTimeFormat)); err != nil {
		return fmt.Errorf("record provider control-token witness for %q: %w", id, err)
	}
	return nil
}

// tokenForNewProvider reuses an authority value left behind when a previous
// attempt persisted the credential but failed before inserting the provider
// row. That makes the external authority write safely retryable.
func (s *sqliteStore) tokenForNewProvider(ctx context.Context, id string) (string, error) {
	set, _, err := s.loadTokenSet()
	if err != nil {
		return "", err
	}
	if token := set[id]; token != "" {
		if _, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO provider_control_token_witness (provider_id, minted_at) VALUES (?, ?)`, id, s.clock.Now().UTC().Format(providerTimeFormat)); err != nil {
			return "", fmt.Errorf("record provider control-token witness: %w", err)
		}
		return token, nil
	}
	var witnessed int
	err = s.db.QueryRowContext(ctx, `SELECT 1 FROM provider_control_token_witness WHERE provider_id = ?`, id).Scan(&witnessed)
	if err == nil {
		return "", fmt.Errorf("provider control-token credential is missing for witnessed provider %q", id)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("read provider control-token witness: %w", err)
	}
	token := newControlToken()
	if err := s.storeToken(ctx, id, token); err != nil {
		return "", err
	}
	return token, nil
}

// newControlToken returns a random 128-bit hex token (stdlib only — no external
// uuid dep, matching the aisearch reindex job-id generator's posture). A
// crypto/rand failure is fatal to security, so it panics rather than minting a
// predictable token.
func newControlToken() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Sprintf("registry: crypto/rand failed minting control token: %v", err))
	}
	return hex.EncodeToString(b[:])
}

func (s *sqliteStore) List(ctx context.Context, filter ListFilter) ([]*registryv1.ProviderDescriptor, error) {
	var (
		clauses []string
		args    []any
	)
	if filter.Bucket != 0 {
		clauses = append(clauses, "bucket = ?")
		args = append(args, filter.Bucket)
	}
	if strings.TrimSpace(filter.Type) != "" {
		clauses = append(clauses, "type = ?")
		args = append(args, filter.Type)
	}
	if filter.State != 0 {
		clauses = append(clauses, "state = ?")
		args = append(args, filter.State)
	}

	query := "SELECT descriptor FROM providers"
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY provider_id ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query providers: %w", err)
	}
	defer rows.Close()

	out := make([]*registryv1.ProviderDescriptor, 0)
	for rows.Next() {
		d, scanErr := scanDescriptor(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate providers: %w", err)
	}
	return out, nil
}

func (s *sqliteStore) Get(ctx context.Context, id string) (*registryv1.ProviderDescriptor, error) {
	var blob string
	err := s.db.QueryRowContext(ctx, `SELECT descriptor FROM providers WHERE provider_id = ?`, id).Scan(&blob)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProviderNotFound{ProviderID: id}
	}
	if err != nil {
		return nil, fmt.Errorf("get provider: %w", err)
	}
	return unmarshalDescriptor(blob)
}

func (s *sqliteStore) Delete(ctx context.Context, id string) (bool, error) {
	controlTokenSetMu.Lock()
	defer controlTokenSetMu.Unlock()

	set, _, err := s.loadTokenSet()
	if err != nil {
		return false, err
	}
	previousToken, hadToken := set[id]
	if hadToken {
		delete(set, id)
		if err := s.putTokenSet(set); err != nil {
			return false, fmt.Errorf("remove provider control-token credential: %w", err)
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, s.restoreDeletedToken(ctx, id, previousToken, hadToken, fmt.Errorf("begin provider deletion: %w", err))
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `DELETE FROM providers WHERE provider_id = ?`, id)
	if err != nil {
		return false, s.restoreDeletedToken(ctx, id, previousToken, hadToken, fmt.Errorf("delete provider: %w", err))
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, s.restoreDeletedToken(ctx, id, previousToken, hadToken, fmt.Errorf("delete provider rows affected: %w", err))
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM provider_control_token_witness WHERE provider_id = ?`, id); err != nil {
		return false, s.restoreDeletedToken(ctx, id, previousToken, hadToken, fmt.Errorf("delete provider control-token witness: %w", err))
	}
	if err := tx.Commit(); err != nil {
		return false, s.restoreDeletedToken(ctx, id, previousToken, hadToken, fmt.Errorf("commit provider deletion: %w", err))
	}
	return n > 0, nil
}

// restoreDeletedToken puts the credential back when the database deletion did
// not commit. If this recovery write also fails, the registry witness keeps
// the remaining provider row fail-closed and the returned error reports both
// failures.
func (s *sqliteStore) restoreDeletedToken(ctx context.Context, id, token string, hadToken bool, cause error) error {
	if !hadToken {
		return cause
	}
	set, _, err := s.loadTokenSet()
	if err == nil {
		set[id] = token
		err = s.putTokenSet(set)
	}
	if err != nil {
		return fmt.Errorf("%w; restore provider control-token credential after failed deletion: %v", cause, err)
	}
	return cause
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanDescriptor(s scanner) (*registryv1.ProviderDescriptor, error) {
	var blob string
	if err := s.Scan(&blob); err != nil {
		return nil, fmt.Errorf("scan provider: %w", err)
	}
	return unmarshalDescriptor(blob)
}

func unmarshalDescriptor(blob string) (*registryv1.ProviderDescriptor, error) {
	d := &registryv1.ProviderDescriptor{}
	if err := protojson.Unmarshal([]byte(blob), d); err != nil {
		// Rows written before lifecycle became an enum contain the descriptor
		// policy strings. Preserve those registrations and normalize them on
		// read so the wire-contract migration is backward compatible.
		legacy := strings.ReplaceAll(blob, `"lifecycle":"production"`, `"lifecycle":"LIFECYCLE_PRODUCTION"`)
		legacy = strings.ReplaceAll(legacy, `"lifecycle":"fixture"`, `"lifecycle":"LIFECYCLE_FIXTURE"`)
		legacy = strings.ReplaceAll(legacy, `"lifecycle":"experimental"`, `"lifecycle":"LIFECYCLE_EXPERIMENTAL"`)
		if legacy == blob || protojson.Unmarshal([]byte(legacy), d) != nil {
			return nil, fmt.Errorf("unmarshal descriptor: %w", err)
		}
	}
	return d, nil
}
