import { describe, expect, it } from "vitest";
import { isExternalHref, looksLikeFileReference, looksLikeInlineFileReference, matchProseFilePaths } from "../lib/fileReferences";

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

describe("matchProseFilePaths", () => {
  const paths = (text: string) => matchProseFilePaths(text).map((m) => m.path);

  it("matches anchored absolute paths", () => {
    expect(paths("wrote /tmp/report.md today")).toEqual(["/tmp/report.md"]);
    expect(paths("config lives in /etc/hosts")).toEqual(["/etc/hosts"]);
    expect(paths("see ~/notes/todo.txt")).toEqual(["~/notes/todo.txt"]);
    expect(paths("run ./scripts/build.sh first")).toEqual(["./scripts/build.sh"]);
    expect(paths("open file:///var/log/syslog now")).toEqual(["file:///var/log/syslog"]);
  });

  it("matches anchored single-segment paths only with an extension", () => {
    expect(paths("saved to /report.md")).toEqual(["/report.md"]);
    expect(paths("mounted at /data")).toEqual([]);
  });

  it("matches relative paths with depth and extension", () => {
    expect(paths("edit scenarios/web-console/ui/src/App.tsx please")).toEqual([
      "scenarios/web-console/ui/src/App.tsx",
    ]);
    expect(paths("docs/plan.md has details")).toEqual(["docs/plan.md"]);
  });

  // Directories are previewable, so a relative directory reference has to be
  // linkable too. A trailing slash is the unambiguous signal that running
  // prose never produces by accident.
  it("matches an extensionless relative directory when it ends in a slash", () => {
    expect(paths("evidence lives in docs/evidence/ now")).toEqual(["docs/evidence"]);
    expect(paths("see scenarios/web-console/bas/ for cases")).toEqual(["scenarios/web-console/bas"]);
  });

  it("still rejects an extensionless relative directory without the slash", () => {
    expect(paths("evidence lives in docs/evidence now")).toEqual([]);
  });

  it("does not let the trailing slash admit prose that merely contains one", () => {
    // Single-segment bodies stay out regardless of the slash.
    expect(paths("either and/or works")).toEqual([]);
    expect(paths("we speak TCP/IP here")).toEqual([]);
    expect(paths("read/write access")).toEqual([]);
  });

  it("keeps :line suffixes", () => {
    expect(paths("bug at ui/src/App.tsx:42 here")).toEqual(["ui/src/App.tsx:42"]);
  });

  it("strips trailing sentence punctuation", () => {
    expect(paths("see src/App.tsx.")).toEqual(["src/App.tsx"]);
    expect(paths("in docs/a.md, then more")).toEqual(["docs/a.md"]);
  });

  it("rejects prose that merely contains slashes or dots", () => {
    expect(paths("use and/or logic")).toEqual([]);
    expect(paths("the TCP/IP stack")).toEqual([]);
    expect(paths("built on node.js runtime")).toEqual([]);
    expect(paths("visit vrooli.com for info")).toEqual([]);
    expect(paths("I/O bound work")).toEqual([]);
    expect(paths("a 50/50 split")).toEqual([]);
  });

  it("rejects extensionless relative paths", () => {
    expect(paths("the foo/bar module")).toEqual([]);
  });

  it("does not match inside URLs", () => {
    expect(paths("https://example.com/docs/guide.md is external")).toEqual([]);
  });

  it("matches deep absolute paths with dot-dirs, hyphens, and underscores", () => {
    const text =
      "Looks like it’s matching the negatives /home/matthalloran8/.vrooli/cache/vrooli/web-console/uploads/e802040e-8e0a-4fed-a776-34d1eed75bb1/IMG_9951.png";
    expect(paths(text)).toEqual([
      "/home/matthalloran8/.vrooli/cache/vrooli/web-console/uploads/e802040e-8e0a-4fed-a776-34d1eed75bb1/IMG_9951.png",
    ]);
  });

  it("returns positions usable for splitting", () => {
    const text = "start /a/b.md middle ./c/d.ts end";
    const matches = matchProseFilePaths(text);
    expect(matches).toHaveLength(2);
    const [first, second] = matches;
    expect(text.slice(first?.start, first?.end)).toBe("/a/b.md");
    expect(text.slice(second?.start, second?.end)).toBe("./c/d.ts");
  });
});

describe("looksLikeInlineFileReference", () => {
  it("accepts strict paths and plain filenames with extensions", () => {
    expect(looksLikeInlineFileReference("/tmp/report.md")).toBe(true);
    expect(looksLikeInlineFileReference("~/notes/todo.txt")).toBe(true);
    expect(looksLikeInlineFileReference("ui/src/App.tsx:42")).toBe(true);
    expect(looksLikeInlineFileReference("README.md")).toBe(true);
    expect(looksLikeInlineFileReference("package.json")).toBe(true);
    expect(looksLikeInlineFileReference("/etc/hosts")).toBe(true);
  });

  it("accepts an extensionless relative directory written with a trailing slash", () => {
    expect(looksLikeInlineFileReference("docs/evidence/")).toBe(true);
    expect(looksLikeInlineFileReference("scenarios/web-console/bas/")).toBe(true);
    // Without the slash it stays out, exactly as before.
    expect(looksLikeInlineFileReference("docs/evidence")).toBe(false);
  });

  it("rejects slashed prose, bare domains, and non-path tokens", () => {
    expect(looksLikeInlineFileReference("and/or")).toBe(false);
    expect(looksLikeInlineFileReference("TCP/IP")).toBe(false);
    expect(looksLikeInlineFileReference("50/50")).toBe(false);
    expect(looksLikeInlineFileReference("vrooli.com")).toBe(false);
    expect(looksLikeInlineFileReference("file://")).toBe(false);
    expect(looksLikeInlineFileReference("foo/bar")).toBe(false);
    expect(looksLikeInlineFileReference("message_link")).toBe(false);
    expect(looksLikeInlineFileReference("onLinkClick")).toBe(false);
    expect(looksLikeInlineFileReference("two words.md")).toBe(false);
    expect(looksLikeInlineFileReference("https://example.com/a.md")).toBe(false);
  });
});
