package signing

import (
	"context"
	"fmt"
	"time"

	"scenario-to-desktop-api/signing/types"

	"connectrpc.com/connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain/domainconnect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/shared"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ConnectService delegates the typed transport to the same collaborators used
// by the REST handler. The embedded implementation deliberately makes any
// newly-declared RPC fail loudly until it has an equivalent domain mapping.
type ConnectService struct {
	domainconnect.UnimplementedSigningServiceHandler
	handler *Handler
}

var _ domainconnect.SigningServiceHandler = (*ConnectService)(nil)

func NewConnectService(handler *Handler) *ConnectService { return &ConnectService{handler: handler} }

func (s *ConnectService) GetSigningConfig(ctx context.Context, req *connect.Request[domainv1.SigningScenarioRequest]) (*connect.Response[domainv1.SigningConfigResponse], error) {
	config, err := s.handler.repo.Get(ctx, req.Msg.GetScenarioName())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get signing config: %w", err))
	}
	return connect.NewResponse(s.configResponse(config)), nil
}

func (s *ConnectService) PutSigningConfig(ctx context.Context, req *connect.Request[domainv1.UpsertSigningConfigRequest]) (*connect.Response[domainv1.SigningConfigResponse], error) {
	config := configFromProto(req.Msg.GetConfig())
	if result := s.handler.validator.ValidateConfig(config); !result.Valid {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("signing configuration validation failed"))
	}
	if err := s.handler.repo.Save(ctx, req.Msg.GetScenarioName(), config); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("save signing config: %w", err))
	}
	return connect.NewResponse(s.configResponse(config)), nil
}

func (s *ConnectService) ValidateSigningConfig(ctx context.Context, req *connect.Request[domainv1.ValidateSigningRequest]) (*connect.Response[domainv1.SigningValidationResult], error) {
	config := configFromProto(req.Msg.GetConfig())
	if req.Msg.GetConfig() == nil {
		var err error
		config, err = s.handler.repo.Get(ctx, req.Msg.GetScenarioName())
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if config == nil {
			config = NewSigningConfig()
		}
	}
	result := s.handler.validator.ValidateConfig(config)
	result.Merge(s.handler.prereqChecker.CheckPrerequisites(ctx, config))
	return connect.NewResponse(validationToProto(result, config.Enabled)), nil
}

func (s *ConnectService) GetSigningReadiness(ctx context.Context, req *connect.Request[domainv1.SigningScenarioRequest]) (*connect.Response[domainv1.ReadinessResponse], error) {
	config, err := s.handler.repo.Get(ctx, req.Msg.GetScenarioName())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &domainv1.ReadinessResponse{Message: "Signing is not enabled for this scenario"}
	if config == nil || !config.Enabled {
		return connect.NewResponse(response), nil
	}
	response.Message = "Signing readiness evaluated"
	for _, entry := range []struct {
		platform string
		value    any
	}{{PlatformWindows, config.Windows}, {PlatformMacOS, config.MacOS}, {PlatformLinux, config.Linux}} {
		ready, message := false, "Not configured"
		if entry.value != nil {
			result := s.handler.validator.ValidateForPlatform(config, entry.platform)
			ready = result.Valid
			if !ready && len(result.Errors) > 0 {
				message = result.Errors[0].Message
			}
		}
		response.Platforms = append(response.Platforms, &domainv1.PlatformStatus{Platform: platformProto(entry.platform), Ready: ready, Enabled: entry.value != nil, Message: message})
		response.Ready = response.Ready || ready
	}
	return connect.NewResponse(response), nil
}

