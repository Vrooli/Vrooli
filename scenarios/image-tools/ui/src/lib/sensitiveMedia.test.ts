import { describe, expect, it } from "vitest";

import {
  inheritSensitivity,
  isSensitiveFile,
  isSensitiveRef,
  isSensitiveUrl,
  markSensitive,
  sensitiveRefsOf,
} from "./sensitiveMedia";

describe("sensitiveRefsOf", () => {
  it("reads the server's refs only when the job is flagged", () => {
    expect(sensitiveRefsOf({ sensitive: "true", sensitive_refs: "out/a.png, out/b.png" })).toEqual(["out/a.png", "out/b.png"]);
    expect(sensitiveRefsOf({ sensitive_refs: "out/a.png" })).toEqual([]);
    expect(sensitiveRefsOf({ sensitive: "true" })).toEqual([]);
    expect(sensitiveRefsOf(undefined)).toEqual([]);
  });

  it("matches a ref exactly", () => {
    const job = { resultMeta: { sensitive: "true", sensitive_refs: "out/a.png" } };
    expect(isSensitiveRef(job, "out/a.png")).toBe(true);
    expect(isSensitiveRef(job, "out/a.png.bak")).toBe(false);
  });
});

describe("marks follow the image", () => {
  it("marks urls and files independently of each other", () => {
    const file = new File(["x"], "a.png");
    markSensitive({ url: "blob:one" });
    markSensitive({ file });
    expect(isSensitiveUrl("blob:one")).toBe(true);
    expect(isSensitiveUrl("blob:two")).toBe(false);
    expect(isSensitiveUrl(null)).toBe(false);
    expect(isSensitiveFile(file)).toBe(true);
    expect(isSensitiveFile(new File(["x"], "a.png"))).toBe(false);
  });

  it("carries sensitivity to whatever an edit produces, and only then", () => {
    const sensitive = new File(["s"], "s.png");
    const clean = new File(["c"], "c.png");
    markSensitive({ file: sensitive });

    const derivedFile = new File(["d"], "d.png");
    expect(inheritSensitivity(sensitive, { url: "blob:derived", file: derivedFile })).toBe(true);
    expect(isSensitiveUrl("blob:derived")).toBe(true);
    expect(isSensitiveFile(derivedFile)).toBe(true);

    expect(inheritSensitivity(clean, { url: "blob:clean-derived" })).toBe(false);
    expect(isSensitiveUrl("blob:clean-derived")).toBe(false);
    // A second generation keeps the mark.
    expect(inheritSensitivity(derivedFile, { url: "blob:grandchild" })).toBe(true);
  });
});
