import { describe, expect, it } from "vitest";

import { clearSavedForm, emptyForm, loadForm, missing, move, saveForm, toSpec } from "./spec";

const defaults = { supervisorModel: "gpt-6-luna", supervisorEffort: "medium", ownerModelTier: "inherit" };

describe("toSpec", () => {
  it("sends exactly the packet builder's keys, leaving unused options out", () => {
    const form = { ...emptyForm(defaults), topics: ["debt-reduction"], quotes: ["  make it simple ", ""], emphasis: " depth " };
    expect(toSpec(form)).toEqual({
      topics: ["debt-reduction"],
      allowance_hours: 24,
      emphasis: "depth",
      stance_quotes: ["make it simple"],
      supervisor_model: "gpt-6-luna",
      supervisor_effort: "medium",
      owner_model_tier: "inherit",
    });
  });

  it("keeps per-topic choices only for topics still selected", () => {
    const form = {
      ...emptyForm(defaults),
      topics: ["a"],
      quotes: ["q"],
      order: { a: ["A2", "A1"], b: ["B1"] },
      notes: { a: " look at A2 first ", b: "gone" },
      readFirst: { a: "/home/x/research/topics/a/runs/c/CHECKPOINT.md", b: "/y" },
      choices: { "programme-four-20261110": "alongside" as const, "research-night-20261031": "" as const },
      slug: "focus",
    };
    expect(toSpec(form)).toMatchObject({
      order: { a: ["A2", "A1"] },
      card_notes: { a: "look at A2 first" },
      read_first: { a: "/home/x/research/topics/a/runs/c/CHECKPOINT.md" },
      slug: "focus",
    });
    // Choices about earlier commissions travel in the request, never in the packet (rulings P2).
    expect(toSpec(form)).not.toHaveProperty("run_alongside");
    expect(toSpec(form)).not.toHaveProperty("supersedes");
  });
});

describe("missing", () => {
  it("names what blocks a preview", () => {
    expect(missing({ ...emptyForm(defaults), quotes: [" "], allowanceHours: 0 })).toEqual(["topics", "quotes", "allowance"]);
  });
});

describe("move", () => {
  it("swaps neighbours and ignores moves past either end", () => {
    expect(move(["a", "b", "c"], 2, -1)).toEqual(["a", "c", "b"]);
    expect(move(["a", "b"], 0, -1)).toEqual(["a", "b"]);
  });
});

describe("saved form", () => {
  const offered = { supervisorModels: ["gpt-6-luna", "gpt-6-sol"], supervisorEfforts: ["medium", "high"], ownerModelTiers: ["inherit"] };

  it("comes back as it was left, in this tab", () => {
    const form = { ...emptyForm(defaults), topics: ["a", "b"], quotes: ["be bold"], supervisorModel: "gpt-6-sol", choices: { c1: "alongside" as const } };
    saveForm(form);
    expect(loadForm(defaults, offered)).toEqual(form);
    clearSavedForm();
    expect(loadForm(defaults, offered)).toEqual(emptyForm(defaults));
  });

  it("drops fields of the wrong shape and models no longer offered", () => {
    sessionStorage.setItem("research-desk.launch-form", JSON.stringify({ topics: "a", quotes: [], supervisorModel: "gone", allowanceHours: 12 }));
    expect(loadForm(defaults, offered)).toEqual({ ...emptyForm(defaults), allowanceHours: 12 });
    sessionStorage.setItem("research-desk.launch-form", "{not json");
    expect(loadForm(defaults, offered)).toEqual(emptyForm(defaults));
    clearSavedForm();
  });
});
