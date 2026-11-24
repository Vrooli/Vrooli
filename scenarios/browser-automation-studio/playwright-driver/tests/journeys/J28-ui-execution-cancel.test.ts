import { mkdtemp, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { randomUUID } from 'node:crypto';
import { chromium, expect } from '@playwright/test';
import { create, toJson } from '@bufbuild/protobuf';
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
  createWorkflowThroughEditor,
  env,
  fixtureAfter,
  fixtureState,
  measureJourney,
  elapsedSinceMonotonic,
  journeyResponsivenessBudget,
  recordJourneyMeasurement,
  recordJourneyRuntimeIdentity,
  recordJourneyReceipt,
  recordNotApplicableResponsivenessMetric,
  rpc,
  trackJourneySocketTraffic,
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
      const browser = await chromium.launch({ headless: true });
      resources.add('close J28 UI browser', () => browser.close());
      await recordJourneyRuntimeIdentity('J28', browser.version());
      const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });
      const socketTraffic = trackJourneySocketTraffic(page);
      const definitionJson = JSON.stringify(
        toJson(
          WorkflowDefinitionV2Schema,
          create(WorkflowDefinitionV2Schema, {
            metadata: { name: workflowName, executionMode: ExecutionMode.OBSERVER },
            settings: { headless: true, timeoutMs: 30000 },
            nodes: [
              {
                id: nodeId,
                action: action('navigate', { url: `${env.fixture}/effect/hold`, timeoutMs: 20000 }),
              },
            ],
          })
        )
      );
      const workflowId = await createWorkflowThroughEditor(page, projectId, definitionJson);
      const { workflow } = await rpc(WorkflowsService.method.getWorkflow, { workflowId });
      if (!workflow?.id || workflow.id !== workflowId)
        throw new Error('BAS did not retain the J28 editor workflow');
      const editorWorkflowName = workflow.name || workflowName;
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

      await measureJourney(
        'J28',
        'execution.viewer_ready',
        async () => {
          await page.goto(`${env.ui}/executions`, {
            waitUntil: 'domcontentloaded',
            timeout: 30000,
          });
          await page.getByTestId('app-ready').waitFor({ state: 'attached', timeout: 30000 });
          const row = page.getByText(editorWorkflowName, { exact: true });
          await row.waitFor({ state: 'visible', timeout: 30000 });
          await row.click();
          await page.waitForURL(
            (url) => url.pathname === `/projects/${projectId}/workflows/${workflowId}`,
            { timeout: 30000 }
          );
          await page.getByTestId('execution-viewer').waitFor({ state: 'visible', timeout: 30000 });
          await expect(page.getByTestId('execution-status')).toHaveText(/running/i, {
            timeout: 30000,
          });
          return true;
        },
        () => ({ execution_id: executionId, workflow_id: workflowId }),
        { budgetMetrics: ['setup_to_ready_ms'] }
      );

      const socketBeforeCancel = socketTraffic.snapshot();
      const cancelStarted = process.hrtime.bigint();
      await measureJourney(
        'J28',
        'execution.cancel_click_to_visible_terminal',
        async () => {
          await page.getByTestId('execution-stop-button').click();
          await page
            .getByTestId('execution-status-cancelled')
            .waitFor({ state: 'visible', timeout: 30000 });
          return true;
        },
        () => ({ execution_id: executionId, workflow_id: workflowId }),
        { budgetMetrics: ['click_to_paint_ms'] }
      );
      const cancelTraffic = socketTraffic.snapshot();
      const cancelFanoutFrames = cancelTraffic.received_frames - socketBeforeCancel.received_frames;
      const cancelFanoutBytes = cancelTraffic.received_bytes - socketBeforeCancel.received_bytes;
      expect(cancelFanoutFrames).toBeGreaterThan(0);
      expect(cancelFanoutBytes).toBeLessThanOrEqual(
        journeyResponsivenessBudget('J28', 'event_bytes')
      );
      if (cancelTraffic.last_received_at_ns === null)
        throw new Error('Cancellation event has no receive timestamp');
      const cancelFanoutMs = elapsedSinceMonotonic(
        cancelStarted,
        cancelTraffic.last_received_at_ns
      );
      recordJourneyMeasurement(
        'J28',
        'execution.cancel_event_fanout',
        cancelFanoutMs,
        'event_fanout_ms',
        {
          execution_id: executionId,
          received_frames: cancelFanoutFrames,
          received_bytes: cancelFanoutBytes,
        }
      );
      recordJourneyReceipt('J28', 'execution.cancel_event_bytes', {
        execution_id: executionId,
        received_frames: cancelFanoutFrames,
        received_bytes: cancelFanoutBytes,
        event_bytes_budget: journeyResponsivenessBudget('J28', 'event_bytes'),
        event_bytes_budget_met:
          cancelFanoutBytes <= journeyResponsivenessBudget('J28', 'event_bytes'),
      });

      const terminalEvidence = await measureJourney(
        'J28',
        'execution.cancel_to_terminal',
        async () => {
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
      recordNotApplicableResponsivenessMetric(
        'J28',
        'execution.tab_switch_not_sampled',
        'tab_switch_to_first_frame_ms',
        'The execution cancellation journey uses one execution viewer and has no page-tab switch.'
      );
    }));
});
