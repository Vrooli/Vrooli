import { describe, expect, it } from "vitest";

import { LookKind, StepKind, looksClient } from "./looks";

describe("api/looks", () => {
  it("constructs the LooksService client", () => {
    expect(looksClient).toBeDefined();
    expect(typeof looksClient.listLooks).toBe("function");
    expect(typeof looksClient.compileLook).toBe("function");
    expect(typeof looksClient.renderPreview).toBe("function");
  });

  it("re-exports the proto enums with stable values", () => {
    expect(LookKind.FILM).toBe(2);
    expect(LookKind.STYLE).toBe(1);
    expect(StepKind.DETERMINISTIC).toBe(1);
    expect(StepKind.AI).toBe(2);
  });
});
