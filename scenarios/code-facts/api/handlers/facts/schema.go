package facts

import internalfacts "code-facts/internal/facts"

func Schema() string {
	return internalfacts.CacheSchema()
}
