import { describe, expect, it } from "vitest";

import { validationRunClient, ValidationRunStatus } from "./validationRun";

describe("api/validationRun", () => {
  it("exposes the ValidationRunService RPCs as client methods", () => {
    expect(typeof validationRunClient.start).toBe("function");
    expect(typeof validationRunClient.get).toBe("function");
    expect(typeof validationRunClient.listActive).toBe("function");
  });

  it("re-exports the Status enum", () => {
    expect(ValidationRunStatus).toBeDefined();
  });
});
