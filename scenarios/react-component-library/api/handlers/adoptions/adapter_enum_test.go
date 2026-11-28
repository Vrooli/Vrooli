package adoptions

import (
	"testing"

	"github.com/stretchr/testify/require"
	adoptionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/adoptions"
	"react-component-library/internal/adoptions"
)

func TestEnumAdaptersCoverDomainValuesAndUnknowns(t *testing.T) {
	for _, tt := range []struct {
		name string
		in   adoptions.ReconvergeAction
		want adoptionsv1.ReconvergeAction
	}{
		{"reapplied", adoptions.ReconvergeActionReapplied, adoptionsv1.ReconvergeAction_RECONVERGE_ACTION_REAPPLIED},
		{"would reapply", adoptions.ReconvergeActionWouldReapply, adoptionsv1.ReconvergeAction_RECONVERGE_ACTION_WOULD_REAPPLY},
		{"modified", adoptions.ReconvergeActionFlaggedModified, adoptionsv1.ReconvergeAction_RECONVERGE_ACTION_FLAGGED_MODIFIED},
		{"unresolved", adoptions.ReconvergeActionSkippedUnresolved, adoptionsv1.ReconvergeAction_RECONVERGE_ACTION_SKIPPED_UNRESOLVED},
		{"error", adoptions.ReconvergeActionError, adoptionsv1.ReconvergeAction_RECONVERGE_ACTION_ERROR},
		{"blocked tokens", adoptions.ReconvergeActionBlockedTokens, adoptionsv1.ReconvergeAction_RECONVERGE_ACTION_BLOCKED_TOKENS},
		{"unknown", "future", adoptionsv1.ReconvergeAction_RECONVERGE_ACTION_UNSPECIFIED},
	} {
		t.Run(tt.name, func(t *testing.T) { require.Equal(t, tt.want, reconvergeActionToProto(tt.in)) })
	}
	for _, tt := range []struct {
		in   adoptions.LibraryVersionStatus
		want adoptionsv1.LibraryVersionStatus
	}{
		{adoptions.LibraryVersionStatusCurrent, adoptionsv1.LibraryVersionStatus_LIBRARY_VERSION_STATUS_CURRENT},
		{adoptions.LibraryVersionStatusBehind, adoptionsv1.LibraryVersionStatus_LIBRARY_VERSION_STATUS_BEHIND},
		{adoptions.LibraryVersionStatusDeprecated, adoptionsv1.LibraryVersionStatus_LIBRARY_VERSION_STATUS_DEPRECATED},
		{adoptions.LibraryVersionStatusMissing, adoptionsv1.LibraryVersionStatus_LIBRARY_VERSION_STATUS_MISSING},
		{adoptions.LibraryVersionStatusUnknown, adoptionsv1.LibraryVersionStatus_LIBRARY_VERSION_STATUS_UNKNOWN},
		{adoptions.LibraryVersionStatusSourceDrifted, adoptionsv1.LibraryVersionStatus_LIBRARY_VERSION_STATUS_SOURCE_DRIFTED},
		{"future", adoptionsv1.LibraryVersionStatus_LIBRARY_VERSION_STATUS_UNSPECIFIED},
	} {
		require.Equal(t, tt.want, libraryStatusToProto(tt.in))
	}
	for _, tt := range []struct {
		in   adoptions.LocalStatus
		want adoptionsv1.LocalStatus
	}{
		{adoptions.LocalStatusClean, adoptionsv1.LocalStatus_LOCAL_STATUS_CLEAN},
		{adoptions.LocalStatusModified, adoptionsv1.LocalStatus_LOCAL_STATUS_MODIFIED},
		{adoptions.LocalStatusMissing, adoptionsv1.LocalStatus_LOCAL_STATUS_MISSING},
		{adoptions.LocalStatusUnknown, adoptionsv1.LocalStatus_LOCAL_STATUS_UNKNOWN},
		{"future", adoptionsv1.LocalStatus_LOCAL_STATUS_UNSPECIFIED},
	} {
		require.Equal(t, tt.want, localStatusToProto(tt.in))
	}
}

func TestResolveSourceAdapter(t *testing.T) {
	for _, tt := range []struct {
		in   adoptions.ResolveSource
		want adoptionsv1.ResolveSource
	}{
		{adoptions.SourceExplicit, adoptionsv1.ResolveSource_RESOLVE_SOURCE_EXPLICIT},
		{adoptions.SourceTemplateManifest, adoptionsv1.ResolveSource_RESOLVE_SOURCE_TEMPLATE_MANIFEST},
		{adoptions.SourceHeuristic, adoptionsv1.ResolveSource_RESOLVE_SOURCE_HEURISTIC},
		{adoptions.SourceFallback, adoptionsv1.ResolveSource_RESOLVE_SOURCE_FALLBACK},
		{"future", adoptionsv1.ResolveSource_RESOLVE_SOURCE_UNSPECIFIED},
	} {
		require.Equal(t, tt.want, toProtoSource(tt.in))
	}
}
