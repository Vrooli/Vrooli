package sketch_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	previewconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/preview/preview_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/shared"
	workspaceassistconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspaceassist/workspaceassist_v1connect"
	workspacecandidateconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspacecandidate/workspacecandidate_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestSharedSupportContractsKeepProtoJSONAndConnectPaths(t *testing.T) {
	response := &sharedv1.CandidateResponse{
		Candidate: &sharedv1.CandidateReference{Scenario: "demo", DesignId: "draft", Hash: "candidate-hash"},
		Sketch:    &sharedv1.Sketch{Viewport: "desktop", Template: &sharedv1.AssetReference{Asset: "templates.page", Version: "1.0.0"}},
	}
	encoded, err := protojson.Marshal(response)
	require.NoError(t, err)
	require.JSONEq(t, `{"candidate":{"scenario":"demo","designId":"draft","hash":"candidate-hash"},"sketch":{"viewport":"desktop","template":{"asset":"templates.page","version":"1.0.0"}}}`, string(encoded))
	require.Equal(t, "vrooli.react_component_library.v1.shared.CandidateResponse", string(response.ProtoReflect().Descriptor().FullName()))
	require.Equal(t, "vrooli.react_component_library.v1.shared.RenderCompositionResponse", string((&sharedv1.RenderCompositionResponse{}).ProtoReflect().Descriptor().FullName()))
	require.Equal(t, "/vrooli.react_component_library.v1.preview.PreviewService/RenderComposition", previewconnect.PreviewServiceRenderCompositionProcedure)
	require.Equal(t, "/vrooli.react_component_library.v1.workspacecandidate.WorkspaceCandidateService/RenderCandidate", workspacecandidateconnect.WorkspaceCandidateServiceRenderCandidateProcedure)
	require.Equal(t, "/vrooli.react_component_library.v1.workspaceassist.WorkspaceAssistService/ProposeSketch", workspaceassistconnect.WorkspaceAssistServiceProposeSketchProcedure)
}
