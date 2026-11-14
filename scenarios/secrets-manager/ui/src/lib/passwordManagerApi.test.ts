import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => "/api",
  buildApiUrl: (path: string, options: { baseUrl: string }) => `${options.baseUrl}${path}`
}));

import {
  approveAccessRequest,
  bindVaultItem,
  commitVaultImport,
  completeEnrollment,
  createAccessRequest,
  createGrant,
  createSource,
  createVaultExport,
  createVaultItem,
  denyAccessRequest,
  getGrantEffectiveAccess,
  generatePassword,
  getAudit,
  getEnrollmentStatus,
  getRecoveryStatus,
  issueAssurance,
  getSourceHealth,
  getVaultStatus,
  listAccessRequests,
  listGrants,
  listSources,
  listVaultItemHistory,
  listVaultItems,
  loginWithAuthenticator,
  lockVault,
  previewVaultImport,
  redeemVaultExport,
  restoreVaultItem,
  revokeGrant,
  revealVaultItem,
  setConfiguredOwnerToken,
  setOwnerToken,
  trashVaultItem,
  unlockVault,
  updateVaultItem
} from "./passwordManagerApi";

const fetchMock = vi.fn();

function response(body: unknown = {}) {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" }
  });
}

describe("password manager API client", () => {
  afterEach(() => {
    fetchMock.mockReset();
    setOwnerToken("");
    Object.defineProperty(window, "__SECRETS_MANAGER_CONFIG__", {
      configurable: true,
      value: undefined
    });
  });

  it("covers the vault, policy, source, and recovery operations", async () => {
    fetchMock.mockImplementation((url: string) => {
      if (url.endsWith("/assurance")) return Promise.resolve(response({ assurance_token: "assurance" }));
      if (url.includes("/reveal")) return Promise.resolve(response({ fields: { password: "value" } }));
      return Promise.resolve(response());
    });
    Object.defineProperty(globalThis, "fetch", { configurable: true, value: fetchMock, writable: true });

    await getVaultStatus("vault/one");
    await getEnrollmentStatus();
    await completeEnrollment("bootstrap");
    await unlockVault("vault/one");
    await lockVault("vault/one");
    await listVaultItems("site", true, "vault/one");
    await createVaultItem({ name: "site", password: "value" }, "vault/one");
    await updateVaultItem("item/one", 3, { name: "renamed" }, "vault/one");
    await trashVaultItem("item/one", "vault/one");
    await restoreVaultItem("item/one", "vault/one");
    await listVaultItemHistory("item/one", "vault/one");
    await getRecoveryStatus();
    await previewVaultImport("bundle", "vault/one");
    await commitVaultImport("bundle", "rename", "vault/one");
    await revealVaultItem("item/one", ["password", "username"], "vault/one");
    await generatePassword(32);
    await createGrant({ item_id: "item/one" });
    await listGrants();
    await getGrantEffectiveAccess("grant/one", "item/one", "use");
    await revokeGrant("grant/one");
    await createAccessRequest({ grant_id: "grant/one" });
    await listAccessRequests();
    await getAudit();
    await approveAccessRequest("request/one", "digest");
    await issueAssurance("access-request:request/one:approve", "digest");
    await denyAccessRequest("request/one", "digest");
    await createVaultExport("native", "vault/one", ["item/one"]);
    await createVaultExport("plaintext", "vault/one", ["item/one"]);
    await redeemVaultExport({
      download_handle: "handle/one",
      download_token: "download-token",
      expires_at: "2026-09-06T00:00:00Z",
      format: "native",
      secret_bearing: false
    });
    await listSources();
    await createSource({ kind: "native", label: "Local" });
    await getSourceHealth("source/one");
    await bindVaultItem("item/one", "source/one", "external/one", "vault/one");

    expect(fetchMock).toHaveBeenCalledTimes(35);
  });

  it("uses configured authentication and reports failed requests", async () => {
    Object.defineProperty(window, "__SECRETS_MANAGER_CONFIG__", {
      configurable: true,
      value: { apiBaseUrl: "https://configured.test/api", ownerToken: "configured-token" }
    });
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify({ message: "denied" }), { status: 403 }));
    Object.defineProperty(globalThis, "fetch", { configurable: true, value: fetchMock, writable: true });

    await expect(getVaultStatus()).rejects.toThrow("denied");
    const failedRequest = fetchMock.mock.calls[0]?.[1] as RequestInit;
    expect(failedRequest.cache).toBe("no-store");
    expect((failedRequest.headers as Headers).get("Authorization")).toBe("Bearer configured-token");

    setConfiguredOwnerToken("explicit-token");
    fetchMock.mockResolvedValueOnce(response({ status: "unlocked" }));
    await getVaultStatus();
    const explicitRequest = fetchMock.mock.calls[1]?.[1] as RequestInit;
    expect((explicitRequest.headers as Headers).get("Authorization")).toBe("Bearer explicit-token");
  });

  it("uses the same-origin relying-party login endpoint", async () => {
    fetchMock.mockResolvedValueOnce(response({ access_token: "session-token" }));
    Object.defineProperty(globalThis, "fetch", { configurable: true, value: fetchMock, writable: true });

    await expect(loginWithAuthenticator("owner@example.com", "password-for-test")).resolves.toEqual({ access_token: "session-token" });
    expect(fetchMock).toHaveBeenCalledWith("/api/auth/login", expect.objectContaining({ method: "POST" }));
    expect(JSON.parse((fetchMock.mock.calls[0]?.[1] as RequestInit).body as string)).toEqual({ email: "owner@example.com", password: "password-for-test" });
  });
});
