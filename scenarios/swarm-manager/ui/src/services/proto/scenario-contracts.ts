import type { Scenario } from "@vrooli/proto-types/swarm-manager/v1/domain/scenario_pb";
import {
  ListScenariosResponseSchema,
  ScenarioResponseSchema,
  DeleteScenarioResponseSchema,
  ScenarioFilesResponseSchema,
  DeleteScenarioRequestSchema,
  PreserveFilesRequestSchema,
  SpecSyncArchiveRequestSchema,
  SpecSyncArchiveResponseSchema,
  type DeleteScenarioResponse,
  type SpecSyncArchiveResponse,
} from "@vrooli/proto-types/swarm-manager/v1/api/scenarios_pb";
import type { ScenarioFile } from "@vrooli/proto-types/swarm-manager/v1/api/scenarios_pb";
import type {
  Scenario as ScenarioDomain,
  ScenarioFile as ScenarioFileDomain,
  DeleteScenarioResponse as DeleteScenarioDomain,
} from "../../types";
import { SCENARIO_STATUSES } from "../../types";
import { createProtoSchema, isFileType, toFiniteNumber } from "./shared";

const scenarioStatusSet = new Set<string>(SCENARIO_STATUSES);

function isScenarioStatus(value: unknown): value is ScenarioDomain["status"] {
  return typeof value === "string" && scenarioStatusSet.has(value);
}

export const listScenariosResponseSchema = createProtoSchema(
  ListScenariosResponseSchema,
  "scenarios list"
);
export const scenarioResponseSchema = createProtoSchema(
  ScenarioResponseSchema,
  "scenario"
);
export const deleteScenarioResponseSchema = createProtoSchema(
  DeleteScenarioResponseSchema,
  "scenario delete"
);
export const specSyncArchiveResponseSchema = createProtoSchema(
  SpecSyncArchiveResponseSchema,
  "spec-sync-archive"
);
export const scenarioFilesResponseSchema = createProtoSchema(
  ScenarioFilesResponseSchema,
  "scenario files"
);

export { DeleteScenarioRequestSchema, PreserveFilesRequestSchema, SpecSyncArchiveRequestSchema };

export function mapProtoScenario(protoScenario: Scenario): ScenarioDomain {
  const status = isScenarioStatus(protoScenario.status) ? protoScenario.status : "unknown";
  return {
    name: protoScenario.name ?? "",
    displayName: protoScenario.displayName ?? "",
    description: protoScenario.description ?? "",
    status,
    priority: protoScenario.priority ?? 0,
    completenessScore:
      typeof protoScenario.completenessScore === "number"
        ? protoScenario.completenessScore
        : undefined,
    isGreenfield: protoScenario.isGreenfield ?? false,
    tags: protoScenario.tags ?? [],
  };
}

export function mapProtoScenarioFile(protoFile: ScenarioFile): ScenarioFileDomain {
  const size = toFiniteNumber(protoFile.size);
  const children = protoFile.children?.map(mapProtoScenarioFile) ?? [];
  const fileType = isFileType(protoFile.type) ? protoFile.type : "file";
  return {
    name: protoFile.name ?? "",
    path: protoFile.path ?? "",
    type: fileType,
    ...(size !== undefined ? { size } : {}),
    ...(children.length > 0 ? { children } : {}),
  };
}

export function mapDeleteScenarioResponse(
  protoResponse: DeleteScenarioResponse
): DeleteScenarioDomain {
  return {
    name: protoResponse.name,
    archived: protoResponse.archived,
    message: protoResponse.message,
    backlogIdeaName: protoResponse.backlogIdeaName,
    preservedFiles: protoResponse.preservedFiles,
  };
}

export function mapSpecSyncArchiveResponse(
  protoResponse: SpecSyncArchiveResponse
): { executionId: string; status: string; message: string } {
  return {
    executionId: protoResponse.executionId,
    status: protoResponse.status,
    message: protoResponse.message,
  };
}
