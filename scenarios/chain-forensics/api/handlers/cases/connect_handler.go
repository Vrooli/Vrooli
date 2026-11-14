package cases

import (
	"context"
	"fmt"
	"log"
	"strings"

	"chain-forensics/internal/cases"
	"connectrpc.com/connect"
	forensics "github.com/vrooli/chain-forensics"
	proto "github.com/vrooli/vrooli/packages/proto/gen/go/chain-forensics/v1/cases"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	repo           cases.Repository
	logger         *log.Logger
	readers        map[forensics.Chain]EVMReader
	bitcoin        BitcoinReader
	addressHistory BitcoinAddressReader
	wallets        map[forensics.Chain]EVMWalletReader
	solana         SolanaAddressReader
}

type EVMReader interface {
	FetchERC20Transfers(context.Context, forensics.Chain, string) (forensics.CapturedEVMTransaction, error)
}

type BitcoinReader interface {
	FetchTransaction(context.Context, string) (forensics.CapturedTransaction, error)
}

type BitcoinAddressReader interface {
	FetchAddressPage(context.Context, string, string) (forensics.CapturedBitcoinAddressPage, error)
}

type EVMWalletReader interface {
	FetchWalletSnapshot(context.Context, forensics.Chain, string, []string) (forensics.CapturedEVMWalletSnapshot, error)
}

type SolanaAddressReader interface {
	FetchAddressPage(context.Context, string, string) (forensics.CapturedSolanaPage, error)
}

func NewHandler(repo cases.Repository, logger *log.Logger, readers ...map[forensics.Chain]EVMReader) *Handler {
	if logger == nil {
		logger = log.Default()
	}
	h := &Handler{repo: repo, logger: logger, readers: make(map[forensics.Chain]EVMReader)}
	if len(readers) > 0 {
		h.readers = readers[0]
	}
	return h
}

func NewHandlerWithReaders(repo cases.Repository, logger *log.Logger, readers map[forensics.Chain]EVMReader, bitcoin BitcoinReader, addressHistory ...BitcoinAddressReader) *Handler {
	h := NewHandler(repo, logger, readers)
	h.bitcoin = bitcoin
	if len(addressHistory) > 0 {
		h.addressHistory = addressHistory[0]
	}
	return h
}

func (h *Handler) WithWalletReaders(wallets map[forensics.Chain]EVMWalletReader) *Handler {
	h.wallets = wallets
	return h
}

func (h *Handler) WithSolanaReader(reader SolanaAddressReader) *Handler { h.solana = reader; return h }

func (h *Handler) CaptureSolanaAddressPage(ctx context.Context, req *connect.Request[proto.CaptureSolanaAddressPageRequest]) (*connect.Response[proto.CaptureSolanaAddressPageResponse], error) {
	if h.solana == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("Solana public reader is unavailable"))
	}
	record, err := h.repo.Get(ctx, req.Msg.GetCaseId())
	if err != nil {
		return nil, mapError(err)
	}
	page, err := h.solana.FetchAddressPage(ctx, req.Msg.GetAddress(), req.Msg.GetBeforeSignature())
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	sources := []forensics.Source{page.Source}
	occurrences := make([]forensics.Occurrence, 0)
	for _, tx := range page.Transactions {
		sources = append(sources, tx.Source)
		occurrences = append(occurrences, tx.Occurrences...)
	}
	updated, err := h.repo.MergeCapture(ctx, record.Case.ID, sources, occurrences)
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&proto.CaptureSolanaAddressPageResponse{Case: toProto(updated), CapturedTransactionCount: uint32(len(page.Transactions)), CapturedOccurrenceCount: uint32(len(occurrences)), NextBeforeSignature: page.NextBefore}), nil
}

