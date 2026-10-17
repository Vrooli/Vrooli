import { describe, it, expect } from "vitest";
import { getFileTypeInfo, buildImagePreviewSrc, isBinaryExtension } from "./fileTypes";

describe("getFileTypeInfo", () => {
  it("gives SVG the MIME type browsers actually accept", () => {
    // `image/svg` is not a real MIME type; a data URL using it fails to decode
    // and renders as a broken-image placeholder.
    expect(getFileTypeInfo("logo.svg").mimeType).toBe("image/svg+xml");
  });

  it("maps every supported image extension to a valid MIME type", () => {
    const expected: Record<string, string> = {
      "a.png": "image/png",
      "a.jpg": "image/jpeg",
      "a.jpeg": "image/jpeg",
      "a.gif": "image/gif",
      "a.svg": "image/svg+xml",
      "a.webp": "image/webp",
      "a.ico": "image/x-icon",
      "a.bmp": "image/bmp",
      "a.tiff": "image/tiff"
    };
    for (const [path, mimeType] of Object.entries(expected)) {
      const info = getFileTypeInfo(path);
      expect(info.category).toBe("image");
      expect(info.canPreview).toBe(true);
      expect(info.mimeType).toBe(mimeType);
    }
  });

  it("is case insensitive", () => {
    expect(getFileTypeInfo("LOGO.SVG").mimeType).toBe("image/svg+xml");
  });

  it("marks SVG as text and raster images as base64, matching the API", () => {
    expect(getFileTypeInfo("logo.svg").encoding).toBe("text");
    expect(getFileTypeInfo("photo.png").encoding).toBe("base64");
  });

  it("still classifies non-image files", () => {
    expect(getFileTypeInfo("README.md").category).toBe("markdown");
    expect(getFileTypeInfo("main.ts").category).toBe("code");
    expect(getFileTypeInfo("Makefile").category).toBe("code");
    expect(getFileTypeInfo("doc.pdf").canPreview).toBe(false);
  });
});

describe("buildImagePreviewSrc", () => {
  it("percent-encodes SVG source rather than treating it as base64", () => {
    const svg = '<svg xmlns="http://www.w3.org/2000/svg"><rect width="1"/></svg>';
    const src = buildImagePreviewSrc(getFileTypeInfo("logo.svg"), svg);
    expect(src).toBe(`data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`);
    expect(src).not.toContain("base64");
  });

  it("handles SVG containing characters btoa() would reject", () => {
    const svg = '<svg xmlns="http://www.w3.org/2000/svg"><text>→ 日本語</text></svg>';
    expect(() => buildImagePreviewSrc(getFileTypeInfo("chart.svg"), svg)).not.toThrow();
    const src = buildImagePreviewSrc(getFileTypeInfo("chart.svg"), svg) ?? "";
    expect(decodeURIComponent(src.split(",")[1] ?? "")).toBe(svg);
  });

  it("escapes characters that would truncate the data URL", () => {
    // A literal '#' starts a fragment and would silently cut the image short.
    const svg = '<svg><style>.a{fill:#fff}</style></svg>';
    const src = buildImagePreviewSrc(getFileTypeInfo("a.svg"), svg);
    expect(src).not.toContain("#");
  });

  it("keeps raster images base64", () => {
    const src = buildImagePreviewSrc(getFileTypeInfo("photo.png"), "iVBORw0KGgo=");
    expect(src).toBe("data:image/png;base64,iVBORw0KGgo=");
  });

  it("returns null for non-image files", () => {
    expect(buildImagePreviewSrc(getFileTypeInfo("README.md"), "# hi")).toBeNull();
    expect(buildImagePreviewSrc(getFileTypeInfo("doc.pdf"), "abc")).toBeNull();
  });
});

describe("isBinaryExtension", () => {
  it("does not call SVG binary", () => {
    expect(isBinaryExtension("logo.svg")).toBe(false);
  });

  it("still calls raster images and PDFs binary", () => {
    expect(isBinaryExtension("photo.png")).toBe(true);
    expect(isBinaryExtension("doc.pdf")).toBe(true);
    expect(isBinaryExtension("main.ts")).toBe(false);
  });
});
