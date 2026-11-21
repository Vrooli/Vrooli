/**
 * opParams tests. Two pure helpers: `defaultParamsFor` seeds the controlled
 * form from an op's spec, and `toRequestParams` coerces controlled values into
 * the protojson shape the server expects (int64 `target_bytes` → string). The
 * branches that matter are the unknown-op fallback (no spec → empty values) and
 * the `target_bytes` stringify-vs-passthrough split.
 */
import { describe, expect, it } from "vitest";

import { defaultParamsFor, toRequestParams } from "./opParams";
import { OP_SPECS } from "./opSpecs";

describe("defaultParamsFor", () => {
  it("seeds every field of a known op with its spec default", () => {
    const params = defaultParamsFor("resize");
    // resize declares width/height/fit/gravity in opSpecs.
    expect(params).toEqual({ width: 256, height: 0, fit: "fit", gravity: "" });
  });

  it("carries each field's declared default value verbatim", () => {
    const params = defaultParamsFor("compress");
    const spec = OP_SPECS.compress;
    expect(spec).toBeDefined();
    for (const field of spec?.fields ?? []) {
      expect(params[field.name]).toBe(field.default);
    }
  });

  it("returns an empty object for an unknown op (no spec branch)", () => {
    expect(defaultParamsFor("does-not-exist")).toEqual({});
  });
});

describe("toRequestParams", () => {
  it("stringifies the int64 target_bytes field", () => {
    const out = toRequestParams({ target_bytes: 204800, quality: 80 });
    expect(out.target_bytes).toBe("204800");
    // Everything else maps 1:1 (the else branch).
    expect(out.quality).toBe(80);
  });

  it("passes through values 1:1 when there is no target_bytes key", () => {
    const input = { width: 256, fit: "fill", lossless: true };
    expect(toRequestParams(input)).toEqual(input);
  });

  it("stringifies a zero target_bytes (falsy value still hits the string branch)", () => {
    const out = toRequestParams({ target_bytes: 0 });
    expect(out.target_bytes).toBe("0");
  });

  it("returns an empty object for empty input", () => {
    expect(toRequestParams({})).toEqual({});
  });
});
