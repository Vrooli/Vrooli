import { describe, it, expect } from "vitest";
import { parseDeploymentHash, buildDeploymentHash } from "./useDeploymentUrl";
import { DEFAULT_DEPLOYMENT_URL_STATE } from "../types/url";

describe("parseDeploymentHash", () => {
  describe("deployment ID parsing", () => {
    it("returns null deploymentId for #deployments", () => {
      expect(parseDeploymentHash("#deployments").deploymentId).toBe(null);
    });

    it("returns null deploymentId for empty hash", () => {
      expect(parseDeploymentHash("").deploymentId).toBe(null);
    });

    it("parses deployment ID from #deployments/<id>", () => {
      expect(parseDeploymentHash("#deployments/abc123").deploymentId).toBe("abc123");
    });

    it("parses deployment ID with dashes and underscores", () => {
      expect(parseDeploymentHash("#deployments/my-deploy_123").deploymentId).toBe("my-deploy_123");
    });

    it("parses UUID-style deployment IDs", () => {
      expect(parseDeploymentHash("#deployments/550e8400-e29b-41d4-a716-446655440000").deploymentId)
        .toBe("550e8400-e29b-41d4-a716-446655440000");
    });

    it("returns defaults for non-deployment hashes", () => {
      expect(parseDeploymentHash("#dashboard")).toEqual(DEFAULT_DEPLOYMENT_URL_STATE);
      expect(parseDeploymentHash("#wizard")).toEqual(DEFAULT_DEPLOYMENT_URL_STATE);
      expect(parseDeploymentHash("#docs/guide")).toEqual(DEFAULT_DEPLOYMENT_URL_STATE);
    });
  });

  describe("tab parsing", () => {
    it("defaults to overview when no tab parameter", () => {
      expect(parseDeploymentHash("#deployments/abc").tab).toBe("overview");
    });

    it("parses valid tab parameter", () => {
      expect(parseDeploymentHash("#deployments/abc?tab=live-state").tab).toBe("live-state");
      expect(parseDeploymentHash("#deployments/abc?tab=files").tab).toBe("files");
      expect(parseDeploymentHash("#deployments/abc?tab=drift").tab).toBe("drift");
      expect(parseDeploymentHash("#deployments/abc?tab=secrets").tab).toBe("secrets");
      expect(parseDeploymentHash("#deployments/abc?tab=history").tab).toBe("history");
      expect(parseDeploymentHash("#deployments/abc?tab=investigations").tab).toBe("investigations");
      expect(parseDeploymentHash("#deployments/abc?tab=terminal").tab).toBe("terminal");
    });

    it("falls back to overview for invalid tab", () => {
      expect(parseDeploymentHash("#deployments/abc?tab=invalid").tab).toBe("overview");
      expect(parseDeploymentHash("#deployments/abc?tab=").tab).toBe("overview");
    });
  });

  describe("subtab parsing", () => {
    it("defaults to processes when no subtab parameter", () => {
      expect(parseDeploymentHash("#deployments/abc?tab=live-state").subtab).toBe("processes");
    });

    it("parses valid subtab parameter", () => {
      expect(parseDeploymentHash("#deployments/abc?tab=live-state&subtab=ports").subtab).toBe("ports");
      expect(parseDeploymentHash("#deployments/abc?tab=live-state&subtab=system").subtab).toBe("system");
      expect(parseDeploymentHash("#deployments/abc?tab=live-state&subtab=caddy").subtab).toBe("caddy");
      expect(parseDeploymentHash("#deployments/abc?tab=live-state&subtab=management").subtab).toBe("management");
    });

    it("falls back to processes for invalid subtab", () => {
      expect(parseDeploymentHash("#deployments/abc?tab=live-state&subtab=invalid").subtab).toBe("processes");
    });
  });

  describe("modal parsing", () => {
    it("defaults to null when no modal parameter", () => {
      expect(parseDeploymentHash("#deployments/abc").modal).toBe(null);
    });

    it("parses valid modal parameter", () => {
      expect(parseDeploymentHash("#deployments/abc?modal=redeploy").modal).toBe("redeploy");
      expect(parseDeploymentHash("#deployments/abc?modal=spawn-agent").modal).toBe("spawn-agent");
      expect(parseDeploymentHash("#deployments/abc?modal=delete").modal).toBe("delete");
      expect(parseDeploymentHash("#deployments/abc?modal=investigation-report").modal).toBe("investigation-report");
    });

    it("falls back to null for invalid modal", () => {
      expect(parseDeploymentHash("#deployments/abc?modal=invalid").modal).toBe(null);
    });
  });

  describe("modal params parsing", () => {
    it("returns empty object when no extra params", () => {
      expect(parseDeploymentHash("#deployments/abc?modal=redeploy").modalParams).toEqual({});
    });

    it("parses taskType for spawn-agent modal", () => {
      const result = parseDeploymentHash("#deployments/abc?modal=spawn-agent&taskType=fix");
      expect(result.modalParams).toEqual({ taskType: "fix" });
    });

    it("parses invId for investigation-report modal", () => {
      const result = parseDeploymentHash("#deployments/abc?modal=investigation-report&invId=inv-123");
      expect(result.modalParams).toEqual({ invId: "inv-123" });
    });

    it("parses multiple modal params", () => {
      const result = parseDeploymentHash("#deployments/abc?modal=spawn-agent&taskType=investigate&effort=logs");
      expect(result.modalParams).toEqual({ taskType: "investigate", effort: "logs" });
    });

    it("excludes reserved params (tab, subtab, modal) from modalParams", () => {
      const result = parseDeploymentHash("#deployments/abc?tab=live-state&subtab=ports&modal=redeploy&extra=value");
      expect(result.modalParams).toEqual({ extra: "value" });
    });
  });

  describe("complex URL parsing", () => {
    it("parses full URL with all parameters", () => {
      const result = parseDeploymentHash("#deployments/abc123?tab=investigations&modal=investigation-report&invId=inv-456");
      expect(result).toEqual({
        deploymentId: "abc123",
        tab: "investigations",
        subtab: "processes",
        modal: "investigation-report",
        modalParams: { invId: "inv-456" },
      });
    });

    it("handles URL without leading #", () => {
      expect(parseDeploymentHash("deployments/abc").deploymentId).toBe("abc");
    });
  });
});

