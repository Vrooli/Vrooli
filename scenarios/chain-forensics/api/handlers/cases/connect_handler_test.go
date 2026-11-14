package cases

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"testing"
	"time"

	"chain-forensics/internal/cases"
	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	db "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/scheduletest"
	forensics "github.com/vrooli/chain-forensics"
	proto "github.com/vrooli/vrooli/packages/proto/gen/go/chain-forensics/v1/cases"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestProtoRoundTripPreservesTypedEvidenceAndCaveats(t *testing.T) {
	doc, err := forensics.ExampleCase()
	require.NoError(t, err)
	wire := toProto(casesRecord(doc))
	got, err := fromProto(wire)
	require.NoError(t, err)
	require.Equal(t, doc, got)
}

func TestProtoRoundTripPreservesExactHTTPResponseEnvelope(t *testing.T) {
	doc, err := forensics.ExampleCase()
	require.NoError(t, err)
	response := []byte(`{"jsonrpc":"2.0","id":17,"result":[]}`)
	digest := sha256.Sum256(response)
	doc.Sources[0].RawResponseBody = response
	doc.Sources[0].RawResponseSHA256 = hex.EncodeToString(digest[:])
	wire := toProto(casesRecord(doc))
	got, err := fromProto(wire)
	require.NoError(t, err)
	require.Equal(t, response, got.Sources[0].RawResponseBody)
	require.Equal(t, hex.EncodeToString(digest[:]), got.Sources[0].RawResponseSHA256)
}

func TestPrivatePacketEndpointPersistsPackagePacketBeforeReturningBytes(t *testing.T) {
	ctx := context.Background()
	handle := db.NewSQLite(t)
	_, err := handle.ExecContext(ctx, cases.Schema())
	require.NoError(t, err)
	repo := cases.NewSQLiteRepository(handle, scheduletest.New(testTime()))
	doc, err := forensics.ExampleCase()
	require.NoError(t, err)
	_, err = repo.Import(ctx, doc)
	require.NoError(t, err)
	h := NewHandler(repo, log.New(io.Discard, "", 0))
	response, err := h.ExportPrivatePacket(ctx, connect.NewRequest(&proto.ExportPrivatePacketRequest{Id: doc.ID}))
	require.NoError(t, err)
	packet, err := forensics.BuildPrivatePacket(doc)
	require.NoError(t, err)
	require.Equal(t, packet.SchemaVersion, response.Msg.GetSchemaVersion())
	require.Equal(t, packet.Bytes, response.Msg.GetPacketJson())
	require.Equal(t, packet.SHA256, response.Msg.GetSha256())
	var persistedDigest string
	err = handle.QueryRowContext(ctx, `SELECT sha256 FROM forensic_packets WHERE case_id = ?`, doc.ID).Scan(&persistedDigest)
	require.NoError(t, err)
	require.Equal(t, packet.SHA256, persistedDigest)
}

