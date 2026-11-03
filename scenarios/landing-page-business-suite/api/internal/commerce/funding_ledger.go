package commerce

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	ingestconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest/ingest_v1connect"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// DispatchFundingLedger drains committed, mode-matched funding postings through
// Money Ledger's canonical typed ingestion edge. A failed or uncertain delivery
// leaves the immutable outbox row retryable under the same external identity.
func DispatchFundingLedger(ctx context.Context, db UsageStore) (int, error) {
	rows, err := db.QueryContext(ctx, `SELECT o.id,o.lot_id,o.category,o.amount_minor,o.currency,o.created_at,l.source_kind,s.endpoint,s.adapter_id,s.book_id,CASE o.category WHEN 'loss' THEN s.loss_account_id WHEN 'fee' THEN s.fee_account_id ELSE s.refund_account_id END FROM funding_ledger_outbox o JOIN credit_funding_lots l ON l.id=o.lot_id JOIN funding_ledger_sinks s ON s.livemode=l.livemode WHERE o.delivered_at IS NULL ORDER BY o.id LIMIT 100`)
	if err != nil {
		return 0, err
	}
	type posting struct {
		id, amount                                                        int64
		lot, category, currency, source, endpoint, adapter, book, account string
		at                                                                time.Time
	}
	var pending []posting
	for rows.Next() {
		var p posting
		if err := rows.Scan(&p.id, &p.lot, &p.category, &p.amount, &p.currency, &p.at, &p.source, &p.endpoint, &p.adapter, &p.book, &p.account); err != nil {
			rows.Close()
			return 0, err
		}
		pending = append(pending, p)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return 0, err
	}
	sent := 0
	client := &http.Client{Timeout: 10 * time.Second}
	for _, p := range pending {
		event := &sharedpb.MoneyEvent{ExternalId: fmt.Sprintf("lpbs-funding:%s:%d", p.lot, p.id), AdapterId: p.adapter, BookId: p.book, AccountId: p.account, AmountMinor: p.amount, Currency: p.currency, OccurredAt: timestamppb.New(p.at.UTC()), FetchedAt: timestamppb.New(p.at.UTC()), Basis: sharedpb.Basis_BASIS_DERIVED, Category: "funding." + p.source + "." + p.category, Description: "LPBS attributed funding reversal"}
		_, err := ingestconnect.NewIngestServiceClient(client, p.endpoint).IngestEvent(ctx, connect.NewRequest(&ingestpb.IngestEventRequest{Event: event}))
		if err != nil {
			return sent, err
		}
		if _, err = db.ExecContext(ctx, `UPDATE funding_ledger_outbox SET delivered_at=NOW() WHERE id=$1 AND delivered_at IS NULL`, p.id); err != nil {
			return sent, err
		}
		sent++
	}
	return sent, nil
}
