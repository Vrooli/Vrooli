import { createClient } from "@connectrpc/connect";
import {
  CommercialUse,
  ModelsService,
  type AddCustomModelResponse,
  type BackendReadiness,
  type BackendStatus,
  type BlocklistEntry,
  type CandidateModel,
  type CapabilityLabels,
  type DoctorBackendsResponse,
  type EnsureBackendResponse,
  type GetHostSummaryResponse,
  type Hardware,
  type HostSummary,
  type InstallModelResponse,
  type InstallState,
  type ListBlocklistResponse,
  type ListDefaultsResponse,
  type ListModelsResponse,
  type ListOperationModelsResponse,
  type ListOperationsResponse,
  type Model,
  type ModelFit,
  type OpDefault,
  type RemoveModelResponse,
  type SetDefaultModelResponse,
  type SetModelEnabledResponse,
} from "@vrooli/proto-types/image-tools/v1/models/models_pb";

import { transport } from "./client";

export const modelsClient = createClient(ModelsService, transport);

/**
 * listOperationModels returns every model serving an operation, each annotated
 * for THIS host (hardware fit + backend readiness + a single ready_state). It is
 * the data source for the model picker — the host-aware menu behind every AI
 * action — where `selectModel` only returns the one model that would run.
 */
export async function listOperationModels(operation: string): Promise<ListOperationModelsResponse> {
  return modelsClient.listOperationModels({ operation });
}

/**
 * getHostSummary returns this machine's AI-relevant hardware snapshot — used by
 * the model catalog to render hardware-fit affirmatively ("Runs on your GPU")
 * rather than as a static requirement chip.
 */
export async function getHostSummary(): Promise<GetHostSummaryResponse> {
  return modelsClient.getHostSummary({});
}

export { CommercialUse };
export type {
  Model,
  ListModelsResponse,
  ListOperationsResponse,
  SetModelEnabledResponse,
  InstallState,
  InstallModelResponse,
  RemoveModelResponse,
  AddCustomModelResponse,
  SetDefaultModelResponse,
  ListDefaultsResponse,
  ListBlocklistResponse,
  BlocklistEntry,
  CapabilityLabels,
  Hardware,
  OpDefault,
  BackendStatus,
  DoctorBackendsResponse,
  EnsureBackendResponse,
  ListOperationModelsResponse,
  CandidateModel,
  ModelFit,
  BackendReadiness,
  HostSummary,
  GetHostSummaryResponse,
};