func TestReviseCaseEndpointUsesTypedAppendOnlyRequestAndExposesHistory(t *testing.T) {
	ctx := context.Background()
	handle := db.NewSQLite(t)
	_, err := handle.ExecContext(ctx, cases.Schema())
	require.NoError(t, err)
	repo := cases.NewSQLiteRepository(handle, scheduletest.New(testTime()))
	doc, err := forensics.ExampleCase()
	require.NoError(t, err)
	doc.Gaps = append(doc.Gaps, "gap to resolve", "gap to retain")
	created, err := repo.Import(ctx, doc)
	require.NoError(t, err)
	h := NewHandler(repo, log.New(io.Discard, "", 0))
	source := &proto.Source{Id: "endpoint-revision", Name: "Public source", Url: "https://example.org/evidence", RetrievedAt: "2026-11-12T12:00:00Z", Sha256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	claim := &proto.Claim{Id: "endpoint-claim", Text: "Address label is reported by the source.", SourceId: source.GetId(), Kind: proto.EvidenceKind_EVIDENCE_KIND_LEGAL_CLAIM}
	oldOccurrence := doc.Occurrences[0]
	occurrence := &proto.Occurrence{Id: "endpoint-occurrence", Chain: chainToProto(oldOccurrence.Chain), TransactionId: oldOccurrence.Transaction, BlockHeight: oldOccurrence.Block, Timestamp: oldOccurrence.Timestamp, EventIndex: oldOccurrence.EventIndex, Direction: oldOccurrence.Direction, PreviousTransactionId: oldOccurrence.PreviousTx, PreviousOutputIndex: oldOccurrence.PreviousIndex, From: oldOccurrence.From, To: oldOccurrence.To, Asset: oldOccurrence.Asset, AtomicQuantity: oldOccurrence.Atomic, Scale: oldOccurrence.Scale, Kind: kindToProto(oldOccurrence.Kind), SourceId: source.GetId()}
	edge := &proto.FlowEdge{Id: "endpoint-revised-grade", FromOccurrence: occurrence.GetId(), ToOccurrence: doc.Edges[0].To, Grade: proto.EvidenceGrade_EVIDENCE_GRADE_HYPOTHESIS, Rule: "attribution hypothesis", EvidenceIds: []string{claim.GetId()}, Caveats: []string{"Does not prove control."}}
	oldEdge := doc.Edges[0]
	replacement := &proto.FlowEdge{Id: "endpoint-reassessed-grade", FromOccurrence: oldEdge.From, ToOccurrence: oldEdge.To, Grade: proto.EvidenceGrade_EVIDENCE_GRADE_HYPOTHESIS, Rule: "reassessed after source provenance review", EvidenceIds: []string{oldEdge.From}, Caveats: []string{"Prior grade retained in revision snapshot."}}
	request := &proto.ReviseCaseRequest{CaseId: doc.ID, ExpectedUpdatedAt: timestamppb.New(created.UpdatedAt), Summary: "Revised summary with newly captured public evidence.", ResolvedGaps: []string{"gap to resolve"}, Sources: []*proto.Source{source}, Claims: []*proto.Claim{claim}, Occurrences: []*proto.Occurrence{occurrence}, Edges: []*proto.FlowEdge{edge}, EdgeRevisions: []*proto.EdgeRevision{{SupersedesEdgeId: oldEdge.ID, Replacement: replacement}}}
	response, err := h.ReviseCase(ctx, connect.NewRequest(request))
	require.NoError(t, err)
	activeEdgeIDs := make([]string, 0, len(response.Msg.GetCase().GetEdges()))
	for _, active := range response.Msg.GetCase().GetEdges() {
		activeEdgeIDs = append(activeEdgeIDs, active.GetId())
	}
	require.Contains(t, activeEdgeIDs, edge.GetId())
	require.Equal(t, occurrence.GetId(), response.Msg.GetCase().GetOccurrences()[len(doc.Occurrences)].GetId())
	for _, active := range response.Msg.GetCase().GetEdges() {
		require.NotEqual(t, oldEdge.ID, active.GetId())
	}
	require.Equal(t, replacement.GetId(), response.Msg.GetCase().GetEdges()[len(response.Msg.GetCase().GetEdges())-1].GetId())
	require.Equal(t, request.GetSummary(), response.Msg.GetCase().GetSummary())
	require.NotContains(t, response.Msg.GetCase().GetGaps(), "gap to resolve")
	require.Contains(t, response.Msg.GetCase().GetGaps(), "gap to retain")
	require.NotEmpty(t, response.Msg.GetRevisionId())
	_, err = h.ReviseCase(ctx, connect.NewRequest(request))
	require.Error(t, err)
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeAborted, connectErr.Code())
	history, err := h.ListCaseRevisions(ctx, connect.NewRequest(&proto.ListCaseRevisionsRequest{CaseId: doc.ID}))
	require.NoError(t, err)
	require.Len(t, history.Msg.GetRevisions(), 1)
	require.Equal(t, claim.GetId(), history.Msg.GetRevisions()[0].GetAdditions().GetClaims()[0].GetId())
	require.Equal(t, occurrence.GetId(), history.Msg.GetRevisions()[0].GetAdditions().GetOccurrences()[0].GetId())
	require.Equal(t, request.GetSummary(), history.Msg.GetRevisions()[0].GetAdditions().GetSummary())
	require.Equal(t, []string{"gap to resolve"}, history.Msg.GetRevisions()[0].GetAdditions().GetResolvedGaps())
	require.Equal(t, doc.Summary, history.Msg.GetRevisions()[0].GetPreviousCase().GetSummary())
	require.Contains(t, history.Msg.GetRevisions()[0].GetPreviousCase().GetGaps(), "gap to resolve")
	require.Contains(t, history.Msg.GetRevisions()[0].GetPreviousCase().GetEdges(), toProto(casesRecord(doc)).GetEdges()[0])
	require.Equal(t, replacement.GetId(), history.Msg.GetRevisions()[0].GetAdditions().GetEdgeRevisions()[0].GetReplacement().GetId())
}

func testTime() time.Time { return time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC) }

func casesRecord(doc forensics.Case) cases.Record { return cases.Record{Case: doc} }
