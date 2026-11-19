import { describe, expect, it } from "vitest";

import { node } from "../templates/spec";
import { extractionCandidates } from "./extract";

/** A custom "card": two text layers under a surface. */
function card(id: string, title: string, body: string, pad: number) {
  return node(`${id}.card`, "surface", { pad }, [
    node(`${id}.title`, "text", { text: title }),
    node(`${id}.body`, "text", { text: body }),
  ]);
}

describe("extractionCandidates", () => {
  it("groups copies whose copy, ids and prop values differ", () => {
    const spec = node("page", "page", {}, [
      card("a", "Starter", "For one seat", 16),
      card("b", "Team", "For the whole crew", 24),
    ]);

    const candidates = extractionCandidates(spec);
    expect(candidates).toHaveLength(1);
    expect(candidates[0]).toMatchObject({
      nodeCount: 3,
      occurrences: 2,
      exampleIds: ["a.card", "b.card"],
      score: 3,
    });
    expect(candidates[0]?.label).toContain("surface");
  });

  it("keeps shapes apart when a structural prop key differs", () => {
    const spec = node("page", "page", {}, [
      node("a", "surface", { pad: 16 }, [
        node("a.title", "text", { text: "One" }),
        node("a.body", "text", { text: "Two" }),
      ]),
      node("b", "surface", { gap: 16 }, [
        node("b.title", "text", { text: "One" }),
        node("b.body", "text", { text: "Two" }),
      ]),
    ]);

    expect(extractionCandidates(spec)).toEqual([]);
  });

  it("excludes repeated RCL primitives and their whole root", () => {
    const spec = node("page", "page", {}, [
      node("card.a", "card", {}, [
        node("card.a.title", "text", { text: "A" }),
        node("card.a.body", "text", { text: "B" }),
        node("card.a.cta", "button", { text: "Open" }),
      ]),
      node("card.b", "card", {}, [
        node("card.b.title", "text", { text: "C" }),
        node("card.b.body", "text", { text: "D" }),
        node("card.b.cta", "button", { text: "Close" }),
      ]),
      node("stat.a", "stat", {}, [
        node("stat.a.label", "text", { text: "Uptime" }),
        node("stat.a.value", "text", { text: "99%" }),
        node("stat.a.detail", "text", { text: "30 days" }),
      ]),
      node("stat.b", "stat", {}, [
        node("stat.b.label", "text", { text: "Latency" }),
        node("stat.b.value", "text", { text: "40ms" }),
        node("stat.b.detail", "text", { text: "p95" }),
      ]),
    ]);

    expect(extractionCandidates(spec)).toEqual([]);
  });

  it("excludes slots and pages as roots", () => {
    const slot = (id: string) =>
      node(`${id}.slot`, "slot", {}, [
        node(`${id}.hint`, "text", { text: "Metric" }),
        node(`${id}.fallback`, "text", { text: "Chart" }),
      ]);
    const spec = node("page", "page", {}, [
      slot("a"),
      slot("b"),
      node("inner.a", "page", {}, [
        node("inner.a.one", "text", { text: "A" }),
        node("inner.a.two", "text", { text: "B" }),
      ]),
      node("inner.b", "page", {}, [
        node("inner.b.one", "text", { text: "C" }),
        node("inner.b.two", "text", { text: "D" }),
      ]),
    ]);

    expect(extractionCandidates(spec)).toEqual([]);
  });

  it("keeps a repeated two-layer structure below the threshold", () => {
    const spec = node("page", "page", {}, [
      node("a", "surface", { pad: 4 }, [node("a.text", "text", { text: "A" })]),
      node("b", "surface", { pad: 8 }, [node("b.text", "text", { text: "B" })]),
    ]);

    expect(extractionCandidates(spec)).toEqual([]);
  });

  it("keeps the larger structure and drops the nested one with the same occurrences", () => {
    const panel = (id: string, title: string) =>
      node(`${id}.panel`, "surface", { pad: 12 }, [
        node(`${id}.title`, "text", { text: title }),
        node(`${id}.row`, "row", { gap: 4 }, [
          node(`${id}.a`, "text", { text: "A" }),
          node(`${id}.b`, "text", { text: "B" }),
        ]),
      ]);
    const spec = node("page", "page", {}, [panel("one", "One"), panel("two", "Two")]);

    const candidates = extractionCandidates(spec);
    expect(candidates).toHaveLength(1);
    expect(candidates[0]).toMatchObject({
      nodeCount: 5,
      occurrences: 2,
      exampleIds: ["one.panel", "two.panel"],
      score: 5,
    });
  });

  it("produces the same signature across id and copy changes", () => {
    const left = node("page", "page", {}, [card("x", "Hi", "There", 8), card("y", "Hello", "World", 22)]);
    const right = node("other", "page", {}, [card("p", "T", "B", 2), card("q", "U", "C", 6)]);

    expect(extractionCandidates(left)[0]?.signature).toBe(extractionCandidates(right)[0]?.signature);
  });

  it("returns nothing for a frame with no repeated custom structure", () => {
    expect(extractionCandidates(node("page", "page", {}))).toEqual([]);
    expect(extractionCandidates(node("page", "page", {}, [card("only", "A", "B", 8)]))).toEqual([]);
  });

  it("ranks by score and returns at most five candidates", () => {
    const shape = (id: string, leaves: number) =>
      node(
        id,
        "stack",
        {},
        Array.from({ length: leaves }, (_, index) =>
          node(`${id}.${index}`, "text", { text: `${index}` }),
        ),
      );
    const spec = node(
      "page",
      "page",
      {},
      [2, 3, 4, 5, 6, 7].flatMap((leaves) => [
        shape(`a${leaves}`, leaves),
        shape(`b${leaves}`, leaves),
      ]),
    );

    const candidates = extractionCandidates(spec);
    expect(candidates).toHaveLength(5);
    expect(candidates.map((candidate) => candidate.score)).toEqual([8, 7, 6, 5, 4]);
  });

  it("uses an authored label once, not twice", () => {
    const spec = node("page", "page", {}, [
      node("a", "stack", { gap: 14, label: "Group · Signals" }, [
        node("a1", "text", { text: "Signals" }),
        node("a2", "text", { text: "Every change gets a trail." }),
      ]),
      node("b", "stack", { gap: 14, label: "Group · Signals" }, [
        node("b1", "text", { text: "Signals" }),
        node("b2", "text", { text: "Every change gets a trail." }),
      ]),
    ]);
    const label = extractionCandidates(spec)[0]?.label ?? "";
    expect(label.startsWith("Group · ")).toBe(true);
    expect(label).not.toContain("Group · Signals · Signals");
  });
});
