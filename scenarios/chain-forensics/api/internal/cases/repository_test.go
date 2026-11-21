package cases_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"chain-forensics/internal/cases"
	"github.com/stretchr/testify/require"
	db "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/scheduletest"
	forensics "github.com/vrooli/chain-forensics"
)

func TestSQLiteImportIsValidatedIdempotentAndConflictSafe(t *testing.T) {
	ctx := context.Background()
	handle := db.NewSQLite(t)
	_, err := handle.ExecContext(ctx, cases.Schema())
	require.NoError(t, err)
	clock := scheduletest.New(time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC))
	repo := cases.NewSQLiteRepository(handle, clock)
	doc, err := forensics.ExampleCase()
	require.NoError(t, err)
	first, err := repo.Import(ctx, doc)
	require.NoError(t, err)
	require.Equal(t, doc.ID, first.Case.ID)
	replay, err := repo.Import(ctx, doc)
	require.NoError(t, err)
	require.Equal(t, first, replay)
	changed := doc
	changed.Title += " changed"
	_, err = repo.Import(ctx, changed)
	require.ErrorAs(t, err, new(cases.ErrConflict))
	invalid := forensics.Case{ID: "invalid"}
	_, err = repo.Import(ctx, invalid)
	require.ErrorAs(t, err, new(cases.ErrInvalid))
}

func TestSQLiteLabelsRemainSeparateAndAppendOnly(t *testing.T) {
	ctx := context.Background()
	handle := db.NewSQLite(t)
	_, err := handle.ExecContext(ctx, cases.Schema())
	require.NoError(t, err)
	repo := cases.NewSQLiteRepository(handle, scheduletest.New(time.Now().UTC()))
	doc, err := forensics.ExampleCase()
	require.NoError(t, err)
	_, err = repo.Import(ctx, doc)
	require.NoError(t, err)
	label, err := repo.AddLabel(ctx, cases.AnalystLabel{CaseID: doc.ID, Subject: "1HQ3", Label: "operator hypothesis", SourceURL: "https://example.org/case", SourceCitation: "paragraph 1", Grade: forensics.GradeHypothesis, CounterExplanation: "shared custody remains possible", UnresolvedGap: "ownership remains unknown"})
	require.NoError(t, err)
	got, err := repo.Get(ctx, doc.ID)
	require.NoError(t, err)
	require.Equal(t, doc, got.Case)
	labels, err := repo.ListLabels(ctx, doc.ID)
	require.NoError(t, err)
	require.Equal(t, []cases.AnalystLabel{label}, labels)
}

func TestSQLitePrivatePacketsPersistCanonicallyAndReplayIdempotently(t *testing.T) {
	ctx := context.Background()
	handle := db.NewSQLite(t)
	_, err := handle.ExecContext(ctx, cases.Schema())
	require.NoError(t, err)
	clock := scheduletest.New(time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC))
	repo := cases.NewSQLiteRepository(handle, clock)
	doc, err := forensics.ExampleCase()
	require.NoError(t, err)
	_, err = repo.Import(ctx, doc)
	require.NoError(t, err)
	packet, err := forensics.BuildPrivatePacket(doc)
	require.NoError(t, err)
	first, err := repo.SavePacket(ctx, packet)
	require.NoError(t, err)
	require.Equal(t, doc.ID, first.CaseID)
	require.Equal(t, packet.SchemaVersion, first.SchemaVersion)
	require.Equal(t, packet.SHA256, first.SHA256)
	require.Equal(t, packet.Bytes, first.JSON)
	clock.Advance(time.Minute)
	replay, err := repo.SavePacket(ctx, packet)
	require.NoError(t, err)
	require.Equal(t, first, replay)

	corrupt := packet
	corrupt.Bytes = []byte(`{"schema_version":"spoofed"}`)
	_, err = repo.SavePacket(ctx, corrupt)
	require.ErrorAs(t, err, new(cases.ErrInvalid))
}