func (h *Handler) CaptureEVMWalletSnapshot(ctx context.Context, req *connect.Request[proto.CaptureEVMWalletSnapshotRequest]) (*connect.Response[proto.CaptureEVMWalletSnapshotResponse], error) {
	chain := chainFromProto(req.Msg.GetChain())
	reader := h.wallets[chain]
	if reader == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("only Ethereum and Base wallet snapshots are supported"))
	}
	record, err := h.repo.Get(ctx, req.Msg.GetCaseId())
	if err != nil {
		return nil, mapError(err)
	}
	captured, err := reader.FetchWalletSnapshot(ctx, chain, req.Msg.GetAddress(), req.Msg.GetTokenContracts())
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	updated, err := h.repo.MergeCapture(ctx, record.Case.ID, captured.Sources, nil, []forensics.EVMWalletSnapshot{captured.Snapshot})
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&proto.CaptureEVMWalletSnapshotResponse{Case: toProto(updated)}), nil
}

func (h *Handler) CaptureBitcoinTransaction(ctx context.Context, req *connect.Request[proto.CaptureBitcoinTransactionRequest]) (*connect.Response[proto.CaptureBitcoinTransactionResponse], error) {
	if h.bitcoin == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("Bitcoin public reader is unavailable"))
	}
	record, err := h.repo.Get(ctx, req.Msg.GetCaseId())
	if err != nil {
		return nil, mapError(err)
	}
	capture, err := h.bitcoin.FetchTransaction(ctx, req.Msg.GetTransactionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	occurrences, err := forensics.NormalizeBitcoinTransaction(capture.Source, capture.Transaction)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	updated, err := h.repo.MergeCapture(ctx, record.Case.ID, []forensics.Source{capture.Source}, occurrences)
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&proto.CaptureBitcoinTransactionResponse{Case: toProto(updated), CapturedOccurrenceCount: uint32(len(occurrences))}), nil
}

func (h *Handler) CaptureBitcoinAddressPage(ctx context.Context, req *connect.Request[proto.CaptureBitcoinAddressPageRequest]) (*connect.Response[proto.CaptureBitcoinAddressPageResponse], error) {
	if h.addressHistory == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("Bitcoin address-history reader is unavailable"))
	}
	record, err := h.repo.Get(ctx, req.Msg.GetCaseId())
	if err != nil {
		return nil, mapError(err)
	}
	page, err := h.addressHistory.FetchAddressPage(ctx, req.Msg.GetAddress(), req.Msg.GetAfterTxid())
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	occurrences := make([]forensics.Occurrence, 0)
	for _, tx := range page.Transactions {
		normalized, err := forensics.NormalizeBitcoinTransaction(page.Source, tx)
		if err != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		occurrences = append(occurrences, normalized...)
	}
	updated, err := h.repo.MergeCapture(ctx, record.Case.ID, []forensics.Source{page.Source}, occurrences)
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&proto.CaptureBitcoinAddressPageResponse{Case: toProto(updated), CapturedTransactionCount: uint32(len(page.Transactions)), CapturedOccurrenceCount: uint32(len(occurrences)), NextAfterTxid: page.NextAfterTxID}), nil
}

func (h *Handler) CaptureEVMTransaction(ctx context.Context, req *connect.Request[proto.CaptureEVMTransactionRequest]) (*connect.Response[proto.CaptureEVMTransactionResponse], error) {
	chain := chainFromProto(req.Msg.GetChain())
	reader := h.readers[chain]
	if reader == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("only Ethereum and Base public transaction receipts can be captured"))
	}
	record, err := h.repo.Get(ctx, req.Msg.GetCaseId())
	if err != nil {
		return nil, mapError(err)
	}
	capture, err := reader.FetchERC20Transfers(ctx, chain, req.Msg.GetTransactionHash())
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	updated, err := h.repo.MergeCapture(ctx, record.Case.ID, capture.Sources, capture.Occurrences)
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&proto.CaptureEVMTransactionResponse{Case: toProto(updated), CapturedOccurrenceCount: uint32(len(capture.Occurrences))}), nil
}

