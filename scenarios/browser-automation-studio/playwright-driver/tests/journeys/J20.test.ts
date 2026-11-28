import { mkdtemp } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { randomUUID } from 'node:crypto';
import { create } from '@bufbuild/protobuf';
import { ExecutionStatus } from '@vrooli/proto-types/browser-automation-studio/v1/base/shared_pb';
import { ArtifactKind } from '@vrooli/proto-types/browser-automation-studio/v1/evidence/evidence_pb';
import { ExecutionsService, WorkflowsService } from '@vrooli/proto-types/browser-automation-studio/v1/api/service_pb';
import { ProjectsService } from '@vrooli/proto-types/browser-automation-studio/v1/projects/project_pb';
import { ExecutionMode, WorkflowDefinitionV2Schema } from '@vrooli/proto-types/browser-automation-studio/v1/workflows/definition_pb';
import { action, env, fixtureAfter, fixtureState, openLease, registerLeaseCleanup, releaseLease, rpc, runAction, withJourneyResources } from './support';

const sharedWorkflow = () => {
  const navigateId = randomUUID();
  const effectId = randomUUID();
  return create(WorkflowDefinitionV2Schema, {
      metadata: { name: 'J20 shared-context evidence fixture', executionMode: ExecutionMode.OBSERVER },
    settings: { headless: true, timeoutMs: 30000 },
    nodes: [
      { id: navigateId, action: action('navigate', { url: `${env.fixture}/`, timeoutMs: 20000 }) },
      { id: effectId, action: action('click', { selector: '#counter' }) },
    ],
    edges: [{ id: randomUUID(), source: navigateId, target: effectId }],
  });
};

describe('[REQ:BAS-RH-J20] shared context evidence ownership', () => {
  it('given compatible executions and changed evidence requirements, when runs finish, then each manifest and artifact remains execution-owned', () => withJourneyResources(async (resources) => {
    const projectFolder = await mkdtemp(join(tmpdir(), 'bas-j20-'));
    const { project } = await rpc(ProjectsService.method.createProject, { name: `J20 ${randomUUID()}`, folderPath: projectFolder });
    const projectId = project?.id ?? '';
    resources.add('delete project', async () => {
      await rpc(ProjectsService.method.deleteProject, { id: projectId, deleteFiles: true });
    });
    let workflowId = '';
      const created = await rpc(WorkflowsService.method.createWorkflow, {
        projectId, name: 'J20 shared-context evidence fixture', folderPath: 'workflows', flowDefinition: sharedWorkflow(),
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

      // Two released driver leases share one compatible browsing context and
      // carry different execution ownership tokens across the handoff.
      const labels = { journey: 'J20-shared-context' };
      const ownerA = await openLease({ labels });
      registerLeaseCleanup(resources, ownerA);
      await runAction(ownerA, action('navigate', { url: `${env.fixture}/` }));
      const firstEffectCount = (await fixtureState()).effects.length;
      await runAction(ownerA, action('click', { selector: '#counter' }));
      const firstEffect = await fixtureAfter('effect', firstEffectCount);
      expect(firstEffect.effects.at(-1)?.context).toContain('fixture_context=');
      await releaseLease(ownerA);
      const ownerB = await openLease({ labels, reuseMode: 'reuse' });
      registerLeaseCleanup(resources, ownerB);
      expect(ownerB.sessionId).toBe(ownerA.sessionId);
      expect(ownerB.executionId).not.toBe(ownerA.executionId);
      const secondEffectCount = (await fixtureState()).effects.length;
      await runAction(ownerB, action('click', { selector: '#counter' }));
      const secondEffect = await fixtureAfter('effect', secondEffectCount);
      expect(secondEffect.effects.at(-1)?.context).toBe(firstEffect.effects.at(-1)?.context);

      const execute = async (options: { requiresTrace?: boolean } = {}) => {
        const effectCount = (await fixtureState()).effects.length;
        const result = await rpc(WorkflowsService.method.executeWorkflow, {
          workflowId, workflowVersion: created.workflow?.version, waitForCompletion: true,
          options: { requiresTrace: options.requiresTrace },
        });
        if (result.status !== ExecutionStatus.COMPLETED) {
          const detail = await rpc(ExecutionsService.method.getExecution, { executionId: result.executionId }).catch((error: Error) => error.message);
          throw new Error(`J20 terminal execution status=${result.status}, response=${result.error ?? ''}, detail=${JSON.stringify(detail, (_key, value: unknown) => typeof value === 'bigint' ? String(value) : value)}`);
        }
        const effects = await fixtureAfter('effect', effectCount);
        return { result, context: effects.effects.at(-1)?.context };
      };

      // Adding capture requirements must produce separate, complete manifests.
      const traced = await execute({ requiresTrace: true });
      const ordinary = await execute();
      const [tracePackage, ordinaryPackage] = await Promise.all([
        rpc(ExecutionsService.method.getExecutionReplayPackage, { executionId: traced.result.executionId }),
        rpc(ExecutionsService.method.getExecutionReplayPackage, { executionId: ordinary.result.executionId }),
      ]);
      expect(tracePackage.evidence?.id).not.toBe(ordinaryPackage.evidence?.id);
      expect(tracePackage.executionId).toBe(traced.result.executionId);
      expect(ordinaryPackage.executionId).toBe(ordinary.result.executionId);
      const traceArtifacts = tracePackage.evidence?.artifacts ?? [];
      const ordinaryArtifacts = ordinaryPackage.evidence?.artifacts ?? [];
      expect(traceArtifacts.some(({ kind }) => kind === ArtifactKind.TRACE)).toBe(true);
      expect(ordinaryArtifacts.some(({ kind }) => kind === ArtifactKind.SCREENSHOT)).toBe(true);
      for (const artifact of traceArtifacts) expect(artifact.executionId).toBe(traced.result.executionId);
      for (const artifact of ordinaryArtifacts) expect(artifact.executionId).toBe(ordinary.result.executionId);
      expect(traceArtifacts.map(({ id }) => id)).not.toEqual(expect.arrayContaining(ordinaryArtifacts.map(({ id }) => id)));
  }));
});
