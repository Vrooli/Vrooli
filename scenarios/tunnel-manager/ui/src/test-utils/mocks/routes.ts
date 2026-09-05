/**
 * Mock builders for `api/routes`. Call `makeRoutesMocks()` from inside a
 * `vi.mock("../../api/routes", …)` factory closure (never at module top
 * level — see `test-utils/mocks/api.ts` for the hoisting rationale). The
 * `...actual` spread at the call site keeps re-exported enums/types intact.
 */
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import { vi } from "vitest";
import {
  RouteSchema,
  RouteSource,
  Tier,
  type Route,
} from "@vrooli/proto-types/tunnel-manager/v1/routes/routes_pb";

/** A scenario-backed route; override any field per test. */
export const makeRoute = (overrides: MessageInitShape<typeof RouteSchema> = {}): Route =>
  create(RouteSchema, {
    id: "route-scenario-1",
    subdomain: "agent-manager",
    scenario: "agent-manager",
    domain: "itsagitime.com",
    localPort: 21001,
    tier: Tier.CORE,
    enabled: true,
    healthPath: "/health",
    publicUrl: "https://agent-manager.itsagitime.com",
    source: RouteSource.SCENARIO,
    ...overrides,
  });

/** An external route pointing at an arbitrary local service target. */
export const makeExternalRoute = (overrides: MessageInitShape<typeof RouteSchema> = {}): Route =>
  makeRoute({
    id: "route-external-1",
    subdomain: "my-service",
    scenario: "",
    publicUrl: "https://my-service.itsagitime.com",
    source: RouteSource.EXTERNAL,
    serviceTarget: "http://127.0.0.1:9000",
    ...overrides,
  });

export const makeRoutesMocks = () => ({
  routesClient: {
    listRoutes: vi.fn().mockResolvedValue({ routes: [makeRoute(), makeExternalRoute()] }),
    getRoute: vi.fn().mockResolvedValue({ route: makeRoute() }),
    createRoute: vi.fn().mockResolvedValue({ route: makeExternalRoute() }),
    updateRoute: vi.fn().mockResolvedValue({ route: makeExternalRoute() }),
    deleteRoute: vi.fn().mockResolvedValue({ deleted: true }),
  },
  listRoutes: vi.fn().mockResolvedValue([makeRoute(), makeExternalRoute()]),
  createExternalRoute: vi.fn().mockResolvedValue({ route: makeExternalRoute() }),
  deleteRoute: vi.fn().mockResolvedValue({ deleted: true }),
});
