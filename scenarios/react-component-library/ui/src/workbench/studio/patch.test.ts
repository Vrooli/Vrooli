import { describe, expect, it } from "vitest";

import type { DesignDocument } from "../designs";
import { applyPatch, PatchError } from "./patch";

function node(
  id: string,
  type: string,
  props: Record<string, string | number | boolean> = {},
  children?: DesignDocument["frames"][number]["spec"][],
): DesignDocument["frames"][number]["spec"] {
  return {
    id,
    type: type as DesignDocument["frames"][number]["spec"]["type"],
    props,
    ...(children ? { children } : {}),
  };
}

function sample(): DesignDocument {
  return {
    id: "design-1",
    name: "Sample",
    updatedAt: "2026-11-19T10:00:00.000Z",
    revision: 3,
    theme: {
      bg: "#0b0a0f",
      surface: "#141219",
      surfaceAlt: "#1c1922",
      ink: "#f4f1ea",
      muted: "#8b8592",
      line: "#2b2531",
      accent: "#ffb35c",
      accentInk: "#1a1206",
      accentSoft: "#3a2a14",
      secondary: "#7bd6a4",
    },
    frames: [
      {
        id: "desktop",
        name: "Desktop",
        device: "desktop",
        spec: node("page", "page", {}, [
          node("hero", "stack", { gap: 18 }, [
            node("title", "text", { text: "Old headline" }),
            node("cta", "button", { text: "Open", variant: "primary" }),
            { id: "kpi", type: "slot", props: { minH: 96 }, role: "metric" },
          ]),
          node("foot", "text", { text: "Footer" }),
        ]),
      },
    ],
  };
}

const ids = (parentId: string, type: string, sequence: number) => `${parentId}.${type}-${sequence}`;

describe("applyPatch", () => {
  it("rewrites copy and props without mutating the input", () => {
    const doc = sample();
    const before = JSON.stringify(doc);
    const applied = applyPatch(
      doc,
      [
        { kind: "setText", nodeId: "title", text: "Accounts growing this week" },
        { kind: "setProp", nodeId: "cta", key: "variant", value: "secondary" },
        { kind: "setTheme", key: "accent", value: "#7bd6a4" },
      ],
      { idFactory: ids },
    );
    expect(JSON.stringify(doc)).toBe(before);
    const hero = applied.doc.frames[0]?.spec.children?.[0];
    expect(hero?.children?.[0]?.props.text).toBe("Accounts growing this week");
    expect(hero?.children?.[1]?.props.variant).toBe("secondary");
    expect(applied.doc.theme.accent).toBe("#7bd6a4");
    expect(applied.changed).toEqual(["title", "cta", "theme.accent"]);
  });

  it("adds a layer with one authored level of children", () => {
    const applied = applyPatch(
      sample(),
      [
        {
          kind: "addLayer",
          parentId: "hero",
          node: {
            type: "card",
            props: { pad: 18 },
            children: [{ type: "text", props: { text: "New card" } }],
          },
        },
      ],
      { idFactory: ids },
    );
    const added = applied.doc.frames[0]?.spec.children?.[0]?.children?.at(-1);
    expect(added?.type).toBe("card");
    expect(added?.id).toBe("hero.card-1");
    expect(added?.children?.[0]?.id).toBe("hero.card-1.text-2");
    expect(added?.children?.[0]?.props.text).toBe("New card");
  });

  it("moves and removes in the order given", () => {
    const applied = applyPatch(
      sample(),
      [
        { kind: "moveLayer", nodeId: "foot", parentId: "hero", index: 0 },
        { kind: "removeLayer", nodeId: "cta" },
      ],
      { idFactory: ids },
    );
    const hero = applied.doc.frames[0]?.spec.children?.[0];
    expect(hero?.children?.map((child) => child.id)).toEqual(["foot", "title", "kpi"]);
  });

  it("rejects a patch that no longer matches the document", () => {
    expect(() =>
      applyPatch(sample(), [{ kind: "setText", nodeId: "missing", text: "x" }]),
    ).toThrow(PatchError);
    expect(() =>
      applyPatch(sample(), [{ kind: "removeLayer", nodeId: "page" }]),
    ).toThrow(/page root/);
    expect(() =>
      applyPatch(sample(), [{ kind: "removeLayer", nodeId: "kpi" }]),
    ).toThrow(/structural/);
    expect(() =>
      applyPatch(sample(), [{ kind: "moveLayer", nodeId: "hero", parentId: "title" }]),
    ).toThrow(/cannot own children/);
    expect(() =>
      applyPatch(sample(), [{ kind: "setProp", nodeId: "title", key: "src", value: "/x.png" }]),
    ).toThrow(/cannot be set/);
    expect(() =>
      applyPatch(sample(), [{ kind: "setProp", nodeId: "title", key: "x", value: 4 }]),
    ).toThrow(/only valid on a slot/);
    expect(() =>
      applyPatch(sample(), [
        { kind: "addLayer", parentId: "hero", node: { type: "page", props: {} } },
      ]),
    ).toThrow(/cannot be added/);
  });

  it("refuses a removed layer in a later operation", () => {
    expect(() =>
      applyPatch(sample(), [
        { kind: "removeLayer", nodeId: "title" },
        { kind: "setText", nodeId: "title", text: "again" },
      ]),
    ).toThrow(/does not exist/);
  });

  it("caps the patch size", () => {
    const operations = Array.from({ length: 9 }, () => ({
      kind: "setText" as const,
      nodeId: "title",
      text: "x",
    }));
    expect(() => applyPatch(sample(), operations)).toThrow(/1 to 8/);
  });
});
