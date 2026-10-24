package signing

import (
	"context"
	"strings"
	"testing"

	"scenario-to-desktop-api/signing/types"
	"scenario-to-desktop-api/signing/validation"

	"connectrpc.com/connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/shared"
	"google.golang.org/protobuf/types/known/emptypb"
)

type connectTestRepository struct {
	configs map[string]*types.SigningConfig
}

func (r *connectTestRepository) Get(_ context.Context, scenario string) (*types.SigningConfig, error) {
	return r.configs[scenario], nil
}

func (r *connectTestRepository) Save(_ context.Context, scenario string, config *types.SigningConfig) error {
	r.configs[scenario] = config
	return nil
}

func (r *connectTestRepository) Delete(_ context.Context, scenario string) error {
	delete(r.configs, scenario)
	return nil
}

func (r *connectTestRepository) Exists(_ context.Context, scenario string) (bool, error) {
	_, ok := r.configs[scenario]
	return ok, nil
}
func (r *connectTestRepository) GetPath(scenario string) string { return scenario + "/signing.json" }
func (r *connectTestRepository) GetForPlatform(context.Context, string, string) (interface{}, error) {
	return nil, nil
}

func (r *connectTestRepository) SaveForPlatform(context.Context, string, string, interface{}) error {
	return nil
}

func (r *connectTestRepository) DeleteForPlatform(_ context.Context, scenario, platform string) error {
	config := r.configs[scenario]
	if config == nil {
		return nil
	}
	switch platform {
	case PlatformWindows:
		config.Windows = nil
	case PlatformMacOS:
		config.MacOS = nil
	case PlatformLinux:
		config.Linux = nil
	}
	return nil
}

func (r *connectTestRepository) ListScenarios() ([]string, error) {
	names := make([]string, 0, len(r.configs))
	for name := range r.configs {
		names = append(names, name)
	}
	return names, nil
}

func newConnectSigningService() (*ConnectService, *connectTestRepository) {
	repo := &connectTestRepository{configs: map[string]*types.SigningConfig{}}
	return NewConnectService(&Handler{repo: repo, validator: validation.NewValidator(), prereqChecker: validation.NewPrerequisiteChecker()}), repo
}

type connectTestDetector struct {
	certificates []types.DiscoveredCertificate
	err          error
}

func (d connectTestDetector) DiscoverCertificates(context.Context, string) ([]types.DiscoveredCertificate, error) {
	return d.certificates, d.err
}

func TestSigningConnectConfigLifecycleAndValidation(t *testing.T) {
	service, repo := newConnectSigningService()
	ctx := context.Background()
	put, err := service.PutSigningConfig(ctx, connect.NewRequest(&domainv1.UpsertSigningConfigRequest{
		ScenarioName: "demo", Config: &domainv1.SigningConfig{Enabled: false},
	}))
	if err != nil || put.Msg.GetConfig() == nil || repo.configs["demo"] == nil {
		t.Fatalf("PutSigningConfig = %#v, %v", put, err)
	}
	got, err := service.GetSigningConfig(ctx, connect.NewRequest(&domainv1.SigningScenarioRequest{ScenarioName: "demo"}))
	if err != nil || got.Msg.GetConfig().GetEnabled() {
		t.Fatalf("GetSigningConfig = %#v, %v", got, err)
	}
	validated, err := service.ValidateSigningConfig(ctx, connect.NewRequest(&domainv1.ValidateSigningRequest{ScenarioName: "demo"}))
	if err != nil || !validated.Msg.GetValid() || validated.Msg.GetSigningEnabled() {
		t.Fatalf("ValidateSigningConfig = %#v, %v", validated, err)
	}
	deleted, err := service.DeleteSigningConfig(ctx, connect.NewRequest(&domainv1.DeleteSigningConfigRequest{ScenarioName: "demo"}))
	if err != nil || deleted.Msg.GetScenarioName() != "demo" || repo.configs["demo"] != nil {
		t.Fatalf("DeleteSigningConfig = %#v, %v", deleted, err)
	}
}

