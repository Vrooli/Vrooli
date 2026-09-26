import { describe, expect, it } from "vitest";

import { AUTH_REQUIRED_ROUTES, PUBLIC_ROUTES, ROUTES, ROUTE_IDS, ROUTE_PATTERNS } from "./routes.generated";

describe("generated routes", () => {
  it("builds every dynamic route and keeps route metadata aligned", () => {
    expect(ROUTES.goldenDetail("gold")).toBe("/goldens/gold");
    expect(ROUTES.manifestEditor("skill", "gold")).toBe("/manifests/skill/gold");
    expect(ROUTES.runDetail("run")).toBe("/runs/run");
    expect(ROUTES.skillDetail("skill")).toBe("/skills/skill");
    expect(ROUTES.tupleDetail("gold", "tool", "test-genie")).toBe("/goldens/gold/tool/test-genie");
    expect(ROUTE_PATTERNS.tupleDetail).toBe("/goldens/:slug/:tupleKind/:subjectId");
    expect(PUBLIC_ROUTES).toEqual(ROUTE_IDS);
    expect(AUTH_REQUIRED_ROUTES).toEqual([]);
  });
});
