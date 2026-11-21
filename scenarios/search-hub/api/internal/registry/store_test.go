package registry_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	apidb "github.com/vrooli/api-core/database"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	localdb "search-hub/internal/database"
	"search-hub/internal/registry"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/scheduletest"
)

// newStore returns a SQLite-backed Store with the production schema applied —
// the canonical compose pattern: db.NewSQLite + apidb.EnsureSchemas over the
// system + registry providers, so tests exercise the same shape main.go ships.
func newStore(t *testing.T) (registry.Store, *scheduletest.FakeClock) {
	store, _, _, clk := newStoreWithAuthority(t)
	return store, clk
}

type memoryCredentialAuthority struct {
	mu     sync.Mutex
	values map[string]string
	putErr error
}

func newMemoryCredentialAuthority() *memoryCredentialAuthority {
	return &memoryCredentialAuthority{values: make(map[string]string)}
}

func (a *memoryCredentialAuthority) Resolve(_ credentialauthority.Identity, field string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	value := a.values[field]
	if value == "" {
		return "", credentialauthority.ErrUnconfigured
	}
	return value, nil
}

func (a *memoryCredentialAuthority) Put(_ credentialauthority.Identity, field, value string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.putErr != nil {
		return a.putErr
	}
	a.values[field] = value
	return nil
}

func newStoreWithAuthority(t *testing.T) (registry.Store, *sql.DB, *memoryCredentialAuthority, *scheduletest.FakeClock) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(registry.Schema),
	))
	clk := scheduletest.New(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	credentials := newMemoryCredentialAuthority()
	return registry.NewSQLiteStoreWithCredentialAuthority(d, clk, credentials, nil), d, credentials, clk
}

func TestStoreUpsertInsertThenUpdate(t *testing.T) {
	store, db, credentials, _ := newStoreWithAuthority(t)
	ctx := context.Background()
	d := validActive()

	created, token, err := store.Upsert(ctx, d, "")
	require.NoError(t, err)
	require.True(t, created, "first upsert inserts")
	require.NotEmpty(t, token, "first registration mints a control token")

	// Re-register the same leaf with a changed description → update, not insert.
	d2 := validActive()
	d2.Description = "Updated description."
	created, token2, err := store.Upsert(ctx, d2, "")
	require.NoError(t, err)
	require.False(t, created, "second upsert updates")
	require.Equal(t, token, token2, "re-register echoes the same token (empty presented)")
	var persisted string
	require.NoError(t, db.QueryRow(`SELECT control_token FROM providers WHERE provider_id = ?`, "cli-health.commands").Scan(&persisted))
	require.Empty(t, persisted, "SQLite must not retain the bearer token")
	var set map[string]string
	require.NoError(t, json.Unmarshal([]byte(credentials.values["provider-control-token-set"]), &set))
	require.Equal(t, token, set["cli-health.commands"], "credential authority retains the token for Search Hub")

	got, err := store.Get(ctx, "cli-health.commands")
	require.NoError(t, err)
	require.Equal(t, "Updated description.", got.GetDescription())

	all, err := store.List(ctx, registry.ListFilter{})
	require.NoError(t, err)
	require.Len(t, all, 1, "upsert must not duplicate the leaf")
}

func TestStoreUpsertTokenOwnership(t *testing.T) {
	store, _ := newStore(t)
	ctx := context.Background()

	_, token, err := store.Upsert(ctx, validActive(), "")
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// A re-register presenting a WRONG non-empty token is rejected (ownership).
	_, _, err = store.Upsert(ctx, validActive(), "deadbeef-not-the-token")
	require.Error(t, err)
	var mismatch registry.ErrTokenMismatch
	require.ErrorAs(t, err, &mismatch)

	// Presenting the CORRECT token succeeds and echoes it.
	_, echoed, err := store.Upsert(ctx, validActive(), token)
	require.NoError(t, err)
	require.Equal(t, token, echoed)
}

func TestStoreToken(t *testing.T) {
	store, _ := newStore(t)
	ctx := context.Background()

	_, token, err := store.Upsert(ctx, validActive(), "")
	require.NoError(t, err)

	got, err := store.Token(ctx, "cli-health.commands")
	require.NoError(t, err)
	require.Equal(t, token, got)

	_, err = store.Token(ctx, "nope.unknown")
	var notFound registry.ErrProviderNotFound
	require.ErrorAs(t, err, &notFound)
}

func TestStoreTokenFailsClosedWhenAuthorityLosesWitnessedToken(t *testing.T) {
	store, _, credentials, _ := newStoreWithAuthority(t)
	ctx := context.Background()
	_, _, err := store.Upsert(ctx, validActive(), "")
	require.NoError(t, err)
	credentials.mu.Lock()
	delete(credentials.values, "provider-control-token-set")
	credentials.mu.Unlock()

	_, err = store.Token(ctx, "cli-health.commands")
	require.ErrorContains(t, err, "credential is missing")
	_, _, err = store.Upsert(ctx, validActive(), "")
	require.ErrorContains(t, err, "credential is missing")
}

