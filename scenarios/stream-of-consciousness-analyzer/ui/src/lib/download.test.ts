import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { downloadJSON, slugify } from "./download";

// [REQ:P1-002] Export utilities

describe("slugify", () => {
  it("converts spaces to hyphens and lowercases", () => {
    expect(slugify("My Scheme Name")).toBe("my-scheme-name");
  });

  it("collapses multiple spaces into a single hyphen", () => {
    expect(slugify("hello   world")).toBe("hello-world");
  });

  it("handles already-lowercase input", () => {
    expect(slugify("simple")).toBe("simple");
  });

  it("handles empty string", () => {
    expect(slugify("")).toBe("");
  });

  it("handles tabs and mixed whitespace", () => {
    expect(slugify("a\tb\nc")).toBe("a-b-c");
  });
});

describe("downloadJSON", () => {
  let createObjectURLSpy: ReturnType<typeof vi.fn>;
  let revokeObjectURLSpy: ReturnType<typeof vi.fn>;
  let clickSpy: ReturnType<typeof vi.fn>;
  let capturedBlob: Blob | undefined;
  let anchorDownload = "";
  let anchorHref = "";

  beforeEach(() => {
    capturedBlob = undefined;
    anchorDownload = "";
    anchorHref = "";

    createObjectURLSpy = vi.fn().mockImplementation((blob: Blob) => {
      capturedBlob = blob;
      return "blob:fake-url";
    });
    revokeObjectURLSpy = vi.fn();
    clickSpy = vi.fn();

    globalThis.URL.createObjectURL = createObjectURLSpy;
    globalThis.URL.revokeObjectURL = revokeObjectURLSpy;

    vi.spyOn(document, "createElement").mockImplementation((tag: string) => {
      if (tag === "a") {
        return {
          click: clickSpy,
          get href() { return anchorHref; },
          set href(v: string) { anchorHref = v; },
          get download() { return anchorDownload; },
          set download(v: string) { anchorDownload = v; },
        } as unknown as HTMLAnchorElement;
      }
      return document.createElement(tag);
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("creates a blob, clicks a link, and revokes the URL", () => {
    downloadJSON({ key: "value" }, "test.json");

    expect(createObjectURLSpy).toHaveBeenCalledOnce();
    expect(capturedBlob).toBeInstanceOf(Blob);

    expect(anchorDownload).toBe("test.json");
    expect(anchorHref).toBe("blob:fake-url");
    expect(clickSpy).toHaveBeenCalledOnce();
    expect(revokeObjectURLSpy).toHaveBeenCalledWith("blob:fake-url");
  });

  it("serializes data with pretty printing", async () => {
    downloadJSON({ a: 1 }, "out.json");

    expect(capturedBlob).toBeDefined();
    const text = await capturedBlob?.text();
    expect(text).toBe(JSON.stringify({ a: 1 }, null, 2));
  });
});
