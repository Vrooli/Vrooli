package androidadb

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProcessExitsBoundExactPackageAndExcludePrivateDescriptions(t *testing.T) {
	data := "ApplicationExitInfo #0:\n timestamp=2026-10-22 19:03:00\n process=com.example\n reason=3 (LOW_MEMORY)\n description=private contents\n trace=/private/path\nApplicationExitInfo #1:\n process=com.examples\n reason=4 (CRASH)\nApplicationExitInfo #2:\n process=com.example:worker\n reason=10 (USER_REQUESTED)\n"
	got := parseProcessExits(data, "com.example")
	require.Len(t, got, 2)
	require.Equal(t, "low-memory", got[0].Reason)
	require.Equal(t, "user-requested", got[1].Reason)
	require.Equal(t, "2026-10-22 19:03:00", got[0].Timestamp)
	require.Empty(t, parseProcessExits("unknown output", "com.example"))
}
