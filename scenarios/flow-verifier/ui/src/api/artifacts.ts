// API client for the artifacts (codegen lifecycle) surface. Thin
// wrapper over the generated ArtifactsService + ScenariosService Connect
// clients; preserves the public types existing consumers depend on.
import { createClient } from "@connectrpc/connect";

import { transport } from "./client";
import { scenariosClient } from "./scenarios";

import {
  ArtifactsService,
  ArtifactStatus as ProtoArtifactStatus,
  type ArtifactReport as ProtoArtifactReport,
  type ArtifactFile as ProtoArtifactFile,
  type ClearArtifactsResponse as ProtoClearArtifactsResponse,
} from "@vrooli/proto-types/flow-verifier/v1/artifacts/artifacts_pb";

export const artifactsClient = createClient(ArtifactsService, transport);

export type ArtifactStatus = "fresh" | "missing";

export type ArtifactFile = {
  path: string;
  exists: boolean;
  size?: number;
  mtime?: string;
};

export type ArtifactReport = {
  flowId: string;
  scenarioPath: string;
  generatedDir: string;
  status: ArtifactStatus;
  files: ArtifactFile[];
  missing: string[];
};

export type ClearResult = {
  flowId: string;
  removed: string[];
};

export type ScenarioArtifactsResult = {
  scenarioId: string;
  flows: ArtifactReport[];
};

export type ScenarioClearResult = {
  scenarioId: string;
  flows: ClearResult[];
};

export async function fetchArtifactsStatus(
  flowId: string,
  opts: { scenarioId?: string } = {},
): Promise<ArtifactReport> {
  const resp = await artifactsClient.getArtifactStatus({
    flowId,
    scenarioId: opts.scenarioId ?? "",
    root: "",
  });
  if (!resp.report) throw new Error("server returned no artifact report");
  return reportFromProto(resp.report);
}

export async function generateArtifacts(
  flowId: string,
  opts: { scenarioId?: string } = {},
): Promise<ArtifactReport> {
  const resp = await artifactsClient.generateArtifacts({
    flowId,
    scenarioId: opts.scenarioId ?? "",
    root: "",
  });
  if (!resp.report) throw new Error("server returned no artifact report");
  return reportFromProto(resp.report);
}

export async function clearArtifacts(
  flowId: string,
  opts: { scenarioId?: string } = {},
): Promise<ClearResult> {
  const resp = await artifactsClient.clearArtifacts({
    flowId,
    scenarioId: opts.scenarioId ?? "",
    root: "",
  });
  return { flowId: resp.flowId, removed: resp.removed };
}

// generateScenarioArtifacts consumes the server-streaming RPC and
// collects every per-flow message into a final array. UIs that want
// live progress can call scenariosClient.generateScenarioArtifacts
// directly and iterate the stream themselves.
export async function generateScenarioArtifacts(
  scenarioId: string,
): Promise<ScenarioArtifactsResult> {
  const flows: ArtifactReport[] = [];
  const stream = scenariosClient.generateScenarioArtifacts({ scenarioId });
  for await (const msg of stream) {
    if (msg.report) {
      flows.push(reportFromProto(msg.report));
    } else if (msg.errorMessage) {
      throw new Error(`generate failed for ${msg.flowId}: ${msg.errorMessage}`);
    }
  }
  return { scenarioId, flows };
}

export async function clearScenarioArtifacts(
  scenarioId: string,
): Promise<ScenarioClearResult> {
  const resp = await scenariosClient.clearScenarioArtifacts({ scenarioId });
  return {
    scenarioId,
    flows: resp.flows.map((f: ProtoClearArtifactsResponse) => ({
      flowId: f.flowId,
      removed: f.removed,
    })),
  };
}

function reportFromProto(r: ProtoArtifactReport): ArtifactReport {
  return {
    flowId: r.flowId,
    scenarioPath: r.scenarioPath,
    generatedDir: r.generatedDir,
    status: statusFromProto(r.status),
    files: r.files.map(fileFromProto),
    missing: r.missing,
  };
}

function fileFromProto(f: ProtoArtifactFile): ArtifactFile {
  return {
    path: f.path,
    exists: f.exists,
    size: f.size ? Number(f.size) : undefined,
    mtime: f.mtime ? new Date(Number(f.mtime.seconds) * 1000).toISOString() : undefined,
  };
}

function statusFromProto(s: ProtoArtifactStatus): ArtifactStatus {
  switch (s) {
    case ProtoArtifactStatus.FRESH:
      return "fresh";
    case ProtoArtifactStatus.MISSING:
    case ProtoArtifactStatus.STALE:
      return "missing";
  }
  return "missing";
}
