package main

import (
	"os"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/ai"
	apiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	capturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
	consumerdeclarationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/consumer_declarations"
	drillsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/drills"
	entitlementv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/entitlement"
	exportsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/exports"
	observabilityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/observability"
	project_filesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/project_files"
	projectsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/projects"
	recordingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/recordings"
	replayconfigv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/replay_config"
	scenariosv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/scenarios"
	schedulesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/schedules"
	schemav1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/schema"
	sessionprofilesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/session_profiles"
	uxmetricsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/uxmetrics"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestManifestCoversProtoServices(t *testing.T) {
	manifest, err := os.ReadFile("manifest.json")
	if err != nil {
		t.Fatalf("read manifest.json: %v", err)
	}

	services := []struct {
		name    string
		file    protoreflect.FileDescriptor
		service string
	}{
		{name: "AIService", file: aiv1.File_browser_automation_studio_v1_ai_ai_proto, service: "AIService"},
		{name: "VisionNavigationService", file: aiv1.File_browser_automation_studio_v1_ai_ai_proto, service: "VisionNavigationService"},
		{name: "CaptureService", file: capturev1.File_browser_automation_studio_v1_capture_capture_proto, service: "CaptureService"},
		{name: "ConsumerDeclarationsService", file: consumerdeclarationsv1.File_browser_automation_studio_v1_consumer_declarations_consumer_declarations_proto, service: "ConsumerDeclarationsService"},
		{name: "FailureDrillService", file: drillsv1.File_browser_automation_studio_v1_drills_drills_proto, service: "FailureDrillService"},
		{name: "EntitlementService", file: entitlementv1.File_browser_automation_studio_v1_entitlement_entitlement_proto, service: "EntitlementService"},
		{name: "ExecutionsService", file: apiv1.File_browser_automation_studio_v1_api_service_proto, service: "ExecutionsService"},
		{name: "WorkflowsService", file: apiv1.File_browser_automation_studio_v1_api_service_proto, service: "WorkflowsService"},
		{name: "ExportsService", file: exportsv1.File_browser_automation_studio_v1_exports_exports_proto, service: "ExportsService"},
		{name: "ObservabilityService", file: observabilityv1.File_browser_automation_studio_v1_observability_observability_proto, service: "ObservabilityService"},
		{name: "ProjectFilesService", file: project_filesv1.File_browser_automation_studio_v1_project_files_project_files_proto, service: "ProjectFilesService"},
		{name: "ProjectsService", file: projectsv1.File_browser_automation_studio_v1_projects_project_proto, service: "ProjectsService"},
		{name: "RecordingsService", file: recordingsv1.File_browser_automation_studio_v1_recordings_recordings_proto, service: "RecordingsService"},
		{name: "ReplayConfigService", file: replayconfigv1.File_browser_automation_studio_v1_replay_config_replay_config_proto, service: "ReplayConfigService"},
		{name: "ScenariosService", file: scenariosv1.File_browser_automation_studio_v1_scenarios_scenarios_proto, service: "ScenariosService"},
		{name: "SchedulesService", file: schedulesv1.File_browser_automation_studio_v1_schedules_schedules_proto, service: "SchedulesService"},
		{name: "SchemaService", file: schemav1.File_browser_automation_studio_v1_schema_schema_proto, service: "SchemaService"},
		{name: "SessionProfilesService", file: sessionprofilesv1.File_browser_automation_studio_v1_session_profiles_session_profiles_proto, service: "SessionProfilesService"},
		{name: "UXMetricsService", file: uxmetricsv1.File_browser_automation_studio_v1_uxmetrics_uxmetrics_proto, service: "UXMetricsService"},
	}

	for _, test := range services {
		t.Run(test.name, func(t *testing.T) {
			cliapp.RequireProtoServiceCoverage(t, manifest, test.file, test.service)
		})
	}
}
