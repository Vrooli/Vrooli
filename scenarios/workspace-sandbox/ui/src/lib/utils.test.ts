import { describe, it, expect } from "vitest";
import { truncatePath, shortId, sandboxDisplayName, formatOwner, splitPath } from "./utils";

describe("truncatePath", () => {
  it("returns short paths unchanged", () => {
    expect(truncatePath("/home/user", 30)).toBe("/home/user");
  });

  it("returns empty string for empty input", () => {
    expect(truncatePath("", 30)).toBe("");
  });

  it("truncates from the left preserving last 2 segments", () => {
    const path = "/home/user/Vrooli/scenarios/web-console";
    const result = truncatePath(path, 25);
    expect(result).toMatch(/^…\//);
    expect(result).toContain("web-console");
    expect(result).toContain("scenarios");
  });

  it("keeps more segments when they fit", () => {
    const path = "/home/user/Vrooli/scenarios/web-console";
    const result = truncatePath(path, 40);
    expect(result).toContain("web-console");
    expect(result.length).toBeLessThanOrEqual(40);
  });

  it("handles paths with only 2 segments", () => {
    expect(truncatePath("/scenarios/web-console", 5)).toBe("/scenarios/web-console");
  });

  it("handles root path", () => {
    expect(truncatePath("/", 10)).toBe("/");
  });

  it("handles path equal to maxLength", () => {
    const path = "/a/b/c";
    expect(truncatePath(path, path.length)).toBe(path);
  });
});

describe("shortId", () => {
  it("shortens a UUID to 8 chars", () => {
    expect(shortId("407578d1-ee26-475a-9ac9-3745b92d5dc3")).toBe("407578d1");
  });

  it("returns non-UUID strings unchanged", () => {
    expect(shortId("alice")).toBe("alice");
  });

  it("returns empty string for empty input", () => {
    expect(shortId("")).toBe("");
  });

  it("handles uppercase UUIDs", () => {
    expect(shortId("407578D1-EE26-475A-9AC9-3745B92D5DC3")).toBe("407578D1");
  });
});

describe("sandboxDisplayName", () => {
  it("uses name when provided", () => {
    expect(
      sandboxDisplayName({ name: "my-sandbox", scopePath: "/a/b/c", id: "407578d1-ee26-475a-9ac9-3745b92d5dc3" }),
    ).toBe("my-sandbox");
  });

  it("derives from scopePath when no name", () => {
    expect(
      sandboxDisplayName({ scopePath: "/home/user/Vrooli/scenarios/agent-manager/api", id: "407578d1-ee26-475a-9ac9-3745b92d5dc3" }),
    ).toBe("agent-manager/api");
  });

  it("uses single segment from short scopePath", () => {
    expect(
      sandboxDisplayName({ scopePath: "/workspace", id: "407578d1-ee26-475a-9ac9-3745b92d5dc3" }),
    ).toBe("workspace");
  });

  it("falls back to shortId when no name or scopePath", () => {
    expect(
      sandboxDisplayName({ id: "407578d1-ee26-475a-9ac9-3745b92d5dc3" }),
    ).toBe("407578d1");
  });

  it("falls back to shortId when scopePath is empty", () => {
    expect(
      sandboxDisplayName({ scopePath: "", id: "407578d1-ee26-475a-9ac9-3745b92d5dc3" }),
    ).toBe("407578d1");
  });
});

describe("formatOwner", () => {
  it("shortens UUID owners and prepends type", () => {
    expect(formatOwner("407578d1-ee26-475a-9ac9-3745b92d5dc3", "agent")).toBe("agent:407578d1");
  });

  it("returns non-UUID owners with type prefix", () => {
    expect(formatOwner("alice", "user")).toBe("user:alice");
  });

  it("returns 'Unknown' for undefined owner", () => {
    expect(formatOwner(undefined)).toBe("Unknown");
  });

  it("returns display without prefix when no ownerType", () => {
    expect(formatOwner("407578d1-ee26-475a-9ac9-3745b92d5dc3")).toBe("407578d1");
  });
});

describe("splitPath", () => {
  it("splits a file path into dir and file", () => {
    expect(splitPath("scenarios/web-console/api/session.go")).toEqual({
      dir: "scenarios/web-console/api",
      file: "session.go",
    });
  });

  it("handles filename without directory", () => {
    expect(splitPath("README.md")).toEqual({ dir: "", file: "README.md" });
  });

  it("handles root-level file", () => {
    expect(splitPath("/session.go")).toEqual({ dir: "", file: "session.go" });
  });

  it("handles deeply nested paths", () => {
    expect(splitPath("a/b/c/d/e.ts")).toEqual({ dir: "a/b/c/d", file: "e.ts" });
  });
});
