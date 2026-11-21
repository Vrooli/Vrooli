import { ExecutionStatus } from '@vrooli/proto-types/browser-automation-studio/v1/base/shared_pb';
import { ExecutionsService, WorkflowsService } from '@vrooli/proto-types/browser-automation-studio/v1/api/service_pb';
import { action, adhoc, env, fixtureAfter, fixtureState, rpc } from './support';

describe('[REQ:BAS-RH-J18] cancellation retains failure evidence', () => {
  it('given a workflow blocked on a fixture effect, when cancelled, then terminal status and replay evidence identify the cancelled execution', async () => {
    const before = await fixtureState();
    const { executionId } = await rpc(WorkflowsService.method.executeAdhocWorkflow,
      adhoc('J18 cancellation evidence', action('navigate', { url: `${env.fixture}/effect/hold`, timeoutMs: 20000 }), false));
    const held = await fixtureAfter('effect', before.effects.length);
    expect(held.effects.slice(before.effects.length)).toEqual([expect.objectContaining({ held: true })]);
    const stopped = await rpc(ExecutionsService.method.stopExecution, { executionId });
    expect(stopped.status).toBe('stopped');
    const terminal = await rpc(ExecutionsService.method.getExecution, { executionId });
    expect(terminal.execution?.status).toBe(ExecutionStatus.CANCELLED);
    const released = await fixtureAfter('release', before.released.length);
    expect(released.released.slice(before.released.length)).toHaveLength(1);
    const evidence = await rpc(ExecutionsService.method.getExecutionReplayPackage, { executionId });
    expect(evidence).toMatchObject({ executionId, schemaVersion: 'bas-replay/v1' });
    expect(evidence.timeline.length).toBeGreaterThan(0);
  });
});
