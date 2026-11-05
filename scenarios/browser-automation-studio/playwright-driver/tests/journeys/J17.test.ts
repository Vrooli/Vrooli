import { randomUUID } from 'node:crypto';
import { create } from '@bufbuild/protobuf';
import { ActionDefinitionSchema, LoopParamsSchema, LoopType } from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';
import { ExecutionStatus } from '@vrooli/proto-types/browser-automation-studio/v1/base/shared_pb';
import { ExecutionsService, WorkflowsService } from '@vrooli/proto-types/browser-automation-studio/v1/api/service_pb';
import { ExecutionMode, WorkflowDefinitionV2Schema } from '@vrooli/proto-types/browser-automation-studio/v1/workflows/definition_pb';
import { action, adhoc, env, fixtureAfter, fixtureState, rpc } from './support';

describe('[REQ:BAS-RH-J17] bounded repeated actions and recovery retry', () => {
  it('given a bounded repeat loop and a transient navigation failure, when executed, then loop effects and the retry reach the fixture', async () => {
    const before = await fixtureState();
    const startId = randomUUID();
    const loopId = randomUUID();
    const bodyId = randomUUID();
    const endId = randomUUID();
    const loop = create(ActionDefinitionSchema, {
      type: 27,
      params: { case: 'loop', value: create(LoopParamsSchema, { loopType: LoopType.REPEAT, count: 2, maxIterations: 2 }) },
    });
    const request = adhoc('J17 repeat loop', action('navigate', { url: `${env.fixture}/forms` }), true);
    request.flowDefinition = create(WorkflowDefinitionV2Schema, {
      metadata: { name: 'J17 repeat loop', executionMode: ExecutionMode.MUTATING },
      settings: { headless: true, timeoutMs: 20000 },
      nodes: [
        { id: startId, action: action('navigate', { url: `${env.fixture}/` }) },
        { id: loopId, action: loop },
        { id: bodyId, action: action('click', { selector: '#counter' }) },
        { id: endId, action: action('navigate', { url: `${env.fixture}/forms` }) },
      ],
      edges: [
        { id: randomUUID(), source: startId, target: loopId },
        { id: randomUUID(), source: loopId, target: bodyId, sourceHandle: 'loopbody' },
        { id: randomUUID(), source: bodyId, target: loopId, targetHandle: 'loopcontinue' },
        { id: randomUUID(), source: loopId, target: endId },
      ],
    });
    const repeated = await rpc(WorkflowsService.method.executeAdhocWorkflow, request);
    const repeatedRun = await rpc(ExecutionsService.method.getExecution, { executionId: repeated.executionId });
    if (repeated.status !== ExecutionStatus.COMPLETED) throw new Error(`Loop execution failed: ${JSON.stringify(repeatedRun, (_key, value: unknown) => typeof value === 'bigint' ? String(value) : value)}`);
    expect((await fixtureState()).effects).toHaveLength(before.effects.length + 2);

    const retryAttemptsBefore = (await fixtureState()).retryAttempts;
    const retryRequest = adhoc('J17 transient retry', action('navigate', { url: `${env.fixture}/retry-once` }), true, { maxAttempts: 2, delayMs: 1 });
    const retried = await rpc(WorkflowsService.method.executeAdhocWorkflow, retryRequest);
    expect(retried.status).toBe(ExecutionStatus.COMPLETED);
    expect((await fixtureState()).retryAttempts).toBe(retryAttemptsBefore + 2);
  });
});
