package eval

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
)

func TestEvaluationIdentityEvidenceSeparatesProviderDirectAndSearchHubWorkload(t *testing.T) {
	providerDirect, searchHubWorkload := evaluationIdentityEvidence(nil, "sha256:search-hub-source")
	require.Nil(t, providerDirect.Caller, "provider-direct evaluation does not execute Search Hub's task behavior")
	require.NotNil(t, searchHubWorkload.Caller)

	identity, err := searchHubWorkload.Caller.ResolveCallerIdentity(context.Background(), &evalv1.EvalSuite{SuiteId: "provider-owned-suite", ProviderId: "provider.corpus"})
	require.NoError(t, err)
	require.Equal(t, "search-hub", identity.ID)
	require.Equal(t, "sha256:search-hub-source", identity.Version)
	require.NotEqual(t, "provider.corpus", identity.ID, "corpus ownership must not become workload identity")

	nilSuite, err := searchHubWorkload.Caller.ResolveCallerIdentity(context.Background(), nil)
	require.NoError(t, err)
	require.Empty(t, nilSuite.ID)
	require.Empty(t, nilSuite.Version)

	unknownVersionDirect, unknownVersionWorkload := evaluationIdentityEvidence(nil, "")
	require.Nil(t, unknownVersionDirect.Caller)
	unknownVersion, err := unknownVersionWorkload.Caller.ResolveCallerIdentity(context.Background(), &evalv1.EvalSuite{SuiteId: "provider-owned-suite"})
	require.NoError(t, err)
	require.Empty(t, unknownVersion.ID, "missing lifecycle build identity must leave the caller wholly unknown")
	require.Empty(t, unknownVersion.Version)
}
