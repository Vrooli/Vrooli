import { createClient } from "@connectrpc/connect";
import {
  CommercialUse,
  ModelsService,
  type AddCustomModelResponse,
  type BlocklistEntry,
  type CapabilityLabels,
  type Hardware,
  type InstallModelResponse,
  type InstallState,
  type ListBlocklistResponse,
  type ListDefaultsResponse,
  type ListModelsResponse,
  type ListOperationsResponse,
  type Model,
  type OpDefault,
  type RemoveModelResponse,
  type SetDefaultModelResponse,
  type SetModelEnabledResponse,
} from "@vrooli/proto-types/image-tools/v1/models/models_pb";

import { transport } from "./client";

export const modelsClient = createClient(ModelsService, transport);

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
};
