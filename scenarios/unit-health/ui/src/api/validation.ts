import { createClient } from "@connectrpc/connect";
import {
  ValidationService,
  type ValidateScenarioRequest,
  type ValidateScenarioResponse,
} from "@vrooli/proto-types/unit-health/v1/validation/validation_pb";

import { transport } from "./client";

export const validationClient = createClient(ValidationService, transport);

export type { ValidateScenarioRequest, ValidateScenarioResponse };