func TestSigningConnectRoundTripsManagedKey(t *testing.T) {
	service, _ := newConnectSigningService()
	ctx := context.Background()
	_, err := service.PutSigningConfig(ctx, connect.NewRequest(&domainv1.UpsertSigningConfigRequest{
		ScenarioName: "demo",
		Config: &domainv1.SigningConfig{Enabled: true, Linux: &domainv1.LinuxSigningConfig{
			GpgKeyId:   optional("ABC123"),
			ManagedKey: &domainv1.ManagedSigningKey{LogicalId: "vrooli/desktop-signing"},
		}},
	}))
	if err != nil {
		t.Fatalf("PutSigningConfig = %v", err)
	}
	got, err := service.GetSigningConfig(ctx, connect.NewRequest(&domainv1.SigningScenarioRequest{ScenarioName: "demo"}))
	if err != nil {
		t.Fatalf("GetSigningConfig = %v", err)
	}
	managed := got.Msg.GetConfig().GetLinux().GetManagedKey()
	if managed.GetLogicalId() != "vrooli/desktop-signing" {
		t.Fatalf("managed key did not round-trip: %#v", managed)
	}
}

func TestSigningConnectGenerateKeyHonorsLogicalID(t *testing.T) {
	writeFakeGPG(t)
	authority := newFakeAuthority()
	withFakeAuthority(t, authority)
	service, repo := newConnectSigningService()

	resp, err := service.GenerateLinuxSigningKey(context.Background(), connect.NewRequest(&domainv1.GenerateLinuxSigningKeyRequest{
		ScenarioName:  "demo",
		Name:          "Example Publisher",
		Email:         "publisher@example.test",
		LogicalId:     optional("vrooli/desktop-signing"),
		PassphraseEnv: optional("VROOLI_GPG_PASSPHRASE"),
	}))
	if err != nil {
		t.Fatalf("GenerateLinuxSigningKey = %v", err)
	}
	if resp.Msg.GetLogicalId() != "vrooli/desktop-signing" {
		t.Fatalf("response logical id = %q", resp.Msg.GetLogicalId())
	}
	config := repo.configs["demo"]
	if config == nil || config.Linux == nil || config.Linux.ManagedKey == nil {
		t.Fatalf("managed linux config not persisted: %#v", config)
	}
	if config.Linux.ManagedKey.LogicalID != "vrooli/desktop-signing" {
		t.Fatalf("custody identity = %q", config.Linux.ManagedKey.LogicalID)
	}
	if config.Linux.GPGKeyID != "TESTFINGERPRINT123" {
		t.Fatalf("fingerprint = %q", config.Linux.GPGKeyID)
	}
	if config.Linux.GPGPassphraseEnv != "VROOLI_GPG_PASSPHRASE" {
		t.Fatalf("passphrase env = %q", config.Linux.GPGPassphraseEnv)
	}
}

func TestSigningConnectReportsRotationImpact(t *testing.T) {
	writeFakeGPG(t)
	authority := newFakeAuthority()
	withFakeAuthority(t, authority)
	service, repo := newConnectSigningService()
	repo.configs["other-app"] = &types.SigningConfig{
		Enabled: true,
		Linux: &types.LinuxSigningConfig{
			GPGKeyID:   "OLD",
			ManagedKey: &types.ManagedSigningKey{LogicalID: "vrooli/desktop-signing"},
		},
	}

	resp, err := service.GenerateLinuxSigningKey(context.Background(), connect.NewRequest(&domainv1.GenerateLinuxSigningKeyRequest{
		ScenarioName: "demo",
		Name:         "Example Publisher",
		Email:        "publisher@example.test",
		LogicalId:    optional("vrooli/desktop-signing"),
		Force:        true,
	}))
	if err != nil {
		t.Fatalf("GenerateLinuxSigningKey = %v", err)
	}
	if !strings.Contains(resp.Msg.GetMessage(), "other-app") {
		t.Fatalf("rotation message = %q, want it to name other-app", resp.Msg.GetMessage())
	}
}

