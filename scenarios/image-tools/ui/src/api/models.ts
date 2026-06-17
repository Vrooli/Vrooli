import { createClient } from "@connectrpc/connect";
import {
  ModelsService,
  type Model,
  type ListModelsResponse,
  type ListOperationsResponse,
  type SetModelEnabledResponse,
} from "@vrooli/proto-types/image-tools/v1/models/models_pb";

import { transport } from "./client";

export const modelsClient = createClient(ModelsService, transport);

export type { Model, ListModelsResponse, ListOperationsResponse, SetModelEnabledResponse };
