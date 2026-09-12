package journal_test

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
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

	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	source, err := os.ReadFile(filepath.Join(filepath.Dir(currentFile), "repository.go"))
	require.NoError(t, err)
	forbidden := regexp.MustCompile(`(?i)(update\s+journal_events|delete\s+from\s+journal_events)`)
	require.False(t, forbidden.Match(source), "journal event SQL must be insert/read only")
}
