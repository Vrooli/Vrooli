import { createRouterTransport } from "@connectrpc/connect";
import { describe, expect, it } from "vitest";
import { RunsService } from "@vrooli/proto-types/test-genie/v1/runs/runs_pb";
import { createRunsApi } from "./runs";

describe("RunsService API client", () => {
  it("uses generated requests and responses for self-health and artifact catalog calls", async () => {
    let artifactTarget = "";
    let artifactRunId = "";
    let selfHealthOptions: { windowDays: number; skipConformance: boolean; includeTrend: boolean } | undefined;
    const transport = createRouterTransport(({ service }) => {
      service(RunsService, {
        listRunArtifacts: (request) => {
          artifactTarget = request.target;
          artifactRunId = request.runId;
          return { schemaVersion: 1, digest: "sha256:catalog", artifacts: [], legacyDiscovered: false, degradedReasons: [] };
        },
        getSelfHealth: (request) => {
          selfHealthOptions = {
            windowDays: request.windowDays,
            skipConformance: request.skipConformance,
            includeTrend: request.includeTrend,
          };
          return { selfHealth: { conformanceFreshness: "cached" } };
        },
      });
    });
    const api = createRunsApi(transport);

    const catalog = await api.listRunArtifacts("scenario:demo", "run-1");
    const health = await api.getSelfHealth({ windowDays: 14, skipConformance: true, includeTrend: true });

    expect(artifactTarget).toBe("scenario:demo");
    expect(artifactRunId).toBe("run-1");
    expect(catalog.digest).toBe("sha256:catalog");
    expect(selfHealthOptions).toEqual({ windowDays: 14, skipConformance: true, includeTrend: true });
    expect(health.conformanceFreshness).toBe("cached");
  });
});
