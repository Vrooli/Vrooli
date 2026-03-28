import { describe, it, expect } from "vitest";
import { slugify, dedupeSlug, extractHeadings } from "./heading-utils";

describe("slugify", () => {
  it("lowercases and converts spaces to hyphens", () => {
    expect(slugify("Hello World")).toBe("hello-world");
  });

  it("strips special characters", () => {
    expect(slugify("What's New? (v2.0)")).toBe("whats-new-v20");
  });

  it("collapses multiple hyphens", () => {
    expect(slugify("a   ---  b")).toBe("a-b");
  });

  it("trims leading/trailing hyphens", () => {
    expect(slugify(" -hello- ")).toBe("hello");
  });

  it("handles empty string", () => {
    expect(slugify("")).toBe("");
  });
});

describe("dedupeSlug", () => {
  it("returns slug unchanged on first occurrence", () => {
    const seen = new Map<string, number>();
    expect(dedupeSlug("intro", seen)).toBe("intro");
  });

  it("appends counter on duplicates", () => {
    const seen = new Map<string, number>();
    dedupeSlug("intro", seen);
    expect(dedupeSlug("intro", seen)).toBe("intro-1");
    expect(dedupeSlug("intro", seen)).toBe("intro-2");
  });
});

describe("extractHeadings", () => {
  it("extracts h1, h2, h3 with correct levels and line numbers", () => {
    const md = "# Title\n\nSome text\n\n## Section\n\n### Sub-section";
    const result = extractHeadings(md);

    expect(result).toEqual([
      { level: 1, text: "Title", id: "title", line: 1 },
      { level: 2, text: "Section", id: "section", line: 5 },
      { level: 3, text: "Sub-section", id: "sub-section", line: 7 },
    ]);
  });

  it("deduplicates identical heading slugs", () => {
    const md = "## API\n\n## API\n\n## API";
    const result = extractHeadings(md);

    expect(result.map((h) => h.id)).toEqual(["api", "api-1", "api-2"]);
  });

  it("skips headings inside code fences", () => {
    const md = "# Real Heading\n\n```\n# Not a heading\n## Also not\n```\n\n## After Fence";
    const result = extractHeadings(md);

    expect(result).toMatchObject([
      { text: "Real Heading" },
      { text: "After Fence" },
    ]);
  });

  it("handles nested code fences correctly", () => {
    const md = "# Before\n\n```markdown\n# Inside\n```\n\n# After";
    const result = extractHeadings(md);

    expect(result.map((h) => h.text)).toEqual(["Before", "After"]);
  });

  it("returns empty array for empty input", () => {
    expect(extractHeadings("")).toEqual([]);
  });

  it("returns empty array for content with no headings", () => {
    expect(extractHeadings("Just some text\nwith no headings")).toEqual([]);
  });

  it("ignores h4+ headings", () => {
    const md = "# H1\n#### H4\n##### H5";
    const result = extractHeadings(md);

    expect(result).toMatchObject([{ text: "H1" }]);
  });
});