func TestStoreUpsertRetriesAfterAuthorityWriteBeforeProviderInsert(t *testing.T) {
	store, db, credentials, _ := newStoreWithAuthority(t)
	ctx := context.Background()
	_, err := db.Exec(`CREATE TRIGGER reject_provider_insert BEFORE INSERT ON providers BEGIN SELECT RAISE(ABORT, 'injected insert failure'); END`)
	require.NoError(t, err)

	_, _, err = store.Upsert(ctx, validActive(), "")
	require.ErrorContains(t, err, "injected insert failure")
	var firstSet map[string]string
	require.NoError(t, json.Unmarshal([]byte(credentials.values["provider-control-token-set"]), &firstSet))
	firstToken := firstSet["cli-health.commands"]
	require.NotEmpty(t, firstToken)

	_, err = db.Exec(`DROP TRIGGER reject_provider_insert`)
	require.NoError(t, err)
	created, retryToken, err := store.Upsert(ctx, validActive(), "")
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, firstToken, retryToken, "retry must reuse the token already persisted by the credential authority")
}

func TestMigrateLegacyControlTokensWritesAuthorityBeforeClearingSQLite(t *testing.T) {
	store, db, credentials, _ := newStoreWithAuthority(t)
	legacy := "legacy-bearer-token"
	descriptor := validActive()
	blob, err := marshalDescriptor(descriptor)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO providers (provider_id, provider_group, bucket, type, state, scope, descriptor, control_token, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, descriptor.ProviderId, descriptor.ProviderGroup, int32(descriptor.Bucket), descriptor.Type, int32(descriptor.State), int32(descriptor.Scope), blob, legacy, "2026-05-01T12:00:00Z", "2026-05-01T12:00:00Z")
	require.NoError(t, err)
	credentials.putErr = errors.New("secure store locked")
	storeWithMigration := store.(interface{ MigrateLegacyControlTokens(context.Context) error })
	require.Error(t, storeWithMigration.MigrateLegacyControlTokens(context.Background()))
	var retained string
	require.NoError(t, db.QueryRow(`SELECT control_token FROM providers WHERE provider_id = ?`, descriptor.ProviderId).Scan(&retained))
	require.Equal(t, legacy, retained, "failed authority writes must leave the recoverable source untouched")

	credentials.mu.Lock()
	credentials.putErr = nil
	credentials.mu.Unlock()
	require.NoError(t, storeWithMigration.MigrateLegacyControlTokens(context.Background()))
	require.NoError(t, db.QueryRow(`SELECT control_token FROM providers WHERE provider_id = ?`, descriptor.ProviderId).Scan(&retained))
	require.Empty(t, retained)
	got, err := store.Token(context.Background(), descriptor.ProviderId)
	require.NoError(t, err)
	require.Equal(t, legacy, got)
}

