package gates

import (
	"context"
	"path/filepath"
)

func ValidateReleasedVersionImmutable(scope Scope) (Result, error) {
	root := scope.Root
	if scope.DB != nil {
		return validateReleasedVersionImmutableWithDB(root, scope.DB)
	}
	dbPath := filepath.Join(root, "scenarios", "react-component-library", "data", "react-component-library.db")
	db, err := openGateDB(context.Background(), dbPath)
	if err != nil {
		return validateReleasedVersionHashLedger(root)
	}
	defer db.Close()
	return validateReleasedVersionImmutableWithDB(root, db)
}

// ValidateVersionMirrorIntegrity reports an evicted version whose durable
// mirror is empty. The finding is attributed to that version's owning asset;
// one corrupt row must not become a corpus-wide runner failure.
