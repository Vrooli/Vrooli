package cases

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	p "github.com/vrooli/vrooli/packages/proto/gen/go/chain-forensics/v1/cases"
	c "github.com/vrooli/vrooli/packages/proto/gen/go/chain-forensics/v1/cases/cases_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
)

type client struct{ rpc c.CasesServiceClient }

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	httpClient, base := cliapp.NewConnectHTTPClient(core)
	h := &client{rpc: c.NewCasesServiceClient(httpClient, base)}
	prefix := c.CasesServiceName + "."
	bindings := map[string]cliapp.PrimitiveHandler{
		prefix + "ListCases": cliapp.ProtoList(h.list, h.listReport), prefix + "ImportCase": cliapp.ProtoMutation(h.importCase, h.importReport),
		prefix + "GetCase": cliapp.ProtoList(h.get, h.getReport), prefix + "ExportPrivatePacket": cliapp.ProtoList(h.packet, h.packetReport),
		prefix + "AddAnalystLabel": cliapp.ProtoMutation(h.addLabel, h.labelReport), prefix + "ListAnalystLabels": cliapp.ProtoList(h.labels, h.labelsReport),
		prefix + "CaptureEVMTransaction":     cliapp.ProtoMutation(h.captureEVM, h.captureReport),
		prefix + "CaptureBitcoinTransaction": cliapp.ProtoMutation(h.captureBitcoin, h.captureBitcoinReport),
		prefix + "CaptureBitcoinAddressPage": cliapp.ProtoMutation(h.captureBitcoinAddressPage, h.captureBitcoinAddressPageReport),
		prefix + "CaptureEVMWalletSnapshot":  cliapp.ProtoMutation(h.captureEVMWalletSnapshot, h.captureEVMWalletSnapshotReport),
		prefix + "CaptureSolanaAddressPage":  cliapp.ProtoMutation(h.captureSolanaAddressPage, h.captureSolanaAddressPageReport),
		prefix + "ReviseCase":                cliapp.ProtoMutation(h.reviseCase, h.reviseReport),
		prefix + "ListCaseRevisions":         cliapp.ProtoList(h.revisions, h.revisionsReport),
	}
	return cliapp.LoadFromManifestPrimitives(manifest, "cases", bindings)
}

func (h *client) reviseCase(ctx cliapp.OperationContext) (*p.ReviseCaseResponse, error) {
	b, err := os.ReadFile(ctx.Flag("revision"))
	if err != nil {
		return nil, fmt.Errorf("read case revision JSON: %w", err)
	}
	var raw any
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("decode case revision JSON: %w", err)
	}
	normalizeEnums(raw)
	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	request := new(p.ReviseCaseRequest)
	if err := protojson.Unmarshal(encoded, request); err != nil {
		return nil, fmt.Errorf("decode typed case revision: %w", err)
	}
	r, err := h.rpc.ReviseCase(context.Background(), connect.NewRequest(request))
	if err != nil {
		return nil, cliapp.WrapAPIError("revise case", err, nil)
	}
	return r.Msg, nil
}

func (h *client) reviseReport(_ cliapp.OperationContext, r *p.ReviseCaseResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Applied revision %s to case %s.", r.GetRevisionId(), r.GetCase().GetId())}, Changes: []string{fmt.Sprintf("Case now contains %d sources, %d claims, and %d flow edges; prior evidence and revision receipts are retained.", len(r.GetCase().GetSources()), len(r.GetCase().GetClaims()), len(r.GetCase().GetEdges()))}}
}

