import { describe, expect, it } from "vitest";

import { filterHits, type SearchHit } from "../../api/search";

const hit = (overrides: Partial<SearchHit> = {}): SearchHit => ({
  scenario: "demo",
  slot: "Foo",
  kind: "component",
  displayName: "Foo",
  description: "",
  filePath: "ui/src/Foo.tsx",
  score: 1,
  provenance: "custom",
  library: "",
  componentName: "",
  ...overrides,
});

describe("filterHits", () => {
  it("returns all hits when kind = all", () => {
    const hits = [hit({ kind: "component" }), hit({ kind: "page" })];
    expect(filterHits(hits, "all")).toEqual(hits);
  });

  it("returns only matching kinds otherwise", () => {
    const c = hit({ scenario: "a", kind: "component" });
    const p = hit({ scenario: "b", kind: "page" });
    expect(filterHits([c, p], "component")).toEqual([c]);
    expect(filterHits([c, p], "page")).toEqual([p]);
  });

  it("returns an empty list when nothing matches", () => {
    const hits = [hit({ kind: "component" })];
    expect(filterHits(hits, "hook")).toEqual([]);
  });
});
