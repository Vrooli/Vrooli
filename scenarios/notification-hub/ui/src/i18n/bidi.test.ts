import { describe, expect, it } from "vitest";

import { isolateInterpolation } from "./bidi";

const FSI = "\u2068";
const PDI = "\u2069";

describe("isolateInterpolation", () => {
  it("isolates a Latin value inside an Arabic sentence so its punctuation stays attached", () => {
    expect(isolateInterpolation("Set B, keep the character?", undefined, "ar")).toBe(`${FSI}Set B, keep the character?${PDI}`);
    expect(isolateInterpolation("Set B", undefined, "ar-EG")).toBe(`${FSI}Set B${PDI}`);
  });

  it("isolates an Arabic value inside an English or Japanese sentence", () => {
    expect(isolateInterpolation("مرحبا", undefined, "en")).toBe(`${FSI}مرحبا${PDI}`);
    expect(isolateInterpolation("مرحبا", undefined, "ja")).toBe(`${FSI}مرحبا${PDI}`);
  });

  it("leaves a value in the sentence's own direction exactly as written", () => {
    expect(isolateInterpolation("Set B", undefined, "en")).toBe("Set B");
    expect(isolateInterpolation("مرحبا", undefined, "ar")).toBe("مرحبا");
  });

  it("leaves numbers and identifiers without letters alone, in any locale", () => {
    expect(isolateInterpolation(3, undefined, "ar")).toBe("3");
    expect(isolateInterpolation("12:30", undefined, "ar")).toBe("12:30");
    expect(isolateInterpolation("2026-11-14", undefined, "ar")).toBe("2026-11-14");
  });

  it("does not throw on a missing locale or a non-string value", () => {
    expect(isolateInterpolation("Set B")).toBe("Set B");
    expect(isolateInterpolation(undefined)).toBe("undefined");
  });
});
