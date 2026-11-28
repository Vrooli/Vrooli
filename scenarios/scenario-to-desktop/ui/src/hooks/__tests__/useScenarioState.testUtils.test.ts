import { describe, expect, it } from "vitest";
import { createSaveStateResponse } from "./useScenarioState.testUtils";

describe("createSaveStateResponse", () => {
  it("uses the documented deterministic updated_at default", () => {
    expect(createSaveStateResponse().updated_at).toBe("2024-01-03T00:00:00Z");
  });

  it("honors an updated_at override", () => {
    expect(
      createSaveStateResponse({ updated_at: "2024-02-04T05:06:07Z" }).updated_at,
    ).toBe("2024-02-04T05:06:07Z");
  });
});
