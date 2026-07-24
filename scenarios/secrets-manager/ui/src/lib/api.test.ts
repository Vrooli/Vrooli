import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => "/api",
  buildApiUrl: (path: string, options: { baseUrl: string }) => `${options.baseUrl}${path}`
}));

import {
  copyOverridesFromScenario,
  copyOverridesFromTier,
  createAllowlistRule,
  createWatchlistEntry,
  deleteAllowlistRule,
  deleteScenarioOverride,
  deleteWatchlistEntry,
  fetchAllowlistRules,
  fetchCampaigns,
  fetchCompliance,
  fetchDeploymentReadiness,
  fetchEffectiveStrategies,
  fetchHealth,
  fetchOrientationSummary,
  fetchResourceDetail,
  fetchScenarioOverride,
  fetchScenarioOverrides,
  fetchScenarioTierOverrides,
  fetchScenarios,
  fetchVaultStatus,
  fetchVulnerabilities,
  fetchWatchlist,
  generateDeploymentManifest,
  provisionSecrets,
  setScenarioOverride,
  updateAllowlistRule,
  updateResourceSecret,
  updateSecretStrategy,
  updateVulnerabilityStatus
} from "./api";

const fetchMock = vi.fn();

function response(body: unknown = { ok: true }, status = 200, statusText = "OK") {
  return new Response(status === 204 ? null : JSON.stringify(body), {
    status,
    statusText,
    headers: { "Content-Type": "application/json" }
  });
}

async function expectRequest(
  invoke: () => Promise<unknown>,
  path: string,
  init: Partial<RequestInit> = {}
) {
  fetchMock.mockResolvedValueOnce(response());
  await invoke();
  expect(fetchMock).toHaveBeenLastCalledWith(`/api${path}`, expect.objectContaining({
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
    ...init
  }));
}

