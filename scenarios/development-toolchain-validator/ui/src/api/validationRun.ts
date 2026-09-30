import { createClient } from "@connectrpc/connect";
import {
  ValidationRunService,
  Status as ValidationRunStatus,
  type ValidationRun,
  type StartResponse,
  type GetResponse as GetValidationRunResponse,
  type ListActiveResponse,
} from "@vrooli/proto-types/development-toolchain-validator/v1/validation_run/validation_run_pb";

import { transport } from "./client";

export const validationRunClient = createClient(ValidationRunService, transport);

export { ValidationRunStatus };
export type {
  ValidationRun,
  StartResponse,
  GetValidationRunResponse,
  ListActiveResponse,
};
