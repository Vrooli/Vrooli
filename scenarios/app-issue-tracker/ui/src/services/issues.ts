import type { IssueStatus } from '../data/sampleData';
import type { CreateIssueInput, UpdateIssueInput, CreateIssueAttachmentPayload } from '../types/issueCreation';
import { apiJsonRequest, apiVoidRequest } from '../utils/api';
import { buildApiUrl, type ApiIssue, type ApiStatsPayload, formatStatusLabel } from '../utils/issues';

export interface RateLimitStatusPayload {
  rate_limited: boolean;
  rate_limited_count: number;
  reset_time: string;
  seconds_until_reset: number;
  rate_limit_agent: string;
}

export interface ProcessorEndpointPayload {
  processor: Record<string, unknown> | null;
  issuesProcessed: number | null;
  issuesRemaining: number | string | null;
  raw: unknown;
}

export interface AgentSettingsPayload {
  data: Record<string, unknown> | null;
  raw: unknown;
}

export interface IssueStatusMetadata {
  id: IssueStatus;
  label: string;
}

export async function fetchIssueStatuses(baseUrl: string): Promise<IssueStatusMetadata[]> {
  return apiJsonRequest<IssueStatusMetadata[]>(buildApiUrl(baseUrl, '/metadata/statuses'), {
    selector: (payload) => {
      const statuses = (payload as { data?: { statuses?: Array<Record<string, unknown>> } } | null | undefined)
        ?.data?.statuses;
      if (!Array.isArray(statuses)) {
        return [];
      }

      return statuses
        .map((entry) => {
          const idRaw = typeof entry.id === 'string' ? entry.id.trim().toLowerCase() : '';
          if (!idRaw) {
            return null;
          }
          const labelRaw = typeof entry.label === 'string' ? entry.label.trim() : '';
          return {
            id: idRaw as IssueStatus,
            label: labelRaw || formatStatusLabel(idRaw),
          } satisfies IssueStatusMetadata;
        })
        .filter((entry): entry is IssueStatusMetadata => Boolean(entry));
    },
    fallback: [],
  });
}

export async function listIssues(baseUrl: string, limit: number): Promise<ApiIssue[]> {
  return apiJsonRequest<ApiIssue[]>(buildApiUrl(baseUrl, `/issues?limit=${limit}`), {
    selector: (payload) => {
      const issuesPayload = (payload as { data?: { issues?: ApiIssue[] } } | null | undefined)?.data?.issues;
      return Array.isArray(issuesPayload) ? issuesPayload : [];
    },
  });
}

export async function fetchIssueStats(baseUrl: string): Promise<ApiStatsPayload | null> {
  return apiJsonRequest<ApiStatsPayload | null>(buildApiUrl(baseUrl, '/stats'), {
    selector: (payload) => {
      const raw = (payload as { data?: { stats?: Record<string, unknown> } } | null | undefined)?.data?.stats;
      if (!raw) {
        return null;
      }
      return {
        totalIssues: typeof raw.total_issues === 'number' ? raw.total_issues : undefined,
        openIssues: typeof raw.open_issues === 'number' ? raw.open_issues : undefined,
        inProgress: typeof raw.in_progress === 'number' ? raw.in_progress : undefined,
        completedToday: typeof raw.completed_today === 'number' ? raw.completed_today : undefined,
      } satisfies ApiStatsPayload;
    },
  });
}

export async function fetchProcessorState(baseUrl: string): Promise<ProcessorEndpointPayload> {
  const response = await apiJsonRequest<Record<string, unknown>>(buildApiUrl(baseUrl, '/automation/processor'));
  const data = (response?.data as Record<string, unknown> | undefined) ?? undefined;
  const state = (data?.processor ?? response?.processor) as Record<string, unknown> | undefined;

  return {
    processor: state ?? null,
    issuesProcessed: typeof data?.issues_processed === 'number' ? (data.issues_processed as number) : null,
    issuesRemaining:
      data?.issues_remaining !== undefined ? (data.issues_remaining as number | string) : null,
    raw: response,
  };
}

export async function patchProcessorState(
  baseUrl: string,
  body: Record<string, unknown>,
): Promise<ProcessorEndpointPayload> {
  const response = await apiJsonRequest<Record<string, unknown>>(buildApiUrl(baseUrl, '/automation/processor'), {
    method: 'PATCH',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(body),
  });
  const data = (response?.data as Record<string, unknown> | undefined) ?? undefined;
  const state = (data?.processor ?? response?.processor) as Record<string, unknown> | undefined;

  return {
    processor: state ?? null,
    issuesProcessed: typeof data?.issues_processed === 'number' ? (data.issues_processed as number) : null,
    issuesRemaining:
      data?.issues_remaining !== undefined ? (data.issues_remaining as number | string) : null,
    raw: response,
  };
}

