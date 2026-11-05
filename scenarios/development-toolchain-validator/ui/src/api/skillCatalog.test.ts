import { describe, expect, it } from "vitest";

import { skillCatalogClient } from "./skillCatalog";

describe("api/skillCatalog", () => {
  it("exposes the SkillCatalogService RPCs as client methods", () => {
    expect(typeof skillCatalogClient.sync).toBe("function");
    expect(typeof skillCatalogClient.listSkills).toBe("function");
    expect(typeof skillCatalogClient.getSkill).toBe("function");
  });
});
