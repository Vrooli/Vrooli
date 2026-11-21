package androidadb

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestApplicationStatusOnlyAcceptsBoundedMetadataRows(t *testing.T) {
	got := parseApplicationStatus("Row: 0 key=home.connected, value=true\nRow: 1 key=home.pause_reason, value=start-SecurityException\n")
	require.Equal(t, "true", got["home.connected"])
	require.Nil(t, parseApplicationStatus("Row: 0 key=home.connected, value=true\nRow: 1 key=home.connected, value=false\n"))
	require.Nil(t, parseApplicationStatus("Error: permission denied"))
	require.Nil(t, parseApplicationStatus("Row: 0 key=note, value=private text with spaces\n"))
	require.Nil(t, parseApplicationStatus("Row: 0 key=home.connected, value=true\nRow: 1 key=note, value=private text with spaces\n"))
	require.Nil(t, parseApplicationStatus("Row: 0 key=home.connected, value=true\nError: incomplete provider response\n"))
}
