package main

import (
	"context"
	"os"
	"strings"
)

type proposalEnvironmentKey struct{}

// Candidate subprocesses use their own Git storage and repository-local roots.
// Credentials and executable discovery keep their normal inherited contract.
func candidateEnvironment(source, candidate string) []string {
	result := []string{}
	for _, entry := range os.Environ() {
		key, value, _ := strings.Cut(entry, "=")
		switch key {
		case "GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_OBJECT_DIRECTORY", "GIT_ALTERNATE_OBJECT_DIRECTORIES", "GIT_COMMON_DIR":
			continue
		}
		if value == source || strings.HasPrefix(value, source+"/") {
			value = candidate + strings.TrimPrefix(value, source)
		}
		result = append(result, key+"="+value)
	}
	return result
}

func withCandidateEnvironment(ctx context.Context, source, candidate string) context.Context {
	return context.WithValue(ctx, proposalEnvironmentKey{}, candidateEnvironment(source, candidate))
}
