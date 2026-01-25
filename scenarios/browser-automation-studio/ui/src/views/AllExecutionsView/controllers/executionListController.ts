import { fetchProjectsList } from '@/domains/projects/services/projectApi';
import { fetchWorkflowList } from '@/domains/workflows/services/workflowApi';
import { fetchExecutionsList } from '@/domains/executions/services/executionApi';

export type ExecutionStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';

export interface GlobalExecutionItem {
  id: string;
  workflowId: string;
  workflowName: string;
  projectId?: string;
  projectName?: string;
  status: ExecutionStatus;
  startedAt: Date;
  completedAt?: Date;
  duration?: number;
  error?: string;
}

const toDate = (value: string | undefined): Date | null => {
  if (!value) return null;
  const parsed = new Date(value);
  return Number.isNaN(parsed.valueOf()) ? null : parsed;
};

const normalizeExecutionStatus = (rawStatus: unknown): ExecutionStatus => {
  const statusStr = String(rawStatus ?? 'pending').toUpperCase();
  if (statusStr.includes('RUNNING')) return 'running';
  if (statusStr.includes('COMPLETED')) return 'completed';
  if (statusStr.includes('FAILED')) return 'failed';
  if (statusStr.includes('CANCELLED')) return 'cancelled';
  if (statusStr.includes('PENDING')) return 'pending';
  return 'pending';
};

export const loadGlobalExecutions = async (limit = 200): Promise<GlobalExecutionItem[]> => {
  const [projects, workflows, executions] = await Promise.all([
    fetchProjectsList(),
    fetchWorkflowList(500),
    fetchExecutionsList(limit),
  ]);

  const projectsMap = new Map<string, string>();
  projects.forEach((project) => {
    projectsMap.set(project.id, project.name);
  });

  const workflowsMap = new Map<string, { name: string; projectId?: string; projectName?: string }>();
  workflows.forEach((workflow) => {
    const projectId = workflow.project_id ?? workflow.projectId;
    workflowsMap.set(workflow.id, {
      name: workflow.name ?? 'Untitled',
      projectId,
      projectName: projectId ? projectsMap.get(projectId) : undefined,
    });
  });

  const items = executions.map((execution) => {
    const workflowId = execution.workflow_id ?? execution.workflowId ?? '';
    const workflowInfo = workflowsMap.get(workflowId);
    const status = normalizeExecutionStatus(execution.status);
    const startedAt = toDate(execution.started_at ?? execution.startedAt) ?? new Date();
    const completedAt = toDate(execution.completed_at ?? execution.completedAt ?? undefined) ?? undefined;
    const duration = completedAt ? Math.max(0, completedAt.getTime() - startedAt.getTime()) : undefined;

    return {
      id: execution.id ?? execution.executionId ?? execution.execution_id ?? '',
      workflowId,
      workflowName: workflowInfo?.name ?? 'Unknown Workflow',
      projectId: workflowInfo?.projectId,
      projectName: workflowInfo?.projectName,
      status,
      startedAt,
      completedAt,
      duration,
      error: execution.error ?? undefined,
    };
  });

  return items.sort((a, b) => b.startedAt.getTime() - a.startedAt.getTime());
};
