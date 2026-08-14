package byokstore_test

import (
	"context"
	"crypto/rand"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	"audio-tools/internal/byokstore"
	"audio-tools/internal/store"
	db "github.com/vrooli/api-core/databasetest"
)

func newEnc(t *testing.T) *byokstore.Encryptor {
	t.Helper()
	k := make([]byte, 32)
	_, _ = rand.Read(k)
	e, err := byokstore.NewEncryptor(k)
	require.NoError(t, err)
	return e
}

func TestEncryptor_RoundTrip(t *testing.T) {
	e := newEnc(t)
	ct, err := e.Seal([]byte("hello world"))
	require.NoError(t, err)
	pt, err := e.Open(ct)
	require.NoError(t, err)
	require.Equal(t, "hello world", string(pt))
}

func TestEncryptor_InvalidCiphertext(t *testing.T) {
	e := newEnc(t)
	_, err := e.Open([]byte{1, 2, 3})
	require.ErrorIs(t, err, byokstore.ErrInvalidCiphertext)
}

func TestFingerprint(t *testing.T) {
	require.Equal(t, "***", byokstore.Fingerprint("abc"))
	require.Equal(t, "s***cd", byokstore.Fingerprint("sk-abcd"))
	require.Equal(t, "sk***wxyz", byokstore.Fingerprint("sk-abcdefwxyz"))
}

func TestLoadOrCreateKey_Persistence(t *testing.T) {
	t.Setenv(byokstore.KeyEnvVar, "")
	dir := t.TempDir()
	path := filepath.Join(dir, "key")
	k1, err := byokstore.LoadOrCreateKey(path)
	require.NoError(t, err)
	require.Len(t, k1, 32)
	k2, err := byokstore.LoadOrCreateKey(path)
	require.NoError(t, err)
	require.Equal(t, k1, k2)
}

func TestStore_UpsertListGetDelete(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(byokstore.Schema)))
	enc := newEnc(t)
	s := byokstore.New(enc, store.NewBYOKStore(apidb.NewFromPrimary(d)))
	ctx := context.Background()

	c, err := s.Upsert(ctx, "openai-tts", "tts", "sk-supersecret-key")
	require.NoError(t, err)
	require.Equal(t, "sk***-key", c.Fingerprint)

	got, ok, err := s.Get(ctx, "openai-tts", "tts")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "sk-supersecret-key", got)

	list, err := s.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)

	_, ok, err = s.Get(ctx, "nonexistent", "tts")
	require.NoError(t, err)
	require.False(t, ok)

	deleted, err := s.Delete(ctx, "openai-tts", "tts")
	require.NoError(t, err)
	require.True(t, deleted)
}
