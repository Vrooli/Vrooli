package commerce

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
)

type (
	CreditHold struct {
		Reason  string `json:"reason"`
		Credits int64  `json:"credits"`
	}
	FeatureHistory struct {
		FeatureID string `json:"feature_id"`
		Credits   int64  `json:"credits"`
		Calls     int64  `json:"calls"`
	}
	AccountWalletView struct {
		Available int64            `json:"available"`
		OnHold    int64            `json:"on_hold"`
		Holds     []CreditHold     `json:"holds"`
		History   []FeatureHistory `json:"history"`
		LiveMode  bool             `json:"livemode"`
	}
)

// WalletView reads one account-locked snapshot. Funding lots are the balance;
// reservations reduce availability but never manufacture or double-count funds.
func (s *BusinessAccountUsageService) WalletView(ctx context.Context, account string) (*AccountWalletView, error) {
	tx, err := s.beginAccountTransaction(ctx, account)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	now := s.now()
	gross, err := spendableFundingBalance(ctx, tx, s.dialect, account, s.liveMode, now)
	if err != nil {
		return nil, err
	}
	view := &AccountWalletView{Available: gross, LiveMode: s.liveMode, Holds: []CreditHold{}, History: []FeatureHistory{}}
	rows, err := tx.QueryContext(ctx, `SELECT status,settlement_state,spendable_after,remaining FROM credit_funding_lots WHERE business_account_id=$1 AND livemode=$2 AND remaining>0 AND status IN ('open','frozen') AND settlement_state IN ('pending','settled') AND (expires_at IS NULL OR expires_at>$3)`, account, s.liveMode, now)
	if err != nil {
		return nil, err
	}
	holds := map[string]int64{}
	for rows.Next() {
		var status, settlement string
		var after sql.NullTime
		var credits int64
		if err := rows.Scan(&status, &settlement, &after, &credits); err != nil {
			rows.Close()
			return nil, err
		}
		switch {
		case status == "frozen":
			holds["Funding review"] += credits
		case settlement != "settled":
			holds["Payment settling"] += credits
		case after.Valid && after.Time.After(now):
			holds["Safety hold"] += credits
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	reserved, err := pendingFundingReservations(ctx, tx, s.dialect, account, s.liveMode, now)
	if err != nil {
		return nil, err
	}
	if reserved > gross {
		reserved = gross
	}
	if reserved > 0 {
		holds["Requests in progress"] = reserved
		view.Available -= reserved
	}
	for reason, credits := range holds {
		view.Holds = append(view.Holds, CreditHold{Reason: reason, Credits: credits})
		view.OnHold += credits
	}
	sort.Slice(view.Holds, func(i, j int) bool { return view.Holds[i].Reason < view.Holds[j].Reason })
	history, err := tx.QueryContext(ctx, `SELECT e.feature_id,SUM(e.credits),COUNT(*) FROM business_account_usage_events e WHERE e.business_account_id=$1 AND EXISTS (SELECT 1 FROM credit_lot_allocations a JOIN credit_funding_lots l ON l.id=a.lot_id WHERE a.operation_id=e.operation_id AND l.business_account_id=e.business_account_id AND l.livemode=$2) GROUP BY e.feature_id ORDER BY e.feature_id`, account, s.liveMode)
	if err != nil {
		return nil, fmt.Errorf("read account feature history: %w", err)
	}
	for history.Next() {
		var item FeatureHistory
		if err := history.Scan(&item.FeatureID, &item.Credits, &item.Calls); err != nil {
			history.Close()
			return nil, err
		}
		view.History = append(view.History, item)
	}
	err = history.Err()
	history.Close()
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return view, nil
}
