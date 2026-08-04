package generation

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain/domainconnect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/shared"
)

// ConnectService exposes the scenario-analysis and config-synthesis domain over
// the generated ConfigService contract. It deliberately delegates to the same
// analyzer used by pipeline generation, so Connect and pipeline callers cannot
// drift in their metadata or default-config decisions.
type ConnectService struct {
	domainconnect.UnimplementedConfigServiceHandler
	analyzer ScenarioAnalyzer
}

var _ domainconnect.ConfigServiceHandler = (*ConnectService)(nil)

func NewConnectService(analyzer ScenarioAnalyzer) *ConnectService {
	return &ConnectService{analyzer: analyzer}
}

func (s *ConnectService) GetScenarioMetadata(_ context.Context, req *connect.Request[domainv1.GetScenarioMetadataRequest]) (*connect.Response[sharedv1.ScenarioMetadata], error) {
	metadata, err := s.analyzer.AnalyzeScenario(req.Msg.GetScenarioName())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(metadataToProto(metadata)), nil
}

func (s *ConnectService) CreateDesktopConfig(_ context.Context, req *connect.Request[domainv1.CreateDesktopConfigRequest]) (*connect.Response[domainv1.DesktopConfig], error) {
	if req.Msg.GetMetadata() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("metadata is required"))
	}
	metadata := metadataFromProto(req.Msg.GetMetadata())
	if metadata.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("metadata.name is required"))
	}
	config, err := s.analyzer.CreateDesktopConfigFromMetadata(metadata, templateTypeString(req.Msg.GetTemplateType()))
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(desktopConfigToProto(config)), nil
}

func metadataToProto(value *ScenarioMetadata) *sharedv1.ScenarioMetadata {
	result := &sharedv1.ScenarioMetadata{
		Name:            value.Name,
		DisplayName:     optionalString(value.DisplayName),
		Description:     optionalString(value.Description),
		Version:         optionalString(value.Version),
		Author:          optionalString(value.Author),
		License:         optionalString(value.License),
		AppId:           optionalString(value.AppID),
		HasUi:           value.HasUI,
		UiDistPath:      optionalString(value.UIDistPath),
		ScenarioPath:    value.ScenarioPath,
		Category:        optionalString(value.Category),
		Tags:            value.Tags,
		ServiceJsonPath: optionalString(value.ServiceJSONPath),
		PackageJsonPath: optionalString(value.PackageJSONPath),
	}
	if port, ok := value.Ports["ui"]; ok && port.Port > 0 {
		result.UiPort = optionalInt32(port.Port)
	}
	if port, ok := value.Ports["api"]; ok && port.Port > 0 {
		result.ApiPort = optionalInt32(port.Port)
	}
	return result
}

func metadataFromProto(value *sharedv1.ScenarioMetadata) *ScenarioMetadata {
	metadata := &ScenarioMetadata{
		Name:            value.GetName(),
		DisplayName:     value.GetDisplayName(),
		Description:     value.GetDescription(),
		Version:         value.GetVersion(),
		Author:          value.GetAuthor(),
		License:         value.GetLicense(),
		AppID:           value.GetAppId(),
		HasUI:           value.GetHasUi(),
		UIDistPath:      value.GetUiDistPath(),
		ScenarioPath:    value.GetScenarioPath(),
		Category:        value.GetCategory(),
		Tags:            value.GetTags(),
		ServiceJSONPath: value.GetServiceJsonPath(),
		PackageJSONPath: value.GetPackageJsonPath(),
		Ports:           map[string]PortConfig{},
	}
	if value.UiPort != nil {
		metadata.Ports["ui"] = PortConfig{Port: int(value.GetUiPort())}
	}
	if value.ApiPort != nil {
		metadata.Ports["api"] = PortConfig{Port: int(value.GetApiPort())}
	}
	return metadata
}