func TestSQLiteMergeCaptureAppendsAndReplaysWithoutRewritingEvidence(t *testing.T) {
	ctx := context.Background()
	handle := db.NewSQLite(t)
	_, err := handle.ExecContext(ctx, cases.Schema())
	require.NoError(t, err)
	clock := scheduletest.New(time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC))
	repo := cases.NewSQLiteRepository(handle, clock)
	doc, err := forensics.ExampleCase()
	require.NoError(t, err)
	before, err := repo.Import(ctx, doc)
	require.NoError(t, err)
	source := forensics.Source{ID: "captured-receipt", Name: "Public JSON-RPC", URL: "https://rpc.example", Retrieved: "2026-11-12T12:00:00Z", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Citation: "eth_getTransactionReceipt"}
	first, err := repo.MergeCapture(ctx, doc.ID, []forensics.Source{source}, nil)
	require.NoError(t, err)
	require.Len(t, first.Case.Sources, len(doc.Sources)+1)
	require.Equal(t, doc.Occurrences, first.Case.Occurrences)
	second, err := repo.MergeCapture(ctx, doc.ID, []forensics.Source{source}, nil)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Equal(t, before.Case.Occurrences, second.Case.Occurrences)
	reobserved := source
	reobserved.Retrieved = "2026-11-12T12:01:00Z"
	reobservedReplay, err := repo.MergeCapture(ctx, doc.ID, []forensics.Source{reobserved}, nil)
	require.NoError(t, err)
	require.Equal(t, second, reobservedReplay)
	conflict := source
	conflict.Name = "different response under same ID"
	_, err = repo.MergeCapture(ctx, doc.ID, []forensics.Source{conflict}, nil)
	require.ErrorAs(t, err, new(cases.ErrConflict))
}

func TestSQLiteMergeCaptureKeepsFirstExactEnvelopeOnSameRPCResultReplay(t *testing.T) {
	ctx := context.Background()
	handle := db.NewSQLite(t)
	_, err := handle.ExecContext(ctx, cases.Schema())
	require.NoError(t, err)
	repo := cases.NewSQLiteRepository(handle, scheduletest.New(time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)))
	doc, err := forensics.ExampleCase()
	require.NoError(t, err)
	_, err = repo.Import(ctx, doc)
	require.NoError(t, err)
	result := []byte(`"0x1"`)
	resultHash := sha256.Sum256(result)
	source := forensics.Source{ID: "evm-repeat", Name: "RPC", URL: "https://rpc.example", Retrieved: "2026-11-12T12:00:00Z", SHA256: hex.EncodeToString(resultHash[:]), Citation: "result object", Limitations: "full HTTP response not retained yet", ResultBody: result}
	first, err := repo.MergeCapture(ctx, doc.ID, []forensics.Source{source}, nil)
	require.NoError(t, err)
	response1 := []byte(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`)
	responseHash1 := sha256.Sum256(response1)
	enriched := source
	enriched.RawResponseBody = response1
	enriched.RawResponseSHA256 = hex.EncodeToString(responseHash1[:])
	enriched.Limitations = "full HTTP response retained"
	second, err := repo.MergeCapture(ctx, doc.ID, []forensics.Source{enriched}, nil)
	require.NoError(t, err)
	require.NotEqual(t, first.UpdatedAt, second.UpdatedAt)
	require.Equal(t, response1, second.Case.Sources[len(doc.Sources)].RawResponseBody)
	require.Equal(t, "full HTTP response retained", second.Case.Sources[len(doc.Sources)].Limitations)
	response2 := []byte(`{"jsonrpc":"2.0","id":2,"result":"0x1"}`)
	responseHash2 := sha256.Sum256(response2)
	replay := enriched
	replay.RawResponseBody = response2
	replay.RawResponseSHA256 = hex.EncodeToString(responseHash2[:])
	third, err := repo.MergeCapture(ctx, doc.ID, []forensics.Source{replay}, nil)
	require.NoError(t, err)
	require.Equal(t, second, third)
	require.Equal(t, response1, third.Case.Sources[len(doc.Sources)].RawResponseBody)
}
