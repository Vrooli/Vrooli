import { createClient } from "@connectrpc/connect";
import {
  ValidationService,
  Severity,
  type Finding,
  type Summary,
  type ValidateScenarioResponse,
} from "@vrooli/proto-types/security-health/v1/validation/validation_pb";

import { transport } from "./client";

/**
 * Connect-Web client for the ValidationService. The Posture and Secrets
 * surfaces call `validateScenario({ scenario })` and render the normalized
 * Finding list. Severity is the load-bearing contract: ERROR gates, WARNING
 * and INFO are advisory — the UI colors strictly off this enum, never off a
 * scanner's native string.
 */
export const validationClient = createClient(ValidationService, transport);

export { Severity };
export type { Finding, Summary, ValidateScenarioResponse };
