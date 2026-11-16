import { describe, expect, it } from "vitest";
import { createSlugger, slugify } from "./slugger.js";

describe("heading slugs", () => {
  it.each([
    ["Hello world", "hello-world"],
    ["Hello, world!", "hello-world"],
    ["Plan", "plan"],
    ["a  b", "a--b"],
    ["C++ / Rust", "c--rust"],
    ["snake_case and kebab-case", "snake_case-and-kebab-case"],
    ["Über uns", "über-uns"],
    ["日本語 の 見出し", "日本語-の-見出し"],
    ["Step 2: Install", "step-2-install"],
    ["`code` heading", "code-heading"],
  ])("slugifies %j as %j", (heading, expected) => { expect(slugify(heading)).toBe(expected); });

  it("numbers duplicates in document order like GitHub: plan, plan-1, plan-2", () => {
    const slugger = createSlugger();
    expect(["Plan", "Plan", "Other", "Plan"].map((heading) => slugger.slug(heading))).toEqual(["plan", "plan-1", "other", "plan-2"]);
  });

  it("keeps slugs unique when a heading already reads like a counted duplicate", () => {
    const slugger = createSlugger();
    expect(["Plan", "Plan 1", "Plan"].map((heading) => slugger.slug(heading))).toEqual(["plan", "plan-1", "plan-2"]);
  });

  it("starts fresh numbering for each slugger", () => {
    expect(createSlugger().slug("Plan")).toBe("plan");
    expect(createSlugger().slug("Plan")).toBe("plan");
  });
});