func (h *Handler) ListCases(ctx context.Context, req *connect.Request[proto.ListCasesRequest]) (*connect.Response[proto.ListCasesResponse], error) {
	limit := int(req.Msg.GetLimit())
	if limit == 0 {
		limit = 100
	}
	if limit < 1 || limit > 500 {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	records, err := h.repo.List(ctx, limit)
	if err != nil {
		return nil, h.internal("ListCases", err)
	}
	out := &proto.ListCasesResponse{Cases: make([]*proto.CaseSummary, 0, len(records))}
	for _, r := range records {
		out.Cases = append(out.Cases, &proto.CaseSummary{Id: r.Case.ID, Title: r.Case.Title, Private: r.Case.Private, SourceCount: uint32(len(r.Case.Sources)), OccurrenceCount: uint32(len(r.Case.Occurrences)), EdgeCount: uint32(len(r.Case.Edges)), UpdatedAt: timestamppb.New(r.UpdatedAt)})
	}
	return connect.NewResponse(out), nil
}

func (h *Handler) ImportCase(ctx context.Context, req *connect.Request[proto.ImportCaseRequest]) (*connect.Response[proto.CaseResponse], error) {
	if req.Msg.GetCase() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	doc, err := fromProto(req.Msg.GetCase())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	doc = forensics.MergeBridgeCorrelationEdges(doc)
	rec, err := h.repo.Import(ctx, doc)
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&proto.CaseResponse{Case: toProto(rec)}), nil
}

func (h *Handler) GetCase(ctx context.Context, req *connect.Request[proto.GetCaseRequest]) (*connect.Response[proto.CaseResponse], error) {
	rec, err := h.repo.Get(ctx, strings.TrimSpace(req.Msg.GetId()))
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&proto.CaseResponse{Case: toProto(rec)}), nil
}

func (h *Handler) ExportPrivatePacket(ctx context.Context, req *connect.Request[proto.ExportPrivatePacketRequest]) (*connect.Response[proto.ExportPrivatePacketResponse], error) {
	rec, err := h.repo.Get(ctx, req.Msg.GetId())
	if err != nil {
		return nil, mapError(err)
	}
	packet, err := forensics.BuildPrivatePacket(rec.Case)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	saved, err := h.repo.SavePacket(ctx, packet)
	if err != nil {
		return nil, h.internal("ExportPrivatePacket", err)
	}
	return connect.NewResponse(&proto.ExportPrivatePacketResponse{SchemaVersion: saved.SchemaVersion, PacketJson: saved.JSON, Sha256: saved.SHA256}), nil
}

