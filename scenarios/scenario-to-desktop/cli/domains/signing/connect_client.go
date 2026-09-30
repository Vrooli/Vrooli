package signing

import (
	"context"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain/domainconnect"
	"google.golang.org/protobuf/types/known/emptypb"
)

type signingRPC interface {
	GetSigningConfig(context.Context, *connect.Request[domainv1.SigningScenarioRequest]) (*connect.Response[domainv1.SigningConfigResponse], error)
	PutSigningConfig(context.Context, *connect.Request[domainv1.UpsertSigningConfigRequest]) (*connect.Response[domainv1.SigningConfigResponse], error)
	ValidateSigningConfig(context.Context, *connect.Request[domainv1.ValidateSigningRequest]) (*connect.Response[domainv1.SigningValidationResult], error)
	GetSigningReadiness(context.Context, *connect.Request[domainv1.SigningScenarioRequest]) (*connect.Response[domainv1.ReadinessResponse], error)
	DeleteSigningConfig(context.Context, *connect.Request[domainv1.DeleteSigningConfigRequest]) (*connect.Response[domainv1.DeleteSigningResponse], error)
	GenerateLinuxSigningKey(context.Context, *connect.Request[domainv1.GenerateLinuxSigningKeyRequest]) (*connect.Response[domainv1.GenerateLinuxSigningKeyResponse], error)
	ListSigningPrerequisites(context.Context, *connect.Request[emptypb.Empty]) (*connect.Response[domainv1.ListSigningPrerequisitesResponse], error)
	DiscoverSigningCertificates(context.Context, *connect.Request[domainv1.DiscoverSigningCertificatesRequest]) (*connect.Response[domainv1.DiscoverSigningCertificatesResponse], error)
}

func newSigningRPC(app *cliapp.ScenarioApp) signingRPC {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(app)
	return domainconnect.NewSigningServiceClient(httpClient, baseURL)
}