func TestSigningConnectRejectsInvalidPlatformRequests(t *testing.T) {
	service, _ := newConnectSigningService()
	_, err := service.PatchSigningPlatform(context.Background(), connect.NewRequest(&domainv1.PatchSigningPlatformRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("PatchSigningPlatform code = %v", connect.CodeOf(err))
	}
	_, err = service.DeleteSigningPlatform(context.Background(), connect.NewRequest(&domainv1.DeleteSigningPlatformRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("DeleteSigningPlatform code = %v", connect.CodeOf(err))
	}
	_, err = service.ListSigningPrerequisites(context.Background(), connect.NewRequest(&emptypb.Empty{}))
	if err != nil {
		t.Fatalf("ListSigningPrerequisites = %v", err)
	}
}

func TestSigningConnectReadinessPatchingAndDiscovery(t *testing.T) {
	service, repo := newConnectSigningService()
	service.handler.detector = connectTestDetector{certificates: []types.DiscoveredCertificate{{
		ID: "cert-1", Name: "Desktop Certificate", Platform: PlatformLinux, IsCodeSign: true, DaysToExpiry: 30,
	}}}
	ctx := context.Background()
	keyID := "ABC123"
	_, err := service.PatchSigningPlatform(ctx, connect.NewRequest(&domainv1.PatchSigningPlatformRequest{
		ScenarioName: "demo", Platform: sharedv1.Platform_PLATFORM_LINUX,
		Config: &domainv1.PatchSigningPlatformRequest_Linux{Linux: &domainv1.LinuxSigningConfig{GpgKeyId: &keyID}},
	}))
	if err != nil || repo.configs["demo"] == nil || repo.configs["demo"].Linux == nil || repo.configs["demo"].Linux.GPGKeyID != "ABC123" {
		t.Fatalf("PatchSigningPlatform = %#v, %v", repo.configs["demo"], err)
	}
	repo.configs["demo"].Enabled = true
	readiness, err := service.GetSigningReadiness(ctx, connect.NewRequest(&domainv1.SigningScenarioRequest{ScenarioName: "demo"}))
	if err != nil || len(readiness.Msg.GetPlatforms()) != 3 || !readiness.Msg.GetPlatforms()[2].GetEnabled() {
		t.Fatalf("GetSigningReadiness = %#v, %v", readiness, err)
	}
	discovered, err := service.DiscoverSigningCertificates(ctx, connect.NewRequest(&domainv1.DiscoverSigningCertificatesRequest{Platform: sharedv1.Platform_PLATFORM_LINUX}))
	if err != nil || len(discovered.Msg.GetCertificates()) != 1 || discovered.Msg.GetCertificates()[0].GetId() != "cert-1" {
		t.Fatalf("DiscoverSigningCertificates = %#v, %v", discovered, err)
	}
	_, err = service.DiscoverSigningCertificates(ctx, connect.NewRequest(&domainv1.DiscoverSigningCertificatesRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("missing discovery platform code = %v", connect.CodeOf(err))
	}
}

func TestSigningConnectRejectsUnsupportedWindowsCertificateSourcePatch(t *testing.T) {
	service, repo := newConnectSigningService()
	cloudSource := domainv1.CertificateSource_CERTIFICATE_SOURCE_AZURE_KEY_VAULT
	_, err := service.PatchSigningPlatform(context.Background(), connect.NewRequest(&domainv1.PatchSigningPlatformRequest{
		ScenarioName: "demo",
		Platform:     sharedv1.Platform_PLATFORM_WIN,
		Config: &domainv1.PatchSigningPlatformRequest_Windows{Windows: &domainv1.WindowsSigningConfig{
			CertificateSource: cloudSource,
		}},
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("cloud Windows patch code = %v, want invalid argument", connect.CodeOf(err))
	}
	if repo.configs["demo"] != nil {
		t.Fatal("unsupported cloud patch must not persist a configuration")
	}
}
