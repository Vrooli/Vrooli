/**
 * Connect-RPC adapters for the projects domain.
 *
 * All BAS project CRUD + project-scoped workflow operations are owned by the
 * ProjectsService Connect-RPC handler. This module is a thin adapter layer
 * that converts the proto messages into the snake_case shapes the legacy
 * UI components consume, so individual call sites can be migrated
 * incrementally without changing downstream component contracts.
 */

import { projectsClient } from '@/api/projects';
import { projectFilesClient } from '@/api/projectFiles';
import { logger } from '@/utils/logger';
import {
  mapProtoProject,
  mapProtoProjectWithStats,
  type ParsedProject,
} from '@/utils/projectProto';
import type { ProjectEntry, ProjectEntryKind, ProjectWorkflowItem } from '@/shared/api/schemas';
import { PresetKind } from '@vrooli/proto-types/browser-automation-studio/v1/projects/project_pb';
import { ProjectEntryKind as ProtoProjectEntryKind } from '@vrooli/proto-types/browser-automation-studio/v1/project_files/project_files_pb';

/**
 * Fetch the project list with stats.
 */
export const fetchProjectsList = async (): Promise<ParsedProject[]> => {
  const resp = await projectsClient.listProjects({});
  const parsed = (resp.projects ?? [])
    .map((entry) => mapProtoProjectWithStats(entry))
    .filter((p): p is ParsedProject => p !== null);
  if (parsed.length === 0) {
    logger.warn('Projects response contained no entries', {
      component: 'ProjectApi',
      action: 'fetchProjectsList',
    });
  }
  return parsed;
};

/**
 * Fetch a single project with stats.
 */
export const fetchProject = async (projectId: string): Promise<ParsedProject | null> => {
  const resp = await projectsClient.getProject({ id: projectId });
  return mapProtoProjectWithStats({
    $typeName: 'browser_automation_studio.v1.projects.ProjectWithStats',
    project: resp.project,
    stats: resp.stats,
  } as Parameters<typeof mapProtoProjectWithStats>[0]);
};

const PRESET_KIND_MAP: Record<string, PresetKind> = {
  empty: PresetKind.EMPTY,
  recommended: PresetKind.RECOMMENDED,
  custom: PresetKind.CUSTOM,
};

const resolvePreset = (preset: string | undefined): PresetKind => {
  if (!preset) return PresetKind.UNSPECIFIED;
  return PRESET_KIND_MAP[preset.toLowerCase()] ?? PresetKind.UNSPECIFIED;
};

export interface CreateProjectInput {
  name: string;
  description?: string;
  folder_path: string;
  preset?: string;
  preset_paths?: string[];
}

export const createProjectViaApi = async (input: CreateProjectInput): Promise<ParsedProject> => {
  const resp = await projectsClient.createProject({
    name: input.name,
    description: input.description ?? '',
    folderPath: input.folder_path,
    preset: resolvePreset(input.preset),
    presetPaths: input.preset_paths ?? [],
  });
  const parsed = mapProtoProjectWithStats({
    $typeName: 'browser_automation_studio.v1.projects.ProjectWithStats',
    project: resp.project,
    stats: resp.stats,
  } as Parameters<typeof mapProtoProjectWithStats>[0]);
  if (!parsed) throw new Error('Failed to parse create-project response');
  return parsed;
};

export interface UpdateProjectInput {
  name?: string;
  description?: string;
  folder_path?: string;
}

export const updateProjectViaApi = async (
  id: string,
  updates: UpdateProjectInput,
): Promise<ParsedProject> => {
  const resp = await projectsClient.updateProject({
    id,
    name: updates.name ?? '',
    description: updates.description ?? '',
    folderPath: updates.folder_path ?? '',
  });
  const parsed = mapProtoProject(resp.project);
  if (!parsed) throw new Error('Failed to parse update-project response');
  return parsed;
};

export const deleteProjectViaApi = async (id: string, deleteFiles = false): Promise<void> => {
  await projectsClient.deleteProject({ id, deleteFiles });
};

const toIso = (ts: { seconds?: bigint | number; nanos?: number } | undefined): string => {
  if (!ts) return '';
  const seconds = typeof ts.seconds === 'bigint' ? Number(ts.seconds) : Number(ts.seconds ?? 0);
  const nanos = Number(ts.nanos ?? 0);
  if (!seconds && !nanos) return '';
  return new Date(seconds * 1000 + Math.floor(nanos / 1_000_000)).toISOString();
};

/**
 * Fetch workflows for a specific project.
 */
export const fetchProjectWorkflows = async (projectId: string): Promise<ProjectWorkflowItem[]> => {
  const resp = await projectsClient.listProjectWorkflows({ projectId });
  return (resp.workflows ?? []).map((w) => ({
    id: w.id,
    name: w.name,
    description: w.description ?? undefined,
    project_id: w.projectId ?? undefined,
    folder_path: w.folderPath ?? undefined,
    version: Number(w.version ?? 0),
    created_at: toIso(w.createdAt),
    updated_at: toIso(w.updatedAt),
  })) as ProjectWorkflowItem[];
};

export interface BulkExecutionResult {
  message: string;
  executions: Array<{
    workflow_id: string;
    workflow_name: string;
    execution_id?: string;
    status: string;
    error?: string;
  }>;
}

export const executeAllProjectWorkflowsViaApi = async (
  projectId: string,
): Promise<BulkExecutionResult> => {
  const resp = await projectsClient.executeAllProjectWorkflows({ projectId });
  return {
    message: resp.message,
    executions: (resp.executions ?? []).map((e) => ({
      workflow_id: e.workflowId,
      workflow_name: e.workflowName,
      execution_id: e.executionId || undefined,
      status: e.status,
      error: e.error || undefined,
    })),
  };
};

export const bulkDeleteProjectWorkflowsViaApi = async (
  projectId: string,
  workflowIds: string[],
): Promise<{ deleted_count: number; deleted_ids: string[] }> => {
  const resp = await projectsClient.bulkDeleteProjectWorkflows({
    projectId,
    workflowIds,
  });
  return {
    deleted_count: Number(resp.deletedCount ?? 0),
    deleted_ids: resp.deletedIds ?? [],
  };
};

const PROTO_ENTRY_KIND_TO_UI: Record<ProtoProjectEntryKind, ProjectEntryKind | null> = {
  [ProtoProjectEntryKind.UNSPECIFIED]: null,
  [ProtoProjectEntryKind.FOLDER]: 'folder',
  [ProtoProjectEntryKind.WORKFLOW_FILE]: 'workflow_file',
  [ProtoProjectEntryKind.ASSET_FILE]: 'asset_file',
};

/**
 * Fetch project file tree entries via ProjectFilesService.
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
