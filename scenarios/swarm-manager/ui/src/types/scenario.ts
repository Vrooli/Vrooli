/**
 * Scenario domain types.
 */

import type { Scenario as ProtoScenario, ScenarioHealthSnapshot as ProtoScenarioHealthSnapshot, ScenarioHealthPhase as ProtoScenarioHealthPhase } from "@vrooli/proto-types/swarm-manager/v1/domain/scenario_pb";
import type {
  ScenarioFile as ProtoScenarioFile,
  PreserveFilesRequest as ProtoPreserveFilesRequest,
  DeleteScenarioRequest as ProtoDeleteScenarioRequest,
  DeleteScenarioResponse as ProtoDeleteScenarioResponse,
  UpdateScenarioMetadataRequest as ProtoUpdateScenarioMetadataRequest,
} from "@vrooli/proto-types/swarm-manager/v1/api/scenarios_pb";
import type { ProtoMessage } from "./shared";

/**
 * Valid runtime states for a scenario
 */
export type ScenarioStatus = "running" | "stopped" | "error" | "unknown";


/**
 * A scenario represents a deployed application in the Vrooli ecosystem.
 * [REQ:REQ-P0-007] Includes metadata for greenfield toggle
 */
export type Scenario = Omit<ProtoMessage<ProtoScenario>, "status"> & {
  /** Current runtime state */
  status: ScenarioStatus;
};

export type ScenarioHealthPhase = ProtoMessage<ProtoScenarioHealthPhase>;
export type ScenarioHealthSnapshot = ProtoMessage<ProtoScenarioHealthSnapshot>;

/**
 * Request to update scenario metadata
 * [REQ:REQ-P0-007] Update metadata for greenfield
 */
export type UpdateScenarioMetadataRequest = ProtoMessage<ProtoUpdateScenarioMetadataRequest>;

/**
 * File type in the scenario file tree
 */
export type ScenarioFileType = "file" | "directory";

/**
 * Represents a file or directory within a scenario folder.
 */
export type ScenarioFile = Omit<ProtoMessage<ProtoScenarioFile>, "type" | "size" | "children"> & {
  /** Whether this is a file or directory */
  type: ScenarioFileType;
  /** File size in bytes (only for files) */
  size?: number;
  /** Child files (only for directories) */
  children?: ScenarioFile[];
};

/**
 * Available presets for file preservation during archive
 */
export type PreserveFilesPreset = "documentation" | "requirements" | "planning" | "all-planning";

/**
 * Request to specify which files to preserve when archiving
 */
export type PreserveFilesRequest = Partial<
  Omit<ProtoMessage<ProtoPreserveFilesRequest>, "preset">
> & {
  /** Preset name: "documentation", "requirements", "planning", "all-planning" */
  preset?: PreserveFilesPreset;
};

/**
 * Request body for DELETE /api/v1/scenarios/{name}
 */
export type DeleteScenarioRequest = Omit<ProtoMessage<ProtoDeleteScenarioRequest>, "preserveFiles"> & {
  /** Optional file preservation settings when archiving */
  preserveFiles?: PreserveFilesRequest;
};

/**
 * Response from scenario deletion
 * [REQ:REQ-P0-008] Deletion confirmation with archive status
 */
export type DeleteScenarioResponse = ProtoMessage<ProtoDeleteScenarioResponse>;

/**
 * Response from spec-sync-archive
 * Contains execution ID for progress polling
 */
export interface SpecSyncArchiveResponse {
  executionId: string;
  status: string;
  message: string;
}
