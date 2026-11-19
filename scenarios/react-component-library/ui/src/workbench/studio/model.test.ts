import { describe, expect, it } from "vitest";

import type { DesignDocument } from "../designs";
import { DEFAULT_TEMPLATE } from "../templates/library";
import { node } from "../templates/spec";
import {
  PALETTE_KEYS,
  docSignature,
  duplicateInput,
  firstNodeOfType,
  frameForDevice,
  frameTemplate,
  paletteTokens,
  rekeySpec,
  seedInput,
  themePatch,
  visibleLayerRows,
} from "./model";

describe("studio model", () => {
  it("seeds one desktop, tablet and phone frame from a template", () => {
    const input = seedInput(DEFAULT_TEMPLATE);
    expect(input.name).toBe(DEFAULT_TEMPLATE.name);
    expect(input.frames.map((frame) => frame.device)).toEqual(["desktop", "tablet", "phone"]);
    expect(input.frames.map((frame) => frame.spec.type)).toEqual(["page", "page", "page"]);
    expect(input.theme).toEqual(DEFAULT_TEMPLATE.palette);
    expect(input.options).toEqual({
      radius: DEFAULT_TEMPLATE.radius,
      display: DEFAULT_TEMPLATE.display,
    });
    const ids = input.frames.flatMap((frame) => frame.spec.id);
    expect(new Set(ids).size).toBe(3);
  });

  it("rekeys a tree without sharing node ids", () => {
    const spec = node("root", "page", {}, [node("a", "text", { text: "one" }), node("b", "text", { text: "two" })]);
    const first = rekeySpec(spec, "frame.a");
    const second = rekeySpec(spec, "frame.b");
    const collect = (value: typeof spec): string[] => [value.id, ...(value.children ?? []).flatMap(collect)];
    expect(collect(first)).toEqual(["frame.a.root", "frame.a.a", "frame.a.b"]);
    expect(new Set([...collect(first), ...collect(second)]).size).toBe(6);
  });

  it("duplicates a design with fresh frame and layer ids", () => {
    const source = seedInput(DEFAULT_TEMPLATE);
    const doc: DesignDocument = {
      id: "design-1",
      name: source.name,
      updatedAt: new Date(0).toISOString(),
      revision: 2,
      frames: source.frames,
      theme: source.theme,
      options: { radius: 9, display: "sans" },
    };
    const copy = duplicateInput(doc, "Aperture copy");
    expect(copy.name).toBe("Aperture copy");
    expect(copy.options).toEqual({ radius: 9, display: "sans" });
    expect(copy.frames).toHaveLength(3);
    for (const [index, frame] of copy.frames.entries()) {
      const original = doc.frames[index];
      expect(frame.id).not.toBe(original?.id);
      expect(frame.spec.id).not.toBe(original?.spec.id);
    }

    // A fixture from before options existed duplicates to an empty chrome
    // object, not undefined.
    const legacy: DesignDocument = {
      id: "design-2",
      name: source.name,
      updatedAt: new Date(0).toISOString(),
      revision: 1,
      frames: source.frames,
      theme: source.theme,
    };
    expect(duplicateInput(legacy, "Legacy copy").options).toEqual({});
  });

  it("fingerprints content but ignores revision metadata", () => {
    const source = seedInput(DEFAULT_TEMPLATE);
    const doc: DesignDocument = {
      id: "design-1",
      name: source.name,
      updatedAt: new Date(0).toISOString(),
      revision: 1,
      frames: source.frames,
      theme: source.theme,
      options: source.options,
    };
    expect(docSignature(doc)).toBe(docSignature({ ...doc, revision: 9, updatedAt: new Date().toISOString() }));
    expect(docSignature(doc)).not.toBe(docSignature({ ...doc, theme: { ...doc.theme, accent: "#000000" } }));
    expect(docSignature(doc)).not.toBe(
      docSignature({ ...doc, options: { ...doc.options, radius: DEFAULT_TEMPLATE.radius + 6 } }),
    );
  });

  it("renders a design's own chrome and falls back for older fixtures", () => {
    const source = seedInput(DEFAULT_TEMPLATE);
    const frame = source.frames[0];
    if (!frame) throw new Error("seedInput produced no frames");
    const doc: DesignDocument = {
      id: "design-1",
      name: source.name,
      updatedAt: new Date(0).toISOString(),
      revision: 1,
      frames: source.frames,
      theme: source.theme,
      options: { radius: 36, display: "sans" },
    };
    expect(frameTemplate(doc, frame)).toMatchObject({ radius: 36, display: "sans" });

    // A fixture written before options existed renders with library defaults.
    const legacy: DesignDocument = {
      id: "design-2",
      name: source.name,
      updatedAt: new Date(0).toISOString(),
      revision: 1,
      frames: source.frames,
      theme: source.theme,
    };
    expect(frameTemplate(legacy, frame)).toMatchObject({
      radius: DEFAULT_TEMPLATE.radius,
      display: DEFAULT_TEMPLATE.display,
    });

    // One key set, the other falling back independently.
    expect(frameTemplate({ ...doc, options: { radius: 6 } }, frame)).toMatchObject({
      radius: 6,
      display: DEFAULT_TEMPLATE.display,
    });
  });

  it("folds collapsed groups away and keeps their children in the tree", () => {
    const spec = node("root", "page", {}, [
      node("group", "card", {}, [node("child", "text", {})]),
      node("sibling", "text", {}),
    ]);
    expect(visibleLayerRows(spec, new Set()).map((row) => row.node.id)).toEqual([
      "root",
      "group",
      "child",
      "sibling",
    ]);
    const rows = visibleLayerRows(spec, new Set(["group"]));
    expect(rows.map((row) => row.node.id)).toEqual(["root", "group", "sibling"]);
    expect(rows[1]).toMatchObject({ hasChildren: true, collapsed: true, depth: 1 });
  });

  it("finds the first layer of a type and the frame for a device", () => {
    const spec = node("root", "page", {}, [node("card", "card", {}, [node("button", "button", {})])]);
    expect(firstNodeOfType(spec, "button")).toBe("button");
    expect(firstNodeOfType(spec, "stat")).toBeNull();
    const input = seedInput(DEFAULT_TEMPLATE);
    expect(frameForDevice(input.frames, "phone")?.name).toBe("Phone");
  });

  it("lists palette tokens in source order and patches one", () => {
    const tokens = paletteTokens(DEFAULT_TEMPLATE.palette);
    expect(tokens.map((token) => token.key)).toEqual(PALETTE_KEYS);
    expect(tokens[0]?.value).toBe(DEFAULT_TEMPLATE.palette.bg);
    expect(themePatch(DEFAULT_TEMPLATE.palette, "accent", "#123456").accent).toBe("#123456");
  });
});
