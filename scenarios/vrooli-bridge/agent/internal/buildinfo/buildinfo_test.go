package buildinfo

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFingerprint_NonEmptyAndContainsVersion(t *testing.T) {
	fp := Fingerprint()
	require.NotEmpty(t, fp)
	require.True(t, strings.HasPrefix(fp, Version), "fingerprint leads with the version: %q", fp)
}

func TestShortCommit(t *testing.T) {
	require.Equal(t, "abc1234", shortCommit("abc1234def567"))
	require.Equal(t, "short", shortCommit("short"))
}
