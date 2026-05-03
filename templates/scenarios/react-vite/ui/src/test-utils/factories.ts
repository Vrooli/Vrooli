/**
 * Test data factories.
 *
 * Each `make<Domain>(overrides?)` returns a stable default instance that
 * tests selectively override via `Partial<Domain>`. Defaults are picked
 * so the most common test path is `makeX()` with no args.
 *
 * Naming: `make<Domain>` (camelCase) — the TS analogue of workspace-
 * sandbox's Go-side `Fake<Domain>`. Asymmetry is deliberate: Go fakes
 * are stateful types (`type FakeClock struct{...}`); TS factories
 * return plain proto messages (`HealthResponse`).
 *
 * # Wire shape lives in proto, not here
 *
 * The HealthResponse type is the GENERATED proto message at
 * `packages/proto/gen/typescript/{{SCENARIO_ID}}/v1/health/health_pb.ts`.
 * Factories use `create(ResponseSchema, overrides)` so:
 *
 *   - the runtime instance includes proto's internal `$typeName` /
 *     reflection state (necessary for `toJson` / `fromJson` round-trips
 *     in tests that exercise the full pipeline);
 *   - field defaults match proto3 semantics (numbers default to 0,
 *     strings to "", maps to {} — never `undefined`);
 *   - adding a field to the proto schema makes it instantly available
 *     in factories without editing this file (because `Partial<Type>`
 *     widens with the type).
 *
 * If a test never round-trips through JSON and only reads a few fields,
 * a plain object cast may look tempting — resist it. The factory's
 * point is that every test sees the same instance shape as production
 * code that fetches from the API.
 */
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import { ResponseSchema } from "@vrooli/proto-types/{{SCENARIO_ID}}/v1/health/health_pb";
import type { Response as HealthResponse } from "@vrooli/proto-types/{{SCENARIO_ID}}/v1/health/health_pb";

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
