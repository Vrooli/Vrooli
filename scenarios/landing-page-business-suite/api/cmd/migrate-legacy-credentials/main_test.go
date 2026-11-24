package main

import "testing"

func TestLocalMigrationCopyGuard(t *testing.T) {
	for _, tc := range []struct {
		dsn  string
		want bool
	}{
		{"postgres://user:pw@127.0.0.1:5432/lpbs_credential_migration_copy", true},
		{"postgres://user:pw@[::1]:5432/lpbs_credential_migration_copy", true},
		{"postgres://user:pw@remote.example/lpbs_credential_migration_copy", false},
		{"postgres://user:pw@localhost/lpbs", false},
		{"", false},
	} {
		if got := localMigrationCopy(tc.dsn); got != tc.want {
			t.Errorf("localMigrationCopy(%q)=%t want %t", tc.dsn, got, tc.want)
		}
	}
}
