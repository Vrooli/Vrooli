import { fromJson, type JsonReadOptions, type JsonValue } from '@bufbuild/protobuf';
import {
  ProjectListSchema,
  ProjectSchema,
  ProjectWithStatsSchema,
  type Project as ProtoProject,
  type ProjectWithStats as ProtoProjectWithStats,
} from '@vrooli/proto-types/browser-automation-studio/v1/projects/project_pb';

const protoOptions: Partial<JsonReadOptions> = { ignoreUnknownFields: false };

export type ParsedProject = {
  id: string;
  name: string;
  description?: string;
  folder_path: string;
  created_at: string;
  updated_at: string;
  stats?: {
    workflow_count: number;
    execution_count: number;
    last_execution?: string;
  };
};

/** Type guard for timestamp-like objects */
const isTimestampLike = (value: unknown): value is { seconds?: unknown; nanos?: unknown } =>
  typeof value === 'object' && value !== null && ('seconds' in value || 'nanos' in value);

const toIsoString = (value: unknown): string | undefined => {
  if (!value) return undefined;
  if (typeof value === 'string') {
    const parsed = new Date(value);
    return Number.isNaN(parsed.valueOf()) ? undefined : parsed.toISOString();
  }
  if (!isTimestampLike(value)) return undefined;
  const secondsRaw = value.seconds ?? 0;
  const nanosRaw = value.nanos ?? 0;
  const seconds = typeof secondsRaw === 'bigint' ? Number(secondsRaw) : Number(secondsRaw);
  const nanos = Number(nanosRaw);
  const millis = seconds * 1_000 + Math.floor(nanos / 1_000_000);
  const date = new Date(millis);
  return Number.isNaN(date.valueOf()) ? undefined : date.toISOString();
};

const mapProject = (proto: ProtoProject | undefined): ParsedProject | null => {
  if (!proto) return null;
  const createdAt = toIsoString(proto.createdAt) ?? '';
  const updatedAt = toIsoString(proto.updatedAt) ?? createdAt;
  return {
    id: proto.id,
    name: proto.name,
    description: proto.description ?? '',
    folder_path: proto.folderPath,
    created_at: createdAt,
    updated_at: updatedAt,
  };
};

const mapProjectWithStats = (proto: ProtoProjectWithStats | undefined): ParsedProject | null => {
  if (!proto) return null;
  const project = mapProject(proto.project);
  if (!project) return null;
  const stats = proto.stats
    ? {
        workflow_count: Number(proto.stats.workflowCount ?? 0),
        execution_count: Number(proto.stats.executionCount ?? 0),
        last_execution: toIsoString(proto.stats.lastExecution),
      }
    : undefined;
  return stats ? { ...project, stats } : project;
};

export const parseProjectList = (raw: unknown): ParsedProject[] => {
  try {
    const proto = fromJson(ProjectListSchema, raw as JsonValue, protoOptions);
    return (proto.projects ?? [])
      .map((entry) => mapProjectWithStats(entry))
      .filter((p): p is ParsedProject => p !== null);
  } catch {
    // Fallback for legacy responses shaped as ProjectWithStats entries
    const projects = (raw as { projects?: unknown[] } | null | undefined)?.projects;
    if (!Array.isArray(projects)) return [];
    return projects
      .map((entry) => {
        try {
          const proto = fromJson(ProjectWithStatsSchema, entry as JsonValue, protoOptions);
          return mapProjectWithStats(proto);
        } catch {
          return fallbackProjectWithStats(entry);
        }
      })
      .filter((p): p is ParsedProject => p !== null);
  }
};

export const parseProjectWithStats = (raw: unknown): ParsedProject | null => {
  try {
    const proto = fromJson(ProjectWithStatsSchema, raw as JsonValue, protoOptions);
    return mapProjectWithStats(proto);
  } catch {
    return fallbackProjectWithStats(raw);
  }
};

export const parseProject = (raw: unknown): ParsedProject | null => {
  try {
    const proto = fromJson(ProjectSchema, raw as JsonValue, protoOptions);
    return mapProject(proto);
  } catch {
    return fallbackProject(raw);
  }
};

const fallbackProject = (raw: unknown): ParsedProject | null => {
  if (!raw || typeof raw !== 'object') return null;
  const obj = raw as Record<string, unknown>;
  const createdAt = toIsoString(obj.created_at ?? obj.createdAt) ?? '';
  const updatedAt = toIsoString(obj.updated_at ?? obj.updatedAt) ?? createdAt;
  const id = typeof obj.id === 'string' ? obj.id : '';
  const name = typeof obj.name === 'string' ? obj.name : '';
  const folder = typeof obj.folder_path === 'string' ? obj.folder_path : typeof obj.folderPath === 'string' ? obj.folderPath : '';
  if (!id || !name || !folder) return null;
  return {
    id,
    name,
    description: typeof obj.description === 'string' ? obj.description : '',
    folder_path: folder,
    created_at: createdAt,
    updated_at: updatedAt || createdAt,
  };
};

const fallbackProjectWithStats = (raw: unknown): ParsedProject | null => {
  if (!raw || typeof raw !== 'object') return null;
  const obj = raw as Record<string, unknown>;
  const base = fallbackProject(obj.project ?? obj);
  if (!base) return null;
  const statsObj = obj.stats as Record<string, unknown> | undefined;
  const stats = statsObj
    ? {
        workflow_count: Number(statsObj.workflow_count ?? statsObj.workflowCount ?? 0),
        execution_count: Number(statsObj.execution_count ?? statsObj.executionCount ?? 0),
        last_execution: toIsoString(statsObj.last_execution ?? statsObj.lastExecution),
      }
    : undefined;
  return stats ? { ...base, stats } : base;
};
