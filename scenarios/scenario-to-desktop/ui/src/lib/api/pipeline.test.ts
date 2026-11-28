import { describe, expect, it, vi } from "vitest";
import {
  Platform,
  StageName,
  StageStatus,
} from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";

const client = vi.hoisted(() => ({
  run: vi.fn(),
  get: vi.fn(),
  resume: vi.fn(),
  cancel: vi.fn(),
  getActive: vi.fn(),
  createActive: vi.fn(),
  resetActive: vi.fn(),
  getHistory: vi.fn(),
  startActive: vi.fn(),
}));

vi.mock("./connect", () => ({
  pipelineConnectClient: client,
}));

import {
  cancelPipeline,
  createNewPipeline,
  getActivePipeline,
  getPipelineHistory,
  getPipelineStatus,
  resetPipeline,
  resumePipeline,
  runPipeline,
  runPreflightPipeline,
  startActivePipeline,
} from "./pipeline";

describe("getPipelineStatus", () => {
  it("returns the generated protobuf stage payload without a legacy adapter", async () => {
    client.get.mockResolvedValue({
      pipelineId: "pipeline-1",
      scenarioName: "example",
      status: StageStatus.COMPLETED,
      progressPercent: 100,
      stages: {
        bundle: {
          stage: StageName.BUNDLE,
          status: StageStatus.COMPLETED,
          logs: ["bundle complete"],
          details: {
            kind: {
              case: "bundle",
              value: {
                bundleDir: "/tmp/bundle",
                manifestPath: "",
                runtimeBinaries: {},
                copiedArtifacts: [],
                totalSizeBytes: 0n,
                totalSizeHuman: "",
              },
            },
          },
        },
      },
      stageOrder: [StageName.BUNDLE],
      finalArtifacts: {},
    });

    const status = await getPipelineStatus("pipeline-1", { verbose: true });

    expect(status.stages.bundle).toMatchObject({
      stage: StageName.BUNDLE,
      status: StageStatus.COMPLETED,
      logs: ["bundle complete"],
      details: {
        kind: { case: "bundle", value: { bundleDir: "/tmp/bundle" } },
      },
    });
  });

  it("sends typed run and preflight configuration through PipelineService", async () => {
    client.run.mockResolvedValue({ pipelineId: "pipeline-2" });

    await expect(
      runPipeline({ scenarioName: "calculator", platforms: [Platform.LINUX] }),
    ).resolves.toEqual({ pipelineId: "pipeline-2" });
    await runPreflightPipeline("calculator", { platforms: [Platform.MAC] });

    expect(client.run).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({
        // Vitest asymmetric matchers are intentionally untyped at this boundary.
        // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
        config: expect.objectContaining({
          scenarioName: "calculator",
          platforms: [Platform.LINUX],
        }),
      }),
    );
    expect(client.run).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({
        // Vitest asymmetric matchers are intentionally untyped at this boundary.
        // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
        config: expect.objectContaining({
          scenarioName: "calculator",
          platforms: [Platform.MAC],
          stopAfterStage: StageName.PREFLIGHT,
        }),
      }),
    );
  });

  it("routes lifecycle operations to generated PipelineService methods", async () => {
    const pipeline = { pipelineId: "pipeline-3", stages: {}, stageOrder: [] };
    client.resume.mockResolvedValue({ pipelineId: "pipeline-3" });
    client.cancel.mockResolvedValue({ cancelled: true });
    client.getActive.mockResolvedValue({ pipeline, created: true });
    client.createActive.mockResolvedValue({
      pipeline,
      archivedPipelineId: "old",
    });
    client.resetActive.mockResolvedValue({
      archivedPipelineId: "old",
      cleared: true,
    });
    client.getHistory.mockResolvedValue({ pipelines: [pipeline], total: 1 });
    client.startActive.mockResolvedValue({ pipeline });

    await expect(resumePipeline("pipeline-3")).resolves.toEqual({
      pipelineId: "pipeline-3",
    });
    await expect(cancelPipeline("pipeline-3")).resolves.toEqual({
      cancelled: true,
    });
    await expect(
      getActivePipeline("calculator", { autoCreate: false }),
    ).resolves.toEqual({ pipeline, created: true });
    await expect(createNewPipeline("calculator")).resolves.toEqual({
      pipeline,
      archivedPipelineId: "old",
    });
    await expect(resetPipeline("calculator")).resolves.toEqual({
      archivedPipelineId: "old",
      cleared: true,
    });
    await expect(
      getPipelineHistory("calculator", { limit: 5 }),
    ).resolves.toEqual({ pipelines: [pipeline], total: 1 });
    await expect(
      startActivePipeline("calculator", { platforms: [Platform.WIN] }),
    ).resolves.toMatchObject({ pipeline });

    expect(client.resume).toHaveBeenCalledWith({ pipelineId: "pipeline-3" });
    expect(client.cancel).toHaveBeenCalledWith({ pipelineId: "pipeline-3" });
    expect(client.getActive).toHaveBeenCalledWith({
      scenarioName: "calculator",
      autoCreate: false,
    });
    expect(client.createActive).toHaveBeenCalledWith({
      scenarioName: "calculator",
      config: undefined,
    });
    expect(client.resetActive).toHaveBeenCalledWith({
      scenarioName: "calculator",
    });
    expect(client.getHistory).toHaveBeenCalledWith({
      scenarioName: "calculator",
      limit: 5,
    });
    expect(client.startActive).toHaveBeenCalledWith(
      expect.objectContaining({ scenarioName: "calculator" }),
    );
  });

  it("rejects malformed generated lifecycle responses instead of inventing a legacy fallback", async () => {
    client.createActive.mockResolvedValue({ archivedPipelineId: "old" });
    client.startActive.mockResolvedValue({});

    await expect(createNewPipeline("calculator")).rejects.toThrow(
      "CreateActive returned no pipeline",
    );
    await expect(startActivePipeline("calculator")).rejects.toThrow(
      "StartActive returned no pipeline",
    );
  });
});
