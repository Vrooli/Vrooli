package emaildelivery

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"testing"
)

type seedRecorder struct {
	args [][]any
}

func (r *seedRecorder) ExecContext(_ context.Context, _ string, args ...any) (sql.Result, error) {
	r.args = append(r.args, args)
	return driver.RowsAffected(1), nil
}

func TestSeedDefaultsCatalogsSurveyedProviders(t *testing.T) {
	recorder := &seedRecorder{}
	if err := SeedDefaults(context.Background(), recorder); err != nil {
		t.Fatalf("seed defaults: %v", err)
	}
	want := map[string]bool{
		"mailgun": true, "sendgrid": true, "resend": true, "brevo": true,
		"mailjet": true, "mailtrap": true, "smtp2go": true,
		"mailersend": true, "postmark": true, "ses": true,
	}
	if len(recorder.args) != len(want) {
		t.Fatalf("seeded %d providers, want %d", len(recorder.args), len(want))
	}
	for _, args := range recorder.args {
		id, ok := args[0].(string)
		if !ok || !want[id] {
			t.Fatalf("unexpected provider seed arguments: %#v", args)
		}
		delete(want, id)
	}
	if len(want) != 0 {
		t.Fatalf("missing provider catalog entries: %#v", want)
	}
}
