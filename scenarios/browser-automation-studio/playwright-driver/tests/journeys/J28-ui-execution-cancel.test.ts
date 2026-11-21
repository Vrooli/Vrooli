import { mkdtemp, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { randomUUID } from 'node:crypto';
import { chromium, expect } from '@playwright/test';
import { create } from '@bufbuild/protobuf';
import { ExecutionStatus } from '@vrooli/proto-types/browser-automation-studio/v1/base/shared_pb';
import {
  ExecutionsService,
  WorkflowsService,
} from '@vrooli/proto-types/browser-automation-studio/v1/api/service_pb';
import { ProjectsService } from '@vrooli/proto-types/browser-automation-studio/v1/projects/project_pb';
import {
  ExecutionMode,
  WorkflowDefinitionV2Schema,
} from '@vrooli/proto-types/browser-automation-studio/v1/workflows/definition_pb';
import {
  action,
  env,
  fixtureAfter,
  fixtureState,
  measureJourney,
  recordJourneyReceipt,
  rpc,
  withJourneyResources,
} from './support';

jest.setTimeout(120000);

describe('[REQ:BAS-RH-J28] BAS UI execution cancellation', () => {
  it('cancels a held execution from its BAS viewer and waits for browser and server cleanup', () =>
    withJourneyResources(async (resources) => {
      const projectFolder = await mkdtemp(join(tmpdir(), 'bas-j28-'));
      resources.add('remove temporary project directory', () =>
        rm(projectFolder, { recursive: true, force: true })
      );
      const { project } = await rpc(ProjectsService.method.createProject, {
        name: `J28 ${randomUUID()}`,
        folderPath: projectFolder,
      });
      const projectId = project?.id ?? '';
      if (!projectId) throw new Error('BAS did not create the J28 project');
      resources.add('delete J28 project', () =>
        rpc(ProjectsService.method.deleteProject, { id: projectId, deleteFiles: true })
      );

      const workflowName = `J28 UI cancellation ${randomUUID()}`;
      const nodeId = randomUUID();
      const { workflow } = await rpc(WorkflowsService.method.createWorkflow, {
        projectId,
        name: workflowName,
        folderPath: 'workflows',
        flowDefinition: create(WorkflowDefinitionV2Schema, {
          metadata: { name: workflowName, executionMode: ExecutionMode.OBSERVER },
          settings: { headless: true, timeoutMs: 30000 },
          nodes: [
            {
              id: nodeId,
              action: action('navigate', { url: `${env.fixture}/effect/hold`, timeoutMs: 20000 }),
            },
          ],
        }),
      });
      const workflowId = workflow?.id ?? '';
      if (!workflowId) throw new Error('BAS did not create the J28 workflow');
      resources.add('delete J28 workflow', () =>
        rpc(WorkflowsService.method.deleteWorkflow, { workflowId })
      );
      resources.add('delete J28 workflow artifacts', () =>
        rpc(ExecutionsService.method.runExecutionArtifactRetention, {
          workflowId,
          projectId,
          maxAgeDays: 0,
          keepLatest: 0,
          confirm: true,
        })
      );

      const before = await fixtureState();
      const started = await rpc(WorkflowsService.method.executeWorkflow, {
        workflowId,
        workflowVersion: workflow?.version,
        waitForCompletion: false,
      });
      const executionId = started.executionId;
      if (!executionId) throw new Error('BAS did not return the J28 execution ID');
      let stopSettled = false;
      resources.add('stop J28 execution if still active', async () => {
        if (stopSettled) return;
        const current = await rpc(ExecutionsService.method.getExecution, { executionId });
        if (
          current.execution?.status === ExecutionStatus.PENDING ||
          current.execution?.status === ExecutionStatus.RUNNING
        ) {
          await rpc(ExecutionsService.method.stopExecution, { executionId });
        }
      });

      const held = await fixtureAfter('effect', before.effects.length);
      expect(held.effects.slice(before.effects.length)).toEqual([
        expect.objectContaining({ held: true }),
      ]);

      const browser = await chromium.launch({ headless: true });
      resources.add('close J28 UI browser', () => browser.close());
      const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });
      await page.goto(`${env.ui}/executions`, { waitUntil: 'domcontentloaded', timeout: 30000 });
      await page.getByTestId('app-ready').waitFor({ state: 'attached', timeout: 30000 });
      const row = page.getByText(workflowName, { exact: true });
      await row.waitFor({ state: 'visible', timeout: 30000 });
      await row.click();
      await page.waitForURL(
        (url) => url.pathname === `/projects/${projectId}/workflows/${workflowId}`,
        { timeout: 30000 }
      );
      await page.getByTestId('execution-viewer').waitFor({ state: 'visible', timeout: 30000 });
      const runningStatus = page.getByTestId('execution-status');
      await expect(runningStatus).toHaveText(/running/i, { timeout: 30000 });

      const terminalEvidence = await measureJourney(
        'J28',
        'execution.cancel_to_terminal',
        async () => {
          await page.getByTestId('execution-stop-button').click();
          await page
            .getByTestId('execution-status-cancelled')
            .waitFor({ state: 'visible', timeout: 30000 });
          const released = await fixtureAfter('release', before.released.length);
          expect(released.released.slice(before.released.length)).toHaveLength(1);
          const terminal = await rpc(ExecutionsService.method.getExecution, { executionId });
          expect(terminal.execution?.status).toBe(ExecutionStatus.CANCELLED);
          const replay = await rpc(ExecutionsService.method.getExecutionReplayPackage, {
            executionId,
          });
          expect(replay.timeline.length).toBeGreaterThan(0);
          return {
            terminal,
            releaseCount: released.released.slice(before.released.length).length,
            timelineEntries: replay.timeline.length,
          };
        },
        ({ terminal, releaseCount, timelineEntries }) => ({
          execution_id: executionId,
          workflow_id: workflowId,
          status: terminal.execution?.status ?? 'unknown',
          release_count: releaseCount,
          timeline_entries: timelineEntries,
        })
      );
      stopSettled = true;
      recordJourneyReceipt('J28', 'execution.cancelled_and_released', {
        execution_id: executionId,
        workflow_id: workflowId,
        status: terminalEvidence.terminal.execution?.status ?? 'unknown',
        release_count: terminalEvidence.releaseCount,
        timeline_entries: terminalEvidence.timelineEntries,
      });
    }));
});
