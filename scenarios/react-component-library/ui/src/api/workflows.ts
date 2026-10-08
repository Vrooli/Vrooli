import { createClient } from "@connectrpc/connect";
import {
  WorkflowsService,
  WorkflowKind,
  type PromotionReadiness,
  type Workflow,
} from "@vrooli/proto-types/react-component-library/v1/workflows/workflows_pb";

import { transport } from "./client";

const client = createClient(WorkflowsService, transport);

// All workflow reads and writes use the generated Connect client. A workflow
// is only assisted execution state; callers must query promotion readiness to
// establish canonicalization evidence.
export const workflowsClient = {
  startWorkflow: (input: {
    kind: WorkflowKind;
    sourceScenario?: string;
    targetScenario?: string;
    sourcePath?: string;
    assetId?: string;
    idempotencyKey: string;
  }) => client.startWorkflow(input),
  listWorkflows: (input: {
    activeOnly?: boolean;
    limit?: number;
    assetId?: string;
    targetScenario?: string;
  }) => client.listWorkflows(input),
  stopWorkflow: (input: { id: string }) => client.stopWorkflow(input),
  retryWorkflow: (input: { id: string; idempotencyKey: string }) => client.retryWorkflow(input),
  getPromotionReadiness: (input: { assetId: string; originScenario: string; version?: string }) =>
    client.getPromotionReadiness(input),
};

export { WorkflowKind };
export type { PromotionReadiness, Workflow };
