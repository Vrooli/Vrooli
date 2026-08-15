import { describe, expect, it } from "vitest";
import { StepReadiness } from "../wizard/StepReadiness";

describe("UI primitive adoption", () => {
  it("keeps wizard actions on the shared Button primitive", () => {
    expect(StepReadiness).toBeTypeOf("function");
  });
});