describe("buildDeploymentHash", () => {
  describe("basic URL building", () => {
    it("builds minimal URL for list view", () => {
      expect(buildDeploymentHash({})).toBe("#deployments");
      expect(buildDeploymentHash({ deploymentId: null })).toBe("#deployments");
    });

    it("builds URL with deployment ID", () => {
      expect(buildDeploymentHash({ deploymentId: "abc123" })).toBe("#deployments/abc123");
    });
  });

  describe("tab parameter", () => {
    it("omits tab when it is overview (default)", () => {
      expect(buildDeploymentHash({ deploymentId: "abc", tab: "overview" })).toBe("#deployments/abc");
    });

    it("includes tab when not default", () => {
      expect(buildDeploymentHash({ deploymentId: "abc", tab: "live-state" })).toBe("#deployments/abc?tab=live-state");
      expect(buildDeploymentHash({ deploymentId: "abc", tab: "files" })).toBe("#deployments/abc?tab=files");
    });
  });

  describe("subtab parameter", () => {
    it("omits subtab when tab is not live-state", () => {
      expect(buildDeploymentHash({ deploymentId: "abc", tab: "files", subtab: "ports" }))
        .toBe("#deployments/abc?tab=files");
    });

    it("omits subtab when it is processes (default)", () => {
      expect(buildDeploymentHash({ deploymentId: "abc", tab: "live-state", subtab: "processes" }))
        .toBe("#deployments/abc?tab=live-state");
    });

    it("includes subtab when on live-state and not default", () => {
      expect(buildDeploymentHash({ deploymentId: "abc", tab: "live-state", subtab: "ports" }))
        .toBe("#deployments/abc?tab=live-state&subtab=ports");
    });
  });

  describe("modal parameter", () => {
    it("omits modal when null", () => {
      expect(buildDeploymentHash({ deploymentId: "abc", modal: null })).toBe("#deployments/abc");
    });

    it("includes modal when set", () => {
      expect(buildDeploymentHash({ deploymentId: "abc", modal: "redeploy" }))
        .toBe("#deployments/abc?modal=redeploy");
    });
  });

  describe("modal params", () => {
    it("includes modal params", () => {
      expect(buildDeploymentHash({ deploymentId: "abc", modal: "spawn-agent", modalParams: { taskType: "fix" } }))
        .toBe("#deployments/abc?modal=spawn-agent&taskType=fix");
    });

    it("skips empty modal params", () => {
      expect(buildDeploymentHash({ deploymentId: "abc", modal: "spawn-agent", modalParams: { taskType: "" } }))
        .toBe("#deployments/abc?modal=spawn-agent");
    });

    it("includes multiple modal params", () => {
      const result = buildDeploymentHash({
        deploymentId: "abc",
        modal: "investigation-report",
        modalParams: { invId: "inv-123", extra: "value" },
      });
      expect(result).toContain("invId=inv-123");
      expect(result).toContain("extra=value");
    });
  });

  describe("complex URL building", () => {
    it("builds full URL with all parameters", () => {
      const result = buildDeploymentHash({
        deploymentId: "abc123",
        tab: "live-state",
        subtab: "ports",
        modal: "redeploy",
        modalParams: {},
      });
      expect(result).toBe("#deployments/abc123?tab=live-state&subtab=ports&modal=redeploy");
    });
  });
});

describe("round-trip parsing and building", () => {
  it("preserves state through parse -> build -> parse", () => {
    const original = "#deployments/abc123?tab=live-state&subtab=ports";
    const parsed = parseDeploymentHash(original);
    const rebuilt = buildDeploymentHash(parsed);
    const reparsed = parseDeploymentHash(rebuilt);

    expect(reparsed).toEqual(parsed);
  });

  it("normalizes URLs by removing defaults", () => {
    const withDefaults = "#deployments/abc?tab=overview&subtab=processes";
    const parsed = parseDeploymentHash(withDefaults);
    const rebuilt = buildDeploymentHash(parsed);

    // Rebuilt URL should not include default values
    expect(rebuilt).toBe("#deployments/abc");
  });
});
