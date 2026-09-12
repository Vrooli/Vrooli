package generation

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/shared"
)

type connectAnalyzerFake struct {
	metadata *ScenarioMetadata
	config   *DesktopConfig
}

func (f connectAnalyzerFake) AnalyzeScenario(string) (*ScenarioMetadata, error) {
	return f.metadata, nil
}
func (f connectAnalyzerFake) ValidateScenarioForDesktop(string) error { return nil }
func (f connectAnalyzerFake) CreateDesktopConfigFromMetadata(*ScenarioMetadata, string) (*DesktopConfig, error) {
	return f.config, nil
}

func TestConnectServiceReturnsScenarioMetadata(t *testing.T) {
	handler := NewConnectService(connectAnalyzerFake{metadata: &ScenarioMetadata{Name: "desktop-demo", HasUI: true, Ports: map[string]PortConfig{"ui": {Port: 22829}, "api": {Port: 19925}}}})

	response, err := handler.GetScenarioMetadata(context.Background(), connect.NewRequest(&domainv1.GetScenarioMetadataRequest{ScenarioName: "desktop-demo"}))
	if err != nil {
		t.Fatalf("GetScenarioMetadata() error = %v", err)
	}
	if response.Msg.GetName() != "desktop-demo" || !response.Msg.GetHasUi() {
		t.Fatalf("metadata = %#v, want desktop-demo with UI", response.Msg)
	}
	if response.Msg.GetUiPort() != 22829 || response.Msg.GetApiPort() != 19925 {
		t.Fatalf("ports = (%d, %d), want (22829, 19925)", response.Msg.GetUiPort(), response.Msg.GetApiPort())
	}
}

func TestConnectServiceSynthesizesDesktopConfig(t *testing.T) {
	handler := NewConnectService(connectAnalyzerFake{config: &DesktopConfig{AppName: "desktop-demo", AppDisplayName: "Desktop Demo", Version: "1.0.0", ServerType: "external", Framework: "electron", TemplateType: "advanced", Platforms: []string{"linux"}, Features: map[string]interface{}{"systemTray": true}}})

	response, err := handler.CreateDesktopConfig(context.Background(), connect.NewRequest(&domainv1.CreateDesktopConfigRequest{Metadata: &sharedv1.ScenarioMetadata{Name: "desktop-demo", HasUi: true}, TemplateType: sharedv1.TemplateType_TEMPLATE_TYPE_ADVANCED}))
	if err != nil {
		t.Fatalf("CreateDesktopConfig() error = %v", err)
	}
	if response.Msg.GetFramework() != sharedv1.Framework_FRAMEWORK_ELECTRON || response.Msg.GetTemplateType() != sharedv1.TemplateType_TEMPLATE_TYPE_ADVANCED {
		t.Fatalf("config enums = (%v, %v), want Electron advanced", response.Msg.GetFramework(), response.Msg.GetTemplateType())
	}
	if !response.Msg.GetFeatures()["systemTray"] {
		t.Fatalf("config features = %#v, want systemTray", response.Msg.GetFeatures())
	}
}
