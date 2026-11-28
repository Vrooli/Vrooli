import { beforeEach, describe, expect, it } from "vitest";

import { fileViewerPreferences } from "../fileViewerPreferences";

describe("fileViewerPreferences", () => {
  beforeEach(() => { window.localStorage.clear(); });

  it("namespaces viewer options so unrelated app settings cannot collide", () => {
    fileViewerPreferences.write("directory.sort", JSON.stringify("mtime_desc"));
    expect(window.localStorage.getItem("web-console.file-viewer.v1.directory.sort")).toBe('"mtime_desc"');
    expect(fileViewerPreferences.read("directory.sort")).toBe('"mtime_desc"');
  });

  it("degrades to a null read when storage is unavailable", () => {
    const original = Object.getOwnPropertyDescriptor(Storage.prototype, "getItem");
    Object.defineProperty(Storage.prototype, "getItem", { configurable: true, value: () => { throw new Error("blocked"); } });
    try {
      expect(fileViewerPreferences.read("html.zoom")).toBeNull();
    } finally {
      if (original) Object.defineProperty(Storage.prototype, "getItem", original);
    }
  });
});