func (h *Handler) AddAnalystLabel(ctx context.Context, req *connect.Request[proto.AddAnalystLabelRequest]) (*connect.Response[proto.AnalystLabelResponse], error) {
	p := req.Msg.GetLabel()
	if p == nil || strings.TrimSpace(p.GetCaseId()) == "" || strings.TrimSpace(p.GetSubject()) == "" || strings.TrimSpace(p.GetLabel()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	if _, err := h.repo.Get(ctx, p.GetCaseId()); err != nil {
		return nil, mapError(err)
	}
	l := cases.AnalystLabel{ID: p.GetId(), CaseID: p.GetCaseId(), Subject: p.GetSubject(), Label: p.GetLabel(), SourceURL: p.GetSourceUrl(), SourceCitation: p.GetSourceCitation(), Grade: gradeFromProto(p.GetGrade()), CounterExplanation: p.GetCounterExplanation(), UnresolvedGap: p.GetUnresolvedGap()}
	if l.Grade == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	saved, err := h.repo.AddLabel(ctx, l)
	if err != nil {
		return nil, h.internal("AddAnalystLabel", err)
	}
	return connect.NewResponse(&proto.AnalystLabelResponse{Label: labelToProto(saved)}), nil
}

func (h *Handler) ListAnalystLabels(ctx context.Context, req *connect.Request[proto.ListAnalystLabelsRequest]) (*connect.Response[proto.ListAnalystLabelsResponse], error) {
	if _, err := h.repo.Get(ctx, req.Msg.GetCaseId()); err != nil {
		return nil, mapError(err)
	}
	rows, err := h.repo.ListLabels(ctx, req.Msg.GetCaseId())
	if err != nil {
		return nil, h.internal("ListAnalystLabels", err)
	}
	out := &proto.ListAnalystLabelsResponse{Labels: make([]*proto.AnalystLabel, 0, len(rows))}
	for _, l := range rows {
		out.Labels = append(out.Labels, labelToProto(l))
	}
	return connect.NewResponse(out), nil
}

func (h *Handler) ReviseCase(ctx context.Context, req *connect.Request[proto.ReviseCaseRequest]) (*connect.Response[proto.ReviseCaseResponse], error) {
	m := req.Msg
	if m.GetExpectedUpdatedAt() == nil || m.GetExpectedUpdatedAt().CheckValid() != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("expected_updated_at must be a valid case version timestamp"))
	}
	additions := cases.RevisionAdditions{Summary: m.GetSummary(), ResolvedGaps: append([]string(nil), m.GetResolvedGaps()...)}
	for _, s := range m.GetSources() {
		additions.Sources = append(additions.Sources, forensics.Source{ID: s.GetId(), Name: s.GetName(), URL: s.GetUrl(), Retrieved: s.GetRetrievedAt(), SHA256: s.GetSha256(), Citation: s.GetCitation(), Limitations: s.GetLimitations(), ResultBody: append([]byte(nil), s.GetResultBody()...), RawResponseBody: append([]byte(nil), s.GetRawResponseBody()...), RawResponseSHA256: s.GetRawResponseSha256()})
	}
	for _, c := range m.GetClaims() {
		additions.Claims = append(additions.Claims, forensics.Claim{ID: c.GetId(), Text: c.GetText(), SourceID: c.GetSourceId(), Kind: kindFromProto(c.GetKind())})
	}
	for _, o := range m.GetOccurrences() {
		additions.Occurrences = append(additions.Occurrences, forensics.Occurrence{ID: o.GetId(), Chain: chainFromProto(o.GetChain()), Transaction: o.GetTransactionId(), Block: o.GetBlockHeight(), Timestamp: o.GetTimestamp(), EventIndex: o.GetEventIndex(), Direction: o.GetDirection(), PreviousTx: o.GetPreviousTransactionId(), PreviousIndex: o.GetPreviousOutputIndex(), From: o.GetFrom(), To: o.GetTo(), Asset: o.GetAsset(), Atomic: o.GetAtomicQuantity(), Scale: o.GetScale(), Kind: kindFromProto(o.GetKind()), SourceID: o.GetSourceId()})
	}
	for _, e := range m.GetEdges() {
		additions.Edges = append(additions.Edges, forensics.Edge{ID: e.GetId(), From: e.GetFromOccurrence(), To: e.GetToOccurrence(), Grade: gradeFromProto(e.GetGrade()), Rule: e.GetRule(), EvidenceIDs: append([]string(nil), e.GetEvidenceIds()...), Caveats: append([]string(nil), e.GetCaveats()...)})
	}
	for _, e := range m.GetEdgeRevisions() {
		replacement := e.GetReplacement()
		if replacement == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("each edge revision must include a replacement edge"))
		}
		additions.EdgeRevisions = append(additions.EdgeRevisions, cases.EdgeRevision{SupersedesEdgeID: e.GetSupersedesEdgeId(), Replacement: forensics.Edge{ID: replacement.GetId(), From: replacement.GetFromOccurrence(), To: replacement.GetToOccurrence(), Grade: gradeFromProto(replacement.GetGrade()), Rule: replacement.GetRule(), EvidenceIDs: append([]string(nil), replacement.GetEvidenceIds()...), Caveats: append([]string(nil), replacement.GetCaveats()...)}})
	}
	additions.Gaps = append([]string(nil), m.GetGaps()...)
	additions.Counterexamples = append([]string(nil), m.GetCounterexamples()...)
	additions.Disagreements = append([]string(nil), m.GetSourceDisagreements()...)
	updated, revision, err := h.repo.Revise(ctx, m.GetCaseId(), m.GetExpectedUpdatedAt().AsTime(), additions)
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&proto.ReviseCaseResponse{RevisionId: revision.ID, Case: toProto(updated), PreviousUpdatedAt: timestamppb.New(revision.PreviousUpdatedAt)}), nil
}

