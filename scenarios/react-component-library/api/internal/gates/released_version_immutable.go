package gates

func ValidateReleasedVersionImmutable(scope Scope) (Result, error) {
	root := scope.Root
	if scope.DB != nil {
		return validateReleasedVersionImmutableWithDB(scope.Context, root, scope.DB)
	}
	return validateReleasedVersionHashLedger(root)
}

// ValidateVersionMirrorIntegrity reports an evicted version whose durable
// mirror is empty. The finding is attributed to that version's owning asset;
// one corrupt row must not become a corpus-wide runner failure.