func desktopConfigToProto(value *DesktopConfig) *domainv1.DesktopConfig {
	result := &domainv1.DesktopConfig{
		App: &domainv1.AppIdentity{
			Name:        value.AppName,
			DisplayName: value.AppDisplayName,
			Description: optionalString(value.AppDescription),
			Version:     value.Version,
			Author:      optionalString(value.Author),
			Email:       optionalString(value.AuthorEmail),
			Icon:        optionalString(value.Icon),
			Homepage:    optionalString(value.Homepage),
			License:     optionalString(value.License),
			AppId:       optionalString(value.AppID),
			AppUrl:      optionalString(value.AppURL),
		},
		Server: &domainv1.ServerConfig{
			ServerType:        optionalString(value.ServerType),
			Port:              optionalInt32(value.ServerPort),
			Path:              optionalString(value.ServerPath),
			ApiEndpoint:       optionalString(value.APIEndpoint),
			ScenarioPath:      optionalString(value.ScenarioPath),
			AutoManageVrooli:  optionalBool(value.AutoManageVrooli),
			VrooliBinaryPath:  optionalString(value.VrooliBinaryPath),
			DeploymentMode:    deploymentModeProto(value.DeploymentMode),
			ProxyUrl:          optionalString(value.ProxyURL),
			ExternalServerUrl: optionalString(value.ExternalServerURL),
			ExternalApiUrl:    optionalString(value.ExternalAPIURL),
		},
		Framework:    sharedv1.Framework_FRAMEWORK_ELECTRON,
		TemplateType: templateTypeProto(value.TemplateType),
		OutputPath:   optionalString(value.OutputPath),
		Features:     boolFeatures(value.Features),
		Styling:      stringStyling(value.Styling),
		SigningEnabled: func() *bool {
			if value.CodeSigning == nil {
				return nil
			}
			return optionalBool(value.CodeSigning.Enabled)
		}(),
	}
	for _, platform := range value.Platforms {
		result.Platforms = append(result.Platforms, platformProto(platform))
	}
	if value.BundleIPC != nil || value.BundleManifestPath != "" || value.BundleRuntimeRoot != "" {
		bundle := &domainv1.BundleConfig{ManifestPath: optionalString(value.BundleManifestPath), RuntimeRoot: optionalString(value.BundleRuntimeRoot), UiServiceId: optionalString(value.BundleUISvcID), PortName: optionalString(value.BundleUIPortName), TelemetryUploadUrl: optionalString(value.BundleTelemetryUploadURL)}
		if value.BundleIPC != nil {
			bundle.Ipc = &domainv1.BundleIPCConfig{Host: optionalString(value.BundleIPC.Host), Port: optionalInt32(value.BundleIPC.Port), AuthTokenRelPath: optionalString(value.BundleIPC.AuthTokenRel)}
		}
		result.Bundle = bundle
	}
	if value.UpdateConfig != nil {
		update := &sharedv1.UpdateConfig{Channel: optionalString(value.UpdateConfig.Channel), Provider: optionalString(value.UpdateConfig.Provider), AutoCheck: optionalBool(value.UpdateConfig.AutoCheck)}
		if value.UpdateConfig.GitHub != nil {
			update.Github = &sharedv1.GitHubUpdateConfig{Owner: value.UpdateConfig.GitHub.Owner, Repo: value.UpdateConfig.GitHub.Repo, Private: optionalBool(value.UpdateConfig.GitHub.Private)}
		}
		if value.UpdateConfig.Generic != nil {
			update.Generic = &sharedv1.GenericUpdateConfig{Url: value.UpdateConfig.Generic.URL, ChannelPath: optionalString(value.UpdateConfig.Generic.ChannelPath)}
		}
		result.Update = update
	}
	if value.Window != nil {
		result.Window = &domainv1.WindowConfig{Width: mapInt(value.Window, "width"), Height: mapInt(value.Window, "height"), MinWidth: mapInt(value.Window, "minWidth"), MinHeight: mapInt(value.Window, "minHeight"), Resizable: mapBool(value.Window, "resizable"), Frame: mapBool(value.Window, "frame"), DevTools: mapBool(value.Window, "devTools")}
	}
	return result
}

func templateTypeString(value sharedv1.TemplateType) string {
	switch value {
	case sharedv1.TemplateType_TEMPLATE_TYPE_ADVANCED:
		return "advanced"
	case sharedv1.TemplateType_TEMPLATE_TYPE_MULTI_WINDOW:
		return "multi_window"
	case sharedv1.TemplateType_TEMPLATE_TYPE_KIOSK:
		return "kiosk"
	default:
		return "basic"
	}
}

func templateTypeProto(value string) sharedv1.TemplateType {
	switch value {
	case "advanced":
		return sharedv1.TemplateType_TEMPLATE_TYPE_ADVANCED
	case "multi_window":
		return sharedv1.TemplateType_TEMPLATE_TYPE_MULTI_WINDOW
	case "kiosk":
		return sharedv1.TemplateType_TEMPLATE_TYPE_KIOSK
	default:
		return sharedv1.TemplateType_TEMPLATE_TYPE_BASIC
	}
}

func platformProto(value string) sharedv1.Platform {
	switch value {
	case "win", "windows":
		return sharedv1.Platform_PLATFORM_WIN
	case "mac", "macos":
		return sharedv1.Platform_PLATFORM_MAC
	case "linux":
		return sharedv1.Platform_PLATFORM_LINUX
	default:
		return sharedv1.Platform_PLATFORM_UNSPECIFIED
	}
}

func deploymentModeProto(value string) sharedv1.DeploymentMode {
	switch value {
	case "bundled":
		return sharedv1.DeploymentMode_DEPLOYMENT_MODE_BUNDLED
	case "proxy":
		return sharedv1.DeploymentMode_DEPLOYMENT_MODE_PROXY
	case "external-server":
		return sharedv1.DeploymentMode_DEPLOYMENT_MODE_PROXY
	default:
		return sharedv1.DeploymentMode_DEPLOYMENT_MODE_UNSPECIFIED
	}
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func optionalInt32(value int) *int32 {
	if value == 0 {
		return nil
	}
	result := int32(value)
	return &result
}
func optionalBool(value bool) *bool { return &value }
func boolFeatures(values map[string]interface{}) map[string]bool {
	result := map[string]bool{}
	for key, value := range values {
		if enabled, ok := value.(bool); ok {
			result[key] = enabled
		}
	}
	return result
}

func stringStyling(values map[string]interface{}) map[string]string {
	result := map[string]string{}
	for key, value := range values {
		if text, ok := value.(string); ok {
			result[key] = text
		}
	}
	return result
}

func mapInt(values map[string]interface{}, key string) *int32 {
	value, ok := values[key].(int)
	if !ok {
		return nil
	}
	return optionalInt32(value)
}

func mapBool(values map[string]interface{}, key string) *bool {
	value, ok := values[key].(bool)
	if !ok {
		return nil
	}
	return optionalBool(value)
}
