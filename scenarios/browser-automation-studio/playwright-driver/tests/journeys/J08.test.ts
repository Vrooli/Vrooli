import { ExecutionStatus } from '@vrooli/proto-types/browser-automation-studio/v1/base/shared_pb';
import { ExecutionsService, WorkflowsService } from '@vrooli/proto-types/browser-automation-studio/v1/api/service_pb';
import { action, adhoc, env, rpc } from './support';

describe('[REQ:BAS-RH-J08] replay evidence package', () => {
  it('given a completed fixture execution, when its replay package is read, then it is versioned and carries screenshot telemetry', async () => {
    const run = await rpc(WorkflowsService.method.executeAdhocWorkflow,
      adhoc('J08 fixture evidence', action('navigate', { url: `${env.fixture}/forms`, timeoutMs: 10000 }), true));
    expect(run.status).toBe(ExecutionStatus.COMPLETED);

    const pack = await rpc(ExecutionsService.method.getExecutionReplayPackage, { executionId: run.executionId });
    expect(pack).toMatchObject({ schemaVersion: 'bas-replay/v1', executionId: run.executionId, id: expect.any(String) });
    expect(pack.evidence).toMatchObject({ schemaVersion: 'bas-evidence/v1', executionId: run.executionId });
    expect(pack.timeline.some((entry) => entry.action !== undefined)).toBe(true);
    expect(pack.timeline.some(({ telemetry }) => (telemetry?.screenshot?.width ?? 0) > 0 && (telemetry?.screenshot?.height ?? 0) > 0)).toBe(true);
  });
});
