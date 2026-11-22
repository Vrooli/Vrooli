import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  fetchCaptureStorageStats,
  fetchScreenshotPath,
  fetchVisualCaptureDetail,
  fetchVisualCaptures,
} from "./api-visual";

const mocks = vi.hoisted(() => ({
  listCaptures: vi.fn(),
  getCaptureDetail: vi.fn(),
  getStorageStats: vi.fn(),
  getScreenshotPath: vi.fn(),
}));

vi.mock("./connect", () => ({
  visualCaptureClient: {
    listCaptures: mocks.listCaptures,
    getCaptureDetail: mocks.getCaptureDetail,
    getStorageStats: mocks.getStorageStats,
    getScreenshotPath: mocks.getScreenshotPath,
  },
}));

describe("visual capture typed read adapters", () => {
  beforeEach(() => vi.clearAllMocks());

  it("preserves scoped list metadata and staleness", async () => {
    mocks.listCaptures.mockResolvedValue({
      snapshots: [{ id: "cap-1", scenarioSlug: "demo", role: "capture", createdAt: "2026-11-21T10:00:00Z", status: "complete", pages: ["/"], presets: [] }],
      total: 1,
      staleness: { isStale: true, lastFileChange: "2026-11-21T10:01:00Z", captureCreatedAt: "2026-11-21T10:00:00Z" },
    });

    const result = await fetchVisualCaptures("demo", "42");

    expect(mocks.listCaptures).toHaveBeenCalledWith(expect.objectContaining({ repositoryId: 42n, scenarioSlug: "demo" }));
    expect(result.total).toBe(1);
    expect(result.snapshots[0]).toMatchObject({ id: "cap-1", role: "capture", createdAt: "2026-11-21T10:00:00Z" });
    expect(result.staleness).toEqual({ isStale: true, lastFileChange: "2026-11-21T10:01:00Z", captureCreatedAt: "2026-11-21T10:00:00Z" });
  });

  it("converts capture detail while retaining media descriptors", async () => {
    mocks.getCaptureDetail.mockResolvedValue({
      snapshot: { id: "cap-2", scenarioSlug: "demo", createdAt: "2026-11-21T10:00:00Z", status: "complete", pages: ["/"], presets: [] },
      screenshots: [{ filename: "home.png", sizeBytes: 123n, viewportWidth: 1440, viewportHeight: 900, theme: "dark" }],
      videos: [{ filename: "home.webm", sizeBytes: 456n }],
    });

    const result = await fetchVisualCaptureDetail("cap-2", "demo", "42");

    expect(mocks.getCaptureDetail).toHaveBeenCalledWith(expect.objectContaining({ repositoryId: 42n, scenarioSlug: "demo", captureId: "cap-2" }));
    expect(result).toMatchObject({ id: "cap-2", screenshots: [{ filename: "home.png", sizeBytes: 123, viewportWidth: 1440 }], videos: [{ filename: "home.webm", sizeBytes: 456 }] });
  });

  it("converts storage totals without losing per-scenario accounting", async () => {
    mocks.getStorageStats.mockResolvedValue({ totalSizeBytes: 2048n, snapshotCount: 3, perScenario: [{ scenarioSlug: "demo", snapshotCount: 3, sizeBytes: 2048n }] });

    await expect(fetchCaptureStorageStats("42")).resolves.toEqual({
      totalSizeBytes: 2048,
      snapshotCount: 3,
      perScenario: [{ scenarioSlug: "demo", snapshotCount: 3, sizeBytes: 2048 }],
    });
    expect(mocks.getStorageStats).toHaveBeenCalledWith(expect.objectContaining({ repositoryId: 42n }));
  });

  it("uses the typed path lookup for local agent context", async () => {
    mocks.getScreenshotPath.mockResolvedValue({ path: "/data/gct/42/demo/cap-1/home.png" });

    await expect(fetchScreenshotPath("cap-1", "demo", "home.png", "42")).resolves.toBe("/data/gct/42/demo/cap-1/home.png");
    expect(mocks.getScreenshotPath).toHaveBeenCalledWith(expect.objectContaining({ repositoryId: 42n, scenarioSlug: "demo", captureId: "cap-1", filename: "home.png" }));
  });
});
