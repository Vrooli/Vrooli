package eval

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/provenance"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	journalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal"
	journalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal/journal_v1connect"

	internaleval "search-hub/internal/eval"
)

const denominatorReviewScope = "local-model-performance-denominator-review"
const denominatorReviewKind = "lm2-independent-denominator-review-v1"

type journalEntryReader interface {
	GetEntry(context.Context, *connect.Request[journalv1.GetEntryRequest]) (*connect.Response[journalv1.GetEntryResponse], error)
}

type independentDenominatorReviewVerifier struct {
	reader journalEntryReader
}

func newIndependentDenominatorReviewVerifier() *independentDenominatorReviewVerifier {
	return &independentDenominatorReviewVerifier{reader: &discoveredJournalEntryReader{
		resolver: discovery.NewResolver(discovery.ResolverConfig{}),
		http:     &http.Client{Timeout: 10 * time.Second, Transport: provenance.ForwardingTransport{}},
	}}
}

func (v *independentDenominatorReviewVerifier) VerifyIndependentDenominatorReview(ctx context.Context, suite *evalv1.EvalSuite, reviewerID, evidenceURI string) error {
	if v == nil || v.reader == nil {
		return fmt.Errorf("Source Ledger review receipt reader is unavailable")
	}
	entryID, digest, err := parseDenominatorReviewURI(evidenceURI)
	if err != nil {
		return err
	}
	response, err := v.reader.GetEntry(ctx, connect.NewRequest(&journalv1.GetEntryRequest{Id: entryID, Scope: denominatorReviewScope}))
	if err != nil {
		return fmt.Errorf("load immutable Source Ledger review receipt: %w", err)
	}
	if response == nil || response.Msg == nil || response.Msg.GetEntry() == nil {
		return fmt.Errorf("Source Ledger did not return the referenced review receipt")
	}
	entry := response.Msg.GetEntry()
	if entry.GetId() != entryID || entry.GetKind() != denominatorReviewKind || entry.GetCreatedAt() == nil {
		return fmt.Errorf("Source Ledger entry is not a valid LM2 independent review receipt")
	}
	if err := entry.GetCreatedAt().CheckValid(); err != nil {
		return fmt.Errorf("Source Ledger review receipt timestamp is invalid")
	}
	body := []byte(entry.GetBody())
	bodyDigest := sha256.Sum256(body)
	if hex.EncodeToString(bodyDigest[:]) != digest {
		return fmt.Errorf("Source Ledger receipt digest does not match evidence URI")
	}
	attribution := entry.GetAttribution()
	if attribution == nil {
		return fmt.Errorf("Source Ledger review receipt has no verified actor attribution")
	}
	return internaleval.ValidateIndependentReviewReceipt(
		suite,
		reviewerID,
		attribution.GetActorId(),
		attribution.GetVerificationStatus(),
		entry.GetCreatedAt().AsTime().UTC().Format(time.RFC3339Nano),
		body,
	)
}

func parseDenominatorReviewURI(raw string) (entryID, digest string, err error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u == nil || u.Scheme != "source-ledger" || u.Host != "entries" || u.User != nil || u.Fragment != "" {
		return "", "", fmt.Errorf("review_evidence_uri must reference an immutable Source Ledger entry")
	}
	entryID = strings.TrimPrefix(u.Path, "/")
	if entryID == "" || strings.Contains(entryID, "/") || u.Path != "/"+entryID {
		return "", "", fmt.Errorf("review_evidence_uri must contain one Source Ledger entry id")
	}
	query := u.Query()
	if len(query) != 2 || len(query["scope"]) != 1 || query.Get("scope") != denominatorReviewScope || len(query["sha256"]) != 1 {
		return "", "", fmt.Errorf("review_evidence_uri must include the required scope and receipt digest")
	}
	digest = strings.ToLower(query.Get("sha256"))
	if len(digest) != 64 {
		return "", "", fmt.Errorf("review_evidence_uri receipt digest must be SHA-256")
	}
	if _, err := hex.DecodeString(digest); err != nil {
		return "", "", fmt.Errorf("review_evidence_uri receipt digest must be SHA-256")
	}
	return entryID, digest, nil
}

type discoveredJournalEntryReader struct {
	resolver *discovery.Resolver
	http     *http.Client
}

func (r *discoveredJournalEntryReader) GetEntry(ctx context.Context, req *connect.Request[journalv1.GetEntryRequest]) (*connect.Response[journalv1.GetEntryResponse], error) {
	if r == nil || r.resolver == nil || r.http == nil {
		return nil, fmt.Errorf("Source Ledger resolver is unavailable")
	}
	base, err := r.resolver.ResolveScenarioURLDefault(ctx, "source-ledger")
	if err != nil {
		return nil, err
	}
	client := journalconnect.NewJournalServiceClient(r.http, base)
	return client.GetEntry(ctx, req)
}
