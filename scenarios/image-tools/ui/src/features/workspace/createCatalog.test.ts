/**
 * createCatalog tests — the pure pieces of the Create surface: the
 * variation-key parser (which turns the backend's terminal job message into the
 * full set of output blob keys) and the presentation/preset tables.
 */
import { describe, expect, it } from "vitest";

import {
  CREATE_CATALOG,
  DEFAULT_SIZE_PRESET,
  SIZE_PRESETS,
  VARIATION_OPTIONS,
  createPresentation,
  parseVariationKeys,
} from "./createCatalog";

describe("parseVariationKeys", () => {
  it("parses every key from a multi-variation job message", () => {
    const keys = parseVariationKeys("variations: [out/a.png out/b.png out/c.png]", "out/a.png");
    expect(keys).toEqual(["out/a.png", "out/b.png", "out/c.png"]);
  });

  it("includes the primary key (first element) and preserves order", () => {
    const keys = parseVariationKeys("variations: [out/0.png out/1.png]", "out/0.png");
    expect(keys[0]).toBe("out/0.png");
    expect(keys).toHaveLength(2);
  });

  it("falls back to the single primary ref when the message has no marker", () => {
    expect(parseVariationKeys("produced 1/1", "out/only.png")).toEqual(["out/only.png"]);
  });

  it("falls back to the primary ref for an empty bracket list", () => {
    expect(parseVariationKeys("variations: []", "out/only.png")).toEqual(["out/only.png"]);
  });

  it("returns an empty array when there is neither a list nor a primary ref", () => {
    expect(parseVariationKeys("", "")).toEqual([]);
  });

  it("tolerates extra surrounding text and whitespace", () => {
    const keys = parseVariationKeys("done — variations: [  k0   k1  ] ok", "k0");
    expect(keys).toEqual(["k0", "k1"]);
  });
});

describe("createPresentation", () => {
  it("maps every generation op to a label/desc/icon", () => {
    for (const op of [
      "text_to_image",
      "image_to_image",
      "edit_instruct",
      "inpaint",
      "object_removal",
      "outpaint",
      "background_replace",
    ]) {
      const meta = createPresentation(op);
      expect(meta).toBeDefined();
      // lucide icons are forwardRef components (objects), not plain functions.
      expect(meta?.Icon).toBeTruthy();
    }
    expect(Object.keys(CREATE_CATALOG)).toContain("text_to_image");
    expect(Object.keys(CREATE_CATALOG)).toContain("outpaint");
    expect(Object.keys(CREATE_CATALOG)).toContain("background_replace");
  });

  it("returns undefined for an unknown op", () => {
    expect(createPresentation("nope")).toBeUndefined();
  });
});

describe("presets", () => {
  it("offers square as the default and three sizes", () => {
    expect(DEFAULT_SIZE_PRESET.key).toBe("square");
    expect(SIZE_PRESETS).toHaveLength(3);
    expect(SIZE_PRESETS[0]).toBe(DEFAULT_SIZE_PRESET);
  });

  it("offers 1..4 variation counts", () => {
    expect(VARIATION_OPTIONS).toEqual(["1", "2", "3", "4"]);
  });
});
