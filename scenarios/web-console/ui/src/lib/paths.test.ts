import { describe, expect, it } from "vitest";

import { basename, isWindowsPath, joinPath, pathCrumbs, separatorFor } from "./paths";

// These paths arrive from the API in the *server's* native form, so the UI has
// to handle both flavours regardless of what the browser is running on.

describe("isWindowsPath", () => {
  it("recognizes drive letters and UNC shares", () => {
    expect(isWindowsPath("C:\\Users\\me")).toBe(true);
    expect(isWindowsPath("C:/Users/me")).toBe(true);
    expect(isWindowsPath("\\\\server\\share\\docs")).toBe(true);
    expect(isWindowsPath("docs\\evidence")).toBe(true);
  });

  it("treats POSIX paths as POSIX", () => {
    expect(isWindowsPath("/home/me/docs")).toBe(false);
    expect(isWindowsPath("docs/evidence")).toBe(false);
  });
});

describe("separatorFor", () => {
  it("matches the path's own flavour", () => {
    expect(separatorFor("/home/me")).toBe("/");
    expect(separatorFor("C:\\Users")).toBe("\\");
  });
});

describe("joinPath", () => {
  it("joins with the parent's separator", () => {
    expect(joinPath("/home/me", "docs")).toBe("/home/me/docs");
    expect(joinPath("C:\\Users\\me", "docs")).toBe("C:\\Users\\me\\docs");
  });

  it("does not double a separator that is already there", () => {
    expect(joinPath("/", "home")).toBe("/home");
    expect(joinPath("C:\\", "Users")).toBe("C:\\Users");
  });

  it("returns the name when there is no parent", () => {
    expect(joinPath("", "docs")).toBe("docs");
  });
});

describe("basename", () => {
  it("takes the last segment of either flavour", () => {
    expect(basename("/home/me/docs/evidence")).toBe("evidence");
    expect(basename("C:\\Users\\me\\notes.md")).toBe("notes.md");
  });

  it("ignores a trailing separator", () => {
    expect(basename("/home/me/docs/")).toBe("docs");
  });

  it("returns roots unchanged", () => {
    expect(basename("/")).toBe("/");
  });
});

describe("pathCrumbs", () => {
  it("builds absolute POSIX crumbs from root to leaf", () => {
    expect(pathCrumbs("/home/me/docs")).toEqual([
      { label: "/", path: "/" },
      { label: "home", path: "/home" },
      { label: "me", path: "/home/me" },
      { label: "docs", path: "/home/me/docs" },
    ]);
  });

  it("roots Windows crumbs at the drive", () => {
    expect(pathCrumbs("C:\\Users\\me")).toEqual([
      { label: "C:\\", path: "C:\\" },
      { label: "Users", path: "C:\\Users" },
      { label: "me", path: "C:\\Users\\me" },
    ]);
  });

  it("roots UNC crumbs at the share, which is the smallest addressable unit", () => {
    expect(pathCrumbs("\\\\server\\share\\docs")).toEqual([
      { label: "\\\\server\\share", path: "\\\\server\\share" },
      { label: "docs", path: "\\\\server\\share\\docs" },
    ]);
  });

  it("returns a single crumb for a relative path, which has no addressable chain", () => {
    expect(pathCrumbs("docs/evidence")).toEqual([{ label: "docs/evidence", path: "docs/evidence" }]);
  });

  it("returns nothing for an empty path", () => {
    expect(pathCrumbs("")).toEqual([]);
  });
});
