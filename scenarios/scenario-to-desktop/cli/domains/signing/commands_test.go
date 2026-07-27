package signing

import (
	"context"
	"encoding/json"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/shared"
	"google.golang.org/protobuf/types/known/emptypb"
)

type fakeSigningRPC struct {
	put      *domainv1.UpsertSigningConfigRequest
	discover *domainv1.DiscoverSigningCertificatesRequest
	key      *domainv1.GenerateLinuxSigningKeyRequest
}

func (f *fakeSigningRPC) GetSigningConfig(context.Context, *connect.Request[domainv1.SigningScenarioRequest]) (*connect.Response[domainv1.SigningConfigResponse], error) {
	return connect.NewResponse(&domainv1.SigningConfigResponse{}), nil
}

func (f *fakeSigningRPC) PutSigningConfig(_ context.Context, r *connect.Request[domainv1.UpsertSigningConfigRequest]) (*connect.Response[domainv1.SigningConfigResponse], error) {
	f.put = r.Msg
	return connect.NewResponse(&domainv1.SigningConfigResponse{}), nil
}

func (f *fakeSigningRPC) ValidateSigningConfig(context.Context, *connect.Request[domainv1.ValidateSigningRequest]) (*connect.Response[domainv1.SigningValidationResult], error) {
	return connect.NewResponse(&domainv1.SigningValidationResult{Valid: true}), nil
}

func (f *fakeSigningRPC) GetSigningReadiness(context.Context, *connect.Request[domainv1.SigningScenarioRequest]) (*connect.Response[domainv1.ReadinessResponse], error) {
	return connect.NewResponse(&domainv1.ReadinessResponse{}), nil
}

func (f *fakeSigningRPC) DeleteSigningConfig(context.Context, *connect.Request[domainv1.DeleteSigningConfigRequest]) (*connect.Response[domainv1.DeleteSigningResponse], error) {
	return connect.NewResponse(&domainv1.DeleteSigningResponse{}), nil
}

func (f *fakeSigningRPC) GenerateLinuxSigningKey(_ context.Context, r *connect.Request[domainv1.GenerateLinuxSigningKeyRequest]) (*connect.Response[domainv1.GenerateLinuxSigningKeyResponse], error) {
	f.key = r.Msg
	return connect.NewResponse(&domainv1.GenerateLinuxSigningKeyResponse{Fingerprint: "test"}), nil
}

func (f *fakeSigningRPC) ListSigningPrerequisites(context.Context, *connect.Request[emptypb.Empty]) (*connect.Response[domainv1.ListSigningPrerequisitesResponse], error) {
	return connect.NewResponse(&domainv1.ListSigningPrerequisitesResponse{}), nil
}

func (f *fakeSigningRPC) DiscoverSigningCertificates(_ context.Context, r *connect.Request[domainv1.DiscoverSigningCertificatesRequest]) (*connect.Response[domainv1.DiscoverSigningCertificatesResponse], error) {
	f.discover = r.Msg
	return connect.NewResponse(&domainv1.DiscoverSigningCertificatesResponse{}), nil
}

func assertPrimitiveModes(t *testing.T, handler cliapp.PrimitiveHandler, schema cliapp.ArgSchema, args []string) {
	t.Helper()
	modes := cliapptest.RunPrimitiveHandlerModes(t, handler, schema, args, nil)
	if modes.HumanErr != nil || modes.JSONErr != nil {
		t.Fatalf("primitive errors: human=%v json=%v", modes.HumanErr, modes.JSONErr)
	}
	var result any
	if err := json.Unmarshal([]byte(modes.JSON), &result); err != nil {
		t.Fatalf("JSON result: %v", err)
	}
}

func TestSetPrimitiveUsesCanonicalProtoConfig(t *testing.T) {
	rpc := &fakeSigningRPC{}
	c := &Commands{rpc: rpc}
	assertPrimitiveModes(t, c.setPrimitive(), cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario", Required: true}}, Flags: []cliapp.Flag{{Name: "config", Required: true}}}, []string{"demo", "--config", `{"enabled":true,"linux":{"gpg_key_id":"key"}}`})
	if rpc.put.GetScenarioName() != "demo" || !rpc.put.GetConfig().GetEnabled() || rpc.put.GetConfig().GetLinux().GetGpgKeyId() != "key" {
		t.Fatalf("unexpected request %#v", rpc.put)
	}
}

func TestDiscoverPrimitiveUsesTypedPlatform(t *testing.T) {
	rpc := &fakeSigningRPC{}
	c := &Commands{rpc: rpc}
	assertPrimitiveModes(t, c.discoverPrimitive(), scenarioSchema("platform"), []string{"windows"})
	if rpc.discover.GetPlatform() != sharedv1.Platform_PLATFORM_WIN {
		t.Fatalf("platform=%v", rpc.discover.GetPlatform())
	}
}

func TestGenerateKeyProductionParserRejectsInlineSecret(t *testing.T) {
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario", Required: true}}, Flags: []cliapp.Flag{{Name: "name", Required: true}, {Name: "email", Required: true}, {Name: "passphrase-env"}, {Name: "force", Bool: true}}}
	if _, err := cliapptest.NewTestRunContextFromArgs(schema, []string{"demo", "--name", "n", "--email", "a@example.com", "--passphrase", "secret"}, nil, nil, nil); err == nil {
		t.Fatal("inline passphrase accepted")
	}
}

func TestGenerateKeyPrimitiveUsesEnvironmentReference(t *testing.T) {
	rpc := &fakeSigningRPC{}
	c := &Commands{rpc: rpc}
	assertPrimitiveModes(t, c.generateKeyPrimitive(), cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario", Required: true}}, Flags: []cliapp.Flag{{Name: "name", Required: true}, {Name: "email", Required: true}, {Name: "passphrase-env"}, {Name: "force", Bool: true}}}, []string{"demo", "--name", "n", "--email", "a@example.com", "--passphrase-env", "GPG_SECRET", "--force"})
	if rpc.key.GetPassphraseEnv() != "GPG_SECRET" || !rpc.key.GetForce() {
		t.Fatalf("unexpected request %#v", rpc.key)
	}
}
