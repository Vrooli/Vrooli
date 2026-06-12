import { create } from "@bufbuild/protobuf";
import { createClient } from "@connectrpc/connect";
import {
  CodeTargetSchema,
  CodeFactsService,
  FactFamily,
  TargetKind,
  type CodeFactsReport,
  type CodeTarget,
  type ListSurfacesResponse,
} from "@vrooli/proto-types/code-facts/v1/facts/facts_pb";

import { transport } from "./client";

export const factsClient = createClient(CodeFactsService, transport);

export function scenarioTarget(scenario: string): CodeTarget {
  return create(CodeTargetSchema, { kind: TargetKind.SCENARIO, scenario });
}

export function pathTarget(path: string): CodeTarget {
  return create(CodeTargetSchema, { kind: TargetKind.PATH, path });
}

export function moduleTarget(path: string): CodeTarget {
  return create(CodeTargetSchema, { kind: TargetKind.MODULE, path });
}

export function projectTarget(path: string): CodeTarget {
  return create(CodeTargetSchema, { kind: TargetKind.PROJECT, path });
}

export interface DescribeCodeFactsOptions {
  target: CodeTarget;
  include: FactFamily[];
  useCache: boolean;
}

export async function describeCodeFacts({
  target,
  include,
  useCache,
}: DescribeCodeFactsOptions): Promise<CodeFactsReport> {
  return factsClient.describeCodeFacts({
    target,
    include,
    endpointIds: [],
    commandIds: [],
    widgetIds: [],
    useCache,
  });
}

export async function describeScenario(scenario = "code-facts"): Promise<CodeFactsReport> {
  return describeCodeFacts({
    target: scenarioTarget(scenario),
    include: [FactFamily.SURFACES, FactFamily.PARSE_UNITS, FactFamily.PROTO_ADOPTION],
    useCache: true,
  });
}

export async function listScenarioSurfaces(scenario = "code-facts"): Promise<ListSurfacesResponse> {
  return factsClient.listSurfaces({ target: scenarioTarget(scenario), useCache: true });
}

export { FactFamily, TargetKind };
export type { CodeFactsReport, CodeTarget, ListSurfacesResponse };
