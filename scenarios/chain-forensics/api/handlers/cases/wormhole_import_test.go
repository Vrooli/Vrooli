package cases

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	internalcases "chain-forensics/internal/cases"
	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	db "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/scheduletest"
	forensics "github.com/vrooli/chain-forensics"
	proto "github.com/vrooli/vrooli/packages/proto/gen/go/chain-forensics/v1/cases"
)

func TestImportCaseServesRetainedWormholeBaseArbitrumCorrelation(t *testing.T) {
	ctx := context.Background()
	handle := db.NewSQLite(t)
	_, err := handle.ExecContext(ctx, internalcases.Schema())
	require.NoError(t, err)
	repo := internalcases.NewSQLiteRepository(handle, scheduletest.New(testTime()))
	h := NewHandler(repo, log.New(io.Discard, "", 0))
	evidenceDir := filepath.Join("..", "..", "..", "..", "..", "packages", "chain-forensics", "evidence", "cf6-wormhole-live-pair-20261114")

	operation := readEvidenceSource(t, evidenceDir, "wormholescan-operation.json", "wormholescan-index", "https://api.wormholescan.io/api/v1/operations/30/0000000000000000000000008d2de8d2f73f1f4cab472ac9a881c9b123c79627/86175", "GET", "Public indexer metadata identifies the paired source/destination transaction; raw chain RPC receipts remain primary.")
	baseSource, baseBlockSource, baseReceipt, baseTimestamp := readEvidenceReceipt(t, evidenceDir, "base-source-receipt.json", "base-source-block.json", "base-source", "https://mainnet.base.org")
	arbSource, arbBlockSource, arbReceipt, arbTimestamp := readEvidenceReceipt(t, evidenceDir, "arbitrum-destination-receipt.json", "arbitrum-destination-block.json", "arbitrum-destination", "https://arb1.arbitrum.io/rpc")
	occurrences := append(evidenceTransfers(t, forensics.ChainBase, baseSource, baseReceipt, baseTimestamp), evidenceTransfers(t, forensics.ChainArbitrum, arbSource, arbReceipt, arbTimestamp)...)
	doc := forensics.Case{
		ID: "private-wormhole-base-arbitrum-import", Title: "Wormhole Base→Arbitrum source-bound protocol pair", Summary: "Private local API fixture from retained successful public chain receipts.", Private: true,
		Sources: []forensics.Source{operation, baseSource, baseBlockSource, arbSource, arbBlockSource}, Occurrences: occurrences,
		Gaps: []string{"Same atomic amount and protocol recipient across different ERC-20 contracts do not prove token equivalence or economic continuity.", "No identity, ownership, intent, exchange custody, or full transfer history claim is made."},
	}
	response, err := h.ImportCase(ctx, connect.NewRequest(&proto.ImportCaseRequest{Case: toProto(casesRecord(doc))}))
	require.NoError(t, err)
	require.True(t, response.Msg.GetCase().GetPrivate())
	var bridgeEdges []*proto.FlowEdge
	for _, edge := range response.Msg.GetCase().GetEdges() {
		if strings.HasPrefix(edge.GetId(), "bridge:wormhole:") {
			bridgeEdges = append(bridgeEdges, edge)
		}
	}
	require.Len(t, bridgeEdges, 1)
	require.Equal(t, proto.EvidenceGrade_EVIDENCE_GRADE_PROTOCOL_CORRELATED, bridgeEdges[0].GetGrade())
	stored, err := h.GetCase(ctx, connect.NewRequest(&proto.GetCaseRequest{Id: doc.ID}))
	require.NoError(t, err)
	require.Contains(t, stored.Msg.GetCase().GetEdges(), bridgeEdges[0])
	require.Len(t, stored.Msg.GetCase().GetSources(), 5)
	require.Len(t, stored.Msg.GetCase().GetOccurrences(), len(occurrences))
	for _, expected := range doc.Sources {
		var persisted *proto.Source
		for _, candidate := range stored.Msg.GetCase().GetSources() {
			if candidate.GetId() == expected.ID {
				persisted = candidate
				break
			}
		}
		require.NotNil(t, persisted, "missing source after SQLite readback: %s", expected.ID)
		require.Equal(t, expected.SHA256, persisted.GetSha256())
		require.Equal(t, expected.ResultBody, persisted.GetResultBody())
		require.Equal(t, expected.RawResponseSHA256, persisted.GetRawResponseSha256())
		require.Equal(t, expected.RawResponseBody, persisted.GetRawResponseBody())
	}

	packet, err := h.ExportPrivatePacket(ctx, connect.NewRequest(&proto.ExportPrivatePacketRequest{Id: doc.ID}))
	require.NoError(t, err)
	require.Equal(t, "chain-forensics.packet/v1", packet.Msg.GetSchemaVersion())
	packetDigest := sha256.Sum256(packet.Msg.GetPacketJson())
	require.Equal(t, hex.EncodeToString(packetDigest[:]), packet.Msg.GetSha256())
	var storedSchema, storedDigest string
	var storedJSON []byte
	require.NoError(t, handle.QueryRowContext(ctx, `SELECT schema_version, sha256, packet_json FROM forensic_packets WHERE case_id = ?`, doc.ID).Scan(&storedSchema, &storedDigest, &storedJSON))
	require.Equal(t, packet.Msg.GetSchemaVersion(), storedSchema)
	require.Equal(t, packet.Msg.GetSha256(), storedDigest)
	require.Equal(t, packet.Msg.GetPacketJson(), storedJSON)
	var packetDocument struct {
		Case forensics.Case `json:"case"`
	}
	require.NoError(t, json.Unmarshal(storedJSON, &packetDocument))
	require.True(t, packetDocument.Case.Private)
	require.Contains(t, packetDocument.Case.Edges, forensics.Edge{ID: bridgeEdges[0].GetId(), From: bridgeEdges[0].GetFromOccurrence(), To: bridgeEdges[0].GetToOccurrence(), Grade: forensics.GradeProtocolCorrelated, Rule: bridgeEdges[0].GetRule(), EvidenceIDs: bridgeEdges[0].GetEvidenceIds(), Caveats: bridgeEdges[0].GetCaveats()})
	_, err = h.ExportPrivatePacket(ctx, connect.NewRequest(&proto.ExportPrivatePacketRequest{Id: doc.ID}))
	require.NoError(t, err)
	var packetCount int
	require.NoError(t, handle.QueryRowContext(ctx, `SELECT COUNT(*) FROM forensic_packets WHERE case_id = ?`, doc.ID).Scan(&packetCount))
	require.Equal(t, 1, packetCount, "packet replay should remain idempotent")
}