func (s *ConnectService) PatchSigningPlatform(ctx context.Context, req *connect.Request[domainv1.PatchSigningPlatformRequest]) (*connect.Response[domainv1.SigningConfigResponse], error) {
	platform := platformString(req.Msg.GetPlatform())
	if platform == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("platform is required"))
	}
	config, err := s.handler.repo.Get(ctx, req.Msg.GetScenarioName())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if config == nil {
		config = NewSigningConfig()
	}
	switch value := req.Msg.Config.(type) {
	case *domainv1.PatchSigningPlatformRequest_Windows:
		if platform != PlatformWindows || value.Windows == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("windows config requires windows platform"))
		}
		config.Windows = configFromProto(&domainv1.SigningConfig{Windows: value.Windows}).Windows
	case *domainv1.PatchSigningPlatformRequest_Macos:
		if platform != PlatformMacOS || value.Macos == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("macos config requires macos platform"))
		}
		config.MacOS = configFromProto(&domainv1.SigningConfig{Macos: value.Macos}).MacOS
	case *domainv1.PatchSigningPlatformRequest_Linux:
		if platform != PlatformLinux || value.Linux == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("linux config requires linux platform"))
		}
		config.Linux = configFromProto(&domainv1.SigningConfig{Linux: value.Linux}).Linux
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("matching platform configuration is required"))
	}
	if err := s.handler.repo.Save(ctx, req.Msg.GetScenarioName(), config); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(s.configResponse(config)), nil
}

func (s *ConnectService) DeleteSigningConfig(ctx context.Context, req *connect.Request[domainv1.DeleteSigningConfigRequest]) (*connect.Response[domainv1.DeleteSigningResponse], error) {
	if err := s.handler.repo.Delete(ctx, req.Msg.GetScenarioName()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&domainv1.DeleteSigningResponse{ScenarioName: req.Msg.GetScenarioName()}), nil
}

func (s *ConnectService) DeleteSigningPlatform(ctx context.Context, req *connect.Request[domainv1.DeleteSigningPlatformRequest]) (*connect.Response[domainv1.DeleteSigningResponse], error) {
	platform := platformString(req.Msg.GetPlatform())
	if platform == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("platform is required"))
	}
	if err := s.handler.repo.DeleteForPlatform(ctx, req.Msg.GetScenarioName(), platform); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&domainv1.DeleteSigningResponse{ScenarioName: req.Msg.GetScenarioName(), Platform: &req.Msg.Platform}), nil
}

func (s *ConnectService) GenerateLinuxSigningKey(ctx context.Context, req *connect.Request[domainv1.GenerateLinuxSigningKeyRequest]) (*connect.Response[domainv1.GenerateLinuxSigningKeyResponse], error) {
	keyCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	result, err := s.handler.generateLinuxKey(keyCtx, generateLinuxKeyParams{Name: req.Msg.GetName(), Email: req.Msg.GetEmail(), PassphraseEnv: req.Msg.GetPassphraseEnv(), KeyType: req.Msg.GetKeyType(), Expiry: req.Msg.GetExpiry(), Homedir: req.Msg.GetHomedir(), Force: req.Msg.GetForce(), ExportPublic: req.Msg.GetExportPublic(), Scenario: req.Msg.GetScenarioName(), WorkingDirRoot: resolveVrooliRoot()})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	config, err := s.handler.repo.Get(keyCtx, req.Msg.GetScenarioName())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if config == nil {
		config = NewSigningConfig()
	}
	config.Enabled = true
	if config.Linux == nil {
		config.Linux = &types.LinuxSigningConfig{}
	}
	config.Linux.GPGKeyID = result.Fingerprint
	config.Linux.GPGHomedir = result.Homedir
	if req.Msg.GetPassphraseEnv() != "" {
		config.Linux.GPGPassphraseEnv = req.Msg.GetPassphraseEnv()
	}
	if err := s.handler.repo.Save(keyCtx, req.Msg.GetScenarioName(), config); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&domainv1.GenerateLinuxSigningKeyResponse{KeyId: result.Fingerprint, Fingerprint: result.Fingerprint, Homedir: result.Homedir, PublicKey: optional(result.PublicKey), PublicKeyPath: optional(result.PublicPath)}), nil
}

