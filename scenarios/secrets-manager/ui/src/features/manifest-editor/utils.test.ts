import { describe, expect, it } from "vitest";
import type { DeploymentManifestResponse } from "../../lib/api";
import { secretIdToString, stringToSecretId } from "./types";
import {
  computeManifestSummary,
  createExportManifest,
  filterResourceGroups,
  filterSecretsInGroup,
  getStrategyColorClass,
  getStrategyLabel,
  groupSecretsByResource,
  isSecretBlocking
} from "./utils";

const manifest: DeploymentManifestResponse = {
  scenario: "secrets-manager",
  tier: "tier-2-desktop",
  generated_at: "2026-07-23T00:00:00Z",
  resources: ["redis", "vault"],
  secrets: [
    {
      resource_name: "vault",
      secret_key: "VAULT_TOKEN",
      secret_type: "token",
      required: true,
      classification: "service",
      handling_strategy: "prompt",
      requires_user_input: true
    },
    {
      resource_name: "vault",
      secret_key: "VAULT_ADDR",
      secret_type: "endpoint",
      required: true,
      classification: "infrastructure",
      handling_strategy: "none",
      requires_user_input: false
    },
    {
      resource_name: "redis",
      secret_key: "REDIS_PASSWORD",
      secret_type: "password",
      required: false,
      classification: "service",
      handling_strategy: "generate",
      requires_user_input: false
    }
  ],
  summary: {
    total_secrets: 3,
    strategized_secrets: 2,
    requires_action: 1,
    blocking_secrets: ["vault/VAULT_ADDR"],
    classification_weights: {},
    strategy_breakdown: {},
    scope_readiness: {}
  }
};

describe("manifest-editor deployment utilities", () => {
  it("groups, sorts, and identifies excluded resources without losing blocker counts", () => {
    const groups = groupSecretsByResource(
      manifest.secrets,
      new Set(["redis"]),
      new Set(["vault:VAULT_ADDR"])
    );

    expect(groups.map((group) => group.resourceName)).toEqual(["redis", "vault"]);
    expect(groups[0]).toMatchObject({ totalCount: 1, strategizedCount: 1, blockingCount: 0, isFullyExcluded: true });
    expect(groups[1]).toMatchObject({ totalCount: 2, strategizedCount: 1, blockingCount: 1, isFullyExcluded: false });
  });

  it("filters the deployment tree by search, blockers, overrides, and exclusions", () => {
    const groups = groupSecretsByResource(manifest.secrets, new Set(), new Set());
    const overridden = new Set(["vault:VAULT_TOKEN"]);
    const excluded = new Set(["vault:VAULT_ADDR"]);

    expect(filterResourceGroups(groups, "blocking", new Set(), excluded, overridden, "").map((group) => group.resourceName)).toEqual(["vault"]);
    expect(filterResourceGroups(groups, "overridden", new Set(), excluded, overridden, "").map((group) => group.resourceName)).toEqual(["vault"]);
    expect(filterResourceGroups(groups, "excluded", new Set(), excluded, overridden, "").map((group) => group.resourceName)).toEqual(["vault"]);
    expect(filterResourceGroups(groups, "all", new Set(), excluded, overridden, "redis").map((group) => group.resourceName)).toEqual(["redis"]);
    expect(filterSecretsInGroup(manifest.secrets, "blocking", excluded, overridden, "").map((secret) => secret.secret_key)).toEqual(["VAULT_ADDR"]);
    expect(filterSecretsInGroup(manifest.secrets, "overridden", excluded, overridden, "vault").map((secret) => secret.secret_key)).toEqual(["VAULT_TOKEN"]);
    expect(filterSecretsInGroup(manifest.secrets, "excluded", excluded, overridden, "").map((secret) => secret.secret_key)).toEqual(["VAULT_ADDR"]);
  });

  it("derives operator-facing summary counts while respecting session exclusions", () => {
    expect(
      computeManifestSummary(
        manifest,
        new Set(["redis"]),
        new Set(["vault:VAULT_ADDR"]),
        new Set(["vault:VAULT_TOKEN"])
      )
    ).toEqual({
      totalSecrets: 3,
      strategizedSecrets: 2,
      blockingSecrets: 0,
      excludedSecrets: 2,
      overriddenSecrets: 1,
      resourceCount: 2
    });
  });

  it("exports only permitted secrets and recomputes the downstream bundle summary", () => {
    const exported = createExportManifest(manifest, new Set(["redis"]), new Set(["vault:VAULT_TOKEN"]));

    expect(exported.resources).toEqual(["vault"]);
    expect(exported.secrets.map((secret) => secret.secret_key)).toEqual(["VAULT_ADDR"]);
    expect(exported.summary).toMatchObject({
      total_secrets: 1,
      strategized_secrets: 0,
      requires_action: 1,
      blocking_secrets: ["vault/VAULT_ADDR"]
    });
  });

  it("maps strategy states and stable secret identifiers predictably", () => {
    const vaultToken = manifest.secrets.find((secret) => secret.secret_key === "VAULT_TOKEN");
    const vaultAddress = manifest.secrets.find((secret) => secret.secret_key === "VAULT_ADDR");
    if (!vaultToken || !vaultAddress) {
      throw new Error("manifest fixture is missing expected Vault secrets");
    }

    expect(isSecretBlocking(vaultAddress)).toBe(true);
    expect(isSecretBlocking(vaultToken)).toBe(false);
    expect(getStrategyLabel("prompt")).toBe("Prompt (ask user)");
    expect(getStrategyLabel("unknown")).toBe("unknown");
    expect(getStrategyColorClass("generate")).toContain("cyan");
    expect(getStrategyColorClass("unknown")).toContain("white");
    expect(secretIdToString({ resource: "vault", key: "VAULT_TOKEN" })).toBe("vault:VAULT_TOKEN");
    expect(stringToSecretId("vault:VAULT_TOKEN")).toEqual({ resource: "vault", key: "VAULT_TOKEN" });
    expect(stringToSecretId("invalid")).toBeNull();
  });
});
