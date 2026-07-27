package signing

import (
	"context"
	"fmt"
	"os"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"
)

func signingCallError(operation string, err error) error {
	return cliapp.WrapAPIError(operation, err, nil)
}

func (c *Commands) getPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoList(func(ctx cliapp.OperationContext) (*domainv1.SigningConfigResponse, error) {
		r, err := c.rpc.GetSigningConfig(context.Background(), connect.NewRequest(&domainv1.SigningScenarioRequest{ScenarioName: ctx.Positional("scenario")}))
		if err != nil {
			return nil, signingCallError("get signing config", err)
		}
		return r.Msg, nil
	}, func(ctx cliapp.OperationContext, _ *domainv1.SigningConfigResponse) cliapp.ListReport {
		return cliapp.ListReport{Summary: []string{"Signing configuration for " + ctx.Positional("scenario")}}
	})
}

func (c *Commands) setPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*domainv1.SigningConfigResponse, error) {
		raw := []byte(ctx.Flag("config"))
		var err error
		if strings.HasPrefix(ctx.Flag("config"), "@") {
			raw, err = os.ReadFile(strings.TrimPrefix(ctx.Flag("config"), "@"))
			if err != nil {
				return nil, fmt.Errorf("read signing config: %w", err)
			}
		}
		config := &domainv1.SigningConfig{}
		if err := protojson.Unmarshal(raw, config); err != nil {
			return nil, fmt.Errorf("invalid SigningConfig Proto JSON: %w", err)
		}
		r, err := c.rpc.PutSigningConfig(context.Background(), connect.NewRequest(&domainv1.UpsertSigningConfigRequest{ScenarioName: ctx.Positional("scenario"), Config: config}))
		if err != nil {
			return nil, signingCallError("set signing config", err)
		}
		return r.Msg, nil
	}, func(ctx cliapp.OperationContext, _ *domainv1.SigningConfigResponse) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"Signing configuration updated"}, Changes: []string{"Scenario: " + ctx.Positional("scenario")}, NextCommand: []string{"scenario-to-desktop signing validate " + ctx.Positional("scenario")}}
	})
}

func (c *Commands) deletePrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*domainv1.DeleteSigningResponse, error) {
		r, err := c.rpc.DeleteSigningConfig(context.Background(), connect.NewRequest(&domainv1.DeleteSigningConfigRequest{ScenarioName: ctx.Positional("scenario")}))
		if err != nil {
			return nil, signingCallError("delete signing config", err)
		}
		return r.Msg, nil
	}, func(ctx cliapp.OperationContext, _ *domainv1.DeleteSigningResponse) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"Signing configuration deleted for " + ctx.Positional("scenario")}}
	})
}

func (c *Commands) validatePrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoOperational(func(ctx cliapp.OperationContext) (*domainv1.SigningValidationResult, error) {
		r, err := c.rpc.ValidateSigningConfig(context.Background(), connect.NewRequest(&domainv1.ValidateSigningRequest{ScenarioName: ctx.Positional("scenario")}))
		if err != nil {
			return nil, signingCallError("validate signing config", err)
		}
		return r.Msg, nil
	}, func(_ cliapp.OperationContext, r *domainv1.SigningValidationResult) cliapp.OperationalReport {
		report := cliapp.OperationalReport{}
		if r.GetValid() {
			report.Status = []string{"Signing configuration is valid."}
		} else {
			report.Status = []string{"Signing configuration has issues."}
			for _, issue := range r.GetErrors() {
				report.Triage = append(report.Triage, cliapp.TriageGroup{Heading: "Validation issues", Items: []string{issue.GetMessage()}})
			}
		}
		return report
	})
}

func (c *Commands) readyPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoOperational(func(ctx cliapp.OperationContext) (*domainv1.ReadinessResponse, error) {
		r, err := c.rpc.GetSigningReadiness(context.Background(), connect.NewRequest(&domainv1.SigningScenarioRequest{ScenarioName: ctx.Positional("scenario")}))
		if err != nil {
			return nil, signingCallError("get signing readiness", err)
		}
		return r.Msg, nil
	}, func(_ cliapp.OperationContext, r *domainv1.ReadinessResponse) cliapp.OperationalReport {
		report := cliapp.OperationalReport{Status: []string{r.GetMessage()}}
		for _, item := range r.GetPlatforms() {
			report.Triage = append(report.Triage, cliapp.TriageGroup{Heading: item.GetPlatform().String(), Items: []string{item.GetMessage()}})
		}
		return report
	})
}

func (c *Commands) prerequisitesPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoList(func(cliapp.OperationContext) (*domainv1.ListSigningPrerequisitesResponse, error) {
		r, err := c.rpc.ListSigningPrerequisites(context.Background(), connect.NewRequest(&emptypb.Empty{}))
		if err != nil {
			return nil, signingCallError("list signing prerequisites", err)
		}
		return r.Msg, nil
	}, func(_ cliapp.OperationContext, r *domainv1.ListSigningPrerequisitesResponse) cliapp.ListReport {
		return cliapp.ListReport{Summary: []string{fmt.Sprintf("Signing prerequisites: %d", len(r.GetTools()))}}
	})
}

func (c *Commands) discoverPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoList(func(ctx cliapp.OperationContext) (*domainv1.DiscoverSigningCertificatesResponse, error) {
		platform, err := signingPlatform(ctx.Positional("platform"))
		if err != nil {
			return nil, err
		}
		r, err := c.rpc.DiscoverSigningCertificates(context.Background(), connect.NewRequest(&domainv1.DiscoverSigningCertificatesRequest{Platform: platform}))
		if err != nil {
			return nil, signingCallError("discover signing certificates", err)
		}
		return r.Msg, nil
	}, func(ctx cliapp.OperationContext, _ *domainv1.DiscoverSigningCertificatesResponse) cliapp.ListReport {
		return cliapp.ListReport{Summary: []string{"Signing certificates for " + ctx.Positional("platform")}}
	})
}

func (c *Commands) generateKeyPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*domainv1.GenerateLinuxSigningKeyResponse, error) {
		req := &domainv1.GenerateLinuxSigningKeyRequest{ScenarioName: ctx.Positional("scenario"), Name: ctx.Flag("name"), Email: ctx.Flag("email"), Force: ctx.BoolFlag("force")}
		if ctx.FlagProvided("passphrase-env") {
			req.PassphraseEnv = stringPtr(ctx.Flag("passphrase-env"))
		}
		r, err := c.rpc.GenerateLinuxSigningKey(context.Background(), connect.NewRequest(req))
		if err != nil {
			return nil, signingCallError("generate Linux signing key", err)
		}
		return r.Msg, nil
	}, func(_ cliapp.OperationContext, r *domainv1.GenerateLinuxSigningKeyResponse) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"GPG key generated: " + r.GetFingerprint()}}
	})
}

func stringPtr(value string) *string { return &value }