func (s *ConnectService) ListSigningPrerequisites(ctx context.Context, _ *connect.Request[emptypb.Empty]) (*connect.Response[domainv1.ListSigningPrerequisitesResponse], error) {
	tools, err := s.handler.prereqChecker.DetectTools(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &domainv1.ListSigningPrerequisitesResponse{}
	for _, tool := range tools {
		response.Tools = append(response.Tools, &domainv1.SigningToolStatus{Platform: platformProto(tool.Platform), Tool: tool.Tool, Installed: tool.Installed, Path: optional(tool.Path), Version: optional(tool.Version), Diagnostic: optional(tool.Error), Remediation: optional(tool.Remediation)})
	}
	return connect.NewResponse(response), nil
}

func (s *ConnectService) DiscoverSigningCertificates(ctx context.Context, req *connect.Request[domainv1.DiscoverSigningCertificatesRequest]) (*connect.Response[domainv1.DiscoverSigningCertificatesResponse], error) {
	platform := platformString(req.Msg.GetPlatform())
	if platform == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("platform is required"))
	}
	certificates, err := s.handler.detector.DiscoverCertificates(ctx, platform)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &domainv1.DiscoverSigningCertificatesResponse{}
	for _, cert := range certificates {
		response.Certificates = append(response.Certificates, &domainv1.DiscoveredSigningCertificate{Id: cert.ID, Name: cert.Name, Subject: optional(cert.Subject), Issuer: optional(cert.Issuer), ExpiresAt: optional(cert.ExpiresAt), DaysToExpiry: int32(cert.DaysToExpiry), Expired: cert.IsExpired, CodeSigning: cert.IsCodeSign, Type: optional(cert.Type), Platform: platformProto(cert.Platform), UsageHint: optional(cert.UsageHint)})
	}
	return connect.NewResponse(response), nil
}

func (s *ConnectService) configResponse(config *types.SigningConfig) *domainv1.SigningConfigResponse {
	if config == nil {
		return &domainv1.SigningConfigResponse{}
	}
	return &domainv1.SigningConfigResponse{Config: configToProto(config), Validation: validationToProto(s.handler.validator.ValidateConfig(config), config.Enabled)}
}

func configFromProto(value *domainv1.SigningConfig) *types.SigningConfig {
	if value == nil {
		return NewSigningConfig()
	}
	result := &types.SigningConfig{Enabled: value.GetEnabled(), SchemaVersion: value.GetSchemaVersion()}
	if v := value.GetWindows(); v != nil {
		result.Windows = &types.WindowsSigningConfig{CertificateSource: certificateSourceString(v.GetCertificateSource()), CertificateFile: v.GetCertificatePath(), CertificatePasswordEnv: v.GetPasswordEnv(), CertificateThumbprint: v.GetCertificateThumbprint(), TimestampServer: v.GetTimestampServer(), SignAlgorithm: signAlgorithmString(v.GetSignAlgorithm())}
	}
	if v := value.GetMacos(); v != nil {
		result.MacOS = &types.MacOSSigningConfig{Identity: v.GetIdentity(), TeamID: v.GetTeamId(), HardenedRuntime: v.GetHardenedRuntime(), Notarize: v.GetNotarize(), EntitlementsFile: v.GetEntitlementsPath(), ProvisioningProfile: v.GetProvisioningProfile()}
	}
	if v := value.GetLinux(); v != nil {
		result.Linux = &types.LinuxSigningConfig{GPGKeyID: v.GetGpgKeyId(), GPGPassphraseEnv: v.GetPassphraseEnv(), GPGHomedir: v.GetKeyringPath()}
	}
	return result
}

func configToProto(value *types.SigningConfig) *domainv1.SigningConfig {
	result := &domainv1.SigningConfig{Enabled: value.Enabled}
	if value.SchemaVersion != "" {
		result.SchemaVersion = &value.SchemaVersion
	}
	if v := value.Windows; v != nil {
		result.Windows = &domainv1.WindowsSigningConfig{CertificateSource: certificateSourceProto(v.CertificateSource), CertificatePath: optional(v.CertificateFile), PasswordEnv: optional(v.CertificatePasswordEnv), CertificateThumbprint: optional(v.CertificateThumbprint), TimestampServer: optional(v.TimestampServer), SignAlgorithm: signAlgorithmProto(v.SignAlgorithm)}
	}
	if v := value.MacOS; v != nil {
		result.Macos = &domainv1.MacOSSigningConfig{Identity: optional(v.Identity), TeamId: optional(v.TeamID), HardenedRuntime: optional(v.HardenedRuntime), Notarize: v.Notarize, EntitlementsPath: optional(v.EntitlementsFile), ProvisioningProfile: optional(v.ProvisioningProfile)}
	}
	if v := value.Linux; v != nil {
		result.Linux = &domainv1.LinuxSigningConfig{GpgKeyId: optional(v.GPGKeyID), PassphraseEnv: optional(v.GPGPassphraseEnv), KeyringPath: optional(v.GPGHomedir)}
	}
	return result
}

