// API client for the scenario isolation endpoint, which proxies test-genie's
// routed-test-db eligibility decision.

import { create } from "@bufbuild/protobuf";
import { GetScenarioIsolationRequestSchema } from "@vrooli/proto-types/git-control-tower/v1/scenarios/scenarios_pb";
import { scenarioClient } from "./connect";

export type IsolationStatus = "routed" | "not_routed" | "unknown";

export interface IsolationViolation {
  rule_id: string;
  severity: string;
  file?: string;
  line?: number;
  excerpt?: string;
}

export interface IsolationResponse {
  status: IsolationStatus;
  reasons?: string[];
  violations?: IsolationViolation[];
}

export async function fetchScenarioIsolation(slug: string): Promise<IsolationResponse> {
  const response = await scenarioClient.getScenarioIsolation(create(GetScenarioIsolationRequestSchema, { slug }));
  return {
    status: response.status as IsolationStatus,
    reasons: response.reasons,
    violations: response.violations.map((v) => ({ rule_id: v.ruleId, severity: v.severity, file: v.file, line: v.line, excerpt: v.excerpt })),
  };
}
