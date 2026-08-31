package gates

func ValidateTypes(scope Scope) (Result, error) {
	root := scope.Root
	return validateTypes(root, scope.Assets)
}

// validateTypes runs catalog conformance over a subset of the library.
//
// assets names the library directories whose content changed, plus everything
// that depends on them — the caller derives that closure from the generated
// per-version locks. A nil or empty slice runs the full corpus, which is what
// a cold pass and every direct caller do.
//
// The conformance script decides separately whether the catalog app has to be
// recompiled, from the app's own transitive dependency closure. That is the
// expensive half: a full pass measured 24.6s, a scoped pass touching something
// the app depends on 21.7s, and a scoped pass touching something it does not
// 1.0s. 147 of 238 assets fall in that last case.
