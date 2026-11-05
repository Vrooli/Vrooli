/**
 * Self-tests for the models-domain proto-typed test factories. Co-located
 * with the feature so deleting `features/models/` takes them along.
 *
 * Same contract as the central `test-utils/factories.test.ts`:
 *
 *   - sane defaults make the most common test path `makeX()` no-args
 *   - overrides merge field-level (no all-or-nothing replacement)
 *   - the returned instance round-trips through proto's
 *     `toJson` / `fromJson` byte-identically
 */
import { fromJson, toJson } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";

import {
  ModelSchema,
  type Model,
} from "@vrooli/proto-types/image-tools/v1/models/models_pb";

import { makeListModelsResponse, makeModel } from "./factories";

describe("makeModel", () => {
  it("returns a model with non-empty id/name/tier/backend and enabled default", () => {
    const m = makeModel();
    expect(m.id).not.toBe("");
    expect(m.name).not.toBe("");
    expect(m.tier).not.toBe("");
    expect(m.backend).not.toBe("");
    expect(m.enabled).toBe(true);
  });

  it("merges overrides without dropping defaults", () => {
    const m = makeModel({ id: "custom-1", enabled: false });
    expect(m.id).toBe("custom-1");
    expect(m.enabled).toBe(false);
    expect(m.name).not.toBe("");
  });

  it("round-trips through ModelSchema JSON encode + decode", () => {
    const original = makeModel({ id: "rt-1", tier: "quality" });
    const decoded = fromJson(ModelSchema, toJson(ModelSchema, original));
    expect(decoded.id).toBe("rt-1");
    expect(decoded.tier).toBe("quality");
    expect(decoded.enabled).toBe(original.enabled);
  });
});

describe("makeListModelsResponse", () => {
  it("defaults to an empty models array (proto3: arrays default to [], not undefined)", () => {
    const r = makeListModelsResponse();
    expect(r.models).toEqual([]);
  });

  it("accepts model overrides", () => {
    const r = makeListModelsResponse({ models: [makeModel({ id: "a" }), makeModel({ id: "b" })] });
    expect(r.models.map((m: Model) => m.id)).toEqual(["a", "b"]);
  });
});