func (h *Handler) ListCaseRevisions(ctx context.Context, req *connect.Request[proto.ListCaseRevisionsRequest]) (*connect.Response[proto.ListCaseRevisionsResponse], error) {
	revisions, err := h.repo.ListRevisions(ctx, req.Msg.GetCaseId())
	if err != nil {
		return nil, mapError(err)
	}
	out := &proto.ListCaseRevisionsResponse{Revisions: make([]*proto.CaseRevision, 0, len(revisions))}
	for _, revision := range revisions {
		adds := &proto.ReviseCaseRequest{CaseId: revision.CaseID, ExpectedUpdatedAt: timestamppb.New(revision.PreviousUpdatedAt), Summary: revision.Additions.Summary, Gaps: revision.Additions.Gaps, Counterexamples: revision.Additions.Counterexamples, SourceDisagreements: revision.Additions.Disagreements, ResolvedGaps: revision.Additions.ResolvedGaps}
		for _, s := range revision.Additions.Sources {
			adds.Sources = append(adds.Sources, &proto.Source{Id: s.ID, Name: s.Name, Url: s.URL, RetrievedAt: s.Retrieved, Sha256: s.SHA256, Citation: s.Citation, Limitations: s.Limitations, ResultBody: s.ResultBody, RawResponseBody: s.RawResponseBody, RawResponseSha256: s.RawResponseSHA256})
		}
		for _, c := range revision.Additions.Claims {
			adds.Claims = append(adds.Claims, &proto.Claim{Id: c.ID, Text: c.Text, SourceId: c.SourceID, Kind: kindToProto(c.Kind)})
		}
		for _, o := range revision.Additions.Occurrences {
			adds.Occurrences = append(adds.Occurrences, &proto.Occurrence{Id: o.ID, Chain: chainToProto(o.Chain), TransactionId: o.Transaction, BlockHeight: o.Block, Timestamp: o.Timestamp, EventIndex: o.EventIndex, Direction: o.Direction, PreviousTransactionId: o.PreviousTx, PreviousOutputIndex: o.PreviousIndex, From: o.From, To: o.To, Asset: o.Asset, AtomicQuantity: o.Atomic, Scale: o.Scale, Kind: kindToProto(o.Kind), SourceId: o.SourceID})
		}
		for _, e := range revision.Additions.Edges {
			adds.Edges = append(adds.Edges, &proto.FlowEdge{Id: e.ID, FromOccurrence: e.From, ToOccurrence: e.To, Grade: gradeToProto(e.Grade), Rule: e.Rule, EvidenceIds: e.EvidenceIDs, Caveats: e.Caveats})
		}
		for _, e := range revision.Additions.EdgeRevisions {
			replacement := e.Replacement
			adds.EdgeRevisions = append(adds.EdgeRevisions, &proto.EdgeRevision{SupersedesEdgeId: e.SupersedesEdgeID, Replacement: &proto.FlowEdge{Id: replacement.ID, FromOccurrence: replacement.From, ToOccurrence: replacement.To, Grade: gradeToProto(replacement.Grade), Rule: replacement.Rule, EvidenceIds: replacement.EvidenceIDs, Caveats: replacement.Caveats}})
		}
		item := &proto.CaseRevision{Id: revision.ID, CaseId: revision.CaseID, PreviousUpdatedAt: timestamppb.New(revision.PreviousUpdatedAt), AppliedAt: timestamppb.New(revision.AppliedAt), Additions: adds}
		if revision.PreviousCase.ID != "" {
			item.PreviousCase = toProto(cases.Record{Case: revision.PreviousCase, CreatedAt: revision.PreviousCreatedAt, UpdatedAt: revision.PreviousUpdatedAt})
		}
		out.Revisions = append(out.Revisions, item)
	}
	return connect.NewResponse(out), nil
}

