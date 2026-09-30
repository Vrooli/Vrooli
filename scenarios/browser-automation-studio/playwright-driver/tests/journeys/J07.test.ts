import { ExecutionStatus } from '@vrooli/proto-types/browser-automation-studio/v1/base/shared_pb';
import { ExecutionsService, WorkflowsService } from '@vrooli/proto-types/browser-automation-studio/v1/api/service_pb';
import { action, adhoc, env, fixtureAfter, fixtureState, rpc } from './support';

describe('[REQ:BAS-RH-J07] cancellation and effect ownership', () => {
  it('given a workflow holding one fixture effect, when it is cancelled, then it ends CANCELLED and the effect is released once', async () => {
    const before = await fixtureState();
    const { executionId } = await rpc(WorkflowsService.method.executeAdhocWorkflow,
      adhoc('J07 cancellation', action('navigate', { url: `${env.fixture}/effect/hold`, timeoutMs: 20000 }), false));
    let stopped = false;
    try {
      const held = await fixtureAfter('effect', before.effects.length);
      expect(held.effects.slice(before.effects.length)).toEqual([expect.objectContaining({ held: true })]);
      const running = await rpc(ExecutionsService.method.getExecution, { executionId });
      expect(running.execution?.status).toBe(ExecutionStatus.RUNNING);

      const stop = await rpc(ExecutionsService.method.stopExecution, { executionId });
      stopped = true;
      expect(stop.status).toBe('stopped');
      const released = await fixtureAfter('release', before.released.length);
      expect(released.released.slice(before.released.length)).toEqual([{ sequence: before.effects.length + 1 }]);

      const terminal = await rpc(ExecutionsService.method.getExecution, { executionId });
      expect(terminal.execution?.status).toBe(ExecutionStatus.CANCELLED);
      expect((await fixtureState()).effects).toHaveLength(before.effects.length + 1);
    } finally {
      if (!stopped) await rpc(ExecutionsService.method.stopExecution, { executionId }).catch(() => undefined);
    }
  });
});