func TestMigrateLegacyControlTokensRejectsEqualCountsForDifferentProviders(t *testing.T) {
	store, db, _, _ := newStoreWithAuthority(t)
	ctx := context.Background()
	missing := validActive()
	missing.ProviderId = "missing.provider"
	missing.ProviderGroup = "missing"
	legacy := validActive()
	legacy.ProviderId = "legacy.provider"
	legacy.ProviderGroup = "legacy"
	for _, descriptor := range []*registryv1.ProviderDescriptor{missing, legacy} {
		blob, err := marshalDescriptor(descriptor)
		require.NoError(t, err)
		token := ""
		if descriptor == legacy {
			token = "legacy-only-token"
		}
		_, err = db.Exec(`INSERT INTO providers (provider_id, provider_group, bucket, type, state, scope, descriptor, control_token, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, descriptor.ProviderId, descriptor.ProviderGroup, int32(descriptor.Bucket), descriptor.Type, int32(descriptor.State), int32(descriptor.Scope), blob, token, "2026-05-01T12:00:00Z", "2026-05-01T12:00:00Z")
		require.NoError(t, err)
	}
	_, err := db.Exec(`INSERT INTO provider_control_token_witness (provider_id, minted_at) VALUES (?, ?)`, missing.ProviderId, "2026-05-01T12:00:00Z")
	require.NoError(t, err)

	migrator := store.(interface{ MigrateLegacyControlTokens(context.Context) error })
	require.ErrorContains(t, migrator.MigrateLegacyControlTokens(ctx), "witnesses exist")
	var retained string
	require.NoError(t, db.QueryRow(`SELECT control_token FROM providers WHERE provider_id = ?`, legacy.ProviderId).Scan(&retained))
	require.Equal(t, "legacy-only-token", retained, "migration must not clear a legacy token when a different witnessed provider is missing its credential")
}

func marshalDescriptor(descriptor *registryv1.ProviderDescriptor) (string, error) {
	blob, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(descriptor)
	return string(blob), err
}

func TestStoreUpsertRejectsInvalid(t *testing.T) {
	store, _ := newStore(t)
	d := validActive()
	d.Description = ""

	_, _, err := store.Upsert(context.Background(), d, "")
	require.Error(t, err)
	var invalid registry.ErrInvalidDescriptor
	require.ErrorAs(t, err, &invalid)
}

func TestStoreUpsertRoundTripsDescriptor(t *testing.T) {
	store, _ := newStore(t)
	ctx := context.Background()
	in := validActive()
	in.QueryHint = "prefix: find the command that"

	_, _, err := store.Upsert(ctx, in, "")
	require.NoError(t, err)

	got, err := store.Get(ctx, in.GetProviderId())
	require.NoError(t, err)
	require.Equal(t, in.GetProviderGroup(), got.GetProviderGroup())
	require.Equal(t, registryv1.Bucket_BUCKET_DO, got.GetBucket())
	require.Equal(t, "prefix: find the command that", got.GetQueryHint())
	require.Equal(t, "/vrooli.cli_health.v1.search.SearchService/Search", got.GetEndpoint().GetHttpJson().GetPath())
	require.Equal(t, "results", got.GetResultMapping().GetResultsPath())
	// Normalize defaults persisted.
	require.Equal(t, registryv1.ProviderState_PROVIDER_STATE_ACTIVE, got.GetState())
	require.Equal(t, registryv1.Scope_SCOPE_PROJECT, got.GetScope())
}

func TestStoreListFilters(t *testing.T) {
	store, _ := newStore(t)
	ctx := context.Background()

	// One ACTIVE DO command, one ACTIVE KNOW doc, one CAPABILITY_GAP code stub.
	require.NoError(t, mustUpsert(t, store, validActive()))

	doc := validActive()
	doc.ProviderId = "knowledge-observatory.docs"
	doc.ProviderGroup = "knowledge-observatory"
	doc.Bucket = registryv1.Bucket_BUCKET_KNOW
	doc.Type = "doc"
	require.NoError(t, mustUpsert(t, store, doc))

	gap := &registryv1.ProviderDescriptor{
		ProviderId:    "code.symbols",
		ProviderGroup: "code-reference",
		Bucket:        registryv1.Bucket_BUCKET_REUSE,
		Type:          "code",
		Description:   "Source symbols.",
		Lifecycle:     registryv1.Lifecycle_LIFECYCLE_PRODUCTION,
		State:         registryv1.ProviderState_PROVIDER_STATE_CAPABILITY_GAP,
		IntendedHome:  "code-reference",
	}
	require.NoError(t, mustUpsert(t, store, gap))

	t.Run("no filter returns all, id-ordered", func(t *testing.T) {
		all, err := store.List(ctx, registry.ListFilter{})
		require.NoError(t, err)
		require.Len(t, all, 3)
		require.Equal(t, "cli-health.commands", all[0].GetProviderId())
		require.Equal(t, "code.symbols", all[1].GetProviderId())
		require.Equal(t, "knowledge-observatory.docs", all[2].GetProviderId())
	})

	t.Run("filter by bucket", func(t *testing.T) {
		got, err := store.List(ctx, registry.ListFilter{Bucket: int32(registryv1.Bucket_BUCKET_KNOW)})
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, "knowledge-observatory.docs", got[0].GetProviderId())
	})

	t.Run("filter by type", func(t *testing.T) {
		got, err := store.List(ctx, registry.ListFilter{Type: "command"})
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, "cli-health.commands", got[0].GetProviderId())
	})

	t.Run("filter by state finds gaps", func(t *testing.T) {
		got, err := store.List(ctx, registry.ListFilter{State: int32(registryv1.ProviderState_PROVIDER_STATE_CAPABILITY_GAP)})
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, "code.symbols", got[0].GetProviderId())
	})
}

func TestStoreGetNotFound(t *testing.T) {
	store, _ := newStore(t)
	_, err := store.Get(context.Background(), "nope.nope")
	var notFound registry.ErrProviderNotFound
	require.ErrorAs(t, err, &notFound)
}

func TestStoreDeleteIdempotent(t *testing.T) {
	store, _, credentials, _ := newStoreWithAuthority(t)
	ctx := context.Background()
	_, oldToken, err := store.Upsert(ctx, validActive(), "")
	require.NoError(t, err)

	removed, err := store.Delete(ctx, "cli-health.commands")
	require.NoError(t, err)
	require.True(t, removed)
	var set map[string]string
	require.NoError(t, json.Unmarshal([]byte(credentials.values["provider-control-token-set"]), &set))
	require.Empty(t, set["cli-health.commands"], "deleting a provider must remove its secret from the authority")
	removed, err = store.Delete(ctx, "cli-health.commands")
	require.NoError(t, err)
	require.False(t, removed, "second delete is a no-op")

	_, newToken, err := store.Upsert(ctx, validActive(), "")
	require.NoError(t, err)
	require.NotEqual(t, oldToken, newToken, "re-registration after delete must mint a new token")
}

func mustUpsert(t *testing.T, store registry.Store, d *registryv1.ProviderDescriptor) error {
	t.Helper()
	_, _, err := store.Upsert(context.Background(), d, "")
	return err
}
