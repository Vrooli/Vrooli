import { DashboardStats, Issue, IssueStatus, Priority } from '../data/sampleData';

export interface ApiAttachment {
  name?: string;
  type?: string;
  path?: string;
  size?: number;
  category?: string;
  description?: string;
}

export interface ApiIssue {
  id: string;
  title: string;
  description?: string;
  notes?: string;
  priority?: string;
  app_id?: string;
  status?: string;
  metadata?: {
    created_at?: string;
    updated_at?: string;
    resolved_at?: string;
    tags?: string[];
    labels?: Record<string, string>;
    extra?: Record<string, string>;
  };
  reporter?: {
    name?: string;
    email?: string;
    timestamp?: string;
  };
  investigation?: {
    agent_id?: string;
    report?: string;
    confidence_score?: number;
    started_at?: string;
    completed_at?: string;
  };
  attachments?: ApiAttachment[];
}

export interface ApiStatsPayload {
  totalIssues?: number;
  openIssues?: number;
  inProgress?: number;
  completedToday?: number;
}

const FALLBACK_STATUSES: IssueStatus[] = ['open', 'active', 'waiting', 'completed', 'failed', 'archived'];

let validStatuses: IssueStatus[] = [...FALLBACK_STATUSES];

export function getValidStatuses(): IssueStatus[] {
  return [...validStatuses];
}

export function setValidStatuses(statuses: string[] | IssueStatus[]): void {
  const normalized = Array.from(
    new Set(
      statuses
        .map((status) => (typeof status === 'string' ? status.trim().toLowerCase() : status))
        .filter((status): status is string => Boolean(status)),
    ),
  );

  if (!normalized.includes('open')) {
    normalized.unshift('open');
  }

  validStatuses = (normalized.length > 0 ? normalized : FALLBACK_STATUSES) as IssueStatus[];
}

export function getFallbackStatuses(): IssueStatus[] {
  return [...FALLBACK_STATUSES];
}

export function formatStatusLabel(status: string): string {
  return status
    .split('-')
    .filter((part) => part.length > 0)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ');
}

export function buildApiUrl(baseUrl: string, path: string): string {
  const normalizedBase = baseUrl.endsWith('/') ? baseUrl.slice(0, -1) : baseUrl;
  return `${normalizedBase}${path}`;
}

function isLoopbackHost(hostname: string | null): boolean {
  if (!hostname) {
    return false;
  }
  const value = hostname.toLowerCase();
  return value === 'localhost' || value === '127.0.0.1' || value === '0.0.0.0' || value === '::1' || value === '[::1]';
}

export function buildAttachmentUrl(baseUrl: string, issueId: string, attachmentPath: string): string {
  const safeIssueId = encodeURIComponent(issueId);
  const segments = attachmentPath
    .split(/[\\/]+/)
    .filter((segment) => segment.length > 0)
    .map((segment) => encodeURIComponent(segment));
  const normalized = segments.join('/');
  const relativePath = `/issues/${safeIssueId}/attachments/${normalized}`;
  const rawUrl = buildApiUrl(baseUrl, relativePath);

  if (typeof window === 'undefined' || !window.location) {
    return rawUrl;
  }

  try {
    const resolved = new URL(rawUrl, window.location.origin);
    const targetHost = resolved.hostname;
    const currentHost = window.location.hostname;

    if (isLoopbackHost(targetHost) && currentHost && !isLoopbackHost(currentHost)) {
      const origin = window.location.origin.replace(/\/$/, '');
      const pathname = resolved.pathname.startsWith('/') ? resolved.pathname : `/${resolved.pathname}`;
      const search = resolved.search ?? '';
      const hash = resolved.hash ?? '';
      return `${origin}${pathname}${search}${hash}`;
    }

    return resolved.toString();
  } catch (error) {
    console.warn('[IssueTracker] Failed to normalize attachment URL', error);
    if (window.location.origin) {
      const origin = window.location.origin.replace(/\/$/, '');
      const normalizedRelative = rawUrl.startsWith('/') ? rawUrl : `/${rawUrl}`;
      return `${origin}${normalizedRelative}`;
    }
    return rawUrl;
  }
}

export function normalizePriority(value?: string | null): Priority {
  switch ((value ?? '').toLowerCase()) {
    case 'critical':
      return 'Critical';
    case 'high':
      return 'High';
    case 'low':
      return 'Low';
    default:
      return 'Medium';
  }
}

