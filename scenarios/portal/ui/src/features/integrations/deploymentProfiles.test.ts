import { describe, expect, it } from "vitest";

import {
  DEPLOYMENT_PROFILE_STORAGE_KEY,
  evaluateDeploymentProfile,
  loadDeploymentProfileConfig,
  saveDeploymentProfileConfig,
  validateDeploymentEndpoint,
} from "./deploymentProfiles";

function storage(): Storage {
  const values = new Map<string, string>();
  return {
    get length() {
      return values.size;
    },
    clear: () => values.clear(),
    getItem: (key) => values.get(key) ?? null,
    key: (index) => [...values.keys()][index] ?? null,
    removeItem: (key) => values.delete(key),
    setItem: (key, value) => values.set(key, value),
  };
}

describe("deployment profiles", () => {
  it("normalizes safe endpoints and persists no credentials", () => {
    const target = storage();
    const saved = saveDeploymentProfileConfig({ profile: "automation", endpoint: "https://example.test/" }, target);
    expect(saved.endpoint).toBe("https://example.test");
    expect(target.getItem(DEPLOYMENT_PROFILE_STORAGE_KEY)).toBe(
      JSON.stringify({ version: 1, profile: "automation", endpoint: "https://example.test" }),
    );
    expect(loadDeploymentProfileConfig(target)).toEqual(saved);
  });

  it("rejects credentials and query data", () => {
    expect(() => validateDeploymentEndpoint("https://user:secret@example.test"))
      .toThrow(/credentials/);
    expect(() => validateDeploymentEndpoint("https://example.test?token=secret"))
      .toThrow(/credentials/);
  });

  it("keeps client usable while reporting missing optional profile capabilities", () => { // OPT-10
    const client = loadDeploymentProfileConfig(null);
    expect(evaluateDeploymentProfile(client, [])).toMatchObject({ ready: true, missingCapabilities: [] });
    expect(
      evaluateDeploymentProfile({ ...client, profile: "local-control" }, [{ id: "agent-manager", state: "available" }]),
    ).toMatchObject({ ready: false, missingCapabilities: ["device-control"] });
    expect(
      evaluateDeploymentProfile({ ...client, profile: "automation" }, [{ id: "agent-manager", state: "available" }]),
    ).toMatchObject({ ready: true, missingCapabilities: [] });
  });

  it("rejects unsafe endpoint shapes and recovers from invalid stored configuration", () => {
    expect(validateDeploymentEndpoint("   ")).toBe("");
    expect(() => validateDeploymentEndpoint("not a url")).toThrow(/valid URL/);
    expect(() => validateDeploymentEndpoint("ftp://example.test")).toThrow(/http or https/);
    expect(() => validateDeploymentEndpoint(`https://example.test/${"x".repeat(2048)}`)).toThrow(/too long/);
    expect(() => validateDeploymentEndpoint("https://example.test/#fragment")).toThrow(/query data/);

    const target = storage();
    for (const raw of ["", "null", "[]", "{\"version\":2}", "{\"version\":1,\"profile\":\"unknown\",\"endpoint\":\"\"}", "{\"version\":1,\"profile\":\"client\",\"endpoint\":\"bad\"}", "not-json"]) {
      target.setItem(DEPLOYMENT_PROFILE_STORAGE_KEY, raw);
      expect(loadDeploymentProfileConfig(target)).toEqual(defaultConfig());
    }
  });

  it("supports absent storage and preserves a safe no-storage configuration", () => {
    expect(loadDeploymentProfileConfig(undefined)).toEqual(defaultConfig());
    expect(saveDeploymentProfileConfig({ profile: "client", endpoint: "" }, null)).toEqual(defaultConfig());
    expect(() => saveDeploymentProfileConfig({ profile: "unknown" as never, endpoint: "" }, null)).toThrow(/unknown/);
  });
});

function defaultConfig() {
  return { version: 1 as const, profile: "client" as const, endpoint: "" };
}