func validationToProto(value *types.ValidationResult, enabled bool) *domainv1.SigningValidationResult {
	result := &domainv1.SigningValidationResult{SigningEnabled: enabled, Platforms: map[string]*domainv1.PlatformValidation{}, ValidatedAt: timestamppb.New(time.Now())}
	if value == nil {
		return result
	}
	result.Valid = value.Valid
	for name, status := range value.Platforms {
		result.Platforms[name] = &domainv1.PlatformValidation{Platform: platformProto(name), Valid: len(status.Errors) == 0, Enabled: status.Configured, ToolsAvailable: status.ToolInstalled, MissingTools: status.Errors}
	}
	for _, issue := range value.Errors {
		result.Errors = append(result.Errors, &sharedv1.ValidationError{Code: issue.Code, Field: optional(issue.Field), Message: issue.Message})
	}
	for _, issue := range value.Warnings {
		result.Warnings = append(result.Warnings, &sharedv1.ValidationWarning{Code: issue.Code, Message: issue.Message})
	}
	return result
}

func platformProto(value string) sharedv1.Platform {
	if value == PlatformWindows {
		return sharedv1.Platform_PLATFORM_WIN
	}
	if value == PlatformMacOS {
		return sharedv1.Platform_PLATFORM_MAC
	}
	if value == PlatformLinux {
		return sharedv1.Platform_PLATFORM_LINUX
	}
	return sharedv1.Platform_PLATFORM_UNSPECIFIED
}

func platformString(value sharedv1.Platform) string {
	switch value {
	case sharedv1.Platform_PLATFORM_WIN:
		return PlatformWindows
	case sharedv1.Platform_PLATFORM_MAC:
		return PlatformMacOS
	case sharedv1.Platform_PLATFORM_LINUX:
		return PlatformLinux
	default:
		return ""
	}
}

func certificateSourceString(value domainv1.CertificateSource) string {
	return map[domainv1.CertificateSource]string{domainv1.CertificateSource_CERTIFICATE_SOURCE_FILE: types.CertSourceFile, domainv1.CertificateSource_CERTIFICATE_SOURCE_STORE: types.CertSourceStore, domainv1.CertificateSource_CERTIFICATE_SOURCE_AZURE_KEY_VAULT: types.CertSourceAzureKeyVault, domainv1.CertificateSource_CERTIFICATE_SOURCE_AWS_KMS: types.CertSourceAWSKMS}[value]
}

func certificateSourceProto(value string) domainv1.CertificateSource {
	return map[string]domainv1.CertificateSource{types.CertSourceFile: domainv1.CertificateSource_CERTIFICATE_SOURCE_FILE, types.CertSourceStore: domainv1.CertificateSource_CERTIFICATE_SOURCE_STORE, types.CertSourceAzureKeyVault: domainv1.CertificateSource_CERTIFICATE_SOURCE_AZURE_KEY_VAULT, types.CertSourceAWSKMS: domainv1.CertificateSource_CERTIFICATE_SOURCE_AWS_KMS}[value]
}

func signAlgorithmString(value domainv1.SignAlgorithm) string {
	return map[domainv1.SignAlgorithm]string{domainv1.SignAlgorithm_SIGN_ALGORITHM_SHA256: types.SignAlgorithmSHA256, domainv1.SignAlgorithm_SIGN_ALGORITHM_SHA384: types.SignAlgorithmSHA384, domainv1.SignAlgorithm_SIGN_ALGORITHM_SHA512: types.SignAlgorithmSHA512}[value]
}

func signAlgorithmProto(value string) domainv1.SignAlgorithm {
	return map[string]domainv1.SignAlgorithm{types.SignAlgorithmSHA256: domainv1.SignAlgorithm_SIGN_ALGORITHM_SHA256, types.SignAlgorithmSHA384: domainv1.SignAlgorithm_SIGN_ALGORITHM_SHA384, types.SignAlgorithmSHA512: domainv1.SignAlgorithm_SIGN_ALGORITHM_SHA512}[value]
}

func optional[T comparable](value T) *T {
	var zero T
	if value == zero {
		return nil
	}
	return &value
}