func (h *Handler) internal(op string, err error) error {
	h.logger.Printf("cases.%s: %v", op, err)
	return connect.NewError(connect.CodeInternal, nil)
}

func mapError(err error) error {
	switch err.(type) {
	case cases.ErrNotFound:
		return connect.NewError(connect.CodeNotFound, err)
	case cases.ErrConflict:
		return connect.NewError(connect.CodeAlreadyExists, err)
	case cases.ErrStaleRevision:
		return connect.NewError(connect.CodeAborted, err)
	case cases.ErrInvalid:
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

func gradeFromProto(g proto.EvidenceGrade) forensics.Grade {
	switch g {
	case proto.EvidenceGrade_EVIDENCE_GRADE_OBSERVED:
		return forensics.GradeObserved
	case proto.EvidenceGrade_EVIDENCE_GRADE_PROTOCOL_CORRELATED:
		return forensics.GradeProtocolCorrelated
	case proto.EvidenceGrade_EVIDENCE_GRADE_SOURCE_CORROBORATED:
		return forensics.GradeSourceCorroborated
	case proto.EvidenceGrade_EVIDENCE_GRADE_HYPOTHESIS:
		return forensics.GradeHypothesis
	case proto.EvidenceGrade_EVIDENCE_GRADE_UNRESOLVED:
		return forensics.GradeUnresolved
	}
	return ""
}

func toProto(r cases.Record) *proto.EvidenceCase {
	d := r.Case
	p := &proto.EvidenceCase{Id: d.ID, Title: d.Title, Summary: d.Summary, Private: d.Private, Gaps: d.Gaps, Counterexamples: d.Counterexamples, SourceDisagreements: d.Disagreements, CreatedAt: timestamppb.New(r.CreatedAt), UpdatedAt: timestamppb.New(r.UpdatedAt)}
	for _, s := range d.Sources {
		p.Sources = append(p.Sources, &proto.Source{Id: s.ID, Name: s.Name, Url: s.URL, RetrievedAt: s.Retrieved, Sha256: s.SHA256, Citation: s.Citation, Limitations: s.Limitations, ResultBody: s.ResultBody, RawResponseBody: s.RawResponseBody, RawResponseSha256: s.RawResponseSHA256})
	}
	for _, c := range d.Claims {
		p.Claims = append(p.Claims, &proto.Claim{Id: c.ID, Text: c.Text, SourceId: c.SourceID, Kind: kindToProto(c.Kind)})
	}
	for _, o := range d.Occurrences {
		p.Occurrences = append(p.Occurrences, &proto.Occurrence{Id: o.ID, Chain: chainToProto(o.Chain), TransactionId: o.Transaction, BlockHeight: o.Block, Timestamp: o.Timestamp, EventIndex: o.EventIndex, Direction: o.Direction, PreviousTransactionId: o.PreviousTx, PreviousOutputIndex: o.PreviousIndex, From: o.From, To: o.To, Asset: o.Asset, AtomicQuantity: o.Atomic, Scale: o.Scale, Kind: kindToProto(o.Kind), SourceId: o.SourceID})
	}
	for _, e := range d.Edges {
		p.Edges = append(p.Edges, &proto.FlowEdge{Id: e.ID, FromOccurrence: e.From, ToOccurrence: e.To, Grade: gradeToProto(e.Grade), Rule: e.Rule, EvidenceIds: e.EvidenceIDs, Caveats: e.Caveats})
	}
	for _, snapshot := range d.WalletSnapshots {
		ps := &proto.EVMWalletSnapshot{Id: snapshot.ID, Chain: chainToProto(snapshot.Chain), Address: snapshot.Address, BlockHeight: snapshot.Block, BlockHash: snapshot.BlockHash, Timestamp: snapshot.Timestamp, NativeAsset: snapshot.NativeAsset, NativeAtomicQuantity: snapshot.NativeAtomic, NativeScale: snapshot.NativeScale, SourceIds: snapshot.SourceIDs}
		for _, token := range snapshot.Tokens {
			ps.Tokens = append(ps.Tokens, &proto.EVMTokenBalance{Contract: token.Contract, AtomicQuantity: token.Atomic, Scale: token.Scale, ScaleKnown: token.ScaleKnown})
		}
		p.WalletSnapshots = append(p.WalletSnapshots, ps)
	}
	return p
}

func fromProto(p *proto.EvidenceCase) (forensics.Case, error) {
	d := forensics.Case{ID: p.GetId(), Title: p.GetTitle(), Summary: p.GetSummary(), Private: p.GetPrivate(), Gaps: p.GetGaps(), Counterexamples: p.GetCounterexamples(), Disagreements: p.GetSourceDisagreements()}
	for _, s := range p.GetSources() {
		d.Sources = append(d.Sources, forensics.Source{ID: s.GetId(), Name: s.GetName(), URL: s.GetUrl(), Retrieved: s.GetRetrievedAt(), SHA256: s.GetSha256(), Citation: s.GetCitation(), Limitations: s.GetLimitations(), ResultBody: append([]byte(nil), s.GetResultBody()...), RawResponseBody: append([]byte(nil), s.GetRawResponseBody()...), RawResponseSHA256: s.GetRawResponseSha256()})
	}
	for _, c := range p.GetClaims() {
		d.Claims = append(d.Claims, forensics.Claim{ID: c.GetId(), Text: c.GetText(), SourceID: c.GetSourceId(), Kind: kindFromProto(c.GetKind())})
	}
	for _, o := range p.GetOccurrences() {
		d.Occurrences = append(d.Occurrences, forensics.Occurrence{ID: o.GetId(), Chain: chainFromProto(o.GetChain()), Transaction: o.GetTransactionId(), Block: o.GetBlockHeight(), Timestamp: o.GetTimestamp(), EventIndex: o.GetEventIndex(), Direction: o.GetDirection(), PreviousTx: o.GetPreviousTransactionId(), PreviousIndex: o.GetPreviousOutputIndex(), From: o.GetFrom(), To: o.GetTo(), Asset: o.GetAsset(), Atomic: o.GetAtomicQuantity(), Scale: o.GetScale(), Kind: kindFromProto(o.GetKind()), SourceID: o.GetSourceId()})
	}
	for _, e := range p.GetEdges() {
		d.Edges = append(d.Edges, forensics.Edge{ID: e.GetId(), From: e.GetFromOccurrence(), To: e.GetToOccurrence(), Grade: gradeFromProto(e.GetGrade()), Rule: e.GetRule(), EvidenceIDs: e.GetEvidenceIds(), Caveats: e.GetCaveats()})
	}
	for _, snapshot := range p.GetWalletSnapshots() {
		ws := forensics.EVMWalletSnapshot{ID: snapshot.GetId(), Chain: chainFromProto(snapshot.GetChain()), Address: snapshot.GetAddress(), Block: snapshot.GetBlockHeight(), BlockHash: snapshot.GetBlockHash(), Timestamp: snapshot.GetTimestamp(), NativeAsset: snapshot.GetNativeAsset(), NativeAtomic: snapshot.GetNativeAtomicQuantity(), NativeScale: snapshot.GetNativeScale(), SourceIDs: snapshot.GetSourceIds()}
		for _, token := range snapshot.GetTokens() {
			ws.Tokens = append(ws.Tokens, forensics.WalletTokenBalance{Contract: token.GetContract(), Atomic: token.GetAtomicQuantity(), Scale: token.GetScale(), ScaleKnown: token.GetScaleKnown()})
		}
		d.WalletSnapshots = append(d.WalletSnapshots, ws)
	}
	if err := forensics.Validate(d); err != nil {
		return forensics.Case{}, err
	}
	return d, nil
}

func chainToProto(c forensics.Chain) proto.Chain {
	switch c {
	case forensics.ChainBitcoin:
		return proto.Chain_CHAIN_BITCOIN
	case forensics.ChainEthereum:
		return proto.Chain_CHAIN_ETHEREUM
	case forensics.ChainBase:
		return proto.Chain_CHAIN_BASE
	case forensics.ChainArbitrum:
		return proto.Chain_CHAIN_ARBITRUM
	case forensics.ChainSolana:
		return proto.Chain_CHAIN_SOLANA
	}
	return proto.Chain_CHAIN_UNSPECIFIED
}

func chainFromProto(c proto.Chain) forensics.Chain {
	switch c {
	case proto.Chain_CHAIN_BITCOIN:
		return forensics.ChainBitcoin
	case proto.Chain_CHAIN_ETHEREUM:
		return forensics.ChainEthereum
	case proto.Chain_CHAIN_BASE:
		return forensics.ChainBase
	case proto.Chain_CHAIN_ARBITRUM:
		return forensics.ChainArbitrum
	case proto.Chain_CHAIN_SOLANA:
		return forensics.ChainSolana
	}
	return ""
}

func kindToProto(k forensics.EvidenceKind) proto.EvidenceKind {
	switch k {
	case forensics.EvidenceObservation:
		return proto.EvidenceKind_EVIDENCE_KIND_OBSERVATION
	case forensics.EvidenceLegalClaim:
		return proto.EvidenceKind_EVIDENCE_KIND_LEGAL_CLAIM
	case forensics.EvidenceLabel:
		return proto.EvidenceKind_EVIDENCE_KIND_PUBLIC_LABEL
	case forensics.EvidenceDerivation:
		return proto.EvidenceKind_EVIDENCE_KIND_DERIVATION
	}
	return proto.EvidenceKind_EVIDENCE_KIND_UNSPECIFIED
}

func kindFromProto(k proto.EvidenceKind) forensics.EvidenceKind {
	switch k {
	case proto.EvidenceKind_EVIDENCE_KIND_OBSERVATION:
		return forensics.EvidenceObservation
	case proto.EvidenceKind_EVIDENCE_KIND_LEGAL_CLAIM:
		return forensics.EvidenceLegalClaim
	case proto.EvidenceKind_EVIDENCE_KIND_PUBLIC_LABEL:
		return forensics.EvidenceLabel
	case proto.EvidenceKind_EVIDENCE_KIND_DERIVATION:
		return forensics.EvidenceDerivation
	}
	return ""
}

func gradeToProto(g forensics.Grade) proto.EvidenceGrade {
	switch g {
	case forensics.GradeObserved:
		return proto.EvidenceGrade_EVIDENCE_GRADE_OBSERVED
	case forensics.GradeProtocolCorrelated:
		return proto.EvidenceGrade_EVIDENCE_GRADE_PROTOCOL_CORRELATED
	case forensics.GradeSourceCorroborated:
		return proto.EvidenceGrade_EVIDENCE_GRADE_SOURCE_CORROBORATED
	case forensics.GradeHypothesis:
		return proto.EvidenceGrade_EVIDENCE_GRADE_HYPOTHESIS
	case forensics.GradeUnresolved:
		return proto.EvidenceGrade_EVIDENCE_GRADE_UNRESOLVED
	}
	return proto.EvidenceGrade_EVIDENCE_GRADE_UNSPECIFIED
}

func labelToProto(l cases.AnalystLabel) *proto.AnalystLabel {
	return &proto.AnalystLabel{Id: l.ID, CaseId: l.CaseID, Subject: l.Subject, Label: l.Label, SourceUrl: l.SourceURL, SourceCitation: l.SourceCitation, Grade: gradeToProto(l.Grade), CounterExplanation: l.CounterExplanation, UnresolvedGap: l.UnresolvedGap, CreatedAt: timestamppb.New(l.CreatedAt)}
}