type retainedReceipt struct {
	TransactionHash string             `json:"transactionHash"`
	Logs            []forensics.EVMLog `json:"logs"`
}

func readEvidenceSource(t *testing.T, dir, name, id, endpoint, method, limitations string) forensics.Source {
	t.Helper()
	full, err := os.ReadFile(filepath.Join(dir, name))
	require.NoError(t, err)
	var response struct {
		Result json.RawMessage `json:"result"`
	}
	if strings.HasPrefix(name, "wormholescan-") {
		response.Result = append([]byte(nil), full...)
	} else {
		require.NoError(t, json.Unmarshal(full, &response))
	}
	resultDigest := sha256.Sum256(response.Result)
	rawDigest := sha256.Sum256(full)
	retrieved := "2026-11-14T06:09:02Z"
	if strings.HasPrefix(name, "wormholescan-") {
		retrieved = "2026-11-14T06:06:16Z"
	} else if method == "eth_getBlockByNumber" {
		retrieved = "2026-11-14T06:09:01Z"
	} else if method == "eth_getTransactionReceipt" {
		retrieved = "2026-11-14T06:06:54Z"
	}
	return forensics.Source{ID: id, Name: "Retained Wormhole transaction evidence", URL: endpoint, Retrieved: retrieved, Citation: method + " response", Limitations: limitations, SHA256: hex.EncodeToString(resultDigest[:]), ResultBody: response.Result, RawResponseBody: full, RawResponseSHA256: hex.EncodeToString(rawDigest[:])}
}

func readEvidenceReceipt(t *testing.T, dir, receiptName, blockName, id, endpoint string) (forensics.Source, forensics.Source, retainedReceipt, string) {
	t.Helper()
	full, err := os.ReadFile(filepath.Join(dir, receiptName))
	require.NoError(t, err)
	var response struct {
		Result json.RawMessage `json:"result"`
	}
	require.NoError(t, json.Unmarshal(full, &response))
	var receipt retainedReceipt
	require.NoError(t, json.Unmarshal(response.Result, &receipt))
	blockFull, err := os.ReadFile(filepath.Join(dir, blockName))
	require.NoError(t, err)
	var blockResponse struct {
		Result struct {
			Hash      string `json:"hash"`
			Timestamp string `json:"timestamp"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(blockFull, &blockResponse))
	var receiptIdentity struct {
		BlockHash string `json:"blockHash"`
		Status    string `json:"status"`
	}
	require.NoError(t, json.Unmarshal(response.Result, &receiptIdentity))
	require.Equal(t, "0x1", receiptIdentity.Status)
	require.True(t, strings.EqualFold(blockResponse.Result.Hash, receiptIdentity.BlockHash))
	seconds, ok := new(big.Int).SetString(strings.TrimPrefix(blockResponse.Result.Timestamp, "0x"), 16)
	require.True(t, ok)
	require.True(t, seconds.IsInt64())
	blockAt := time.Unix(seconds.Int64(), 0).UTC().Format(time.RFC3339)
	resultDigest := sha256.Sum256(response.Result)
	rawDigest := sha256.Sum256(full)
	source := forensics.Source{ID: id, Name: "Public EVM JSON-RPC receipt", URL: endpoint, Retrieved: "2026-11-14T06:06:54Z", Citation: "eth_getTransactionReceipt", Limitations: "Successful receipt; exact JSON-RPC result and raw HTTP response are retained.", SHA256: hex.EncodeToString(resultDigest[:]), ResultBody: response.Result, RawResponseBody: full, RawResponseSHA256: hex.EncodeToString(rawDigest[:])}
	blockSource := readEvidenceSource(t, dir, blockName, id+"-block", endpoint, "eth_getBlockByNumber", "Anchors the receipt block hash and timestamp; exact JSON-RPC result and raw HTTP response are retained.")
	return source, blockSource, receipt, blockAt
}

func evidenceTransfers(t *testing.T, chain forensics.Chain, source forensics.Source, receipt retainedReceipt, timestamp string) []forensics.Occurrence {
	t.Helper()
	var result []forensics.Occurrence
	for _, event := range receipt.Logs {
		if len(event.Topics) == 0 || !strings.EqualFold(event.Topics[0], forensics.ERC20TransferTopic) {
			continue
		}
		occurrence, err := forensics.NormalizeERC20TransferLog(chain, source, event, timestamp)
		require.NoError(t, err)
		result = append(result, occurrence)
	}
	return result
}
