import { mkdtemp } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { randomUUID } from 'node:crypto';
import { ActionType } from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';
import { ExecutionStatus, StepStatus } from '@vrooli/proto-types/browser-automation-studio/v1/base/shared_pb';
import { ExecutionsService, WorkflowsService } from '@vrooli/proto-types/browser-automation-studio/v1/api/service_pb';
import { ProjectsService } from '@vrooli/proto-types/browser-automation-studio/v1/projects/project_pb';
import {
  CreateRecordingSessionResponseSchema, GenerateWorkflowResponseSchema, GetActionsResponseSchema,
  RecordingStatusResponseSchema, StartRecordingResponseSchema, StopRecordingResponseSchema,
} from '@vrooli/proto-types/browser-automation-studio/v1/recording/session_pb';
import type { TimelineEntry } from '@vrooli/proto-types/browser-automation-studio/v1/timeline/entry_pb';
import type { InputRequest } from '../../src/routes/record-mode/types';
import {
  VIEWPORT, apiRoute, env, fixtureAfter, fixtureInput, fixtureState, rest, rpc, runAction, target, withLeases,
} from './support';

const FINAL = 'final J02 value';
const live = (sessionId: string) => `/recordings/live/${sessionId}`;
const fieldIs = (value: string, context?: string) => fixtureInput((input) => input.value === value && (context === undefined || input.context === context));

/** Passive capture through the API owner: every edit waits on the fixture's own observation, never a sleep. */
async function editFixture(sessionId: string): Promise<void> {
  const input = (request: Omit<InputRequest, 'execution_id' | 'lease_id'>) => apiRoute(`${live(sessionId)}/input`, request);
  const click = (name: Parameters<typeof target>[0]) => input({ type: 'pointer', action: 'click', button: 'left', ...target(name) });
  const key = (name: string, modifiers?: string[]) => input({ type: 'keyboard', key: name, modifiers });

  await click('input');
  await input({ type: 'keyboard', text: 'typed first' });
  await key('a', ['Control']);
  await input({ type: 'keyboard', text: 'replacement' });
  await fieldIs('replacement');
  await key('Home');
  await key('ArrowRight', ['Shift']);
  await key('ArrowRight', ['Shift']);
  await key('Backspace');
  await fieldIs('placement');
  await click('paste');
  await fieldIs('pasted value');
  await click('compose');
  await fieldIs('東京');
  await click('input');
  await key('a', ['Control']);
  await key('Backspace');
  await fieldIs('');
  await input({ type: 'keyboard', text: FINAL });
  await fieldIs(FINAL);
  await click('counter');
}

const inputSnapshots = (entries: TimelineEntry[]) =>
  entries.flatMap(({ action }) => (action?.params.case === 'input' ? [action.params.value.value] : []));

