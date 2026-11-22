import { create } from "@bufbuild/protobuf";
import { createClient, type Transport } from "@connectrpc/connect";
import { createScenarioConnectTransport } from "@vrooli/api-base";
import {
  RunsService,
  SelfHealthSchema,
  type ArtifactRef,
  type ListRunArtifactsResponse,
  type CatalogSummary,
  type PhaseReliability,
  type ProviderConformance,
  type ReliabilityLedger,
  type SelfHealth,
  type SelfHealthTrendPoint,
  type TrendDelta,
} from "@vrooli/proto-types/test-genie/v1/runs/runs_pb";

export type {
  ArtifactRef,
  CatalogSummary,
  ListRunArtifactsResponse,
  PhaseReliability,
  ProviderConformance,
  ReliabilityLedger,
  SelfHealth,
  SelfHealthTrendPoint,
  TrendDelta,
};

export function createRunsApi(transport: Transport = createScenarioConnectTransport()) {
  const client = createClient(RunsService, transport);
  return {
    async listRunArtifacts(target: string, runId: string): Promise<ListRunArtifactsResponse> {
      return client.listRunArtifacts({ target, runId });
    },
    async getSelfHealth(options: {
      windowDays?: number;
      skipConformance?: boolean;
      includeTrend?: boolean;
    } = {}): Promise<SelfHealth> {
      const response = await client.getSelfHealth({
        windowDays: options.windowDays ?? 0,
        skipConformance: options.skipConformance ?? false,
        includeTrend: options.includeTrend ?? false,
      });
      return response.selfHealth ?? create(SelfHealthSchema, {});
    },
  };
}

const runsApi = createRunsApi();
export const { listRunArtifacts, getSelfHealth } = runsApi;
