import { mkdtemp } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { randomUUID } from 'node:crypto';
import { create } from '@bufbuild/protobuf';
import { ExecutionStatus } from '@vrooli/proto-types/browser-automation-studio/v1/base/shared_pb';
import { ExecutionsService, WorkflowsService } from '@vrooli/proto-types/browser-automation-studio/v1/api/service_pb';
import { ProjectsService } from '@vrooli/proto-types/browser-automation-studio/v1/projects/project_pb';
import { ExecutionMode, WorkflowDefinitionV2Schema } from '@vrooli/proto-types/browser-automation-studio/v1/workflows/definition_pb';
import { action, env, fixtureAfter, fixtureState, rpc, withJourneyResources } from './support';

const workflowDefinition = (url: string) => {
  const navigateId = randomUUID();
  const effectId = randomUUID();
  const isHeld = url.endsWith('/effect/hold');
  return create(WorkflowDefinitionV2Schema, {
    metadata: { name: 'J16 retention fixture', executionMode: ExecutionMode.OBSERVER },
    settings: { headless: true, timeoutMs: 20000 },
    nodes: isHeld
      ? [{ id: navigateId, action: action('navigate', { url, timeoutMs: 20000 }) }]
      : [
          { id: navigateId, action: action('navigate', { url: `${env.fixture}/`, timeoutMs: 20000 }) },
          { id: effectId, action: action('click', { selector: '#counter' }) },
        ],
    edges: isHeld ? [] : [{ id: randomUUID(), source: navigateId, target: effectId }],
  });
};

describe('[REQ:BAS-RH-J16] active execution and per-workflow keep_latest retention', () => {
  it('given terminal evidence and a live execution, when bounded retention is previewed and applied, then active and newest evidence survive while older evidence is removed', () => withJourneyResources(async (resources) => {
    const projectFolder = await mkdtemp(join(tmpdir(), 'bas-j16-'));
    const { project } = await rpc(ProjectsService.method.createProject, { name: `J16 ${randomUUID()}`, folderPath: projectFolder });
    const projectId = project?.id ?? '';
    resources.add('delete project', async () => {
      await rpc(ProjectsService.method.deleteProject, { id: projectId, deleteFiles: true });
    });
    let workflowId = '';
    let activeExecutionId = '';
    const completedExecutionIds: string[] = [];
      const created = await rpc(WorkflowsService.method.createWorkflow, {
        projectId, name: 'J16 retention fixture', folderPath: 'workflows', flowDefinition: workflowDefinition(`${env.fixture}/effect`),
      });
      workflowId = created.workflow?.id ?? '';
      resources.add('delete workflow', async () => {
        if (workflowId) await rpc(WorkflowsService.method.deleteWorkflow, { workflowId });
      });
      resources.add('delete workflow execution artifacts', async () => {
        if (workflowId) await rpc(ExecutionsService.method.runExecutionArtifactRetention, {
          workflowId, projectId, maxAgeDays: 0, keepLatest: 0, confirm: true,
        });
      });
      expect(workflowId).not.toBe('');

      for (let index = 0; index < 2; index += 1) {
        const effectCount = (await fixtureState()).effects.length;
        const execution = await rpc(WorkflowsService.method.executeWorkflow, {
          workflowId, workflowVersion: created.workflow?.version, waitForCompletion: true,
        });
        completedExecutionIds.push(execution.executionId);
        if (execution.status !== ExecutionStatus.COMPLETED) {
          const detail = await rpc(ExecutionsService.method.getExecution, { executionId: execution.executionId }).catch((error: Error) => error.message);
          throw new Error(`J16 terminal execution status=${execution.status}, response=${execution.error ?? ''}, detail=${JSON.stringify(detail, (_key, value: unknown) => typeof value === 'bigint' ? String(value) : value)}`);
        }
        const effected = await fixtureAfter('effect', effectCount);
        expect(effected.effects.at(-1)?.context).toContain('fixture_context=');
      }

      const current = await rpc(WorkflowsService.method.getWorkflow, { workflowId });
      const heldDefinition = workflowDefinition(`${env.fixture}/effect/hold`);
      await rpc(WorkflowsService.method.updateWorkflow, {
        workflowId, expectedVersion: current.workflow?.version, flowDefinition: heldDefinition,
        changeDescription: 'Hold one execution open during retention',
      });
      const activeEffectCount = (await fixtureState()).effects.length;
      const active = await rpc(WorkflowsService.method.executeWorkflow, { workflowId, waitForCompletion: false });
      activeExecutionId = active.executionId;
      resources.add('stop active execution', async () => {
        if (activeExecutionId) await rpc(ExecutionsService.method.stopExecution, { executionId: activeExecutionId });
      });
      expect(activeExecutionId).not.toBe('');
      const held = await fixtureAfter('effect', activeEffectCount);
      expect(held.effects.at(-1)).toMatchObject({ held: true });

      const filter = { workflowId, projectId, maxAgeDays: 0, keepLatest: 1 };
      const preview = await rpc(ExecutionsService.method.previewExecutionArtifactRetention, filter);
      expect(preview.dryRun).toBe(true);
      expect(preview.removed.map(({ executionId }) => executionId)).toEqual([completedExecutionIds[0]]);
      expect(preview.skipped.map(({ executionId }) => executionId)).toContain(completedExecutionIds[1]);
      expect(preview.removed.map(({ executionId }) => executionId)).not.toContain(activeExecutionId);

      const applied = await rpc(ExecutionsService.method.runExecutionArtifactRetention, { ...filter, confirm: true });
      expect(applied.removedCount).toBe(1);
      expect(applied.removed.map(({ executionId }) => executionId)).toEqual([completedExecutionIds[0]]);
      await expect(rpc(ExecutionsService.method.getExecution, { executionId: completedExecutionIds[0] })).rejects.toThrow();
      const protectedExecution = await rpc(ExecutionsService.method.getExecution, { executionId: completedExecutionIds[1] });
      expect(protectedExecution.execution?.status).toBe(ExecutionStatus.COMPLETED);
      const stillActive = await rpc(ExecutionsService.method.getExecution, { executionId: activeExecutionId });
      expect(stillActive.execution?.status).toBe(ExecutionStatus.RUNNING);
  }));
});
