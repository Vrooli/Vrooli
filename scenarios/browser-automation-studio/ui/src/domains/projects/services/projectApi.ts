import { getConfig } from '@/config';
import { logger } from '@/utils/logger';
import { parseProjectList, type ParsedProject } from '@/utils/projectProto';
import { safeParse } from '@/shared/api/safeParse';
import {
  ProjectEntriesResponseSchema,
  ProjectWorkflowsResponseSchema,
  type ProjectEntry,
  type ProjectWorkflowItem,
} from '@/shared/api/schemas';

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

/**
 * Fetch project file tree entries with runtime validation.
 */
export const fetchProjectEntries = async (projectId: string): Promise<ProjectEntry[]> => {
  const { API_URL } = await getConfig();
  const response = await fetch(`${API_URL}/projects/${projectId}/files/tree`);

  if (!response.ok) {
    const message = await response.text().catch(() => '');
    throw new Error(message || `Failed to fetch project files (${response.status})`);
  }

  const payload: unknown = await response.json();
  const payloadRecord =
    payload && typeof payload === 'object' ? (payload as Record<string, unknown>) : {};
  const normalized = {
    entries: Array.isArray(payloadRecord.entries) ? payloadRecord.entries : [],
  };
  const result = safeParse(ProjectEntriesResponseSchema, normalized, 'ProjectEntries');
  if (!result.success) {
    throw new Error(result.error);
  }
  return result.data.entries;
};
