package administration

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAdminUserCreditsReadAttributedMemberAccountLots(t *testing.T) {
	matcher := sqlmock.QueryMatcherFunc(func(expected, actual string) error {
		if expected != "account-credit-lots" {
			return sqlmock.QueryMatcherRegexp.Match(expected, actual)
		}
		for _, part := range []string{"FROM credit_funding_lots l", "business_account_members", "m.user_id = $1", "l.status = 'open'", "l.settlement_state = 'settled'", "l.livemode IS NOT NULL", "l.spendable_after", "l.expires_at"} {
			if !strings.Contains(actual, part) {
				return fmt.Errorf("credit lookup lacks %s", part)
			}
		}
		if strings.Contains(actual, "credit_wallets") {
			return fmt.Errorf("legacy wallet cannot supply user credits")
		}
		return nil
	})
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(matcher))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery("SELECT status, plan_tier FROM subscriptions").WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("account-credit-lots").WithArgs("user-one").WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(7))
	mock.ExpectQuery("SELECT COUNT.*FROM user_sessions").WithArgs("user-one").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	user := &UserAccountResponse{ID: "user-one", Email: "member@example.test"}
	NewUserManagementService(db).enrich(context.Background(), user)
	if user.Credits == nil || user.Credits.Balance != 7 || user.Credits.Bonus != 0 {
		t.Fatalf("account credit display=%+v", user.Credits)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
