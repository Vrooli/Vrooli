import { randomUUID } from 'node:crypto';
import { create } from '@bufbuild/protobuf';
import { ActionType } from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';
import { AssertionMode, ExecutionStatus, StepStatus, WorkflowEdgeType } from '@vrooli/proto-types/browser-automation-studio/v1/base/shared_pb';
import { ExecutionsService, WorkflowsService } from '@vrooli/proto-types/browser-automation-studio/v1/api/service_pb';
import { WorkflowDefinitionV2Schema, WorkflowEdgeV2Schema, WorkflowNodeV2Schema } from '@vrooli/proto-types/browser-automation-studio/v1/workflows/definition_pb';
import { action, adhoc, env, rpc } from './support';

describe('[REQ:BAS-RH-J24] typed outcomes and public assertion validation', () => {
  it('carries the public assertion requirement through validation and preserves structured outcomes in execution, timeline, and replay', async () => {
    const assertAction = action('assert', { selector: '#counter', mode: AssertionMode.VISIBLE });
    const request = adhoc('J24 typed assertion outcome', assertAction, true);
    const seedFlow = request.flowDefinition;
    if (!seedFlow) throw new Error('J24 requires the typed adhoc workflow definition');
    const navigateId = randomUUID();
    const assertId = randomUUID();
    const flow = create(WorkflowDefinitionV2Schema, {
      metadata: seedFlow.metadata,
      settings: seedFlow.settings,
      nodes: [
        create(WorkflowNodeV2Schema, { id: navigateId, action: action('navigate', { url: `${env.fixture}/` }) }),
        create(WorkflowNodeV2Schema, { id: assertId, action: assertAction }),
      ],
      edges: [create(WorkflowEdgeV2Schema, { source: navigateId, target: assertId, type: WorkflowEdgeType.DEFAULT })],
    });
    request.flowDefinition = flow;

    const noAssertion = create(WorkflowDefinitionV2Schema, {
      metadata: flow.metadata,
      settings: flow.settings,
      edges: flow.edges,
      nodes: [],
    });
    await expect(rpc(WorkflowsService.method.validateWorkflow, {
      workflow: noAssertion,
      requireAssertion: true,
    })).rejects.toThrow();

    const validation = await rpc(WorkflowsService.method.validateWorkflow, {
      workflow: flow,
      requireAssertion: true,
    });
    expect(validation.result?.valid).toBe(true);

    const assertNode = flow.nodes.find(({ action: nodeAction }) => nodeAction?.type === ActionType.ASSERT);
    expect(assertNode).toBeDefined();
    const execution = await rpc(WorkflowsService.method.executeAdhocWorkflow, request);
    expect(execution.status).toBe(ExecutionStatus.COMPLETED);

    const timeline = await rpc(ExecutionsService.method.getExecutionTimeline, { executionId: execution.executionId });
    expect(timeline.entries).toHaveLength(2);
    const timelineEntry = timeline.entries.find(({ nodeId }) => nodeId === assertNode?.id);
    expect(timelineEntry).toBeDefined();
    expect(timelineEntry.aggregates?.status).toBe(StepStatus.COMPLETED);
    expect(timelineEntry.context?.success).toBe(true);
    expect(timelineEntry.context?.assertion).toMatchObject({
      mode: AssertionMode.VISIBLE,
      selector: '#counter',
      success: true,
    });
    expect(timelineEntry.context?.assertion?.actual).toBeDefined();

    const replay = await rpc(ExecutionsService.method.getExecutionReplayPackage, { executionId: execution.executionId });
    expect(replay).toMatchObject({ schemaVersion: 'bas-replay/v1', executionId: execution.executionId });
    const replayEntry = replay.timeline.find(({ nodeId }) => nodeId === assertNode?.id);
    expect(replayEntry?.aggregates?.status).toBe(StepStatus.COMPLETED);
    expect(replayEntry?.context?.assertion).toMatchObject({
      mode: AssertionMode.VISIBLE,
      selector: '#counter',
      success: true,
    });
    expect(replayEntry?.context?.assertion?.actual).toBeDefined();
  });
});
