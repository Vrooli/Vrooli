package authz

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	applyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/apply/applyv1connect"
	capabilitiesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/capabilities/capabilitiesv1connect"
	credentialconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/credentials/credentialsv1connect"
	glossaryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/glossary/glossaryv1connect"
	hostconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/host/hostv1connect"
	operatorinputsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/operatorinputs/operatorinputsv1connect"
	operatorstateconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/operatorstate/operatorstatev1connect"
	profilesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/profiles/profilesv1connect"
	readinessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/readiness/readinessv1connect"
	resourcesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/resources/resourcesv1connect"
	selectionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/selection/selectionv1connect"
	sessionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/session/sessionv1connect"
)

const MutationCapability = "vrooli-onboarding:write"

// OperationEffect is the admission classification for one generated Connect
// procedure. Every procedure is listed explicitly so a newly added RPC cannot
// silently inherit read access and bypass the mutation boundary.
type OperationEffect uint8

const (
	OperationEffectUnknown OperationEffect = iota
	OperationEffectRead
	OperationEffectMutation
)

var operationEffects = map[string]OperationEffect{
	applyconnect.ApplyServiceStartApplyProcedure:                              OperationEffectMutation,
	applyconnect.ApplyServiceGetApplyRunProcedure:                             OperationEffectRead,
	applyconnect.ApplyServiceGetApplyPlanProcedure:                            OperationEffectRead,
	applyconnect.ApplyServiceReviewApplyProcedure:                             OperationEffectMutation,
	applyconnect.ApplyServiceCancelApplyProcedure:                             OperationEffectMutation,
	capabilitiesconnect.CapabilitiesServiceListCapabilitiesProcedure:          OperationEffectRead,
	capabilitiesconnect.CapabilitiesServiceGetCapabilityStatusProcedure:       OperationEffectRead,
	capabilitiesconnect.CapabilitiesServicePreviewCapabilityProcedure:         OperationEffectMutation,
	capabilitiesconnect.CapabilitiesServiceApplyCapabilityProcedure:           OperationEffectMutation,
	credentialconnect.CredentialsServiceListCredentialsProcedure:              OperationEffectRead,
	credentialconnect.CredentialsServiceProvisionCredentialProcedure:          OperationEffectMutation,
	credentialconnect.CredentialsServiceDiagnoseCredentialsProcedure:          OperationEffectRead,
	hostconnect.HostServiceListHostRequirementsProcedure:                      OperationEffectRead,
	hostconnect.HostServiceGetHostFactsProcedure:                              OperationEffectRead,
	hostconnect.HostServiceListTargetsProcedure:                               OperationEffectRead,
	hostconnect.HostServicePatchHostSafeguardConfigProcedure:                  OperationEffectMutation,
	hostconnect.HostServiceSetNotificationRecipientProcedure:                  OperationEffectMutation,
	operatorinputsconnect.OperatorInputsServiceListOperatorInputsProcedure:    OperationEffectRead,
	operatorinputsconnect.OperatorInputsServiceResolveOperatorInputsProcedure: OperationEffectMutation,
	operatorstateconnect.OperatorStateServiceGetOperatorStateProcedure:        OperationEffectRead,
	operatorstateconnect.OperatorStateServicePatchOperatorStateProcedure:      OperationEffectMutation,
	profilesconnect.ProfileServiceListProfilesProcedure:                       OperationEffectRead,
	profilesconnect.ProfileServiceEvaluateProfileProcedure:                    OperationEffectRead,
	readinessconnect.ReadinessServiceGetReadinessProcedure:                    OperationEffectRead,
	readinessconnect.ReadinessServiceAcknowledgeDegradedReadinessProcedure:    OperationEffectMutation,
	resourcesconnect.ResourcesServiceListResourcesProcedure:                   OperationEffectRead,
	resourcesconnect.ResourcesServiceGetResourceProcedure:                     OperationEffectRead,
	resourcesconnect.ResourcesServiceGetResourceHealthProcedure:               OperationEffectRead,
	resourcesconnect.ResourcesServiceListDerivedResourcesProcedure:            OperationEffectRead,
	selectionconnect.SelectionServiceListScenariosProcedure:                   OperationEffectRead,
	selectionconnect.SelectionServiceGetCoreSetProcedure:                      OperationEffectRead,
	selectionconnect.SelectionServiceGetRecommendationProcedure:               OperationEffectRead,
	selectionconnect.SelectionServiceAcceptRecommendationProcedure:            OperationEffectMutation,
	selectionconnect.SelectionServiceGetClosureProcedure:                      OperationEffectRead,
	selectionconnect.SelectionServiceGetUnionProcedure:                        OperationEffectRead,
	selectionconnect.SelectionServiceCreateHandoffProcedure:                   OperationEffectMutation,
	sessionconnect.SessionServiceGetSessionProcedure:                          OperationEffectRead,
	sessionconnect.SessionServiceAdvanceSessionStepProcedure:                  OperationEffectMutation,
	sessionconnect.SessionServiceGetStepModelProcedure:                        OperationEffectRead,
	sessionconnect.SessionServiceGetDraftProcedure:                            OperationEffectRead,
	sessionconnect.SessionServiceSaveDraftProcedure:                           OperationEffectMutation,
	sessionconnect.SessionServiceDiscardDraftProcedure:                        OperationEffectMutation,
	sessionconnect.SessionServiceGetProfileSessionProcedure:                   OperationEffectRead,
	sessionconnect.SessionServiceSaveProfileSessionProcedure:                  OperationEffectMutation,
	glossaryconnect.GlossaryServiceSearchGlossaryProcedure:                    OperationEffectRead,
	glossaryconnect.GlossaryServiceSearchConfigurationProcedure:               OperationEffectRead,
}

// ClassifyProcedure returns the explicit effect classification for a generated
// procedure. Unknown procedures are intentionally not treated as reads.
func ClassifyProcedure(procedure string) (OperationEffect, bool) {
	effect, ok := operationEffects[procedure]
	return effect, ok
}

// MutationInterceptor is the single onboarding mutation boundary. HTTP
// authentication is installed once by the API server; this boundary consumes
// only the verified principal and the shared capability grammar.
func MutationInterceptor() connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			effect, classified := ClassifyProcedure(req.Spec().Procedure)
			if !classified {
				return nil, connect.NewError(connect.CodeInternal, errors.New("onboarding operation has no authorization classification"))
			}
			if effect != OperationEffectMutation {
				return next(ctx, req)
			}
			if err := RequireMutation(ctx); err != nil {
				return nil, err
			}
			return next(ctx, req)
		}
	})
}

func RequireMutation(ctx context.Context) error {
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok {
		return connect.NewError(connect.CodeUnauthenticated, errors.New("verified onboarding operator required"))
	}
	if _, err := authn.RequireHuman(ctx); err != nil {
		return connect.NewError(connect.CodePermissionDenied, errors.New("human operator authorization required"))
	}
	if _, err := authn.RequireCapability(ctx, MutationCapability); err != nil {
		return connect.NewError(connect.CodePermissionDenied, errors.New("onboarding write capability required"))
	}
	if principal.Subject == "" {
		return connect.NewError(connect.CodeUnauthenticated, errors.New("verified onboarding operator required"))
	}
	return nil
}

func IsMutationProcedure(procedure string) bool {
	effect, classified := ClassifyProcedure(procedure)
	return classified && effect == OperationEffectMutation
}
