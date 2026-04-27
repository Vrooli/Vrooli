import { describe, expect, it } from "vitest";
import { isExternalHref, looksLikeFileReference } from "../lib/fileReferences";

describe("fileReferences", () => {
  it("detects external hrefs", () => {
    expect(isExternalHref("https://example.com")).toBe(true);
    expect(isExternalHref("mailto:test@example.com")).toBe(true);
    expect(isExternalHref("docs/plan.md")).toBe(false);
  });

  it("detects file-like references", () => {
    expect(looksLikeFileReference("/tmp/file.ts:12")).toBe(true);
    expect(looksLikeFileReference("docs/plan.md")).toBe(true);
    expect(looksLikeFileReference("https://example.com")).toBe(false);
    expect(looksLikeFileReference("#anchor")).toBe(false);
  });
});
