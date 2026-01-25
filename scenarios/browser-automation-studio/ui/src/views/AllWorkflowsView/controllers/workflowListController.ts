import { fetchProjectsList } from '@/domains/projects/services/projectApi';
import { fetchWorkflowList } from '@/domains/workflows/services/workflowApi';

export interface GlobalWorkflowItem {
  id: string;
  name: string;
  projectId: string;
  projectName: string;
  folderPath: string;
  updatedAt: Date;
  executionCount?: number;
  lastExecution?: Date;
}

const toDate = (value: string | undefined): Date => {
  if (!value) return new Date();
  const parsed = new Date(value);
  return Number.isNaN(parsed.valueOf()) ? new Date() : parsed;
};

export const loadGlobalWorkflows = async (limit = 500): Promise<GlobalWorkflowItem[]> => {
  const [projects, workflows] = await Promise.all([
    fetchProjectsList(),
    fetchWorkflowList(limit),
  ]);

  const projectsMap = new Map<string, string>();
  projects.forEach((project) => {
    projectsMap.set(project.id, project.name);
  });

  return workflows.map((workflow) => {
    const projectId = workflow.project_id ?? workflow.projectId ?? '';
    const lastExecutionRaw = workflow.last_execution ?? workflow.lastExecution;
    const executionCountRaw = workflow.execution_count ?? workflow.executionCount;

    return {
      id: workflow.id,
      name: workflow.name ?? 'Untitled',
      projectId,
      projectName: projectsMap.get(projectId) ?? 'Unknown Project',
      folderPath: workflow.folder_path ?? workflow.folderPath ?? '/',
      updatedAt: toDate(workflow.updated_at ?? workflow.updatedAt),
      executionCount: typeof executionCountRaw === 'number' ? executionCountRaw : undefined,
      lastExecution: lastExecutionRaw ? toDate(lastExecutionRaw) : undefined,
    };
  });
};