describe('[REQ:BAS-RH-J02] passive capture, fresh replay and saved-workflow execution', () => {
  const cleanup: Array<() => Promise<unknown>> = [];
  afterEach(async () => {
    const failures: unknown[] = [];
    for (const step of cleanup.splice(0).reverse()) await step().catch((error: unknown) => failures.push(error));
    if (failures.length > 0) throw new AggregateError(failures, 'J02 did not remove everything it created');
  });

  it('given passive capture of edited inputs, when replayed fresh and as a saved workflow, then final values and single effects match', async () => {
    const before = (await fixtureState()).effects.length;
    const { sessionId } = await rest(CreateRecordingSessionResponseSchema, '/recordings/live/session', {
      viewport_width: VIEWPORT.width, viewport_height: VIEWPORT.height, initial_url: `${env.fixture}/`, restore_tabs: false,
    });
    cleanup.push(() => apiRoute(`/recordings/live/session/${sessionId}/close`, {}));
    await rest(StartRecordingResponseSchema, '/recordings/live/start', { session_id: sessionId });
    expect((await rest(RecordingStatusResponseSchema, `${live(sessionId)}/status`)).isRecording).toBe(true);

    // Given passive capture, when fixture inputs are edited, then the final snapshot is retained.
    await editFixture(sessionId);
    const recorded = await fixtureAfter('effect', before);
    expect(recorded.effects).toHaveLength(before + 1);
    expect((await rest(StopRecordingResponseSchema, `${live(sessionId)}/stop`, {})).recordingId).not.toBe('');
    const { entries, count } = await rest(GetActionsResponseSchema, `${live(sessionId)}/actions`);
    expect(count).toBe(entries.length);
    expect(new Set(entries.map(({ id }) => id)).size).toBe(entries.length);
    expect(entries.filter(({ action }) => action?.type === ActionType.CLICK && action.params.case === 'click' && action.params.value.selector === '#counter')).toHaveLength(1);
    expect(entries.some(({ action }) => action?.params.case === 'navigate' && action.params.value.url === `${env.fixture}/`)).toBe(true);
    expect(inputSnapshots(entries).at(-1)).toBe(FINAL);

    const folder = join(await mkdtemp(join(tmpdir(), 'bas-j02-')), 'project');
    const { project } = await rpc(ProjectsService.method.createProject, { name: `J02 ${randomUUID()}`, folderPath: folder });
    const projectId = project?.id ?? '';
    cleanup.push(() => rpc(ProjectsService.method.deleteProject, { id: projectId, deleteFiles: true }));
    const generated = await rest(GenerateWorkflowResponseSchema, `${live(sessionId)}/generate-workflow`, { name: 'J02 recorded fixture', project_id: projectId });
    expect(generated.nodeCount).toBeGreaterThan(0);
    cleanup.push(() => rpc(WorkflowsService.method.deleteWorkflow, { workflowId: generated.workflowId }));
    const { workflow } = await rpc(WorkflowsService.method.getWorkflow, { workflowId: generated.workflowId });
    const nodeIds = workflow?.flowDefinition?.nodes.map(({ id }) => id) ?? [];
    expect(nodeIds.length).toBeGreaterThan(0);

    // Committed entries are acknowledged exactly once by reading with clear=true.
    const acknowledged = await rest(GetActionsResponseSchema, `${live(sessionId)}/actions?clear=true`);
    expect(acknowledged.entries.map(({ id }) => id)).toEqual(entries.map(({ id }) => id));

    // When the captured history replays in a fresh driver context, then its final value and one effect recur there.
    await withLeases(async (open) => {
      const replay = await open();
      for (const { action } of entries) if (action) await runAction(replay, action);
    });
    const replayed = await fixtureAfter('effect', before + 1);
    const [recordEffect, replayEffect] = replayed.effects.slice(before);
    expect(replayed.effects).toHaveLength(before + 2);
    expect(replayEffect?.context).not.toBe(recordEffect?.context);
    await fieldIs(FINAL, replayEffect?.context);

    // When the saved workflow executes, then the API owner completes every node and reproduces the final value.
    const execution = await rpc(WorkflowsService.method.executeWorkflow, {
      workflowId: generated.workflowId, workflowVersion: workflow?.version, waitForCompletion: true,
    });
    cleanup.push(() => retainOnly(execution.executionId, generated.workflowId, projectId));
    expect(execution.status).toBe(ExecutionStatus.COMPLETED);
    const saved = await fixtureAfter('effect', before + 2);
    expect(saved.effects).toHaveLength(before + 3);
    await fieldIs(FINAL, saved.effects.at(-1)?.context);
    const timeline = await rpc(ExecutionsService.method.getExecutionTimeline, { executionId: execution.executionId });
    expect(timeline).toMatchObject({ executionId: execution.executionId, workflowId: generated.workflowId, status: ExecutionStatus.COMPLETED });
    expect(timeline.entries.map(({ nodeId }) => nodeId).sort()).toEqual([...nodeIds].sort());
    expect(timeline.entries.every(({ aggregates, context }) => aggregates?.status === StepStatus.COMPLETED && context?.success === true)).toBe(true);
  });
});

/** Remove this journey's execution artifacts, refusing if retention would touch anything else. */
async function retainOnly(executionId: string, workflowId: string, projectId: string): Promise<void> {
  const filter = { workflowId, projectId, maxAgeDays: 0, keepLatest: 0 };
  const preview = await rpc(ExecutionsService.method.previewExecutionArtifactRetention, filter);
  if (preview.removed.some((row) => row.executionId !== executionId)) throw new Error('Retention preview exceeded the journey execution');
  await rpc(ExecutionsService.method.runExecutionArtifactRetention, { ...filter, confirm: true });
}
