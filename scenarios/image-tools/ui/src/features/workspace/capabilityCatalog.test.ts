/**
 * Contract test for the generated capability catalog.
 *
 * The generated file is produced by `make capability-gen` from the Go manifest
 * seed; the Go drift test (`render_test.go`) guards byte-level freshness. This
 * test guards the UI-facing shape: every operation resolves to usable
 * presentation metadata, so an operation missing a hand-written catalog entry can
 * never render as a bare name with no icon.
 */
import { describe, expect, it } from "vitest";

import en from "../../i18n/locales/en.json";
import {
  CAPABILITIES,
  capabilityIcon,
  capabilityMeta,
  capabilitySummary,
  familyIcon,
} from "./capabilityCatalog";

describe("generated capability catalog", () => {
  it("covers every operation and names the engine that runs it", () => {
    const ids = Object.keys(CAPABILITIES);
    expect(ids.length).toBeGreaterThanOrEqual(70);
    expect(new Set(ids).size).toBe(ids.length);
    expect(capabilityMeta("text_to_image")?.engine).toBe("ai");
    expect(capabilityMeta("resize")?.engine).toBe("compute");
    for (const [id, meta] of Object.entries(CAPABILITIES)) {
      expect(meta.engine, id).toMatch(/^(ai|analysis|compute|unwired)$/);
    }
  });

  it("declares panel membership separately from family", () => {
    // The two axes are independent: `background_removal` is an extraction that a
    // person finds on the enhance surface, which is exactly what the old single
    // `category` field could not express.
    expect(capabilityMeta("text_to_image")?.panel).toBe("create");
    expect(capabilityMeta("background_removal")?.panel).toBe("enhance");
    expect(capabilityMeta("background_removal")?.family).toBe("extract");
    expect(capabilityMeta("caption")?.panel).toBe("analyze");
    expect(capabilityMeta("resize")?.panel).toBeUndefined();
    for (const [id, meta] of Object.entries(CAPABILITIES)) {
      if (meta.panel !== undefined) {
        expect(["create", "enhance", "analyze"], id).toContain(meta.panel);
      }
    }
  });

  it("gives every entry a family, mechanism, and summary", () => {
    for (const [id, meta] of Object.entries(CAPABILITIES)) {
      expect(meta.id, id).toBe(id);
      expect(meta.family, id).toBeTruthy();
      expect(meta.mechanism, id).toMatch(/^(compute|infer|fetch)$/);
      expect(meta.summary.length, id).toBeGreaterThan(0);
      expect(meta.outputKind, id).toBeTruthy();
    }
  });

  it("resolves an icon for every operation", () => {
    for (const id of Object.keys(CAPABILITIES)) {
      expect(capabilityIcon(id), id).toBeTruthy();
    }
  });

  it("falls back to a family icon for unknown operations", () => {
    expect(capabilityIcon("no_such_operation")).toBeUndefined();
    expect(familyIcon(undefined)).toBeTruthy();
    expect(familyIcon("not-a-family")).toBeTruthy();
  });

  it("exposes the canonical summary for fallback copy", () => {
    expect(capabilitySummary("face_swap")).toContain("reference image");
    expect(capabilitySummary("no_such_operation")).toBeUndefined();
  });
});

describe("generated presentation keys", () => {
  const resolve = (root: unknown, path: string): unknown =>
    path.split(".").reduce<unknown>((acc, key) => {
      if (acc && typeof acc === "object" && key in (acc as Record<string, unknown>)) {
        return (acc as Record<string, unknown>)[key];
      }
      return undefined;
    }, root);

  it("resolves every manifest label/desc key in the English locale", () => {
    for (const [id, meta] of Object.entries(CAPABILITIES)) {
      if (meta.labelKey) {
        expect(resolve(en, meta.labelKey), `${id} label ${meta.labelKey}`).toBeTypeOf("string");
      }
      if (meta.descKey) {
        expect(resolve(en, meta.descKey), `${id} desc ${meta.descKey}`).toBeTypeOf("string");
      }
    }
  });

  it("resolves every swap-extent stage and preset key in the English locale", () => {
    const extent = CAPABILITIES["face_swap"]?.extent;
    expect(extent, "face_swap extent").toBeTruthy();
    for (const item of [...(extent?.stages ?? []), ...(extent?.presets ?? [])]) {
      expect(resolve(en, item.labelKey), `${item.id} label ${item.labelKey}`).toBeTypeOf("string");
      expect(resolve(en, item.descKey), `${item.id} desc ${item.descKey}`).toBeTypeOf("string");
    }
  });

  it("gives every operation with copy a resolvable icon", () => {
    for (const [id, meta] of Object.entries(CAPABILITIES)) {
      if (meta.labelKey) {
        expect(capabilityIcon(id), id).toBeTruthy();
      }
    }
  });
});