export async function fetchRateLimitStatus(
  baseUrl: string,
): Promise<RateLimitStatusPayload | null> {
  return apiJsonRequest<RateLimitStatusPayload | null>(buildApiUrl(baseUrl, '/rate-limit-status'), {
    selector: (payload) => ((payload as { data?: RateLimitStatusPayload } | null | undefined)?.data ?? null),
  });
}

export async function fetchAgentSettings(baseUrl: string): Promise<AgentSettingsPayload> {
  const response = await apiJsonRequest<Record<string, unknown>>(buildApiUrl(baseUrl, '/agent/settings'));
  const settingsData = (response?.data ?? response) as Record<string, unknown> | undefined;
  return {
    data: settingsData ?? null,
    raw: response,
  };
}

export async function patchAgentSettings(baseUrl: string, body: Record<string, unknown>): Promise<void> {
  await apiVoidRequest(buildApiUrl(baseUrl, '/agent/settings'), {
    method: 'PATCH',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(body),
  });
}

export interface CreateIssueResult {
  issueId: string | null;
  issue: ApiIssue | null;
  raw: unknown;
}

export async function createIssue(
  baseUrl: string,
  input: CreateIssueInput,
): Promise<CreateIssueResult> {
  const payload = {
    title: input.title.trim(),
    description: input.description.trim() || input.title.trim(),
    priority: input.priority.toLowerCase(),
    status: input.status.toLowerCase(),
    app_id: input.appId.trim() || 'unknown',
    tags: input.tags,
    reporter_name: input.reporterName?.trim(),
    reporter_email: input.reporterEmail?.trim(),
    artifacts: input.attachments.map(mapAttachmentPayload),
  };

  const response = await apiJsonRequest<Record<string, unknown>>(buildApiUrl(baseUrl, '/issues'), {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });

  const responseData = (response?.data ?? response) as Record<string, unknown> | undefined;
  const issueId = typeof responseData?.issue_id === 'string' ? (responseData.issue_id as string) : null;
  const issue = (responseData?.issue ?? null) as ApiIssue | null;

  return {
    issueId,
    issue,
    raw: response,
  };
}

export async function updateIssueStatus(
  baseUrl: string,
  issueId: string,
  nextStatus: IssueStatus,
): Promise<{ issue: ApiIssue | null; raw: unknown }> {
  const response = await apiJsonRequest<Record<string, unknown>>(buildApiUrl(baseUrl, `/issues/${encodeURIComponent(issueId)}`), {
    method: 'PATCH',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ status: nextStatus }),
  });

  const responseData = (response?.data ?? response) as Record<string, unknown> | undefined;
  const issue = (responseData?.issue ?? null) as ApiIssue | null;

  return {
    issue,
    raw: response,
  };
}

function mapAttachmentPayload(attachment: CreateIssueAttachmentPayload) {
  return {
    name: attachment.name,
    category: attachment.category ?? 'attachment',
    content: attachment.content,
    encoding: attachment.encoding,
    content_type: attachment.contentType,
  };
}

export async function updateIssue(
  baseUrl: string,
  input: UpdateIssueInput,
): Promise<{ issue: ApiIssue | null; raw: unknown }> {
  const body: Record<string, unknown> = {};

  if (typeof input.title === 'string') {
    body.title = input.title.trim();
  }
  if (typeof input.description === 'string') {
    body.description = input.description.trim();
  }
  if (typeof input.priority === 'string') {
    body.priority = input.priority.toLowerCase();
  }
  if (typeof input.status === 'string') {
    body.status = input.status.toLowerCase();
  }
  if (typeof input.appId === 'string') {
    body.app_id = input.appId.trim();
  }
  if (Array.isArray(input.tags)) {
    body.tags = input.tags;
  }

  if (typeof input.reporterName === 'string' || typeof input.reporterEmail === 'string') {
    body.reporter = {
      ...(typeof input.reporterName === 'string' ? { name: input.reporterName.trim() } : {}),
      ...(typeof input.reporterEmail === 'string' ? { email: input.reporterEmail.trim() } : {}),
    };
  }

  if (Array.isArray(input.attachments) && input.attachments.length > 0) {
    body.artifacts = input.attachments.map(mapAttachmentPayload);
  }

  const response = await apiJsonRequest<Record<string, unknown>>(
    buildApiUrl(baseUrl, `/issues/${encodeURIComponent(input.issueId)}`),
    {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    },
  );

  const responseData = (response?.data ?? response) as Record<string, unknown> | undefined;
  const issue = (responseData?.issue ?? null) as ApiIssue | null;

  return {
    issue,
    raw: response,
  };
}

export async function deleteIssue(baseUrl: string, issueId: string): Promise<void> {
  await apiVoidRequest(buildApiUrl(baseUrl, `/issues/${encodeURIComponent(issueId)}`), {
    method: 'DELETE',
  });
}
