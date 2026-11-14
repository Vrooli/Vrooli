package mfa

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	mfav1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/mfa"
	mfaconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/mfa/mfa_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	client mfaconnect.MFAServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: mfaconnect.NewMFAServiceClient(httpClient, baseURL)}
}

func (h *handlers) beginEnrollment(ctx cliapp.RunContext) error {
	resp, err := h.client.BeginEnrollment(context.Background(), connect.NewRequest(&mfav1.BeginEnrollmentRequest{AccessToken: ctx.Flag("access-token")}))
	if err != nil {
		return cliapp.WrapAPIError("begin MFA enrollment", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no MFA enrollment")
	}
	changes := []string{"Enrollment ID: " + resp.Msg.EnrollmentId, "Provisioning URI: " + resp.Msg.ProvisioningUri}
	if resp.Msg.ExpiresAt != nil {
		changes = append(changes, "Expires: "+resp.Msg.ExpiresAt.AsTime().String())
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{"MFA enrollment started; scan the provisioning URI, then confirm it with `mfa confirm-enrollment`."},
		Changes: changes,
	})
}

func (h *handlers) confirmEnrollment(ctx cliapp.RunContext) error {
	resp, err := h.client.ConfirmEnrollment(context.Background(), connect.NewRequest(&mfav1.ConfirmEnrollmentRequest{
		AccessToken: ctx.Flag("access-token"), EnrollmentId: ctx.Flag("enrollment-id"), TotpCode: ctx.Flag("totp-code"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("confirm MFA enrollment", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no MFA recovery codes")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{"MFA enabled. Store these recovery codes; they are shown only once."},
		Changes: []string{fmt.Sprintf("Recovery codes: %v", resp.Msg.RecoveryCodes)},
	})
}

func (h *handlers) removeEnrollment(ctx cliapp.RunContext) error {
	resp, err := h.client.RemoveEnrollment(context.Background(), connect.NewRequest(&mfav1.RemoveEnrollmentRequest{AccessToken: ctx.Flag("access-token")}))
	if err != nil {
		return cliapp.WrapAPIError("remove MFA enrollment", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{"MFA enrollment removed and recovery codes invalidated."}})
}