func (h *client) revisions(ctx cliapp.OperationContext) (*p.ListCaseRevisionsResponse, error) {
	r, err := h.rpc.ListCaseRevisions(context.Background(), connect.NewRequest(&p.ListCaseRevisionsRequest{CaseId: ctx.Positional("case-id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list case revisions", err, nil)
	}
	return r.Msg, nil
}

func (h *client) revisionsReport(_ cliapp.OperationContext, r *p.ListCaseRevisionsResponse) cliapp.ListReport {
	out := cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d case revision(s).", len(r.GetRevisions()))}, ResultsHeading: "Case revision history", ListShaped: true, ResultCount: len(r.GetRevisions())}
	for _, revision := range r.GetRevisions() {
		adds := revision.GetAdditions()
		out.Results = append(out.Results, fmt.Sprintf("%s at %s: +%d sources, +%d claims, +%d occurrences, +%d edges, +%d caveats, resolved %d gaps; prior summary: %q", revision.GetId(), revision.GetAppliedAt().AsTime().Format(time.RFC3339), len(adds.GetSources()), len(adds.GetClaims()), len(adds.GetOccurrences()), len(adds.GetEdges()), len(adds.GetGaps())+len(adds.GetCounterexamples())+len(adds.GetSourceDisagreements()), len(adds.GetResolvedGaps()), revision.GetPreviousCase().GetSummary()))
	}
	return out
}

func (h *client) captureSolanaAddressPage(ctx cliapp.OperationContext) (*p.CaptureSolanaAddressPageResponse, error) {
	r, e := h.rpc.CaptureSolanaAddressPage(context.Background(), connect.NewRequest(&p.CaptureSolanaAddressPageRequest{CaseId: ctx.Flag("case-id"), Address: ctx.Flag("address"), BeforeSignature: ctx.Flag("before-signature")}))
	if e != nil {
		return nil, cliapp.WrapAPIError("capture Solana address page", e, nil)
	}
	return r.Msg, nil
}

func (h *client) captureSolanaAddressPageReport(_ cliapp.OperationContext, r *p.CaptureSolanaAddressPageResponse) cliapp.MutationReport {
	id := ""
	if r.GetCase() != nil {
		id = r.GetCase().GetId()
	}
	result := fmt.Sprintf("Captured %d finalized Solana transaction(s) and %d token-balance observation(s) into case %s.", r.GetCapturedTransactionCount(), r.GetCapturedOccurrenceCount(), id)
	if r.GetNextBeforeSignature() != "" {
		result += " Next page cursor: " + r.GetNextBeforeSignature()
	}
	return cliapp.MutationReport{Result: []string{result}, Changes: []string{"One bounded finalized signature page and per-transaction token balance deltas were attached with response hashes."}}
}

func (h *client) captureEVMWalletSnapshot(ctx cliapp.OperationContext) (*p.CaptureEVMWalletSnapshotResponse, error) {
	var chain p.Chain
	switch strings.ToLower(ctx.Flag("chain")) {
	case "ethereum":
		chain = p.Chain_CHAIN_ETHEREUM
	case "base":
		chain = p.Chain_CHAIN_BASE
	default:
		return nil, fmt.Errorf("chain must be ethereum or base")
	}
	var tokens []string
	for _, token := range strings.Split(ctx.Flag("token-contracts"), ",") {
		if token = strings.TrimSpace(token); token != "" {
			tokens = append(tokens, token)
		}
	}
	r, e := h.rpc.CaptureEVMWalletSnapshot(context.Background(), connect.NewRequest(&p.CaptureEVMWalletSnapshotRequest{CaseId: ctx.Flag("case-id"), Chain: chain, Address: ctx.Flag("address"), TokenContracts: tokens}))
	if e != nil {
		return nil, cliapp.WrapAPIError("capture EVM wallet snapshot", e, nil)
	}
	return r.Msg, nil
}

func (h *client) captureEVMWalletSnapshotReport(_ cliapp.OperationContext, r *p.CaptureEVMWalletSnapshotResponse) cliapp.MutationReport {
	count := 0
	if r.GetCase() != nil {
		count = len(r.GetCase().GetWalletSnapshots())
	}
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Case %s now contains %d pinned EVM wallet snapshot(s).", r.GetCase().GetId(), count)}, Changes: []string{"Native and selected ERC-20 balances, block identity and source provenance were captured into the local case."}}
}

func (h *client) captureBitcoinAddressPage(ctx cliapp.OperationContext) (*p.CaptureBitcoinAddressPageResponse, error) {
	r, e := h.rpc.CaptureBitcoinAddressPage(context.Background(), connect.NewRequest(&p.CaptureBitcoinAddressPageRequest{CaseId: ctx.Flag("case-id"), Address: ctx.Flag("address"), AfterTxid: ctx.Flag("after-txid")}))
	if e != nil {
		return nil, cliapp.WrapAPIError("capture Bitcoin address page", e, nil)
	}
	return r.Msg, nil
}

func (h *client) captureBitcoinAddressPageReport(_ cliapp.OperationContext, r *p.CaptureBitcoinAddressPageResponse) cliapp.MutationReport {
	result := fmt.Sprintf("Captured %d confirmed Bitcoin transaction(s), %d occurrence(s), into case %s.", r.GetCapturedTransactionCount(), r.GetCapturedOccurrenceCount(), r.GetCase().GetId())
	if r.GetNextAfterTxid() != "" {
		result += " Next page cursor: " + r.GetNextAfterTxid()
	}
	return cliapp.MutationReport{Result: []string{result}, Changes: []string{"One bounded mempool.space confirmed-history page and its response hash were attached to this case."}}
}

func (h *client) captureBitcoin(ctx cliapp.OperationContext) (*p.CaptureBitcoinTransactionResponse, error) {
	r, e := h.rpc.CaptureBitcoinTransaction(context.Background(), connect.NewRequest(&p.CaptureBitcoinTransactionRequest{CaseId: ctx.Flag("case-id"), TransactionId: ctx.Flag("transaction-id")}))
	if e != nil {
		return nil, cliapp.WrapAPIError("capture Bitcoin transaction", e, nil)
	}
	return r.Msg, nil
}

func (h *client) captureBitcoinReport(_ cliapp.OperationContext, r *p.CaptureBitcoinTransactionResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Captured %d Bitcoin input/output occurrence(s) into case %s.", r.GetCapturedOccurrenceCount(), r.GetCase().GetId())}, Changes: []string{"A confirmed transaction was fetched from mempool.space; raw response provenance and normalized inputs/outputs are now attached to this case."}}
}

func (h *client) captureEVM(ctx cliapp.OperationContext) (*p.CaptureEVMTransactionResponse, error) {
	var chain p.Chain
	switch strings.ToLower(ctx.Flag("chain")) {
	case "ethereum":
		chain = p.Chain_CHAIN_ETHEREUM
	case "base":
		chain = p.Chain_CHAIN_BASE
	case "arbitrum":
		chain = p.Chain_CHAIN_ARBITRUM
	default:
		return nil, fmt.Errorf("chain must be ethereum, base, or arbitrum")
	}
	r, e := h.rpc.CaptureEVMTransaction(context.Background(), connect.NewRequest(&p.CaptureEVMTransactionRequest{CaseId: ctx.Flag("case-id"), Chain: chain, TransactionHash: ctx.Flag("transaction-hash")}))
	if e != nil {
		return nil, cliapp.WrapAPIError("capture EVM transaction", e, nil)
	}
	return r.Msg, nil
}

func (h *client) captureReport(_ cliapp.OperationContext, r *p.CaptureEVMTransactionResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Captured %d EVM transfer occurrence(s) into case %s.", r.GetCapturedOccurrenceCount(), r.GetCase().GetId())}, Changes: []string{"A public endpoint read the confirmed receipt, matching block and transaction; ERC-20 logs and native value transfers are now attached with response hashes."}}
}

func (h *client) list(_ cliapp.OperationContext) (*p.ListCasesResponse, error) {
	r, e := h.rpc.ListCases(context.Background(), connect.NewRequest(&p.ListCasesRequest{Limit: 100}))
	if e != nil {
		return nil, cliapp.WrapAPIError("list cases", e, nil)
	}
	return r.Msg, nil
}

func (h *client) listReport(_ cliapp.OperationContext, r *p.ListCasesResponse) cliapp.ListReport {
	out := cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d case(s).", len(r.GetCases()))}, ResultsHeading: "Forensic cases", ListShaped: true, ResultCount: len(r.GetCases())}
	for _, v := range r.GetCases() {
		out.Results = append(out.Results, fmt.Sprintf("%s — %s (%d occurrences, %d edges, private=%t)", v.GetId(), v.GetTitle(), v.GetOccurrenceCount(), v.GetEdgeCount(), v.GetPrivate()))
	}
	return out
}

func (h *client) importCase(ctx cliapp.OperationContext) (*p.CaseResponse, error) {
	b, e := os.ReadFile(ctx.Flag("case"))
	if e != nil {
		return nil, fmt.Errorf("read case file: %w", e)
	}
	var raw any
	if e = json.Unmarshal(b, &raw); e != nil {
		return nil, fmt.Errorf("decode case JSON: %w", e)
	}
	normalizeEnums(raw)
	encoded, e := json.Marshal(raw)
	if e != nil {
		return nil, e
	}
	d := new(p.EvidenceCase)
	if e = protojson.Unmarshal(encoded, d); e != nil {
		return nil, fmt.Errorf("decode case JSON: %w", e)
	}
	r, e := h.rpc.ImportCase(context.Background(), connect.NewRequest(&p.ImportCaseRequest{Case: d}))
	if e != nil {
		return nil, cliapp.WrapAPIError("import case", e, nil)
	}
	return r.Msg, nil
}

// normalizeEnums accepts the canonical package Case JSON vocabulary while
// preserving the strongly typed proto enum wire contract.
func normalizeEnums(v any) {
	switch x := v.(type) {
	case map[string]any:
		for k, value := range x {
			if s, ok := value.(string); ok {
				switch k {
				case "chain":
					x[k] = "CHAIN_" + strings.ToUpper(s)
				case "kind":
					x[k] = "EVIDENCE_KIND_" + strings.ToUpper(s)
				case "grade":
					x[k] = "EVIDENCE_GRADE_" + strings.ToUpper(s)
				}
			}
			normalizeEnums(value)
		}
	case []any:
		for _, value := range x {
			normalizeEnums(value)
		}
	}
}

func (h *client) importReport(_ cliapp.OperationContext, r *p.CaseResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Imported case " + r.GetCase().GetId() + "."}, Changes: []string{fmt.Sprintf("%d sources, %d occurrences, %d edges", len(r.GetCase().GetSources()), len(r.GetCase().GetOccurrences()), len(r.GetCase().GetEdges()))}}
}

func (h *client) get(ctx cliapp.OperationContext) (*p.CaseResponse, error) {
	r, e := h.rpc.GetCase(context.Background(), connect.NewRequest(&p.GetCaseRequest{Id: ctx.Positional("id")}))
	if e != nil {
		return nil, cliapp.WrapAPIError("get case", e, nil)
	}
	return r.Msg, nil
}

func (h *client) getReport(_ cliapp.OperationContext, r *p.CaseResponse) cliapp.ListReport {
	b, _ := json.MarshalIndent(r.GetCase(), "", "  ")
	return cliapp.ListReport{Summary: []string{"Case " + r.GetCase().GetId()}, ResultsHeading: r.GetCase().GetTitle(), Results: []string{string(b)}}
}

func (h *client) packet(ctx cliapp.OperationContext) (*p.ExportPrivatePacketResponse, error) {
	r, e := h.rpc.ExportPrivatePacket(context.Background(), connect.NewRequest(&p.ExportPrivatePacketRequest{Id: ctx.Positional("id")}))
	if e != nil {
		return nil, cliapp.WrapAPIError("export private packet", e, nil)
	}
	return r.Msg, nil
}

func (h *client) packetReport(_ cliapp.OperationContext, r *p.ExportPrivatePacketResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{"Private packet generated locally; no transmission occurred."}, ResultsHeading: r.GetSchemaVersion(), Results: []string{string(r.GetPacketJson())}}
}

func (h *client) addLabel(ctx cliapp.OperationContext) (*p.AnalystLabelResponse, error) {
	b, e := os.ReadFile(ctx.Flag("label"))
	if e != nil {
		return nil, fmt.Errorf("read analyst label file: %w", e)
	}
	d := new(p.AnalystLabel)
	if e = protojson.Unmarshal(b, d); e != nil {
		return nil, fmt.Errorf("decode analyst label JSON: %w", e)
	}
	r, e := h.rpc.AddAnalystLabel(context.Background(), connect.NewRequest(&p.AddAnalystLabelRequest{Label: d}))
	if e != nil {
		return nil, cliapp.WrapAPIError("add analyst label", e, nil)
	}
	return r.Msg, nil
}

func (h *client) labelReport(_ cliapp.OperationContext, r *p.AnalystLabelResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Added analyst label " + r.GetLabel().GetId() + "."}, Changes: []string{r.GetLabel().GetLabel() + " (" + string(r.GetLabel().GetGrade().String()) + ")"}}
}

func (h *client) labels(ctx cliapp.OperationContext) (*p.ListAnalystLabelsResponse, error) {
	r, e := h.rpc.ListAnalystLabels(context.Background(), connect.NewRequest(&p.ListAnalystLabelsRequest{CaseId: ctx.Positional("case-id")}))
	if e != nil {
		return nil, cliapp.WrapAPIError("list analyst labels", e, nil)
	}
	return r.Msg, nil
}

func (h *client) labelsReport(_ cliapp.OperationContext, r *p.ListAnalystLabelsResponse) cliapp.ListReport {
	out := cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d analyst label(s).", len(r.GetLabels()))}, ResultsHeading: "Analyst labels", ListShaped: true, ResultCount: len(r.GetLabels())}
	for _, l := range r.GetLabels() {
		out.Results = append(out.Results, fmt.Sprintf("%s: %s (%s)", l.GetSubject(), l.GetLabel(), l.GetGrade()))
	}
	return out
}
