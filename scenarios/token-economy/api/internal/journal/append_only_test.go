package journal_test

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"token-economy/internal/journal"
)

// [REQ:TKE-P0-010] The repository contract and implementation expose no path
// that rewrites or deletes a journal event.
func TestJournalRepositoryIsStructurallyAppendOnly(t *testing.T) {
	repositoryType := reflect.TypeOf((*journal.Repository)(nil)).Elem()
	require.Equal(t, 3, repositoryType.NumMethod(), "event repository must expose append, read, and compensating reverse only")
	methodNames := make([]string, 0, repositoryType.NumMethod())
	for i := 0; i < repositoryType.NumMethod(); i++ {
		methodNames = append(methodNames, repositoryType.Method(i).Name)
		name := strings.ToLower(repositoryType.Method(i).Name)
		require.NotContains(t, name, "update")
		require.NotContains(t, name, "delete")
		require.NotContains(t, name, "remove")
	}
	require.ElementsMatch(t, []string{"Append", "Read", "Reverse"}, methodNames)

	workingDir, err := os.Getwd()
	require.NoError(t, err)
	sourcePath := filepath.Join(workingDir, "repository.go")
	if _, err := os.Stat(sourcePath); err != nil {
		sourcePath = filepath.Join(workingDir, "internal", "journal", "repository.go")
	}
	source, err := os.ReadFile(sourcePath)
	require.NoError(t, err)
	forbidden := regexp.MustCompile(`(?i)(update\s+journal_events|delete\s+from\s+journal_events)`)
	require.False(t, forbidden.Match(source), "journal event SQL must be insert/read only")
}
