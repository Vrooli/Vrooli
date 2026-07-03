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
import { ResponseSchema } from "@vrooli/proto-types/api-health/v1/shared/health_pb";
import type { Response as HealthResponse } from "@vrooli/proto-types/api-health/v1/shared/health_pb";

export type { HealthResponse };

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
