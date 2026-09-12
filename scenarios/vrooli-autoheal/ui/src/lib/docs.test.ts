import { describe, expect, it } from "vitest";
import { getDocsPath, getDocsPathForCheck, navigateToCheckDocs, navigateToDocs } from "./docs";

describe("documentation navigation", () => {
  it("maps known and unknown topics", () => {
    expect(getDocsPath("watchdog")).toContain("watchdog-installation");
    expect(getDocsPath("unknown-topic")).toBe("getting-started.md");
  });

  it("maps infrastructure, resource, scenario, and unknown checks", () => {
    expect(getDocsPathForCheck("infra-dns")).toBe("reference/checks/infra-dns.md");
    expect(getDocsPathForCheck("resource-postgres")).toBe("reference/checks/resource-check.md");
    expect(getDocsPathForCheck("scenario-demo")).toBe("reference/checks/scenario-check.md");
    expect(getDocsPathForCheck("other-check")).toBe("reference/check-catalog.md");
  });

  it("encodes navigation hashes", () => {
    navigateToDocs("reference/my file.md");
    expect(window.location.hash).toBe("#docs?path=reference%2Fmy%20file.md");
    navigateToCheckDocs("resource-postgres");
    expect(window.location.hash).toContain("resource-check.md");
  });
});
