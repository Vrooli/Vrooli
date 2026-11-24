import { mkdtemp, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { randomUUID } from 'node:crypto';
import { chromium, expect } from '@playwright/test';
import { create, toJson } from '@bufbuild/protobuf';
import { ArtifactKind } from '@vrooli/proto-types/browser-automation-studio/v1/evidence/evidence_pb';
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
  measureJourney,
  recordJourneyRuntimeIdentity,
  recordJourneyReceipt,
  recordNotApplicableResponsivenessMetric,
  rpc,
  withJourneyResources,
} from './support';

jest.setTimeout(120000);

describe('[REQ:BAS-RH-J27] BAS UI execution history and artifacts', () => {
  it('opens a saved run from BAS history and displays its own screenshot artifact', () =>
    withJourneyResources(async (resources) => {
      const projectFolder = await mkdtemp(join(tmpdir(), 'bas-j27-'));
      resources.add('remove temporary project directory', () =>
        rm(projectFolder, { recursive: true, force: true })
      );
      const { project } = await rpc(ProjectsService.method.createProject, {
        name: `J27 ${randomUUID()}`,
        folderPath: projectFolder,
      });
      const projectId = project?.id ?? '';
      if (!projectId) throw new Error('BAS did not create the J27 project');
      resources.add('delete J27 project', () =>
        rpc(ProjectsService.method.deleteProject, { id: projectId, deleteFiles: true })
      );

      const workflowName = `J27 UI execution ${randomUUID()}`;
      const startNode = randomUUID();
      const effectNode = randomUUID();
      const browser = await chromium.launch({ headless: true });
      resources.add('close J27 UI browser', () => browser.close());
      await recordJourneyRuntimeIdentity('J27', browser.version());
      const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });
      const definitionJson = JSON.stringify(
        toJson(
          WorkflowDefinitionV2Schema,
          create(WorkflowDefinitionV2Schema, {
            metadata: { name: workflowName, executionMode: ExecutionMode.OBSERVER },
            settings: { headless: true, timeoutMs: 30000 },
            nodes: [
              {
                id: startNode,
                action: action('navigate', { url: `${env.fixture}/`, timeoutMs: 20000 }),
              },
              { id: effectNode, action: action('click', { selector: '#counter' }) },
            ],
            edges: [{ id: randomUUID(), source: startNode, target: effectNode }],
          })
        )
      );
      const workflowId = await createWorkflowThroughEditor(page, projectId, definitionJson);
      const { workflow } = await rpc(WorkflowsService.method.getWorkflow, { workflowId });
      if (!workflow?.id || workflow.id !== workflowId)
        throw new Error('BAS did not retain the J27 editor workflow');
      const editorWorkflowName = workflow.name || workflowName;
      resources.add('delete J27 workflow', () =>
        rpc(WorkflowsService.method.deleteWorkflow, { workflowId })
      );
      resources.add('delete J27 workflow artifacts', () =>
        rpc(ExecutionsService.method.runExecutionArtifactRetention, {
          workflowId,
          projectId,
          maxAgeDays: 0,
          keepLatest: 0,
          confirm: true,
        })
      );

      const execution = await rpc(WorkflowsService.method.executeWorkflow, {
        workflowId,
        workflowVersion: workflow?.version,
        waitForCompletion: true,
      });
      if (execution.status !== ExecutionStatus.COMPLETED || !execution.executionId) {
        throw new Error(
          `J27 workflow did not complete: status=${execution.status}, error=${execution.error}`
        );
      }
      const replay = await rpc(ExecutionsService.method.getExecutionReplayPackage, {
        executionId: execution.executionId,
      });
      expect(replay.evidence?.artifacts?.some(({ kind }) => kind === ArtifactKind.SCREENSHOT)).toBe(
        true
      );

      await measureJourney(
        'J27',
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
          await page
            .getByTestId('execution-status-completed')
            .waitFor({ state: 'visible', timeout: 30000 });
        },
        () => ({
          execution_id: execution.executionId,
          workflow_id: workflowId,
          status: 'COMPLETED',
        }),
        { budgetMetrics: ['setup_to_ready_ms'] }
      );
      const screenshot = page.getByTestId('execution-screenshot');
      const imageWidth = await measureJourney(
        'J27',
        'execution.artifact_image_ready',
        async () => {
          await page.getByTestId('execution-tab-artifacts').click();
          await screenshot.waitFor({ state: 'visible', timeout: 30000 });
          await expect
            .poll(() => screenshot.evaluate((image) => (image as HTMLImageElement).naturalWidth), {
              timeout: 30000,
            })
            .toBeGreaterThan(0);
          return screenshot.evaluate((image) => (image as HTMLImageElement).naturalWidth);
        },
        (width) => ({
          execution_id: execution.executionId,
          artifact_kind: 'SCREENSHOT',
          image_width: width,
        }),
        { budgetMetrics: ['click_to_paint_ms', 'tab_switch_to_first_frame_ms'] }
      );
      recordJourneyReceipt('J27', 'execution.viewer_artifact_receipt', {
        execution_id: execution.executionId,
        workflow_id: workflowId,
        status: 'COMPLETED',
        screenshot_artifact_present: true,
        image_width: imageWidth,
      });
      recordNotApplicableResponsivenessMetric(
        'J27',
        'execution.event_fanout_not_sampled',
        'event_fanout_ms',
        'This journey opens a completed execution and reads persisted artifacts; it does not trigger a live event fan-out.'
      );
      recordNotApplicableResponsivenessMetric(
        'J27',
        'execution.event_bytes_not_sampled',
        'event_bytes',
        'No live recording or execution event is produced while the completed artifact viewer is open.'
      );
    }));
});
