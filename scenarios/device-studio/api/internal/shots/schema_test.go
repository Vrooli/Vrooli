package shots_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"device-studio/internal/shots"
	"github.com/stretchr/testify/require"
)

func TestRoutedSchemaMatchesEmbeddedDomainSchema(t *testing.T) {
	path := filepath.Join("..", "..", "..", "bas", "seeds", "migrations", "sqlite", "001_shots.sql")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, strings.TrimSpace(shots.Schema()), strings.TrimSpace(string(data)))
}
