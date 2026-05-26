package phases

import (
	"strings"
	"testing"
)

func TestExtractTestDSN(t *testing.T) {
	cases := []struct {
		name          string
		env           map[string]string
		primaryDriver string
		wantDSN       string
		wantErrSubstr string
	}{
		{
			name:          "empty driver disqualifies",
			env:           map[string]string{"POSTGRES_URL": "postgres://x"},
			primaryDriver: "",
			wantErrSubstr: "db-detect did not pick a primary driver",
		},
		{
			name:          "postgres with POSTGRES_URL",
			env:           map[string]string{"POSTGRES_URL": "postgres://example/db"},
			primaryDriver: "postgres",
			wantDSN:       "postgres://example/db",
		},
		{
			name:          "sqlite with SQLITE_PATH",
			env:           map[string]string{"SQLITE_PATH": "/tmp/test.db"},
			primaryDriver: "sqlite",
			wantDSN:       "/tmp/test.db",
		},
		{
			name:          "postgres but only SQLITE_PATH present -> mismatch error",
			env:           map[string]string{"SQLITE_PATH": "/tmp/test.db"},
			primaryDriver: "postgres",
			wantErrSubstr: "primary driver is postgres but isolation env has no POSTGRES_URL",
		},
		{
			name:          "sqlite but only POSTGRES_URL present -> mismatch error",
			env:           map[string]string{"POSTGRES_URL": "postgres://x"},
			primaryDriver: "sqlite",
			wantErrSubstr: "primary driver is sqlite but isolation env has no SQLITE_PATH",
		},
		{
			name:          "postgres falls back to DATABASE_URL",
			env:           map[string]string{"DATABASE_URL": "postgres://fallback/db"},
			primaryDriver: "postgres",
			wantDSN:       "postgres://fallback/db",
		},
		{
			name:          "sqlite falls back to SQLITE_DB",
			env:           map[string]string{"SQLITE_DB": "/tmp/legacy.db"},
			primaryDriver: "sqlite",
			wantDSN:       "/tmp/legacy.db",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := extractTestDSN(tc.env, tc.primaryDriver)
			if tc.wantErrSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Fatalf("want error containing %q; got dsn=%q err=%v", tc.wantErrSubstr, got, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.wantDSN {
				t.Fatalf("dsn = %q; want %q", got, tc.wantDSN)
			}
		})
	}
}
