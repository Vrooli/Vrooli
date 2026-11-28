import { describe, expect, it } from "vitest";

import type { DesignDocument } from "../design-types";
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
    expect(() => applyPatch(sample(), [{ kind: "setText", nodeId: "missing", text: "x" }])).toThrow(
      PatchError,
    );
    expect(() => applyPatch(sample(), [{ kind: "removeLayer", nodeId: "page" }])).toThrow(
      /page root/,
    );
    expect(() => applyPatch(sample(), [{ kind: "removeLayer", nodeId: "kpi" }])).toThrow(
      /structural/,
    );
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

  it("rejects missing operations, invalid ids, and malformed operation fields", () => {
    expect(() => applyPatch(sample(), [])).toThrow(/1 to 8/);
    expect(() => applyPatch(sample(), [{ kind: "setText", nodeId: "bad id", text: "x" }])).toThrow(
      /not a valid id/,
    );
    expect(() =>
      applyPatch(sample(), [{ kind: "setText", nodeId: "title", text: undefined }]),
    ).toThrow(/text is required/);
    expect(() => applyPatch(sample(), [{ kind: "setText", nodeId: "hero", text: "x" }])).toThrow(
      /does not render text/,
    );
    expect(() => applyPatch(sample(), [{ kind: "setProp", nodeId: "title" }])).toThrow(
      /key is required/,
    );
    expect(() =>
      applyPatch(sample(), [{ kind: "setProp", nodeId: "title", key: "title", value: undefined }]),
    ).toThrow(/value is required/);
    expect(() =>
      applyPatch(sample(), [{ kind: "setTheme", key: "notAToken", value: "red" }]),
    ).toThrow(/not a palette token/);
    expect(() => applyPatch(sample(), [{ kind: "addLayer", parentId: "hero" }])).toThrow(
      /node is required/,
    );
    expect(() => applyPatch(sample(), [{ kind: "futureOperation" } as never])).toThrow(
      /unknown kind futureOperation/,
    );
  });

  it("validates text, prop values, and theme colour formats at their boundaries", () => {
    expect(() =>
      applyPatch(sample(), [{ kind: "setText", nodeId: "title", text: "line\u0001break" }]),
    ).toThrow(/control character/);
    expect(() =>
      applyPatch(sample(), [{ kind: "setText", nodeId: "title", text: "x".repeat(2001) }]),
    ).toThrow(/exceeds 2000/);
    expect(() =>
      applyPatch(sample(), [{ kind: "setProp", nodeId: "title", key: "bad-key", value: "x" }]),
    ).toThrow(/key .* is not allowed/);
    expect(() =>
      applyPatch(sample(), [{ kind: "setProp", nodeId: "title", key: "onClick", value: "x" }]),
    ).toThrow(/cannot be set/);
    expect(() =>
      applyPatch(sample(), [{ kind: "setProp", nodeId: "title", key: "_private", value: "x" }]),
    ).toThrow(/key .* is not allowed/);
    expect(() =>
      applyPatch(sample(), [{ kind: "setProp", nodeId: "title", key: "count", value: Infinity }]),
    ).toThrow(/out of range/);
    expect(() =>
      applyPatch(sample(), [{ kind: "setProp", nodeId: "title", key: "count", value: 100001 }]),
    ).toThrow(/out of range/);
    expect(() =>
      applyPatch(sample(), [
        { kind: "setProp", nodeId: "title", key: "count", value: null as unknown as number },
      ]),
    ).toThrow(/must be a string, number or boolean/);
    expect(() => applyPatch(sample(), [{ kind: "setTheme", key: "accent", value: "#xy" }])).toThrow(
      /must be a hex colour or a colour name/,
    );

    const applied = applyPatch(sample(), [
      { kind: "setText", nodeId: "title", text: "line\nwith\ttabs" },
      { kind: "setProp", nodeId: "title", key: "enabled", value: true },
      { kind: "setTheme", key: "accent", value: "coral" },
    ]);
    expect(applied.doc.frames[0]?.spec.children?.[0]?.children?.[0]?.props.text).toBe(
      "line\nwith\ttabs",
    );
    expect(applied.doc.theme.accent).toBe("coral");
  });

  it("enforces add-layer size, depth, and placement constraints", () => {
    expect(() =>
      applyPatch(sample(), [
        {
          kind: "addLayer",
          parentId: "hero",
          node: {
            type: "card",
            props: Object.fromEntries(Array.from({ length: 17 }, (_, i) => [`p${i}`, i])),
          },
        },
      ]),
    ).toThrow(/at most 16 props/);
    expect(() =>
      applyPatch(sample(), [
        {
          kind: "addLayer",
          parentId: "hero",
          node: {
            type: "card",
            props: {},
            children: Array.from({ length: 7 }, () => ({ type: "text" as const, props: {} })),
          },
        },
      ]),
    ).toThrow(/at most 6 children/);
    expect(() =>
      applyPatch(sample(), [
        {
          kind: "addLayer",
          parentId: "hero",
          node: {
            type: "card",
            props: {},
            children: [{ type: "stack", props: {}, children: [{ type: "text", props: {} }] }],
          },
        },
      ]),
    ).toThrow(/only nest one level/);
    const deepDoc = sample();
    let cursor = deepDoc.frames[0]!.spec;
    for (let depth = 2; depth <= 12; depth += 1) {
      const child = node(`deep-${depth}`, "stack");
      cursor.children = [child];
      cursor = child;
    }
    expect(() =>
      applyPatch(deepDoc, [
        { kind: "addLayer", parentId: "deep-11", node: { type: "card", props: {} } },
      ]),
    ).toThrow(/nest deeper than 12/);

    const fullDoc = sample();
    fullDoc.frames[0]!.spec.children![0]!.children!.push(
      ...Array.from({ length: 494 }, (_, index) => node(`extra-${index}`, "text")),
    );
    expect(() =>
      applyPatch(fullDoc, [
        { kind: "addLayer", parentId: "hero", node: { type: "text", props: {} } },
      ]),
    ).toThrow(/exceed 500 layers/);
  });

  it("rejects moves into descendants or outside a destination list", () => {
    const nested = sample();
    nested.frames[0]!.spec.children![0]!.children!.push(node("nested", "stack"));
    expect(() =>
      applyPatch(nested, [{ kind: "moveLayer", nodeId: "hero", parentId: "nested" }]),
    ).toThrow(/move into itself/);
    expect(() =>
      applyPatch(sample(), [{ kind: "moveLayer", nodeId: "hero", parentId: "hero" }]),
    ).toThrow(/move into itself/);
    expect(() =>
      applyPatch(sample(), [{ kind: "moveLayer", nodeId: "hero", parentId: "cta" }]),
    ).toThrow(/cannot own children/);
    expect(() =>
      applyPatch(sample(), [{ kind: "moveLayer", nodeId: "title", parentId: "title" }]),
    ).toThrow(/cannot own children/);
    expect(() =>
      applyPatch(sample(), [{ kind: "moveLayer", nodeId: "kpi", parentId: "hero", index: -1 }]),
    ).toThrow(/outside the destination/);
    expect(() =>
      applyPatch(sample(), [{ kind: "moveLayer", nodeId: "kpi", parentId: "hero", index: 99 }]),
    ).toThrow(/outside the destination/);
    expect(() =>
      applyPatch(sample(), [{ kind: "moveLayer", nodeId: "title", parentId: "hero", index: 0.5 }]),
    ).toThrow(/outside the destination/);
  });
});
