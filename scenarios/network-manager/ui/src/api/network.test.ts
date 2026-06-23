import { beforeEach, describe, expect, it, vi } from "vitest";

const clients = vi.hoisted(() => ({
  adapter: {
    listCapabilities: vi.fn(),
    getPlatformSummary: vi.fn(),
  },
  inventory: {
    refreshInventory: vi.fn(),
    listDevices: vi.fn(),
    updateDeviceGroup: vi.fn(),
  },
  optimization: {
    createOptimizationRun: vi.fn(),
    scoreCandidates: vi.fn(),
    approveCandidate: vi.fn(),
  },
  policy: {
    previewPolicyChange: vi.fn(),
    applyPolicyChange: vi.fn(),
    rollbackPolicyChange: vi.fn(),
  },
  privacy: {
    getRetentionSettings: vi.fn(),
    getVisibilitySettings: vi.fn(),
  },
  resolver: {
    getResolverStatus: vi.fn(),
    updateUpstreams: vi.fn(),
  },
  snapshot: {
    listSnapshots: vi.fn(),
    runSnapshot: vi.fn(),
    exportSnapshotReport: vi.fn(),
  },
}));

const createClientMock = vi.hoisted(() => vi.fn());

vi.mock("@connectrpc/connect", () => ({
  createClient: createClientMock,
}));

vi.mock("./client", () => ({ transport: {} }));

describe("network API wrappers", () => {
  beforeEach(() => {
    vi.resetModules();
    vi.clearAllMocks();
    createClientMock
      .mockReturnValueOnce(clients.adapter)
      .mockReturnValueOnce(clients.inventory)
      .mockReturnValueOnce(clients.optimization)
      .mockReturnValueOnce(clients.policy)
      .mockReturnValueOnce(clients.privacy)
      .mockReturnValueOnce(clients.resolver)
      .mockReturnValueOnce(clients.snapshot);
  });

  it("loads the control-center overview from every domain client", async () => {
    const snapshot = { id: "snapshot-1" };
    const status = { backend: "adguardhome" };
    const capability = { action: "resolver.status" };
    const platform = { os: "linux" };
    const device = { id: "device-1" };
    const retention = { queryLogDays: 0 };
    const visibility = { householdMode: true };
    clients.snapshot.listSnapshots.mockResolvedValueOnce({ snapshots: [snapshot] });
    clients.resolver.getResolverStatus.mockResolvedValueOnce({ status });
    clients.adapter.listCapabilities.mockResolvedValueOnce({ capabilities: [capability] });
    clients.adapter.getPlatformSummary.mockResolvedValueOnce({ summary: platform });
    clients.inventory.listDevices.mockResolvedValueOnce({ devices: [device] });
    clients.privacy.getRetentionSettings.mockResolvedValueOnce({ settings: retention });
    clients.privacy.getVisibilitySettings.mockResolvedValueOnce({ settings: visibility });

    const { fetchControlCenterOverview } = await import("./network");
    const overview = await fetchControlCenterOverview();

    expect(overview).toMatchObject({
      snapshots: [snapshot],
      resolverStatus: status,
      capabilities: [capability],
      platform,
      devices: [device],
      retention,
      visibility,
    });
  });

  it("forwards snapshot and resolver operations", async () => {
    clients.snapshot.runSnapshot.mockResolvedValueOnce({ snapshot: { id: "snapshot-2" } });
    clients.snapshot.exportSnapshotReport.mockResolvedValueOnce({ report: "report" });
    clients.resolver.getResolverStatus.mockResolvedValueOnce({ status: { status: "healthy" } });
    clients.resolver.updateUpstreams.mockResolvedValueOnce({ changes: ["preview"] });

    const { exportSnapshotReport, fetchResolverStatus, previewUpstreams, runSnapshot } = await import("./network");

    await expect(runSnapshot("home")).resolves.toMatchObject({ id: "snapshot-2" });
    await expect(exportSnapshotReport("snapshot-2")).resolves.toBe("report");
    await expect(fetchResolverStatus()).resolves.toMatchObject({ status: "healthy" });
    await expect(previewUpstreams(["https://dns.example.test/dns-query"])).resolves.toEqual(["preview"]);
    expect(clients.resolver.updateUpstreams).toHaveBeenCalledWith({
      upstreams: ["https://dns.example.test/dns-query"],
      dryRun: true,
    });
  });

  it("forwards policy, inventory, optimization, and privacy operations", async () => {
    clients.policy.previewPolicyChange.mockResolvedValueOnce({ preview: { id: "preview-1" } });
    clients.policy.applyPolicyChange.mockResolvedValueOnce({ change: { id: "change-1" } });
    clients.policy.rollbackPolicyChange.mockResolvedValueOnce({ change: { id: "rollback-1" } });
    clients.inventory.refreshInventory.mockResolvedValueOnce({ devices: [{ id: "device-1" }], findings: ["finding"] });
    clients.inventory.listDevices.mockResolvedValueOnce({ devices: [{ id: "device-2" }] });
    clients.inventory.updateDeviceGroup.mockResolvedValueOnce({ device: { id: "device-2", group: "trusted" } });
    clients.optimization.createOptimizationRun.mockResolvedValueOnce({ run: { id: "run-1" } });
    clients.optimization.scoreCandidates.mockResolvedValueOnce({ run: { id: "run-1", status: "scored" } });
    clients.optimization.approveCandidate.mockResolvedValueOnce({ run: { id: "run-1", status: "approved" } });
    clients.privacy.getRetentionSettings.mockResolvedValueOnce({ settings: { queryLogDays: 0 } });
    clients.privacy.getVisibilitySettings.mockResolvedValueOnce({ settings: { householdMode: true } });

    const network = await import("./network");

    await expect(network.previewPolicyChange({ target: "all", action: "denylist", values: ["example.test"] })).resolves.toMatchObject({ id: "preview-1" });
    await expect(network.applyPolicyChange("preview-1")).resolves.toMatchObject({ id: "change-1" });
    await expect(network.rollbackPolicyChange("change-1")).resolves.toMatchObject({ id: "rollback-1" });
    await expect(network.refreshInventory()).resolves.toMatchObject({ findings: ["finding"] });
    await expect(network.fetchDevices("trusted")).resolves.toEqual([{ id: "device-2" }]);
    await expect(network.updateDeviceGroup("device-2", "trusted")).resolves.toMatchObject({ group: "trusted" });
    await expect(network.createOptimizationRun()).resolves.toMatchObject({ id: "run-1" });
    await expect(network.scoreOptimizationRun("run-1")).resolves.toMatchObject({ status: "scored" });
    await expect(network.approveOptimizationCandidate("run-1", "candidate-1")).resolves.toMatchObject({ status: "approved" });
    await expect(network.fetchPrivacySettings()).resolves.toMatchObject({
      retention: { queryLogDays: 0 },
      visibility: { householdMode: true },
    });
  });
});
