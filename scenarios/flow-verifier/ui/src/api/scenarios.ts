// API client for the scenarios aggregate. Thin wrapper over the
// generated ScenariosService Connect client; the public types preserve
// the camelCase field names the existing UI consumes.
import { createClient } from "@connectrpc/connect";

import { transport } from "./client";
import { flowSummaryFromProto, type FlowSummary } from "./inventory";

import {
  ScenariosService,
  type ScenarioSummary as ProtoScenarioSummary,
  type ScenarioDetail as ProtoScenarioDetail,
} from "@vrooli/proto-types/flow-verifier/v1/scenarios/scenarios_pb";

export const scenariosClient = createClient(ScenariosService, transport);

export type ScenarioSummary = {
  id: string;
  displayName: string;
  description?: string;
  path: string;
  flowCount: number;
  discoveryError?: string;
};

export type ScenariosListResponse = {
  vrooliRoot: string;
  scenarios: ScenarioSummary[];
};

export type ScenarioDetail = ScenarioSummary & {
  flows: FlowSummary[];
};

export async function fetchScenarios(): Promise<ScenariosListResponse> {
  const resp = await scenariosClient.listScenarios({});
  return {
    vrooliRoot: resp.vrooliRoot,
    scenarios: resp.scenarios.map(summaryFromProto),
  };
}

export async function fetchScenarioDetail(id: string): Promise<ScenarioDetail> {
  const resp = await scenariosClient.getScenario({ id });
  if (!resp.scenario) throw new Error("server returned no scenario");
  return detailFromProto(resp.scenario);
}

function summaryFromProto(p: ProtoScenarioSummary): ScenarioSummary {
  return {
    id: p.id,
    displayName: p.displayName,
    description: p.description || undefined,
    path: p.path,
    flowCount: p.flowCount,
    discoveryError: p.discoveryError || undefined,
  };
}

function detailFromProto(p: ProtoScenarioDetail): ScenarioDetail {
  const summary = p.summary
    ? summaryFromProto(p.summary)
    : { id: "", displayName: "", path: "", flowCount: 0 };
  return {
    ...summary,
    flows: p.flows.map(flowSummaryFromProto),
  };
}