export function normalizeStatus(value?: string | null): IssueStatus {
  const normalized = (value ?? '').toLowerCase();
  const statuses = getValidStatuses();
  return (statuses.find((status) => status === normalized) ?? 'open') as IssueStatus;
}

export function normalizeLabels(labels?: Record<string, string>): Record<string, string> {
  if (!labels) {
    return {};
  }
  return Object.fromEntries(Object.entries(labels).map(([key, val]) => [key.toLowerCase(), val]));
}

export function getDateKey(value?: string | null): string | null {
  if (!value) {
    return null;
  }
  const parsed = Date.parse(value);
  if (Number.isNaN(parsed)) {
    return null;
  }
  return new Date(parsed).toISOString().slice(0, 10);
}

export function buildStatusTrend(issues: Issue[]): DashboardStats['statusTrend'] {
  const today = new Date();
  const days = 7;
  const trend: DashboardStats['statusTrend'] = [];

  for (let offset = days - 1; offset >= 0; offset -= 1) {
    const day = new Date(today);
    day.setDate(today.getDate() - offset);
    const key = day.toISOString().slice(0, 10);
    const count = issues.reduce((acc, issue) => {
      return getDateKey(issue.createdAt) === key ? acc + 1 : acc;
    }, 0);
    const label = day.toLocaleDateString(undefined, { weekday: 'short' });
    trend.push({ label, value: count });
  }

  return trend;
}

export function buildDashboardStats(issues: Issue[], apiStats?: ApiStatsPayload | null): DashboardStats {
  const priorityBreakdown: DashboardStats['priorityBreakdown'] = {
    Critical: 0,
    High: 0,
    Medium: 0,
    Low: 0,
  };

  issues.forEach((issue) => {
    priorityBreakdown[issue.priority] = (priorityBreakdown[issue.priority] ?? 0) + 1;
  });

  const openStatuses: IssueStatus[] = ['open'];
  const todayKey = new Date().toISOString().slice(0, 10);

  const totalsFromIssues = {
    total: issues.length,
    open: issues.filter((issue) => openStatuses.includes(issue.status)).length,
    inProgress: issues.filter((issue) => issue.status === 'active').length,
    completedToday: issues.filter(
      (issue) => issue.status === 'completed' && getDateKey(issue.resolvedAt) === todayKey,
    ).length,
  };

  return {
    totalIssues: apiStats?.totalIssues ?? totalsFromIssues.total,
    openIssues: apiStats?.openIssues ?? totalsFromIssues.open,
    inProgress: apiStats?.inProgress ?? totalsFromIssues.inProgress,
    completedToday: apiStats?.completedToday ?? totalsFromIssues.completedToday,
    priorityBreakdown,
    statusTrend: buildStatusTrend(issues),
  };
}

function normalizeWhitespace(value: string): string {
  return value.replace(/\s+/g, ' ').trim();
}

function getTitlePrefix(value: string): string {
  const match = value.match(/^(\[[^\]]+\]\s*)/);
  return match ? match[1] : '';
}

function ensureTitlePrefix(prefix: string, candidate: string): string {
  if (!prefix) {
    return candidate.trim();
  }
  const trimmed = candidate.trim();
  if (!trimmed) {
    return trimmed;
  }
  if (trimmed.startsWith(prefix.trim())) {
    return trimmed;
  }
  if (/^\[[^\]]+\]/.test(trimmed)) {
    return trimmed;
  }
  return `${prefix}${trimmed}`;
}

function extractReporterNotesSection(markdown?: string | null): string | null {
  if (!markdown) {
    return null;
  }

  const lines = markdown.split(/\r?\n/);
  let inSection = false;
  const collected: string[] = [];

  for (const rawLine of lines) {
    const line = rawLine.trim();
    if (!inSection) {
      if (/^##\s+reporter notes/i.test(line)) {
        inSection = true;
      }
      continue;
    }
    if (/^##\s+/.test(line)) {
      break;
    }
    collected.push(rawLine);
  }

  const section = collected.join('\n').trim();
  if (!section) {
    return null;
  }

  return section;
}

