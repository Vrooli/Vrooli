package spec

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParsedContractIdentityIncludesExtensionsAndDoesNotFollowLaterEdits(t *testing.T) {
	path := filepath.Join(t.TempDir(), "page.json")
	first := []byte(`{"kind":"experience-page","page":{"id":"home"},"x-policy":{"version":1}}`)
	require.NoError(t, os.WriteFile(path, first, 0600))
	report := &Report{}
	var page PageDocument
	require.True(t, decodeDoc(report, path, &page))
	require.Equal(t, fmt.Sprintf("%x", sha256.Sum256(first)), page.SourceHash)
	require.JSONEq(t, `{"version":1}`, string(page.Extensions["x-policy"]))
	second := []byte(`{"kind":"experience-page","page":{"id":"home"},"x-policy":{"version":2}}`)
	require.NoError(t, os.WriteFile(path, second, 0600))
	require.Equal(t, fmt.Sprintf("%x", sha256.Sum256(first)), page.SourceHash)
	var updated PageDocument
	require.True(t, decodeDoc(report, path, &updated))
	require.NotEqual(t, page.SourceHash, updated.SourceHash)
}

func TestConversationComponentContractsAreRegisteredAndValid(t *testing.T) {
	report, err := ParseScenario(filepath.Join(repoRoot(t), "scenarios", "react-component-library"))
	require.NoError(t, err)
	names := map[string]bool{"approval-prompt": true, "collection-page": true, "composer-attachment-tray": true, "message": true, "message-list": true, "prompt-composer": true}
	for name := range names {
		component, ok := report.Spec.Components[name]
		require.True(t, ok, "unregistered conversation component: %s", name)
		require.NotEmpty(t, component.SourceHash)
		for _, finding := range report.Findings {
			for _, location := range finding.Locations {
				if filepath.Base(location) == name+".json" && finding.Severity == SeverityError {
					t.Errorf("%s: %s: %s", name, finding.Code, finding.Message)
				}
			}
		}
	}
}
