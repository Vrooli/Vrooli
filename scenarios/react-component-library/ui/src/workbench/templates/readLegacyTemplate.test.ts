import { beforeEach, describe, expect, it } from "vitest";

import { readLegacyTemplateForMigration } from "./readLegacyTemplate";

describe("readLegacyTemplateForMigration", () => {
  beforeEach(() => window.localStorage.clear());

  it("reads a valid cached custom template without writing or syncing it", () => {
    const cached = {
      id: "custom-old",
      name: "Custom old",
      spec: { id: "page", type: "page", props: {} },
      palette: { ink: "#111" },
    };
    window.localStorage.setItem("rcl.workbench.templates.v1", JSON.stringify([cached]));

    expect(readLegacyTemplateForMigration("custom-old")).toEqual(cached);
    expect(window.localStorage.getItem("rcl.workbench.templates.v1")).toContain("custom-old");
  });

  it("fails closed for malformed or absent cache entries", () => {
    window.localStorage.setItem("rcl.workbench.templates.v1", "{bad json");
    expect(readLegacyTemplateForMigration("missing")).toBeUndefined();

    window.localStorage.setItem("rcl.workbench.templates.v1", JSON.stringify([{ id: "missing" }]));
    expect(readLegacyTemplateForMigration("missing")).toBeUndefined();
  });
});
