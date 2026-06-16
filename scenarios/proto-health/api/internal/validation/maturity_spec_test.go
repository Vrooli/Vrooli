package validation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/maturity-go/assessment"
)

func TestMaturitySpecCoversProtoHealthFindings(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", ".vrooli", "maturity.json"))
	require.NoError(t, err)
	spec, err := assessment.ParseSpec(raw)
	require.NoError(t, err)

	require.Equal(t, "proto-health", spec.Provider)
	require.Equal(t, "proto", spec.Phase)
	for _, code := range allFindingCodes() {
		_, ok := spec.Findings[code]
		require.Truef(t, ok, "missing maturity mapping for %s", code)
	}
}

func allFindingCodes() []string {
	return []string{
		CodeCycle,
		CodeGenOutOfSync,
		CodePackageMismatch,
		CodeStabilityDishonest,
		CodeCrossDomainImport,
		CodeUnsupportedAnnotation,
		CodeTemplateSource,
		CodeHandRolledTransport,
		CodeVersionNaming,
		CodeDomainMismatch,
		CodeMissingHealthProto,
		CodePossiblyUnused,
		CodeRESTPayloadMissingDeclaration,
		CodeRESTPayloadUnknownMessage,
		CodeRESTPayloadInvalidConformance,
		CodeStabilityDependencyMismatch,
		CodeSharedTypeMisplaced,
		CodeImportKindUnknown,
		CodeCodeFactsUnavailable,
		CodeProtoAdoptionMissing,
		CodeProtoAdoptionUnsupported,
		CodeProtoAdoptionContradicted,
		CodeEndpointProofMissing,
		CodeEndpointProofUnsupported,
		CodeEndpointProofContradicted,
	}
}
