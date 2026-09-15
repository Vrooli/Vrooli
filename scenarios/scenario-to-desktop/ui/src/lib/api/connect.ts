import { createClient } from "@connectrpc/connect";
import { createScenarioConnectTransport } from "@vrooli/api-base";
import { PipelineService } from "@vrooli/proto-types/scenario-to-desktop/v1/pipeline/types_pb";
import { PreflightService } from "@vrooli/proto-types/scenario-to-desktop/v1/domain/preflight_pb";
import { SigningService } from "@vrooli/proto-types/scenario-to-desktop/v1/domain/signing_pb";
import {
  BuildService,
  SmokeTestService,
} from "@vrooli/proto-types/scenario-to-desktop/v1/domain/build_pb";
import {
  ConfigService,
  SystemService,
} from "@vrooli/proto-types/scenario-to-desktop/v1/domain/config_pb";
import { EvidenceService } from "@vrooli/proto-types/scenario-to-desktop/v1/domain/evidence_pb";
import { TaskService } from "@vrooli/proto-types/scenario-to-desktop/v1/domain/tasks_pb";
import { StateService } from "@vrooli/proto-types/scenario-to-desktop/v1/domain/state_pb";
import { TelemetryService } from "@vrooli/proto-types/scenario-to-desktop/v1/domain/telemetry_pb";
import { DesktopRecordsService } from "@vrooli/proto-types/scenario-to-desktop/v1/domain/records_pb";
import { DocumentationService } from "@vrooli/proto-types/scenario-to-desktop/v1/domain/docs_pb";
import { OperationsService } from "@vrooli/proto-types/scenario-to-desktop/v1/domain/operations_pb";

// Every generated client shares the scenario-aware transport so local development,
// host proxies, and deployed desktop shells resolve the same API origin.
const transport = createScenarioConnectTransport();

export const pipelineConnectClient = createClient(PipelineService, transport);
export const preflightConnectClient = createClient(PreflightService, transport);
export const signingConnectClient = createClient(SigningService, transport);
export const buildConnectClient = createClient(BuildService, transport);
export const smokeTestConnectClient = createClient(SmokeTestService, transport);
export const systemConnectClient = createClient(SystemService, transport);
export const configConnectClient = createClient(ConfigService, transport);
export const evidenceConnectClient = createClient(EvidenceService, transport);
export const taskConnectClient = createClient(TaskService, transport);
export const stateConnectClient = createClient(StateService, transport);
export const telemetryConnectClient = createClient(TelemetryService, transport);
export const desktopRecordsConnectClient = createClient(
  DesktopRecordsService,
  transport,
);
export const documentationConnectClient = createClient(
  DocumentationService,
  transport,
);
export const operationsConnectClient = createClient(
  OperationsService,
  transport,
);
