import { afterEach, describe, expect, it, vi } from "vitest";
import * as api from "./api";

describe("deployment-manager API client", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("serializes the complete operator API surface through one fetch boundary", async () => {
    const profile = { id: "p1", name: "Production", scenario: "demo", tiers: [2], version: 1 };
    const connectResponse = (body: unknown) => ({
      ok: true,
      status: 200,
      headers: new Headers({ "content-type": "application/json" }),
      json: async () => body,
    });
    const fetchMock = vi.fn((input: RequestInfo | URL, _init?: RequestInit) => {
      const url = typeof input === "object" && "url" in input ? input.url : String(input);
      if (url.includes("ProfilesService/ListProfiles")) return Promise.resolve(connectResponse({ profiles: [] }));
      if (url.includes("ProfilesService/CreateProfile")) return Promise.resolve(connectResponse({ profile }));
      if (url.includes("ProfilesService/GetProfile") || url.includes("ProfilesService/UpdateProfile")) {
        return Promise.resolve(connectResponse({ profile }));
      }
      return Promise.resolve(connectResponse({ path: "/tmp/telemetry" }));
    });
    vi.stubGlobal("fetch", fetchMock);
    const request = { git_commit_hash: "abc", platform: "linux" };
    const approval = { decision: "approved" as const, reviewer: "operator" };
    const file = { text: vi.fn().mockResolvedValue('{"event":"ready"}\n') } as unknown as File;

    await Promise.all([
      api.fetchHealth(),
      api.analyzeDependencies("demo"),
      api.scoreFitness({ scenario: "demo", tiers: [2] }),
      api.listProfiles(),
      api.createProfile(profile),
      api.getProfile("p1"),
      api.updateProfile("p1", { name: "Updated" }),
      api.deployProfile("p1"),
      api.getDeploymentStatus("d1"),
      api.analyzeSwap("redis", "valkey"),
      api.analyzeSwapCascade("redis", "valkey"),
      api.uploadTelemetry("demo", file),
      api.listTelemetry(),
      api.reportMigrationTask({ scenario: "demo", from_dependency: "redis", to_dependency: "valkey" }),
      api.getMigrationTaskStatus("demo"),
      api.listApprovals("p1", "abc"),
      api.getApproval("a1"),
      api.createApproval("p1", request),
      api.decideApproval("a1", approval),
      api.checkReleaseGate("p1", "abc"),
      api.getEvidenceReview("p1", "abc"),
      api.setRequiredPlatforms("p1", ["linux"]),
      api.getRequiredPlatforms("p1"),
      api.getProfileLPBSConfig("p1"),
      api.saveProfileLPBSConfig("p1", { lpbs_domain: "example.com" }),
      api.listProfileReleases("p1"),
      api.getRelease("r1"),
      api.reverifyRelease("r1", true),
      api.startRelease("p1", { git_commit_hash: "abc", release_version: "1.0.0" }),
    ]);

    expect(fetchMock).toHaveBeenCalledTimes(29);
    expect(fetchMock.mock.calls.some(([, init]) => (init as RequestInit).method === "POST")).toBe(true);
  });

  it("rejects malformed telemetry before sending it", async () => {
    const fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
    const file = { text: vi.fn().mockResolvedValue("not-json") } as unknown as File;
    await expect(api.uploadTelemetry("demo", file)).rejects.toThrow("Line 1 is not valid JSON");
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("projects API error payloads into actionable errors", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({
      ok: false,
      status: 503,
      headers: new Headers({ "content-type": "application/json" }),
      json: async () => ({ code: "unavailable", message: "database unavailable" }),
    }));
    await expect(api.listProfiles()).rejects.toThrow("database unavailable");
  });
});
