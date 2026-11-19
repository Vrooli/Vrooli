package cloudtarget

import (
	"testing"

	"github.com/vrooli/vrooli/internal/stateformat"
	"github.com/vrooli/vrooli/internal/stateformat/stateformattest"
)

func TestStateFormatDescribesDeploymentControlRecords(t *testing.T) {
	store, err := StateFormat()
	if err != nil {
		t.Fatal(err)
	}
	if err := stateformat.SortStores([]stateformat.Store{store}); err != nil {
		t.Fatalf("state format is invalid: %v", err)
	}
	if got := store.Schemas[store.Write]; got != cloudTargetSchemaDigest {
		t.Fatalf("schema digest = %s, want pinned %s", got, cloudTargetSchemaDigest)
	}
	for _, want := range []string{cloudTargetControlPattern, cloudTargetOperationPattern} {
		found := false
		for _, declared := range store.StatePaths {
			if declared == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("state paths %v do not declare %s", store.StatePaths, want)
		}
	}
	stateformattest.AssertVocabularyCoversConstants(t, stateformattest.Options{
		Dir:        ".",
		Vocabulary: store.Vocabulary,
		Allow: []string{
			"cloudTargetSchemaDigest",
			"cloudTargetControlPattern",
			"cloudTargetOperationPattern",
			"DefaultCaddyConfDir",
			"DefaultCaddyDataDir",
			"RevocationLimitation",
			"activationIntentFile",
			"activeReleaseFile",
			"caddyImportLine",
			"credentialsDirName",
			"deploymentsRelPath",
			"edgeActiveFile",
			"edgeDirName",
			"edgePreviousFile",
			"edgeSpecFile",
			"fenceFile",
			"incompleteMarker",
			"legacyCarryDirName",
			"operationsDirName",
			"persistentDataDirName",
			"recoveryPointsDirName",
			"releaseManifestFile",
			"releasesDirName",
			"snippetPrefix",
			"snippetSuffix",
			"stagingInfix",
		},
	})
}
