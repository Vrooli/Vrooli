package consumerdeclarations

import (
	"context"
	"sort"

	"connectrpc.com/connect"
	consumerdeclaration "github.com/vrooli/browser-automation-studio/services/consumer-declaration"
	consumerdeclarationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/consumer_declarations"
)

type service struct{}

func (*service) Validate(_ context.Context, req *connect.Request[consumerdeclarationsv1.ValidateConsumerDeclarationRequest]) (*connect.Response[consumerdeclarationsv1.ValidateConsumerDeclarationResponse], error) {
	declaration, result := consumerdeclaration.Validate([]byte(req.Msg.GetDeclarationJson()))
	profiles := make([]*consumerdeclarationsv1.DeclaredProfile, 0, len(declaration.Profiles))
	for _, profile := range declaration.Profiles {
		profiles = append(profiles, &consumerdeclarationsv1.DeclaredProfile{Key: profile.Key, WorkflowRef: profile.WorkflowRef, AllowedVariables: profile.AllowedVariables})
	}
	sort.Slice(profiles, func(i, j int) bool { return profiles[i].Key < profiles[j].Key })
	return connect.NewResponse(&consumerdeclarationsv1.ValidateConsumerDeclarationResponse{Valid: result.Valid(), Issues: result.Issues, Profiles: profiles}), nil
}
