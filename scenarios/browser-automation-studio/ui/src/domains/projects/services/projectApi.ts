import { getConfig } from '@/config';
import { logger } from '@/utils/logger';
import { parseProjectList, type ParsedProject } from '@/utils/projectProto';
import { safeParse } from '@/shared/api/safeParse';
import {
  ProjectWorkflowsResponseSchema,
  type ProjectEntry,
  type ProjectEntryKind,
  type ProjectWorkflowItem,
} from '@/shared/api/schemas';
import { projectFilesClient } from '@/api/projectFiles';
import { ProjectEntryKind as ProtoProjectEntryKind } from '@vrooli/proto-types/browser-automation-studio/v1/project_files/project_files_pb';

/**
 * Fetch and parse the project list using proto-aware parsing.
 * This acts as a service-layer boundary for UI consumers.
 */
export const fetchProjectsList = async (): Promise<ParsedProject[]> => {
  const { API_URL } = await getConfig();
  const response = await fetch(`${API_URL}/projects`);

  if (!response.ok) {
    const message = await response.text().catch(() => '');
    throw new Error(message || `Failed to fetch projects (${response.status})`);
  }

  const payload: unknown = await response.json();
  const projects = parseProjectList(payload);
  if (projects.length === 0) {
    logger.warn('Projects response contained no entries', {
      component: 'ProjectApi',
      action: 'fetchProjectsList',
    });
  }
  return projects;
};

/**
 * Fetch workflows for a specific project with runtime validation.
 */
export const fetchProjectWorkflows = async (projectId: string): Promise<ProjectWorkflowItem[]> => {
  const { API_URL } = await getConfig();
  const response = await fetch(`${API_URL}/projects/${projectId}/workflows`);

  if (!response.ok) {
    const message = await response.text().catch(() => '');
    throw new Error(message || `Failed to fetch workflows (${response.status})`);
  }

  const payload: unknown = await response.json();
  const payloadRecord =
    payload && typeof payload === 'object' ? (payload as Record<string, unknown>) : {};
  const normalized = {
    workflows: Array.isArray(payloadRecord.workflows) ? payloadRecord.workflows : [],
  };
  const result = safeParse(ProjectWorkflowsResponseSchema, normalized, 'ProjectWorkflows');
  if (!result.success) {
    throw new Error(result.error);
  }
  return result.data.workflows;
};

const PROTO_ENTRY_KIND_TO_UI: Record<ProtoProjectEntryKind, ProjectEntryKind | null> = {
  [ProtoProjectEntryKind.UNSPECIFIED]: null,
  [ProtoProjectEntryKind.FOLDER]: 'folder',
  [ProtoProjectEntryKind.WORKFLOW_FILE]: 'workflow_file',
  [ProtoProjectEntryKind.ASSET_FILE]: 'asset_file',
};

/**
 * Fetch project file tree entries via the Connect-RPC ProjectFilesService.
 * Adapts proto entries to the consumer-facing snake_case shape so existing
 * components do not need changes.
 */
export const fetchProjectEntries = async (projectId: string): Promise<ProjectEntry[]> => {
  const resp = await projectFilesClient.getProjectFileTree({ projectId });
  const out: ProjectEntry[] = [];
  for (const entry of resp.entries ?? []) {
    const kind = PROTO_ENTRY_KIND_TO_UI[entry.kind];
    if (!kind) continue;
    const metadata = entry.metadata as Record<string, unknown> | undefined;
    out.push({
      id: entry.id,
      project_id: entry.projectId,
      path: entry.path,
      kind,
      ...(entry.workflowId ? { workflow_id: entry.workflowId } : {}),
      ...(metadata && Object.keys(metadata).length > 0 ? { metadata } : {}),
    });
  }
  return out;
};