describe("Secrets Manager API contract client", () => {
  afterEach(() => {
    fetchMock.mockReset();
  });

  it("uses the shared API base for health, Vault, compliance, and discovery reads", async () => {
    await expectRequest(fetchHealth, "/health");
    await expectRequest(() => fetchVaultStatus("vault local"), "/vault/secrets/status?resource=vault%20local");
    await expectRequest(() => fetchVaultStatus(), "/vault/secrets/status");
    await expectRequest(fetchCompliance, "/security/compliance");
    await expectRequest(fetchAllowlistRules, "/security/allowlist-rules");
    await expectRequest(fetchWatchlist, "/security/watchlist");
    await expectRequest(fetchOrientationSummary, "/orientation/summary");
    await expectRequest(fetchScenarios, "/scenarios");
    expect(fetchMock).toHaveBeenCalledTimes(8);
  });

  it("serializes security and deployment mutations with encoded path segments", async () => {
    await expectRequest(
      () => createAllowlistRule({ path_pattern: "**/*.env", excluded_types: ["secret"] }),
      "/security/allowlist-rules",
      { method: "POST", body: JSON.stringify({ path_pattern: "**/*.env", excluded_types: ["secret"] }) }
    );
    await expectRequest(
      () => updateAllowlistRule("rule/one", { path_pattern: "tmp/*", excluded_types: [], enabled: false }),
      "/security/allowlist-rules/rule%2Fone",
      { method: "PUT", body: JSON.stringify({ path_pattern: "tmp/*", excluded_types: [], enabled: false }) }
    );
    await expectRequest(() => deleteAllowlistRule("rule/one"), "/security/allowlist-rules/rule%2Fone", { method: "DELETE" });
    await expectRequest(
      () => createWatchlistEntry({ label: "operator", value: "operator@example.test", value_type: "email" }),
      "/security/watchlist",
      { method: "POST", body: JSON.stringify({ label: "operator", value: "operator@example.test", value_type: "email" }) }
    );
    await expectRequest(() => deleteWatchlistEntry("entry/one"), "/security/watchlist/entry%2Fone", { method: "DELETE" });
    await expectRequest(
      () => generateDeploymentManifest({ scenario: "secrets-manager", tier: "tier-2-desktop", include_optional: true }),
      "/deployment/secrets",
      { method: "POST", body: JSON.stringify({ scenario: "secrets-manager", tier: "tier-2-desktop", include_optional: true }) }
    );
    await expectRequest(
      () => fetchDeploymentReadiness({ scenario: "secrets-manager", tier: "tier-1-local-dev", resources: ["vault"] }),
      "/deployment/readiness",
      { method: "POST", body: JSON.stringify({ scenario: "secrets-manager", tier: "tier-1-local-dev", resources: ["vault"] }) }
    );
    expect(fetchMock).toHaveBeenCalledTimes(7);
  });

  it("keeps resource and secret mutations path-safe and preserves request bodies", async () => {
    await expectRequest(() => fetchResourceDetail("vault/local"), "/resources/vault%2Flocal");
    await expectRequest(
      () => updateResourceSecret("vault/local", "VAULT/TOKEN", { required: true, classification: "service" }),
      "/resources/vault%2Flocal/secrets/VAULT%2FTOKEN",
      { method: "PATCH", body: JSON.stringify({ required: true, classification: "service" }) }
    );
    await expectRequest(
      () => updateSecretStrategy("vault/local", "VAULT/TOKEN", { tier: "tier-2-desktop", handling_strategy: "prompt" }),
      "/resources/vault%2Flocal/secrets/VAULT%2FTOKEN/strategy",
      { method: "POST", body: JSON.stringify({ tier: "tier-2-desktop", handling_strategy: "prompt" }) }
    );
    await expectRequest(
      () => updateVulnerabilityStatus("vuln/one", { status: "resolved", assigned_to: "operator" }),
      "/vulnerabilities/vuln%2Fone/status",
      { method: "POST", body: JSON.stringify({ status: "resolved", assigned_to: "operator" }) }
    );
    await expectRequest(
      () => provisionSecrets({ resource: "vault", secrets: { VAULT_TOKEN: "never-rendered" } }),
      "/secrets/provision",
      { method: "POST", body: JSON.stringify({ resource: "vault", secrets: { VAULT_TOKEN: "never-rendered" } }) }
    );
    expect(fetchMock).toHaveBeenCalledTimes(5);
  });

  it("encodes filters, campaign options, and all scenario override operations", async () => {
    await expectRequest(
      () => fetchVulnerabilities({ component: "vault server", componentType: "resource", severity: "high" }),
      "/vulnerabilities?component=vault+server&component_type=resource&severity=high"
    );
    await expectRequest(() => fetchVulnerabilities({}), "/vulnerabilities");
    await expectRequest(() => fetchCampaigns({ includeReadiness: true, scenario: "secrets manager" }), "/campaigns?include_readiness=true&scenario=secrets+manager");
    await expectRequest(() => fetchCampaigns(), "/campaigns");
    await expectRequest(() => fetchScenarioOverrides("secrets manager"), "/scenarios/secrets%20manager/overrides");
    await expectRequest(() => fetchScenarioTierOverrides("secrets manager", "tier/2"), "/scenarios/secrets%20manager/overrides/tier%2F2");
    await expectRequest(() => fetchScenarioOverride("secrets manager", "tier/2", "vault/local", "VAULT/TOKEN"), "/scenarios/secrets%20manager/overrides/tier%2F2/vault%2Flocal/VAULT%2FTOKEN");
    await expectRequest(
      () => setScenarioOverride("secrets manager", "tier/2", "vault/local", "VAULT/TOKEN", { handling_strategy: "prompt" }),
      "/scenarios/secrets%20manager/overrides/tier%2F2/vault%2Flocal/VAULT%2FTOKEN",
      { method: "POST", body: JSON.stringify({ handling_strategy: "prompt" }) }
    );
    await expectRequest(() => deleteScenarioOverride("secrets manager", "tier/2", "vault/local", "VAULT/TOKEN"), "/scenarios/secrets%20manager/overrides/tier%2F2/vault%2Flocal/VAULT%2FTOKEN", { method: "DELETE" });
    await expectRequest(() => fetchEffectiveStrategies("secrets manager", "tier/2", ["vault", "redis"]), "/scenarios/secrets%20manager/effective/tier%2F2?resources=vault%2Credis");
    await expectRequest(() => fetchEffectiveStrategies("secrets manager", "tier/2"), "/scenarios/secrets%20manager/effective/tier%2F2");
    await expectRequest(
      () => copyOverridesFromTier("secrets manager", { source_tier: "tier-1", target_tier: "tier-2", overwrite: true }),
      "/scenarios/secrets%20manager/overrides/copy-from-tier",
      { method: "POST", body: JSON.stringify({ source_tier: "tier-1", target_tier: "tier-2", overwrite: true }) }
    );
    await expectRequest(
      () => copyOverridesFromScenario("secrets manager", { source_scenario: "template", tier: "tier-2" }),
      "/scenarios/secrets%20manager/overrides/copy-from-scenario",
      { method: "POST", body: JSON.stringify({ source_scenario: "template", tier: "tier-2" }) }
    );
    expect(fetchMock).toHaveBeenCalledTimes(13);
  });

  it("returns no payload for successful deletes and surfaces server failures", async () => {
    fetchMock.mockResolvedValueOnce(response(undefined, 204));
    await expect(deleteAllowlistRule("completed")).resolves.toBeUndefined();

    fetchMock.mockResolvedValueOnce(response({ error: "unavailable" }, 503, "Service Unavailable"));
    await expect(fetchHealth()).rejects.toThrow("Request failed (503): Service Unavailable");
  });
});

Object.defineProperty(globalThis, "fetch", { value: fetchMock, writable: true });
