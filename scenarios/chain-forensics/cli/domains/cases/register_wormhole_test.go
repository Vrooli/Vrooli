package cases

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	p "github.com/vrooli/vrooli/packages/proto/gen/go/chain-forensics/v1/cases"
	c "github.com/vrooli/vrooli/packages/proto/gen/go/chain-forensics/v1/cases/cases_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestImportCaseUsesTypedCLIContractForWormholeCase(t *testing.T) {
	bodyDigest := sha256.Sum256([]byte("{}"))
	bodySHA := hex.EncodeToString(bodyDigest[:])
	caseBody := []byte(`{
  "id":"wormhole-live-pair-20240911",
  "title":"Private Wormhole Base to Arbitrum protocol pair",
  "summary":"Retained successful Base source and Arbitrum redemption receipts.",
  "private":true,
  "sources":[
    {"id":"base-source","name":"Base receipt","url":"https://mainnet.base.org","retrieved_at":"2026-11-14T06:09:02Z","sha256":"` + bodySHA + `","result_body":"e30=","raw_response_body":"e30=","raw_response_sha256":"` + bodySHA + `"},
    {"id":"arbitrum-destination","name":"Arbitrum receipt","url":"https://arb1.arbitrum.io/rpc","retrieved_at":"2026-11-14T06:09:02Z","sha256":"` + bodySHA + `","result_body":"e30=","raw_response_body":"e30=","raw_response_sha256":"` + bodySHA + `"}
  ],
  "occurrences":[
    {"id":"base:source-transfer","chain":"base","transaction_id":"0xsource","event_index":41,"from":"0xsender","to":"0x8d2de8d2f73f1f4cab472ac9a881c9b123c79627","asset":"erc20:0xff0c62a4979400841efaa6faadb07ac7d5c98b27","atomic_quantity":"75966","source_id":"base-source","kind":"observation"},
    {"id":"arbitrum:destination-transfer","chain":"arbitrum","transaction_id":"0xdestination","event_index":2,"from":"0xbridge","to":"0xe70946692e2e56ae47bfae2d93d31bd60952b090","asset":"erc20:0xe4728f3e48e94c6da2b53610e677cc241dafb134","atomic_quantity":"75966","source_id":"arbitrum-destination","kind":"observation"}
  ],
  "edges":[
    {"id":"bridge:wormhole:token-bridge:30:emitter:86175","from_occurrence":"base:source-transfer","to_occurrence":"arbitrum:destination-transfer","grade":"protocol_correlated","rule":"bridge.wormhole.token_bridge.transfer_redeemed_emitter_chain_address_sequence","evidence_ids":["base:source-transfer","arbitrum:destination-transfer"],"caveats":["Protocol-message correlation only; token equivalence and unit continuity are not established."]}
  ],
  "gaps":["No identity or exchange-custody claim is made."]
}`)
	require.True(t, json.Valid(caseBody))
	file := filepath.Join(t.TempDir(), "wormhole-case.json")
	require.NoError(t, os.WriteFile(file, caseBody, 0o600))

	var received p.ImportCaseRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, c.CasesServiceImportCaseProcedure, r.URL.Path)
		raw, err := readAll(r)
		require.NoError(t, err)
		require.NoError(t, protojson.Unmarshal(raw, &received))
		response, err := protojson.Marshal(&p.CaseResponse{Case: received.GetCase()})
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(response)
	}))
	defer server.Close()

	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "case"}}}, Flags: map[string]string{"case": file}})
	h := &client{rpc: c.NewCasesServiceClient(server.Client(), server.URL, connect.WithProtoJSON())}
	response, err := h.importCase(ctx)
	require.NoError(t, err)
	require.Equal(t, "wormhole-live-pair-20240911", received.GetCase().GetId())
	require.True(t, received.GetCase().GetPrivate())
	require.Len(t, received.GetCase().GetSources(), 2)
	require.Equal(t, p.Chain_CHAIN_BASE, received.GetCase().GetOccurrences()[0].GetChain())
	require.Equal(t, p.EvidenceGrade_EVIDENCE_GRADE_PROTOCOL_CORRELATED, received.GetCase().GetEdges()[0].GetGrade())
	require.Equal(t, "bridge.wormhole.token_bridge.transfer_redeemed_emitter_chain_address_sequence", received.GetCase().GetEdges()[0].GetRule())
	require.Equal(t, response.GetCase().GetId(), received.GetCase().GetId())
	report := h.importReport(ctx, response)
	require.Contains(t, report.Result[0], "wormhole-live-pair-20240911")
	require.Contains(t, report.Changes[0], "2 sources, 2 occurrences, 1 edges")
}

func readAll(r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	return io.ReadAll(r.Body)
}