export function deriveDisplayTitle(raw: ApiIssue): { displayTitle: string; rawTitle: string } {
  const rawTitle = (raw.title ?? '').trim() || 'Untitled issue';
  const prefix = getTitlePrefix(rawTitle);
  const extras = raw.metadata?.extra ?? {};

  const preferredKeys = [
    'full_title',
    'original_title',
    'source_title',
    'report_message',
    'reporter_message',
    'long_title',
  ];

  for (const key of preferredKeys) {
    const value = extras?.[key];
    if (typeof value === 'string' && value.trim()) {
      return {
        rawTitle,
        displayTitle: ensureTitlePrefix(prefix, value),
      };
    }
  }

  if (rawTitle.endsWith('...')) {
    const reporterNotes = extractReporterNotesSection(raw.description ?? raw.notes);
    if (reporterNotes) {
      const firstMeaningfulLine = reporterNotes
        .split(/\r?\n/)
        .map((line) => line.replace(/^[-*>\s]+/, '').trim())
        .find((line) => line.length > 0);
      if (firstMeaningfulLine) {
        const sanitized = normalizeWhitespace(firstMeaningfulLine.replace(/[*`_~]/g, ''));
        if (sanitized) {
          return {
            rawTitle,
            displayTitle: ensureTitlePrefix(prefix, sanitized),
          };
        }
      }
    }
  }

  return { displayTitle: rawTitle, rawTitle };
}

interface TransformIssueOptions {
  apiBaseUrl: string;
}

export function transformIssue(raw: ApiIssue, options: TransformIssueOptions): Issue {
  const labels = normalizeLabels(raw.metadata?.labels);
  const createdAt = raw.metadata?.created_at ?? raw.reporter?.timestamp ?? new Date().toISOString();
  const resolvedAt = raw.metadata?.resolved_at ?? null;
  const summary = (raw.description ?? raw.notes ?? '').trim() || 'No summary provided yet.';
  const description = (raw.description ?? '').trim();
  const notes = (raw.notes ?? '').trim();
  const tags = Array.isArray(raw.metadata?.tags)
    ? raw.metadata?.tags.filter((tag) => Boolean(tag)).map((tag) => String(tag))
    : [];
  const attachments: Issue['attachments'] = [];
  if (Array.isArray(raw.attachments)) {
    raw.attachments.forEach((attachment) => {
      const path = attachment?.path?.trim();
      if (!path) {
        return;
      }
      attachments.push({
        name: (attachment?.name ?? '').trim() || path.split(/[\\/]+/).pop() || 'Attachment',
        type: (attachment?.type ?? '').trim() || undefined,
        path,
        size: typeof attachment?.size === 'number' ? attachment.size : undefined,
        url: buildAttachmentUrl(options.apiBaseUrl, raw.id, path),
        category: (attachment?.category ?? '').trim() || undefined,
        description: (attachment?.description ?? '').trim() || undefined,
      });
    });
  }
  const assignee =
    labels.assignee ??
    labels.owner ??
    labels.reviewer ??
    raw.reporter?.name ??
    'Unassigned';

  const { displayTitle, rawTitle } = deriveDisplayTitle(raw);

  return {
    id: raw.id,
    title: displayTitle,
    rawTitle,
    summary,
    description: description || summary,
    assignee,
    priority: normalizePriority(raw.priority),
    createdAt,
    status: normalizeStatus(raw.status),
    app: labels.app ?? raw.app_id ?? 'unknown',
    tags,
    attachments,
    resolvedAt,
    updatedAt: raw.metadata?.updated_at ?? null,
    reporterName: raw.reporter?.name?.trim() || undefined,
    reporterEmail: raw.reporter?.email?.trim() || undefined,
    notes: notes || undefined,
    metadata: raw.metadata,
    investigation: raw.investigation,
  };
}

export function buildIssueSnapshot(issue: Issue): Record<string, unknown> {
  return {
    source_issue_id: issue.id,
    source_issue_title_display: issue.title,
    source_issue_title_original: issue.rawTitle ?? issue.title,
    status: issue.status,
    priority: issue.priority,
    app: issue.app,
    assignee: issue.assignee,
    created_at: issue.createdAt,
    updated_at: issue.updatedAt ?? null,
    resolved_at: issue.resolvedAt ?? null,
    summary: issue.summary,
    description: issue.description,
    notes: issue.notes ?? null,
    tags: issue.tags,
    investigation: issue.investigation ?? null,
    metadata: issue.metadata ?? null,
    attachments: issue.attachments.map((attachment) => ({
      name: attachment.name,
      path: attachment.path,
      size: attachment.size ?? null,
      type: attachment.type ?? null,
      category: attachment.category ?? null,
    })),
  };
}
