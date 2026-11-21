/**
 * Cross-domain test data factories.
 *
 * Each `make<Domain>(overrides?)` returns a stable default instance that
 * tests selectively override via `MessageInitShape<Schema>`. Defaults
 * are picked so the most common test path is `makeX()` with no args.
 *
 * Domain-specific factories live next to the feature they double for
 * (for example, `features/validation/mocks/factories.ts`); only truly cross-domain
 * shapes (HealthResponse, error envelopes) live here. Deleting a feature
 * folder takes its factories with it — no central residue.
 *
 * Naming: `make<Domain>` (camelCase) — the TS analogue of the Go-side
 * `Fake<Domain>`. Asymmetry is deliberate: Go fakes are stateful types
 * (`type FakeClock struct{...}`); TS factories return plain proto
 * messages (`HealthResponse`).
 *
 * # Wire shape lives in proto, not here
 *
 * The HealthResponse type is a GENERATED proto message at
 * `packages/proto/gen/typescript/js/api-health/v1/shared/...`.
 * Factories use `create(<Schema>, overrides)` so the runtime instance
 * includes proto's internal `$typeName` / reflection state, field
 * defaults match proto3 semantics, and adding a field to the proto
 * schema makes it instantly available without editing this file.
 */
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import { MaturityAssessmentSchema } from "@vrooli/proto-types/common/v1/maturity_pb";
import { ResponseSchema } from "@vrooli/proto-types/api-health/v1/shared/health_pb";
import type { Response as HealthResponse } from "@vrooli/proto-types/api-health/v1/shared/health_pb";
import {
  FixResponseSchema,
  ValidateScenarioResponseSchema,
  ValidationStatus,
} from "@vrooli/proto-types/scenario-validation/v1/validation_pb";
import type {
  FixResponse,
  ValidateScenarioResponse,
} from "@vrooli/proto-types/scenario-validation/v1/validation_pb";

import type { ValidationNativeDetail, ValidationReport } from "../api/validation";

export type { HealthResponse };
export type { FixResponse, ValidateScenarioResponse, ValidationReport };

// MessageInitShape<typeof ResponseSchema> is the @bufbuild/protobuf-provided
// type for the optional fields you can pass to `create()`. Using it instead
// of `Partial<HealthResponse>` avoids a TS conflict over the required
// `$typeName` literal — `create()` fills that in for you, but
// `Partial<HealthResponse>` would let callers set it to a wrong value.
export const makeHealthResponse = (
  overrides: MessageInitShape<typeof ResponseSchema> = {},
): HealthResponse =>
  create(ResponseSchema, {
    status: "healthy",
    service: "react-vite-test",
    timestamp: "2026-01-01T00:00:00.000Z",
    readiness: true,
    version: "1.0.0",
    ...overrides,
  });

export const makeValidationResponse = (
  overrides: MessageInitShape<typeof ValidateScenarioResponseSchema> = {},
): ValidateScenarioResponse =>
  create(ValidateScenarioResponseSchema, {
    scenario: "api-health",
    status: ValidationStatus.FAILED,
    assessment: create(MaturityAssessmentSchema, {
      scenario: "api-health",
      provider: "api-health",
      phase: "api-health",
      findings: [
        {
          code: "APIH_LIFE_MISSING_HEALTH_METADATA",
          severity: "SEVERITY_ERROR",
          title: "Missing health metadata",
          message: "service.json does not declare API health metadata",
          location: ".vrooli/service.json",
          remediation: "Declare the API health path and check URL.",
          autofixAvailable: true,
          fixClass: "autofix",
        },
      ],
      findingsBySeverity: {
        SEVERITY_ERROR: 1,
        SEVERITY_WARNING: 0,
        SEVERITY_INFO: 0,
      },
      capabilities: [
        {
          id: "api-lifecycle",
          label: "API lifecycle",
          currentLevel: "L1",
          nextLevel: "L2",
          blockingFindingCodes: ["APIH_LIFE_MISSING_HEALTH_METADATA"],
        },
      ],
    }),
    ...overrides,
  });

export const makeValidationNativeDetail = (
  overrides: ValidationNativeDetail = {},
): ValidationNativeDetail => ({
  scenario: "api-health",
  target: {
    scenario: "api-health",
    resolution: "resolved",
    api_kind: "go-api",
    health_probe: {
      requested: true,
      url: "http://127.0.0.1:15001/health",
      status_code: 503,
      content_type: "application/json",
      elapsed_millis: 17,
      failure_class: "degraded",
      schema_valid: true,
      payload: {
        status: "degraded",
        service: "api-health",
        timestamp: "2026-01-01T00:00:00Z",
        readiness: false,
        dependency_count: 1,
      },
    },
  },
  summary: {
    errors: 1,
    warnings: 0,
    infos: 0,
    passed: false,
  },
  ...overrides,
});

export const makeValidationReport = (
  overrides: {
    response?: MessageInitShape<typeof ValidateScenarioResponseSchema>;
    nativeDetail?: ValidationNativeDetail;
  } = {},
): ValidationReport => ({
  response: makeValidationResponse(overrides.response),
  nativeDetail: makeValidationNativeDetail(overrides.nativeDetail),
});

export const makeFixResponse = (
  overrides: MessageInitShape<typeof FixResponseSchema> = {},
): FixResponse =>
  create(FixResponseSchema, {
    scenario: "api-health",
    applied: false,
    candidates: [
      {
        ruleId: "APIH_LIFE_MISSING_HEALTH_METADATA",
        filePath: ".vrooli/service.json",
        description: "Add /health API metadata",
        before: "{}",
        after: "{\"health\":\"/health\"}",
      },
    ],
    ...overrides,
  });
