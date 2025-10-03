import type {
  ChangeEvent,
  DragEvent,
  FormEvent,
  KeyboardEvent as ReactKeyboardEvent,
  MouseEvent as ReactMouseEvent,
  ReactNode,
} from 'react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { useWebSocket } from './hooks/useWebSocket';
import type { WebSocketEvent } from './types/events';
import {
  AlertCircle,
  Brain,
  CalendarClock,
  CircleAlert,
  Clock,
  Cog,
  ExternalLink,
  FileCode,
  FileDown,
  FileText,
  Filter,
  ChevronDown,
  Hash,
  Eye,
  EyeOff,
  Image as ImageIcon,
  KanbanSquare,
  LayoutDashboard,
  Loader2,
  Mail,
  Paperclip,
  Pause,
  Play,
  Plus,
  UploadCloud,
  Tag,
  User,
  X,
} from 'lucide-react';
import { Dashboard } from './pages/Dashboard';
import { IssuesBoard, columnMeta as issueColumnMeta } from './pages/IssuesBoard';
import { SettingsPage } from './pages/Settings';
import {
  AgentSettings,
  AppSettings,
  DashboardStats,
  DisplaySettings,
  Issue,
  IssueAttachment,
  IssueStatus,
  NavKey,
  Priority,
  ProcessorSettings,
  SampleData,
} from './data/sampleData';
import { formatDistanceToNow as formatRelativeDistance } from './utils/date';
import './styles/app.css';

const API_BASE_INPUT = (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? '/api/v1';
const API_BASE_URL = API_BASE_INPUT.endsWith('/') ? API_BASE_INPUT.slice(0, -1) : API_BASE_INPUT;
const ISSUE_FETCH_LIMIT = 200;
const MAX_ATTACHMENT_PREVIEW_CHARS = 8000;
const VALID_STATUSES: IssueStatus[] = ['open', 'active', 'completed', 'failed', 'archived'];

interface ApiIssue {
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
  };
  reporter?: {
    name?: string;
    email?: string;
    timestamp?: string;
  };
  attachments?: ApiAttachment[];
}

interface ApiAttachment {
  name?: string;
  type?: string;
  path?: string;
  size?: number;
}

interface ApiStatsPayload {
  totalIssues?: number;
  openIssues?: number;
  inProgress?: number;
  completedToday?: number;
}

interface CreateIssueInput {
  title: string;
  description: string;
  priority: Priority;
  status: IssueStatus;
  appId: string;
  tags: string[];
  reporterName?: string;
  reporterEmail?: string;
  attachments: CreateIssueAttachmentPayload[];
}

type PriorityFilterValue = Priority | 'all';

interface CreateIssueAttachmentPayload {
  name: string;
  content: string;
  encoding: 'base64' | 'plain';
  contentType: string;
  category?: string;
}

type SnackbarTone = 'info' | 'success' | 'error';

interface SnackbarState {
  message: string;
  tone: SnackbarTone;
}

function buildApiUrl(path: string): string {
  return `${API_BASE_URL}${path}`;
}

function buildAttachmentUrl(issueId: string, attachmentPath: string): string {
  const safeIssueId = encodeURIComponent(issueId);
  const segments = attachmentPath
    .split(/[\\/]+/)
    .filter((segment) => segment.length > 0)
    .map((segment) => encodeURIComponent(segment));
  const normalized = segments.join('/');
  return buildApiUrl(`/issues/${safeIssueId}/attachments/${normalized}`);
}

function escapeHtml(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function renderInlineMarkdown(source: string): string {
  const codeSegments: string[] = [];
  const codePlaceholderPrefix = '\x00CODESEG\x00';

  let output = escapeHtml(source);

  output = output.replace(/`([^`]+)`/g, (_match, code: string) => {
    const placeholder = `${codePlaceholderPrefix}${codeSegments.length}\x00`;
    codeSegments.push(`<code>${escapeHtml(code)}</code>`);
    return placeholder;
  });

  output = output.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
  output = output.replace(/__([^_]+)__/g, '<strong>$1</strong>');
  output = output.replace(/\*([^*]+)\*/g, '<em>$1</em>');
  output = output.replace(/_([^_]+)_/g, '<em>$1</em>');
  output = output.replace(/~~([^~]+)~~/g, '<del>$1</del>');
  output = output.replace(
    /\[([^\]]+)]\(((?:https?:\/\/)[^\s)]+)\)/g,
    '<a href="$2" target="_blank" rel="noopener noreferrer">$1</a>',
  );

  output = output.replace(/\x00CODESEG\x00(\d+)\x00/g, (_match, index: string) => {
    const idx = Number.parseInt(index, 10);
    return codeSegments[idx] ?? '';
  });

  return output;
}

function renderMarkdownToHtml(markdown: string): string {
  const normalized = markdown.replace(/\r\n/g, '\n');
  const lines = normalized.split('\n');
  const blocks: string[] = [];
  let listType: 'ul' | 'ol' | null = null;
  let inCodeBlock = false;
  let codeLanguage = '';
  let codeBuffer: string[] = [];
  let paragraphLines: string[] = [];
  let blockquoteLines: string[] = [];

  const flushParagraph = () => {
    if (paragraphLines.length === 0) {
      return;
    }
    const paragraph = paragraphLines
      .map((line) => renderInlineMarkdown(line))
      .join('<br />');
    blocks.push(`<p>${paragraph}</p>`);
    paragraphLines = [];
  };

  const flushList = () => {
    if (!listType) {
      return;
    }
    blocks.push(listType === 'ul' ? '</ul>' : '</ol>');
    listType = null;
  };

  const flushBlockquote = () => {
    if (blockquoteLines.length === 0) {
      return;
    }
    const quote = blockquoteLines.map((line) => renderInlineMarkdown(line)).join('<br />');
    blocks.push(`<blockquote>${quote}</blockquote>`);
    blockquoteLines = [];
  };

  const flushCodeBlock = () => {
    if (!inCodeBlock) {
      return;
    }
    const languageClass = codeLanguage ? ` class="language-${escapeHtml(codeLanguage)}"` : '';
    blocks.push(`<pre><code${languageClass}>${escapeHtml(codeBuffer.join('\n'))}</code></pre>`);
    codeBuffer = [];
    codeLanguage = '';
    inCodeBlock = false;
  };

  for (const line of lines) {
    const trimmed = line.trimEnd();

    if (inCodeBlock) {
      if (trimmed.startsWith('```')) {
        flushCodeBlock();
        continue;
      }
      codeBuffer.push(line);
      continue;
    }

    if (trimmed.startsWith('```')) {
      flushParagraph();
      flushList();
      flushBlockquote();
      inCodeBlock = true;
      codeLanguage = trimmed.slice(3).trim();
      continue;
    }

    if (trimmed === '') {
      flushParagraph();
      flushList();
      flushBlockquote();
      continue;
    }

    if (/^(?:-{3,}|\*{3,}|_{3,})$/.test(trimmed)) {
      flushParagraph();
      flushList();
      flushBlockquote();
      blocks.push('<hr />');
      continue;
    }

    const headingMatch = trimmed.match(/^(#{1,6})\s+(.*)$/);
    if (headingMatch) {
      flushParagraph();
      flushList();
      flushBlockquote();
      const level = headingMatch[1].length;
      const headingText = headingMatch[2];
      blocks.push(`<h${level}>${renderInlineMarkdown(headingText)}</h${level}>`);
      continue;
    }

    if (trimmed.startsWith('>')) {
      flushParagraph();
      flushList();
      blockquoteLines.push(trimmed.replace(/^>\s?/, ''));
      continue;
    }

    const orderedMatch = trimmed.match(/^(\d+)\.\s+(.*)$/);
    const unorderedMatch = trimmed.match(/^[-*+]\s+(.*)$/);
    if (orderedMatch) {
      flushParagraph();
      flushBlockquote();
      if (listType !== 'ol') {
        flushList();
        listType = 'ol';
        blocks.push('<ol>');
      }
      blocks.push(`<li>${renderInlineMarkdown(orderedMatch[2])}</li>`);
      continue;
    }
    if (unorderedMatch) {
      flushParagraph();
      flushBlockquote();
      if (listType !== 'ul') {
        flushList();
        listType = 'ul';
        blocks.push('<ul>');
      }
      blocks.push(`<li>${renderInlineMarkdown(unorderedMatch[1])}</li>`);
      continue;
    }

    flushList();
    flushBlockquote();
    paragraphLines.push(trimmed);
  }

  flushCodeBlock();
  flushParagraph();
  flushList();
  flushBlockquote();

  if (blocks.length === 0) {
    return `<p>${renderInlineMarkdown(normalized)}</p>`;
  }

  return blocks.join('');
}

function formatFileSize(bytes?: number): string | null {
  if (typeof bytes !== 'number' || !Number.isFinite(bytes) || bytes < 0) {
    return null;
  }
  if (bytes < 1024) {
    return `${Math.round(bytes)} B`;
  }
  const units = ['KB', 'MB', 'GB', 'TB'];
  let value = bytes / 1024;
  let unitIndex = 0;
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }
  const precision = value >= 10 ? 0 : 1;
  return `${value.toFixed(precision)} ${units[unitIndex]}`;
}

function classifyAttachment(attachment: IssueAttachment): 'image' | 'text' | 'json' | 'other' {
  const mime = attachment.type?.toLowerCase() ?? '';
  if (mime.startsWith('image/')) {
    return 'image';
  }
  if (mime === 'application/json' || mime.endsWith('+json')) {
    return 'json';
  }
  if (mime.startsWith('text/')) {
    return 'text';
  }

  const lowerPath = attachment.path.toLowerCase();
  if (lowerPath.endsWith('.json')) {
    return 'json';
  }
  if (['.log', '.txt', '.md', '.markdown', '.yaml', '.yml', '.har'].some((ext) => lowerPath.endsWith(ext))) {
    return 'text';
  }
  return 'other';
}

function normalizePriority(value?: string | null): Priority {
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

function normalizeStatus(value?: string | null): IssueStatus {
  const normalized = (value ?? '').toLowerCase();
  return (VALID_STATUSES.find((status) => status === normalized) ?? 'open') as IssueStatus;
}

function normalizeLabels(labels?: Record<string, string>): Record<string, string> {
  if (!labels) {
    return {};
  }
  return Object.fromEntries(
    Object.entries(labels).map(([key, val]) => [key.toLowerCase(), val]),
  );
}

function getDateKey(value?: string | null): string | null {
  if (!value) {
    return null;
  }
  const parsed = Date.parse(value);
  if (Number.isNaN(parsed)) {
    return null;
  }
  return new Date(parsed).toISOString().slice(0, 10);
}

function formatDateTime(value?: string | null): string {
  if (!value) {
    return '—';
  }
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }
  return parsed.toLocaleString();
}

function formatRelativeTime(value?: string | null): string | undefined {
  if (!value) {
    return undefined;
  }
  return formatRelativeDistance(value);
}

function buildStatusTrend(issues: Issue[]): DashboardStats['statusTrend'] {
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

function toTitleCase(value: string): string {
  return value
    .split(' ')
    .filter(Boolean)
    .map((segment) => segment.charAt(0).toUpperCase() + segment.slice(1))
    .join(' ');
}

const DEFAULT_STATS: DashboardStats = {
  totalIssues: 0,
  openIssues: 0,
  inProgress: 0,
  completedToday: 0,
  priorityBreakdown: {
    Critical: 0,
    High: 0,
    Medium: 0,
    Low: 0,
  },
  statusTrend: buildStatusTrend([]),
};

function buildDashboardStats(issues: Issue[], apiStats?: ApiStatsPayload | null): DashboardStats {
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

function transformIssue(raw: ApiIssue): Issue {
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
        url: buildAttachmentUrl(raw.id, path),
        category: (attachment?.category ?? '').trim() || undefined,
      });
    });
  }
  const assignee =
    labels.assignee ??
    labels.owner ??
    labels.reviewer ??
    raw.reporter?.name ??
    'Unassigned';

  return {
    id: raw.id,
    title: raw.title,
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

function getIssueIdFromLocation(): string | null {
  if (typeof window === 'undefined') {
    return null;
  }

  try {
    const params = new URLSearchParams(window.location.search);
    const value = params.get('issue');
    if (!value) {
      return null;
    }
    const trimmed = value.trim();
    return trimmed.length > 0 ? trimmed : null;
  } catch (error) {
    console.warn('[IssueTracker] Unable to parse issue id from URL', error);
    return null;
  }
}

// Navigation removed - single page app with dialogs

// URL routing simplified - only issues page with query params

function App() {
  const isMountedRef = useRef(true);
  const [metricsDialogOpen, setMetricsDialogOpen] = useState(false);
  const [settingsDialogOpen, setSettingsDialogOpen] = useState(false);
  const [wsConnectionStatus, setWsConnectionStatus] = useState<'connecting' | 'connected' | 'disconnected' | 'error'>('disconnected');
  const [processorSettings, setProcessorSettings] = useState<ProcessorSettings>(SampleData.processor);
  const [agentSettings, setAgentSettings] = useState(AppSettings.agent);
  const [displaySettings, setDisplaySettings] = useState(AppSettings.display);
  const [issues, setIssues] = useState<Issue[]>([]);
  const [priorityFilter, setPriorityFilter] = useState<PriorityFilterValue>('all');
  const [appFilter, setAppFilter] = useState<string>('all');
  const [searchFilter, setSearchFilter] = useState<string>('');
  const [hiddenColumns, setHiddenColumns] = useState<IssueStatus[]>(['archived']);
  const [createIssueOpen, setCreateIssueOpen] = useState(false);
  const [issueDetailOpen, setIssueDetailOpen] = useState(false);
  const [dashboardStats, setDashboardStats] = useState<DashboardStats>(DEFAULT_STATS);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [processorError, setProcessorError] = useState<string | null>(null);
  const [issuesProcessed, setIssuesProcessed] = useState<number>(0);
  const [issuesRemaining, setIssuesRemaining] = useState<number | string>('unlimited');
  const [rateLimitStatus, setRateLimitStatus] = useState<{
    rate_limited: boolean;
    rate_limited_count: number;
    reset_time: string;
    seconds_until_reset: number;
    rate_limit_agent: string;
  } | null>(null);
  const [focusedIssueId, setFocusedIssueId] = useState<string | null>(() => getIssueIdFromLocation());
  const [filtersOpen, setFiltersOpen] = useState(false);
  const [snackbar, setSnackbar] = useState<SnackbarState | null>(null);
  const [runningProcesses, setRunningProcesses] = useState<Map<string, { agent_id: string; start_time: string; duration?: string }>>(new Map());

  const showSnackbar = useCallback((message: string, tone: SnackbarTone = 'info') => {
    setSnackbar({ message, tone });
  }, []);

  // WebSocket event handler
  const handleWebSocketEvent = useCallback((event: WebSocketEvent) => {
    switch (event.type) {
      case 'issue.created':
      case 'issue.updated': {
        const updatedIssue = transformIssue(event.data.issue);
        setIssues(prev => {
          const index = prev.findIndex(i => i.id === updatedIssue.id);
          if (index >= 0) {
            return prev.map((i, idx) => idx === index ? updatedIssue : i);
          }
          return [...prev, updatedIssue];
        });
        break;
      }

      case 'issue.status_changed': {
        const { issue_id, new_status } = event.data;
        setIssues(prev => prev.map(i =>
          i.id === issue_id ? { ...i, status: normalizeStatus(new_status) } : i
        ));
        break;
      }

      case 'issue.deleted': {
        const { issue_id } = event.data;
        setIssues(prev => prev.filter(i => i.id !== issue_id));
        if (focusedIssueId === issue_id) {
          setFocusedIssueId(null);
          setIssueDetailOpen(false);
        }
        break;
      }

      case 'agent.started': {
        const { issue_id, agent_id, start_time } = event.data;
        setRunningProcesses(prev => {
          const next = new Map(prev);
          next.set(issue_id, { agent_id, start_time });
          return next;
        });
        break;
      }

      case 'agent.completed':
      case 'agent.failed': {
        const { issue_id } = event.data;
        setRunningProcesses(prev => {
          const next = new Map(prev);
          next.delete(issue_id);
          return next;
        });
        break;
      }

      default:
        break;
    }
  }, [focusedIssueId]);

  // WebSocket connection
  const wsUrl = useMemo(() => {
    if (typeof window === 'undefined') return '';

    // If API_BASE_URL is absolute (starts with http/https), extract host from it
    if (API_BASE_INPUT.startsWith('http://') || API_BASE_INPUT.startsWith('https://')) {
      const url = new URL(API_BASE_INPUT);
      const wsProtocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
      const finalUrl = `${wsProtocol}//${url.host}${API_BASE_URL}/ws`;
      console.log('[WebSocket] URL (absolute):', finalUrl);
      return finalUrl;
    }

    // Check if we're behind a proxy/tunnel (non-localhost hostname)
    const hostname = window.location.hostname;
    const isProxied = hostname !== 'localhost' && hostname !== '127.0.0.1' && !hostname.match(/^192\.168\./);

    if (isProxied) {
      // Use the proxy - Vite will forward WebSocket connections
      const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const finalUrl = `${wsProtocol}//${window.location.host}${API_BASE_URL}/ws`;
      console.log('[WebSocket] URL (proxied):', finalUrl);
      return finalUrl;
    }

    // For local development, connect directly to API port
    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const apiPort = typeof __API_PORT__ !== 'undefined' ? __API_PORT__ : '8090';
    const finalUrl = `${wsProtocol}//${hostname}:${apiPort}${API_BASE_URL}/ws`;
    console.log('[WebSocket] URL (direct):', finalUrl);
    return finalUrl;
  }, []);

  const { status: wsStatus, error: wsError } = useWebSocket({
    url: wsUrl,
    onEvent: handleWebSocketEvent,
    enabled: typeof window !== 'undefined',
  });

  useEffect(() => {
    setWsConnectionStatus(wsStatus);
  }, [wsStatus]);

  useEffect(() => {
    if (wsError) {
      console.error('[App] WebSocket error:', wsError);
    }
  }, [wsError]);

  const handleProcessorError = useCallback(
    (message: string) => {
      setProcessorError(message);
      showSnackbar(message, 'error');
    },
    [showSnackbar],
  );

  const selectedIssue = useMemo(() => {
    return issues.find((issue) => issue.id === focusedIssueId) ?? null;
  }, [issues, focusedIssueId]);

  const updateStatsFromIssues = useCallback(
    (nextIssues: Issue[]) => {
      setDashboardStats(buildDashboardStats(nextIssues, null));
    },
    [],
  );

  const availableApps = useMemo(() => {
    const apps = new Set<string>();
    issues.forEach((issue) => {
      if (issue.app) {
        apps.add(issue.app);
      }
    });
    return Array.from(apps).sort((first, second) => first.localeCompare(second));
  }, [issues]);

  const filteredIssues = useMemo(() => {
    return issues.filter((issue) => {
      const matchesPriority = priorityFilter === 'all' || issue.priority === priorityFilter;
      const matchesApp = appFilter === 'all' || issue.app === appFilter;

      let matchesSearch = true;
      if (searchFilter.trim()) {
        const query = searchFilter.toLowerCase();
        matchesSearch =
          issue.id.toLowerCase().includes(query) ||
          issue.title.toLowerCase().includes(query) ||
          issue.description.toLowerCase().includes(query) ||
          issue.assignee.toLowerCase().includes(query) ||
          issue.tags.some(tag => tag.toLowerCase().includes(query)) ||
          (issue.reporterName?.toLowerCase().includes(query) ?? false) ||
          (issue.reporterEmail?.toLowerCase().includes(query) ?? false);
      }

      return matchesPriority && matchesApp && matchesSearch;
    });
  }, [issues, priorityFilter, appFilter, searchFilter]);

  useEffect(() => {
    if (appFilter !== 'all' && !availableApps.includes(appFilter)) {
      setAppFilter('all');
    }
  }, [appFilter, availableApps]);

  useEffect(() => {
    if (!snackbar || typeof window === 'undefined') {
      return;
    }
    const timeoutId = window.setTimeout(() => {
      setSnackbar((current) => (current === snackbar ? null : current));
    }, 5000);
    return () => {
      window.clearTimeout(timeoutId);
    };
  }, [snackbar]);

  useEffect(() => {
    isMountedRef.current = true;
    return () => {
      isMountedRef.current = false;
    };
  }, []);

  useEffect(() => {
    if (typeof window === 'undefined') {
      return;
    }

    const handlePopState = () => {
      const nextIssueId = getIssueIdFromLocation();
      setFocusedIssueId(nextIssueId);
      setIssueDetailOpen(Boolean(nextIssueId));
    };

    window.addEventListener('popstate', handlePopState);
    return () => {
      window.removeEventListener('popstate', handlePopState);
    };
  }, []);

  // Removed navigation initialization - single page app

  const fetchAllData = useCallback(async () => {
    if (isMountedRef.current) {
      setLoading(true);
      setLoadError(null);
      setProcessorError(null);
    }

    let mappedIssues: Issue[] = [];

    try {
      const response = await fetch(buildApiUrl(`/issues?limit=${ISSUE_FETCH_LIMIT}`));
      if (!response.ok) {
        throw new Error(`Issues request failed with status ${response.status}`);
      }
      const payload = await response.json();
      const rawIssues: ApiIssue[] = Array.isArray(payload?.data?.issues) ? payload.data.issues : [];
      mappedIssues = rawIssues.map(transformIssue);
      if (isMountedRef.current) {
        setIssues(mappedIssues);
      }
    } catch (error) {
      console.error('Failed to load issues', error);
      if (isMountedRef.current) {
        setLoadError('Failed to load issues from the API.');
        setLoading(false);
      }
      return;
    }

    let statsFromApi: ApiStatsPayload | null = null;
    try {
      const response = await fetch(buildApiUrl('/stats'));
      if (response.ok) {
        const payload = await response.json();
        const stats = payload?.data?.stats ?? null;
        if (stats) {
          statsFromApi = {
            totalIssues: typeof stats.total_issues === 'number' ? stats.total_issues : undefined,
            openIssues: typeof stats.open_issues === 'number' ? stats.open_issues : undefined,
            inProgress: typeof stats.in_progress === 'number' ? stats.in_progress : undefined,
            completedToday:
              typeof stats.completed_today === 'number' ? stats.completed_today : undefined,
          };
        }
      }
    } catch (error) {
      console.warn('Failed to load stats', error);
    }

    const computedStats = buildDashboardStats(mappedIssues, statsFromApi);
    if (isMountedRef.current) {
      setDashboardStats(computedStats);
    }

    try {
      const response = await fetch(buildApiUrl('/automation/processor'));
      if (response.ok) {
        const payload = await response.json();
        const state = payload?.data?.processor ?? payload?.processor;
        const data = payload?.data;
        if (state && isMountedRef.current) {
          setProcessorSettings((prev) => ({
            active: typeof state.active === 'boolean' ? state.active : prev.active,
            concurrentSlots:
              typeof state.concurrent_slots === 'number'
                ? state.concurrent_slots
                : prev.concurrentSlots,
            refreshInterval:
              typeof state.refresh_interval === 'number'
                ? state.refresh_interval
                : prev.refreshInterval,
            maxIssues:
              typeof state.max_issues === 'number'
                ? state.max_issues
                : prev.maxIssues,
          }));

          // Update issues processed/remaining counters
          if (typeof data?.issues_processed === 'number') {
            setIssuesProcessed(data.issues_processed);
          }
          if (data?.issues_remaining !== undefined) {
            setIssuesRemaining(data.issues_remaining);
          }

          setProcessorError(null);
        }
      } else if (isMountedRef.current) {
        handleProcessorError('Failed to load automation status.');
      }
    } catch (error) {
      console.warn('Failed to load processor settings', error);
      if (isMountedRef.current) {
        handleProcessorError('Failed to load automation status.');
      }
    }

    // Fetch rate limit status
    try {
      const response = await fetch(buildApiUrl('/rate-limit-status'));
      if (response.ok) {
        const payload = await response.json();
        const status = payload?.data;
        if (status && isMountedRef.current) {
          setRateLimitStatus(status);
        }
      }
    } catch (error) {
      console.warn('Failed to load rate limit status', error);
    }

    if (isMountedRef.current) {
      setLoading(false);
    }
  }, [handleProcessorError]);

  useEffect(() => {
    void fetchAllData();
  }, [fetchAllData]);

  // Fetch agent backend settings
  useEffect(() => {
    const fetchAgentBackendSettings = async () => {
      try {
        const response = await fetch(buildApiUrl('/agent/settings'));
        if (response.ok) {
          const data = await response.json();
          if (data?.agent_backend && isMountedRef.current) {
            setAgentSettings((prev) => ({
              ...prev,
              backend: {
                provider: data.agent_backend.provider ?? 'codex',
                autoFallback: data.agent_backend.auto_fallback ?? true,
              },
            }));
          }
        }
      } catch (error) {
        console.warn('Failed to load agent backend settings', error);
      }
    };
    void fetchAgentBackendSettings();
  }, []);

  // Running processes duration calculation (updates every second for display)
  useEffect(() => {
    const updateDurations = () => {
      setRunningProcesses(prev => {
        if (prev.size === 0) return prev;

        const next = new Map(prev);
        let changed = false;

        for (const [issueId, proc] of next.entries()) {
          const startTime = new Date(proc.start_time);
          const now = new Date();
          const durationMs = now.getTime() - startTime.getTime();
          const seconds = Math.floor(durationMs / 1000);
          const minutes = Math.floor(seconds / 60);
          const hours = Math.floor(minutes / 60);

          let duration = '';
          if (hours > 0) {
            duration = `${hours}h ${minutes % 60}m ${seconds % 60}s`;
          } else if (minutes > 0) {
            duration = `${minutes}m ${seconds % 60}s`;
          } else {
            duration = `${seconds}s`;
          }

          if (proc.duration !== duration) {
            next.set(issueId, { ...proc, duration });
            changed = true;
          }
        }

        return changed ? next : prev;
      });
    };

    const intervalId = setInterval(updateDurations, 1000);
    return () => clearInterval(intervalId);
  }, []);

  useEffect(() => {
    if (typeof window === 'undefined') {
      return;
    }

    const url = new URL(window.location.href);
    if (focusedIssueId) {
      url.searchParams.set('issue', focusedIssueId);
    } else {
      url.searchParams.delete('issue');
    }

    const nextSearch = url.searchParams.toString();
    const nextUrl = `${url.pathname}${nextSearch ? `?${nextSearch}` : ''}${url.hash}`;
    window.history.replaceState({}, '', nextUrl);
  }, [focusedIssueId]);

  useEffect(() => {
    setIssueDetailOpen(Boolean(focusedIssueId));
  }, [focusedIssueId]);

  useEffect(() => {
    if (typeof document === 'undefined') {
      return;
    }

    const themeClass = displaySettings.theme === 'dark' ? 'dark' : 'light';
    const body = document.body;
    const root = document.documentElement;

    body.classList.remove('light', 'dark');
    body.classList.add(themeClass);

    root.classList.remove('light', 'dark');
    root.classList.add(themeClass);
    root.style.setProperty('color-scheme', themeClass === 'dark' ? 'dark' : 'light');

    return () => {
      body.classList.remove(themeClass);
      root.classList.remove(themeClass);
    };
  }, [displaySettings.theme]);

  useEffect(() => {
    if (typeof document === 'undefined') {
      return;
    }

    const originalOverflow = document.body.style.overflow;
    if (createIssueOpen || (issueDetailOpen && selectedIssue)) {
      document.body.style.overflow = 'hidden';
    } else {
      document.body.style.overflow = '';
    }

    return () => {
      document.body.style.overflow = originalOverflow;
    };
  }, [createIssueOpen, issueDetailOpen, selectedIssue]);

  useEffect(() => {
    if (!focusedIssueId) {
      return;
    }

    const issueExists = issues.some((issue) => issue.id === focusedIssueId);
    if (!issueExists) {
      return;
    }

    if (typeof window === 'undefined' || typeof document === 'undefined') {
      return;
    }

    const element = document.querySelector<HTMLElement>(`[data-issue-id="${focusedIssueId}"]`);
    if (element) {
      element.focus({ preventScroll: true });
      window.setTimeout(() => {
        element.scrollIntoView({ behavior: 'smooth', block: 'center' });
      }, 50);
    }
  }, [focusedIssueId, issues]);

  // Removed navigation sync - single page app

  const createIssue = useCallback(
    async (input: CreateIssueInput): Promise<string | null> => {
      const payload = {
        title: input.title.trim(),
        description: input.description.trim() || input.title.trim(),
        priority: input.priority.toLowerCase(),
        status: input.status.toLowerCase(),
        app_id: input.appId.trim() || 'unknown',
        tags: input.tags,
        reporter_name: input.reporterName?.trim(),
        reporter_email: input.reporterEmail?.trim(),
        artifacts: input.attachments.map((attachment) => ({
          name: attachment.name,
          category: attachment.category ?? 'attachment',
          content: attachment.content,
          encoding: attachment.encoding,
          content_type: attachment.contentType,
        })),
      };

      const response = await fetch(buildApiUrl('/issues'), {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        throw new Error(`Issue creation failed with status ${response.status}`);
      }

      const responseBody = await response.json();
      await fetchAllData();
      const issueId = typeof responseBody?.data?.issue_id === 'string' ? responseBody.data.issue_id : null;
      return issueId;
    },
    [fetchAllData],
  );

  const deleteIssue = useCallback(async (issueId: string) => {
    const response = await fetch(buildApiUrl(`/issues/${encodeURIComponent(issueId)}`), {
      method: 'DELETE',
    });
    if (!response.ok) {
      throw new Error(`Issue deletion failed with status ${response.status}`);
    }
  }, []);

  const updateIssueStatus = useCallback(
    async (issueId: string, nextStatus: IssueStatus) => {
      const response = await fetch(buildApiUrl(`/issues/${encodeURIComponent(issueId)}`), {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ status: nextStatus }),
      });

      if (!response.ok) {
        throw new Error(`Issue update failed with status ${response.status}`);
      }

      const responseBody = await response.json();
      const updatedIssue: ApiIssue | null = responseBody?.data?.issue ?? null;

      setIssues((prev) => {
        let nextIssues: Issue[];
        if (updatedIssue) {
          const transformed = transformIssue(updatedIssue);
          nextIssues = prev.map((issue) => (issue.id === issueId ? transformed : issue));
        } else {
          nextIssues = prev.map((issue) =>
            issue.id === issueId
              ? {
                  ...issue,
                  status: nextStatus,
                }
              : issue,
          );
        }
        updateStatsFromIssues(nextIssues);
        return nextIssues;
      });
    },
    [updateStatsFromIssues],
  );

  const handleCreateIssue = useCallback(() => {
    setFiltersOpen(false);
    setCreateIssueOpen(true);
  }, []);

  const handleCreateIssueClose = useCallback(() => {
    setCreateIssueOpen(false);
  }, []);

  const handleIssueSelect = useCallback((issueId: string) => {
    setFocusedIssueId((current) => {
      if (current === issueId) {
        setIssueDetailOpen(false);
        return null;
      }
      setIssueDetailOpen(true);
      return issueId;
    });
  }, []);

  const handleSubmitNewIssue = useCallback(
    async (input: CreateIssueInput) => {
      const newIssueId = await createIssue(input);
      if (newIssueId) {
        setFocusedIssueId(newIssueId);
        setIssueDetailOpen(true);
      }
    },
    [createIssue],
  );

  const handlePriorityFilterChange = useCallback((value: PriorityFilterValue) => {
    setPriorityFilter(value);
  }, []);

  const handleAppFilterChange = useCallback((value: string) => {
    setAppFilter(value);
  }, []);

  const handleSearchFilterChange = useCallback((value: string) => {
    setSearchFilter(value);
  }, []);

  const activeFilterCount = useMemo(() => {
    let count = 0;
    if (priorityFilter !== 'all') count++;
    if (appFilter !== 'all') count++;
    if (searchFilter.trim()) count++;
    return count;
  }, [priorityFilter, appFilter, searchFilter]);

  const handleIssueArchive = useCallback(
    async (issue: Issue) => {
      try {
        await updateIssueStatus(issue.id, 'archived');
      } catch (error) {
        console.error('Failed to archive issue', error);
        if (typeof window !== 'undefined' && typeof window.alert === 'function') {
          window.alert('Failed to archive issue. Please try again.');
        }
      }
    },
    [updateIssueStatus],
  );

  const handleIssueDelete = useCallback(
    async (issue: Issue) => {
      const issueId = issue.id;
      try {
        await deleteIssue(issueId);
        setIssues((prev) => {
          const nextIssues = prev.filter((item) => item.id !== issueId);
          updateStatsFromIssues(nextIssues);
          return nextIssues;
        });
        if (focusedIssueId === issueId) {
          setIssueDetailOpen(false);
          setFocusedIssueId(null);
        }
      } catch (error) {
        console.error('Failed to delete issue', error);
        if (typeof window !== 'undefined' && typeof window.alert === 'function') {
          window.alert('Failed to delete issue. Please try again.');
        }
      }
    },
    [deleteIssue, focusedIssueId, updateStatsFromIssues],
  );

  const handleIssueStatusChange = useCallback(
    async (issueId: string, _fromStatus: IssueStatus, targetStatus: IssueStatus) => {
      if (_fromStatus === targetStatus) {
        return;
      }
      try {
        await updateIssueStatus(issueId, targetStatus);
      } catch (error) {
        console.error('Failed to update issue status', error);
        if (typeof window !== 'undefined' && typeof window.alert === 'function') {
          window.alert('Failed to move issue. Please try again.');
        }
      }
    },
    [updateIssueStatus],
  );

  const handleHideColumn = useCallback((status: IssueStatus) => {
    setHiddenColumns((current) => (current.includes(status) ? current : [...current, status]));
  }, []);

  const handleToggleColumn = useCallback((status: IssueStatus) => {
    setHiddenColumns((current) =>
      current.includes(status)
        ? current.filter((value) => value !== status)
        : [...current, status],
    );
  }, []);

  const handleResetHiddenColumns = useCallback(() => {
    setHiddenColumns([]);
  }, []);

  const handleIssueDetailClose = useCallback(() => {
    setIssueDetailOpen(false);
    setFocusedIssueId(null);
  }, []);

  const handleOpenMetrics = useCallback(() => {
    setMetricsDialogOpen(true);
  }, []);

  const handleCloseMetrics = useCallback(() => {
    setMetricsDialogOpen(false);
  }, []);

  const handleOpenSettings = useCallback(() => {
    setSettingsDialogOpen(true);
  }, []);

  const handleCloseSettings = useCallback(() => {
    setSettingsDialogOpen(false);
  }, []);

  const handleMetricsOpenIssue = useCallback(
    (issueId: string) => {
      setMetricsDialogOpen(false);
      setFocusedIssueId(issueId);
      setIssueDetailOpen(true);
    },
    [],
  );

  const handleToggleActive = useCallback(() => {
    const previousActive = processorSettings.active;
    const nextActive = !previousActive;

    setProcessorSettings((prev) => ({
      ...prev,
      active: nextActive,
    }));

    setProcessorError(null);

    void (async () => {
      try {
        const response = await fetch(buildApiUrl('/automation/processor'), {
          method: 'PATCH',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({ active: nextActive }),
        });

        if (!response.ok) {
          throw new Error(`Processor update failed with status ${response.status}`);
        }

        const payload = await response.json();
        const state = payload?.data?.processor ?? payload?.processor;
        if (state && isMountedRef.current) {
          setProcessorSettings((prev) => ({
            active: typeof state.active === 'boolean' ? state.active : prev.active,
            concurrentSlots:
              typeof state.concurrent_slots === 'number'
                ? state.concurrent_slots
                : prev.concurrentSlots,
            refreshInterval:
              typeof state.refresh_interval === 'number'
                ? state.refresh_interval
                : prev.refreshInterval,
            maxIssues:
              typeof state.max_issues === 'number'
                ? state.max_issues
                : prev.maxIssues,
          }));
          setProcessorError(null);
        }
      } catch (error) {
        console.error('Failed to update processor state', error);
        if (isMountedRef.current) {
          setProcessorSettings((prev) => ({
            ...prev,
            active: previousActive,
          }));
          handleProcessorError('Failed to update automation status.');
        }
      }
    })();
  }, [processorSettings.active, handleProcessorError]);

  const processorSetter = (settings: ProcessorSettings) => {
    setProcessorSettings(settings);
  };

  // Sync processor settings to API
  // Use ref to skip initial mount to avoid race condition with data fetch
  const processorSettingsInitializedRef = useRef(false);
  useEffect(() => {
    // Skip on initial mount - let the fetch load the real state first
    if (!processorSettingsInitializedRef.current) {
      processorSettingsInitializedRef.current = true;
      return;
    }

    const saveProcessorSettings = async () => {
      try {
        const response = await fetch(buildApiUrl('/automation/processor'), {
          method: 'PATCH',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            active: processorSettings.active,
            concurrent_slots: processorSettings.concurrentSlots,
            refresh_interval: processorSettings.refreshInterval,
            max_issues: processorSettings.maxIssues,
          }),
        });

        if (!response.ok) {
          throw new Error(`Processor settings update failed with status ${response.status}`);
        }

        console.log('Processor settings saved');
      } catch (error) {
        console.error('Failed to save processor settings', error);
        showSnackbar('Failed to save processor settings', 'error');
      }
    };

    void saveProcessorSettings();
  }, [processorSettings.active, processorSettings.concurrentSlots, processorSettings.refreshInterval, processorSettings.maxIssues, showSnackbar]);

  // Sync agent backend settings to API
  // Use ref to skip initial mount to avoid race condition with data fetch
  const agentSettingsInitializedRef = useRef(false);
  useEffect(() => {
    const backend = agentSettings.backend;
    if (!backend) {
      return;
    }

    // Skip on initial mount - let the fetch load the real state first
    if (!agentSettingsInitializedRef.current) {
      agentSettingsInitializedRef.current = true;
      return;
    }

    const saveAgentBackendSettings = async () => {
      try {
        const response = await fetch(buildApiUrl('/agent/settings'), {
          method: 'PATCH',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            provider: backend.provider,
            auto_fallback: backend.autoFallback,
          }),
        });

        if (!response.ok) {
          throw new Error(`Agent settings update failed with status ${response.status}`);
        }

        console.log('Agent backend settings saved:', backend);
      } catch (error) {
        console.error('Failed to save agent backend settings', error);
        showSnackbar('Failed to save agent backend settings', 'error');
      }
    };

    void saveAgentBackendSettings();
  }, [agentSettings.backend, showSnackbar]);


  // Single page - always render issues board

  const showIssueDetailModal = issueDetailOpen && selectedIssue;

  return (
    <>
      <div className={`app-shell ${displaySettings.theme}`}>
        <div className="main-panel">
          {wsConnectionStatus === 'connecting' && (
            <div className="connection-banner connection-banner--connecting">
              <Loader2 size={18} className="spin" style={{ marginRight: '8px' }} />
              <span>Connecting to live updates...</span>
            </div>
          )}
          {wsConnectionStatus === 'error' && (
            <div className="connection-banner connection-banner--error">
              <CircleAlert size={18} style={{ marginRight: '8px' }} />
              <span>Connection error - live updates unavailable</span>
            </div>
          )}
          {rateLimitStatus?.rate_limited && rateLimitStatus.seconds_until_reset > 0 && (
            <div className="rate-limit-banner">
              <Clock size={18} style={{ marginRight: '8px' }} />
              <span>
                Rate limit active ({rateLimitStatus.rate_limited_count} issue{rateLimitStatus.rate_limited_count !== 1 ? 's' : ''} waiting) - Resets in{' '}
                {Math.floor(rateLimitStatus.seconds_until_reset / 60)}m {rateLimitStatus.seconds_until_reset % 60}s
                {rateLimitStatus.rate_limit_agent && ` (${rateLimitStatus.rate_limit_agent})`}
              </span>
            </div>
          )}
          <main className="page-container page-container--issues">
            {loading && <div className="data-banner loading">Loading latest data…</div>}
            {loadError && (
              <div className="data-banner error">
                <span>{loadError}</span>
                <button type="button" onClick={() => void fetchAllData()}>
                  Retry
                </button>
              </div>
            )}
            <IssuesBoard
              issues={filteredIssues}
              focusedIssueId={focusedIssueId}
              runningProcesses={runningProcesses}
              onIssueSelect={handleIssueSelect}
              onIssueDelete={handleIssueDelete}
              onIssueArchive={handleIssueArchive}
              onIssueDrop={handleIssueStatusChange}
              hiddenColumns={hiddenColumns}
              onHideColumn={handleHideColumn}
            />

            <div className="issues-floating-controls">
              <div className="issues-floating-panel">
                <button
                  type="button"
                  className={`icon-button icon-button--status ${processorSettings.active ? 'is-active' : ''}`.trim()}
                  onClick={handleToggleActive}
                  aria-label={
                    processorSettings.active ? 'Pause issue automation' : 'Activate issue automation'
                  }
                >
                  {processorSettings.active ? <Pause size={18} /> : <Play size={18} />}
                </button>
                {runningProcesses.size > 0 && (
                  <div className="running-agents-indicator">
                    <Brain size={16} className="running-agents-icon" />
                    <span className="running-agents-count">{runningProcesses.size}</span>
                  </div>
                )}
                <button
                  type="button"
                  className="icon-button icon-button--primary"
                  onClick={handleCreateIssue}
                  aria-label="Create new issue"
                >
                  <Plus size={18} />
                </button>
                <button
                  type="button"
                  className={`icon-button icon-button--filter ${filtersOpen ? 'is-active' : ''}`.trim()}
                  onClick={() => setFiltersOpen((state) => !state)}
                  aria-label="Toggle filters"
                  aria-expanded={filtersOpen}
                  style={{ position: 'relative' }}
                >
                  <Filter size={18} />
                  {activeFilterCount > 0 && (
                    <span className="filter-badge" aria-label={`${activeFilterCount} active filters`}>
                      {activeFilterCount}
                    </span>
                  )}
                </button>
                <button
                  type="button"
                  className="icon-button icon-button--secondary"
                  onClick={handleOpenMetrics}
                  aria-label="Open metrics"
                >
                  <LayoutDashboard size={18} />
                </button>
                <button
                  type="button"
                  className="icon-button icon-button--secondary"
                  onClick={handleOpenSettings}
                  aria-label="Open settings"
                >
                  <Cog size={18} />
                </button>
                <IssueBoardToolbar
                  open={filtersOpen}
                  onRequestClose={() => setFiltersOpen(false)}
                  priorityFilter={priorityFilter}
                  onPriorityFilterChange={handlePriorityFilterChange}
                  appFilter={appFilter}
                  onAppFilterChange={handleAppFilterChange}
                  searchFilter={searchFilter}
                  onSearchFilterChange={handleSearchFilterChange}
                  appOptions={availableApps}
                  hiddenColumns={hiddenColumns}
                  onToggleColumn={handleToggleColumn}
                  onResetColumns={handleResetHiddenColumns}
                />
              </div>
            </div>
          </main>
        </div>
      </div>

      {createIssueOpen && (
        <CreateIssueModal onClose={handleCreateIssueClose} onSubmit={handleSubmitNewIssue} />
      )}

      {showIssueDetailModal && selectedIssue && (
        <IssueDetailsModal issue={selectedIssue} onClose={handleIssueDetailClose} onStatusChange={updateIssueStatus} />
      )}

      {metricsDialogOpen && (
        <MetricsDialog
          stats={dashboardStats}
          issues={issues}
          processor={processorSettings}
          agentSettings={agentSettings}
          onClose={handleCloseMetrics}
          onOpenIssue={handleMetricsOpenIssue}
        />
      )}

      {settingsDialogOpen && (
        <SettingsDialog
          apiBaseUrl={API_BASE_URL}
          processor={processorSettings}
          agent={agentSettings}
          display={displaySettings}
          onProcessorChange={processorSetter}
          onAgentChange={setAgentSettings}
          onDisplayChange={setDisplaySettings}
          onClose={handleCloseSettings}
          issuesProcessed={issuesProcessed}
          issuesRemaining={issuesRemaining}
        />
      )}

      {snackbar && (
        <Snackbar message={snackbar.message} tone={snackbar.tone} onClose={() => setSnackbar(null)} />
      )}
    </>
  );
}

interface ModalProps {
  onClose: () => void;
  labelledBy?: string;
  panelClassName?: string;
  children: ReactNode;
}

function Modal({ onClose, labelledBy, panelClassName, children }: ModalProps) {
  const handleBackdropMouseDown = (event: ReactMouseEvent<HTMLDivElement>) => {
    if (event.target === event.currentTarget) {
      onClose();
    }
  };

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose();
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
    };
  }, [onClose]);

  const modalContent = (
    <div className="modal-backdrop" onMouseDown={handleBackdropMouseDown}>
      <div
        className={`modal-panel${panelClassName ? ` ${panelClassName}` : ''}`}
        role="dialog"
        aria-modal="true"
        aria-labelledby={labelledBy}
      >
        {children}
      </div>
    </div>
  );

  return typeof document !== 'undefined'
    ? createPortal(modalContent, document.body)
    : modalContent;
}

interface SnackbarProps {
  message: string;
  tone: SnackbarTone;
  onClose: () => void;
}

function Snackbar({ message, tone, onClose }: SnackbarProps) {
  const role = tone === 'error' ? 'alert' : 'status';
  const live = tone === 'error' ? 'assertive' : 'polite';

  return (
    <div className="snackbar-container">
      <div className={`snackbar snackbar--${tone}`} role={role} aria-live={live}>
        <span className="snackbar-message">{message}</span>
        <button
          type="button"
          className="snackbar-close"
          onClick={onClose}
          aria-label="Dismiss notification"
        >
          <X size={14} />
        </button>
      </div>
    </div>
  );
}

interface IssueDetailsModalProps {
  issue: Issue;
  onClose: () => void;
  onStatusChange?: (issueId: string, newStatus: IssueStatus) => void | Promise<void>;
}

function IssueDetailsModal({ issue, onClose, onStatusChange }: IssueDetailsModalProps) {
  const createdText = formatDateTime(issue.createdAt);
  const updatedText = formatDateTime(issue.updatedAt);
  const resolvedText = formatDateTime(issue.resolvedAt);
  const createdHint = formatRelativeTime(issue.createdAt);
  const updatedHint = formatRelativeTime(issue.updatedAt);
  const resolvedHint = formatRelativeTime(issue.resolvedAt);
  const description = issue.description?.trim();
  const notes = issue.notes?.trim();

  const handleStatusChange = async (event: ChangeEvent<HTMLSelectElement>) => {
    const newStatus = event.target.value as IssueStatus;
    if (newStatus !== issue.status && onStatusChange) {
      await onStatusChange(issue.id, newStatus);
    }
  };

  return (
    <Modal onClose={onClose} labelledBy="issue-details-title">
      <div className="modal-header">
        <div>
          <p className="modal-eyebrow">
            <Hash size={14} />
            {issue.id}
          </p>
          <h2 id="issue-details-title" className="modal-title">
            {issue.title}
          </h2>
        </div>
        <button className="modal-close" type="button" aria-label="Close issue details" onClick={onClose}>
          <X size={18} />
        </button>
      </div>

      <div className="modal-body">
        {onStatusChange && (
          <div className="form-field form-field-full">
            <label htmlFor="issue-status-selector">
              <span>Status (Accessibility Feature)</span>
              <div className="select-wrapper">
                <select
                  id="issue-status-selector"
                  value={issue.status}
                  onChange={handleStatusChange}
                >
                  {VALID_STATUSES.map((status) => (
                    <option key={status} value={status}>
                      {toTitleCase(status.replace(/-/g, ' '))}
                    </option>
                  ))}
                </select>
                <ChevronDown size={16} />
              </div>
            </label>
          </div>
        )}

        <div className="issue-detail-grid">
          <IssueMetaTile label="Status" value={toTitleCase(issue.status.replace(/-/g, ' '))} />
          <IssueMetaTile label="Priority" value={issue.priority} />
          <IssueMetaTile label="Assignee" value={issue.assignee || 'Unassigned'} />
          <IssueMetaTile label="App" value={issue.app} />
          <IssueMetaTile
            label="Created"
            value={createdText}
            hint={createdHint}
            icon={<CalendarClock size={14} />}
          />
          {issue.updatedAt && (
            <IssueMetaTile
              label="Updated"
              value={updatedText}
              hint={updatedHint}
              icon={<CalendarClock size={14} />}
            />
          )}
          {issue.resolvedAt && (
            <IssueMetaTile
              label="Resolved"
              value={resolvedText}
              hint={resolvedHint}
              icon={<CalendarClock size={14} />}
            />
          )}
        </div>

        <section className="issue-detail-section">
          <h3>Description</h3>
          {description ? (
            <MarkdownView content={description} />
          ) : (
            <p className="issue-detail-placeholder">No description provided.</p>
          )}
        </section>

        {notes && (
          <section className="issue-detail-section">
            <h3>Notes</h3>
            <MarkdownView content={notes} />
          </section>
        )}

        {issue.attachments.length > 0 && (
          <section className="issue-detail-section">
            <div className="issue-section-heading">
              <h3>Attachments</h3>
              <span className="issue-section-meta">
                {issue.attachments.length} {issue.attachments.length === 1 ? 'file' : 'files'}
              </span>
            </div>
            <div className="issue-attachments-grid">
              {issue.attachments.map((attachment) => (
                <AttachmentPreview key={attachment.path} attachment={attachment} />
              ))}
            </div>
          </section>
        )}

        {issue.tags.length > 0 && (
          <section className="issue-detail-section">
            <h3>Tags</h3>
            <div className="issue-detail-tags">
              <Tag size={14} />
              {issue.tags.map((tag) => (
                <span key={tag} className="issue-detail-tag">
                  {tag}
                </span>
              ))}
            </div>
          </section>
        )}

        {issue.investigation?.report && (
          <section className="issue-detail-section">
            <h3>Investigation Report</h3>
            <div className="investigation-report">
              {issue.investigation.agent_id && (
                <div className="investigation-meta">
                  <Brain size={14} />
                  <span>Agent: {issue.investigation.agent_id}</span>
                </div>
              )}
              <MarkdownView content={issue.investigation.report} />
            </div>
          </section>
        )}

        {issue.metadata?.extra?.agent_last_error && (
          <section className="issue-detail-section">
            <h3>Agent Execution Error</h3>
            <div className="issue-error-details">
              <div className="issue-error-header">
                <AlertCircle size={16} />
                <span className="issue-error-status">
                  Status: {issue.metadata?.extra?.agent_last_status || 'failed'}
                </span>
                {issue.metadata?.extra?.agent_failure_time && (
                  <span className="issue-error-time">
                    Failed at: {formatDateTime(issue.metadata.extra.agent_failure_time)}
                  </span>
                )}
              </div>
              <pre className="issue-error-content">{issue.metadata.extra.agent_last_error}</pre>
            </div>
          </section>
        )}

        {(issue.reporterName || issue.reporterEmail) && (
          <section className="issue-detail-section">
            <h3>Reporter</h3>
            <div className="issue-detail-inline">
              {issue.reporterName && (
                <span>
                  <User size={14} />
                  {issue.reporterName}
                </span>
              )}
              {issue.reporterEmail && (
                <a href={`mailto:${issue.reporterEmail}`}>
                  <Mail size={14} />
                  {issue.reporterEmail}
                </a>
              )}
            </div>
          </section>
        )}
      </div>
    </Modal>
  );
}

interface MarkdownViewProps {
  content: string;
}

function MarkdownView({ content }: MarkdownViewProps) {
  const rendered = useMemo(() => renderMarkdownToHtml(content), [content]);
  return <div className="markdown-body" dangerouslySetInnerHTML={{ __html: rendered }} />;
}

interface AttachmentPreviewProps {
  attachment: IssueAttachment;
}

function AttachmentPreview({ attachment }: AttachmentPreviewProps) {
  const kind = classifyAttachment(attachment);
  const canPreviewText = kind === 'text' || kind === 'json';
  const [expanded, setExpanded] = useState(kind === 'image');
  const [content, setContent] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const previewRequested = useRef(false);
  const isCollapsible = canPreviewText;

  useEffect(() => {
    setExpanded(kind === 'image');
    setContent(null);
    setErrorMessage(null);
    setLoading(false);
    previewRequested.current = false;
  }, [attachment.path, kind]);

  useEffect(() => {
    if (!canPreviewText || !expanded) {
      return;
    }
    if (content !== null || previewRequested.current) {
      return;
    }

    previewRequested.current = true;
    const controller = new AbortController();
    const { signal } = controller;

    setLoading(true);
    setErrorMessage(null);

    fetch(attachment.url, { signal })
      .then((response) => {
        if (!response.ok) {
          throw new Error(`Preview request failed with status ${response.status}`);
        }
        return response.text();
      })
      .then((data) => {
        if (signal.aborted) {
          return;
        }
        let previewText = data;
        if (kind === 'json') {
          try {
            previewText = JSON.stringify(JSON.parse(data), null, 2);
          } catch (parseError) {
            // Keep raw response if parsing fails
          }
        }
        if (previewText.length > MAX_ATTACHMENT_PREVIEW_CHARS) {
          previewText = `${previewText.slice(0, MAX_ATTACHMENT_PREVIEW_CHARS)}\n…`;
        }
        setContent(previewText);
      })
      .catch((error) => {
        if (signal.aborted) {
          return;
        }
        console.error('[IssueTracker] Failed to fetch attachment preview', error);
        setErrorMessage('Failed to load preview');
      })
      .finally(() => {
        if (signal.aborted) {
          return;
        }
        setLoading(false);
      });

    return () => {
      controller.abort();
      previewRequested.current = false;
    };
  }, [attachment.url, canPreviewText, content, expanded, kind]);

  const displayName = attachment.name || attachment.path.split(/[\\/]+/).pop() || 'Attachment';
  const storageName = attachment.path.split(/[\\/]+/).pop() || displayName;
  const sizeLabel = formatFileSize(attachment.size);
  const typeLabel = attachment.type?.split(';')[0];
  const metadataParts = [typeLabel, sizeLabel].filter(Boolean) as string[];
  const Icon =
    kind === 'image' ? ImageIcon : kind === 'json' ? FileCode : kind === 'text' ? FileText : Paperclip;

  const showPreview = kind === 'image' || (expanded && canPreviewText);
  const categoryLabel = useMemo(() => {
    if (attachment.category) {
      return toTitleCase(attachment.category.replace(/[-_]+/g, ' '));
    }
    switch (kind) {
      case 'image':
        return 'Screenshot';
      case 'json':
        return 'JSON';
      case 'text':
        return 'Log';
      default:
        return 'Attachment';
    }
  }, [attachment.category, kind]);

  const cardStateClass = isCollapsible && !expanded ? ' attachment-card--collapsed' : '';

  const handleToggle = () => {
    if (!isCollapsible) {
      return;
    }
    setExpanded((state) => !state);
  };

  return (
    <article className={`attachment-card${cardStateClass}`}>
      <header className="attachment-card-header">
        <div className="attachment-card-heading">
          <span className="attachment-icon" aria-hidden>
            <Icon size={18} />
          </span>
          <div className="attachment-card-meta">
            <span className="attachment-card-name" title={displayName}>
              {displayName}
            </span>
            {metadataParts.length > 0 && (
              <span className="attachment-card-details">{metadataParts.join(' • ')}</span>
            )}
          </div>
        </div>
        <div className="attachment-header-actions">
          <span className="attachment-category" aria-label="Attachment category">
            {categoryLabel}
          </span>
          {isCollapsible && (
            <button
              type="button"
              className="attachment-toggle"
              onClick={handleToggle}
              aria-expanded={expanded}
            >
              <ChevronDown size={16} className="attachment-toggle-icon" />
              <span>{expanded ? 'Hide details' : 'Show details'}</span>
            </button>
          )}
        </div>
      </header>

      {showPreview && (
        <div className="attachment-preview">
          {kind === 'image' && (
            <img className="attachment-preview-image" src={attachment.url} alt={displayName} />
          )}

          {canPreviewText && expanded && (
            <>
              {loading && (
                <div className="attachment-preview-placeholder">
                  <Loader2 size={16} className="attachment-spinner" />
                  <span>Loading preview…</span>
                </div>
              )}
              {!loading && errorMessage && (
                <div className="attachment-preview-error">{errorMessage}</div>
              )}
              {!loading && !errorMessage && content !== null && (
                <pre className="attachment-preview-text">{content}</pre>
              )}
              {!loading && !errorMessage && content === null && (
                <div className="attachment-preview-placeholder">No preview available.</div>
              )}
            </>
          )}
        </div>
      )}

      <div className="attachment-actions">
        <a
          className="attachment-button"
          href={attachment.url}
          target="_blank"
          rel="noopener noreferrer"
        >
          <ExternalLink size={14} />
          <span>Open</span>
        </a>
        <a className="attachment-button" href={attachment.url} download={storageName}>
          <FileDown size={14} />
          <span>Download</span>
        </a>
      </div>
    </article>
  );
}

interface IssueBoardToolbarProps {
  open: boolean;
  onRequestClose: () => void;
  priorityFilter: PriorityFilterValue;
  onPriorityFilterChange: (value: PriorityFilterValue) => void;
  appFilter: string;
  onAppFilterChange: (value: string) => void;
  searchFilter: string;
  onSearchFilterChange: (value: string) => void;
  appOptions: string[];
  hiddenColumns: IssueStatus[];
  onToggleColumn: (status: IssueStatus) => void;
  onResetColumns: () => void;
}

function IssueBoardToolbar({
  open,
  onRequestClose,
  priorityFilter,
  onPriorityFilterChange,
  appFilter,
  onAppFilterChange,
  searchFilter,
  onSearchFilterChange,
  appOptions,
  hiddenColumns,
  onToggleColumn,
  onResetColumns,
}: IssueBoardToolbarProps) {
  const hiddenSet = useMemo(() => new Set(hiddenColumns), [hiddenColumns]);
  const statusOrder = useMemo(() => Object.keys(issueColumnMeta) as IssueStatus[], []);

  const handleClearFilters = () => {
    onPriorityFilterChange('all');
    onAppFilterChange('all');
    onSearchFilterChange('');
  };

  if (!open) {
    return null;
  }

  return (
    <Modal
      onClose={onRequestClose}
      labelledBy="issue-filters-heading"
      panelClassName="modal-panel--compact"
    >
      <div className="issues-toolbar-dialog">
        <div className="issues-toolbar-popover-header">
          <div className="issues-toolbar-heading">
            <Filter size={16} />
            <span id="issue-filters-heading">Filters & Columns</span>
          </div>
          <button
            type="button"
            className="icon-button icon-button--ghost"
            onClick={onRequestClose}
            aria-label="Close filters"
          >
            <X size={14} />
          </button>
        </div>

        <div className="issues-toolbar-section">
          <div className="issues-toolbar-heading">
            <Filter size={16} />
            <span>Filters</span>
          </div>
          <div className="issues-toolbar-fields">
            <label className="issues-toolbar-field" style={{ flex: '1 1 100%' }}>
              <span>Search</span>
              <input
                type="text"
                value={searchFilter}
                onChange={(event) => onSearchFilterChange(event.target.value)}
                placeholder="Search by title, description, ID, tags..."
              />
            </label>

            <label className="issues-toolbar-field">
              <span>Priority</span>
              <div className="select-wrapper">
                <select
                  value={priorityFilter}
                  onChange={(event) => onPriorityFilterChange(event.target.value as PriorityFilterValue)}
                >
                  <option value="all">All priorities</option>
                  {(['Critical', 'High', 'Medium', 'Low'] as Priority[]).map((priority) => (
                    <option key={priority} value={priority}>
                      {priority}
                    </option>
                  ))}
                </select>
                <ChevronDown size={14} />
              </div>
            </label>

            <label className="issues-toolbar-field">
              <span>Application</span>
              <div className="select-wrapper">
                <select value={appFilter} onChange={(event) => onAppFilterChange(event.target.value)}>
                  <option value="all">All applications</option>
                  {appOptions.map((app) => (
                    <option key={app} value={app}>
                      {app}
                    </option>
                  ))}
                </select>
              </div>
            </label>
          </div>
          <button type="button" className="ghost-button issues-toolbar-clear" onClick={handleClearFilters}>
            Reset filters
          </button>
        </div>

        <div className="issues-toolbar-section">
          <div className="issues-toolbar-heading">
            <Eye size={16} />
            <span>Columns</span>
          </div>
          <div className="column-toggle-row">
            {statusOrder.map((status) => {
              const meta = issueColumnMeta[status];
              const hidden = hiddenSet.has(status);
              return (
                <button
                  key={status}
                  type="button"
                  className={`column-toggle ${hidden ? 'column-toggle--inactive' : ''}`}
                  onClick={() => onToggleColumn(status)}
                  aria-pressed={!hidden}
                >
                  <span>{meta.title}</span>
                  {hidden ? <EyeOff size={14} /> : <Eye size={14} />}
                </button>
              );
            })}
          </div>
          {hiddenColumns.length > 0 && (
            <button type="button" className="link-button" onClick={onResetColumns}>
              Show all columns
            </button>
          )}
        </div>
      </div>
    </Modal>
  );
}

interface AttachmentDraft {
  id: string;
  file: File;
  name: string;
  size: number;
  contentType: string;
  content: string | null;
  encoding: 'base64';
  status: 'pending' | 'ready' | 'error';
  errorMessage?: string;
}

interface IssueMetaTileProps {
  label: string;
  value: string;
  hint?: string;
  icon?: ReactNode;
}

function IssueMetaTile({ label, value, hint, icon }: IssueMetaTileProps) {
  return (
    <div className="issue-detail-tile">
      <span className="issue-detail-label">{label}</span>
      <span className="issue-detail-value">
        {icon && <span className="issue-detail-value-icon">{icon}</span>}
        {value}
      </span>
      {hint && <span className="issue-detail-hint">{hint}</span>}
    </div>
  );
}

interface CreateIssueModalProps {
  onClose: () => void;
  onSubmit: (input: CreateIssueInput) => Promise<void>;
}


function CreateIssueModal({ onClose, onSubmit }: CreateIssueModalProps) {
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [priority, setPriority] = useState<Priority>('Medium');
  const [status, setStatus] = useState<IssueStatus>('open');
  const [appId, setAppId] = useState('web-ui');
  const [tags, setTags] = useState('');
  const [reporterName, setReporterName] = useState('');
  const [reporterEmail, setReporterEmail] = useState('');
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [attachments, setAttachments] = useState<AttachmentDraft[]>([]);
  const [isDragActive, setIsDragActive] = useState(false);
  const fileInputRef = useRef<HTMLInputElement | null>(null);

  const handleFileSelection = useCallback(
    (fileSource: FileList | File[]) => {
      const fileArray = Array.from(fileSource instanceof FileList ? Array.from(fileSource) : fileSource);
      if (fileArray.length === 0) {
        return;
      }
      fileArray.forEach((file) => {
        const duplicate = attachments.some(
          (item) =>
            item.file.name === file.name &&
            item.file.size === file.size &&
            item.file.lastModified === file.lastModified,
        );
        if (duplicate) {
          return;
        }

        const id = `${file.name}-${file.size}-${file.lastModified}-${Math.random().toString(16).slice(2)}`;
        const draft: AttachmentDraft = {
          id,
          file,
          name: file.name,
          size: file.size,
          contentType: file.type || 'application/octet-stream',
          content: null,
          encoding: 'base64',
          status: 'pending',
        };

        setAttachments((prev) => [...prev, draft]);

        const reader = new FileReader();
        reader.onload = () => {
          const result = reader.result;
          if (typeof result !== 'string') {
            setAttachments((prev) =>
              prev.map((item) =>
                item.id === id
                  ? { ...item, status: 'error', errorMessage: 'Unsupported attachment format.' }
                  : item,
              ),
            );
            return;
          }

          const base64 = result.includes(',') ? result.split(',')[1] ?? '' : result;
          setAttachments((prev) =>
            prev.map((item) =>
              item.id === id
                ? { ...item, content: base64, status: 'ready', errorMessage: undefined }
                : item,
            ),
          );
        };
        reader.onerror = () => {
          const message = reader.error?.message ?? 'Failed to read file.';
          setAttachments((prev) =>
            prev.map((item) =>
              item.id === id ? { ...item, status: 'error', errorMessage: message } : item,
            ),
          );
        };
        reader.readAsDataURL(file);
      });
    },
    [attachments],
  );

  const handleFileInputChange = (event: ChangeEvent<HTMLInputElement>) => {
    if (event.target.files && event.target.files.length > 0) {
      handleFileSelection(event.target.files);
      event.target.value = '';
    }
  };

  const handleDropzoneClick = () => {
    fileInputRef.current?.click();
  };

  const handleDropzoneKeyDown = (event: ReactKeyboardEvent<HTMLDivElement>) => {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      fileInputRef.current?.click();
    }
  };

  const handleDragEnter = (event: DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    setIsDragActive(true);
  };

  const handleDragOver = (event: DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    event.dataTransfer.effectAllowed = 'copy';
    event.dataTransfer.dropEffect = 'copy';
    setIsDragActive(true);
  };

  const handleDragLeave = (event: DragEvent<HTMLDivElement>) => {
    const nextTarget = event.relatedTarget as Node | null;
    if (nextTarget && event.currentTarget.contains(nextTarget)) {
      return;
    }
    setIsDragActive(false);
  };

  const handleDrop = (event: DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    setIsDragActive(false);
    const fileList = event.dataTransfer?.files;
    if (fileList && fileList.length > 0) {
      handleFileSelection(fileList);
    }
  };

  const handleRemoveAttachment = (id: string) => {
    setAttachments((prev) => prev.filter((attachment) => attachment.id !== id));
  };

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!title.trim()) {
      setErrorMessage('Title is required.');
      return;
    }

    if (attachments.some((attachment) => attachment.status === 'pending')) {
      setErrorMessage('Please wait for attachments to finish processing.');
      return;
    }

    const preparedAttachments = attachments
      .filter((attachment) => attachment.status === 'ready' && attachment.content)
      .map((attachment) => ({
        name: attachment.name,
        content: attachment.content as string,
        encoding: attachment.encoding,
        contentType: attachment.contentType,
        category: 'attachment',
      }));

    setSubmitting(true);
    setErrorMessage(null);

    try {
      await onSubmit({
        title,
        description,
        priority,
        status,
        appId,
        tags: tags
          .split(',')
          .map((tag) => tag.trim())
          .filter(Boolean),
        reporterName: reporterName.trim() || undefined,
        reporterEmail: reporterEmail.trim() || undefined,
        attachments: preparedAttachments,
      });
      onClose();
    } catch (error) {
      console.error('Failed to create issue', error);
      setErrorMessage(error instanceof Error ? error.message : 'Failed to create issue.');
    } finally {
      setSubmitting(false);
    }
  };

  const attachmentCountLabel = attachments.length === 1 ? 'file' : 'files';
  const hasErroredAttachments = attachments.some((attachment) => attachment.status === 'error');

  return (
    <Modal onClose={onClose} labelledBy="create-issue-title">
      <form className="modal-form" onSubmit={handleSubmit}>
        <div className="modal-header">
          <div>
            <p className="modal-eyebrow">New Issue</p>
            <h2 id="create-issue-title" className="modal-title">
              Create Issue
            </h2>
          </div>
          <button className="modal-close" type="button" aria-label="Close create issue" onClick={onClose}>
            <X size={18} />
          </button>
        </div>

        <div className="modal-body">
          <div className="form-grid">
            <label className="form-field">
              <span>Title</span>
              <input
                type="text"
                value={title}
                onChange={(event) => setTitle(event.target.value)}
                placeholder="Summarize the issue"
                required
              />
            </label>
            <label className="form-field">
              <span>Application</span>
              <input
                type="text"
                value={appId}
                onChange={(event) => setAppId(event.target.value)}
                placeholder="e.g. web-ui"
              />
            </label>
            <label className="form-field">
              <span>Priority</span>
              <select value={priority} onChange={(event) => setPriority(event.target.value as Priority)}>
                {(['Critical', 'High', 'Medium', 'Low'] as Priority[]).map((option) => (
                  <option key={option} value={option}>
                    {option}
                  </option>
                ))}
              </select>
            </label>
            <label className="form-field">
              <span>Status</span>
              <select value={status} onChange={(event) => setStatus(event.target.value as IssueStatus)}>
                {VALID_STATUSES.map((option) => (
                  <option key={option} value={option}>
                    {toTitleCase(option.replace(/-/g, ' '))}
                  </option>
                ))}
              </select>
            </label>
            <label className="form-field">
              <span>Reporter name</span>
              <input
                type="text"
                value={reporterName}
                onChange={(event) => setReporterName(event.target.value)}
                placeholder="Optional"
              />
            </label>
            <label className="form-field">
              <span>Reporter email</span>
              <input
                type="email"
                value={reporterEmail}
                onChange={(event) => setReporterEmail(event.target.value)}
                placeholder="Optional"
              />
            </label>
            <label className="form-field form-field-full">
              <span>Tags</span>
              <input
                type="text"
                value={tags}
                onChange={(event) => setTags(event.target.value)}
                placeholder="Comma separated (e.g. backend, regression)"
              />
            </label>
            <label className="form-field form-field-full">
              <span>Description</span>
              <textarea
                value={description}
                onChange={(event) => setDescription(event.target.value)}
                placeholder="Describe the issue, error messages, reproduction steps…"
                rows={5}
              />
            </label>
          </div>

          <div className="form-section attachments-section">
            <div className="form-section-header">
              <span className="form-section-title">Attachments</span>
              {attachments.length > 0 && (
                <span className="form-section-meta">
                  {attachments.length} {attachmentCountLabel}
                </span>
              )}
            </div>
            <p className="form-section-description">
              Drag and drop logs, screenshots, or other helpful context.
            </p>
            <div
              className={`attachment-dropzone${isDragActive ? ' is-active' : ''}`}
              onDragEnter={handleDragEnter}
              onDragOver={handleDragOver}
              onDragLeave={handleDragLeave}
              onDrop={handleDrop}
              onClick={handleDropzoneClick}
              role="button"
              tabIndex={0}
              onKeyDown={handleDropzoneKeyDown}
            >
              <UploadCloud size={20} />
              <div className="attachment-dropzone-copy">
                <strong>Drag & drop files</strong>
                <span>or click to upload</span>
              </div>
            </div>
            <input
              ref={fileInputRef}
              type="file"
              multiple
              onChange={handleFileInputChange}
              className="hidden-input"
            />

            {attachments.length > 0 && (
              <ul className="attachment-pending-list">
                {attachments.map((attachment) => {
                  const sizeLabel = formatFileSize(attachment.size);
                  return (
                    <li
                      key={attachment.id}
                      className={`attachment-pending-item attachment-pending-item--${attachment.status}`}
                    >
                      <div className="attachment-pending-details">
                        <span className="attachment-pending-icon">
                          <Paperclip size={14} />
                        </span>
                        <div className="attachment-pending-meta">
                          <span className="attachment-file-name">{attachment.name}</span>
                          <span className="attachment-file-subtext">{sizeLabel ?? 'Unknown size'}</span>
                          {attachment.status === 'error' && attachment.errorMessage && (
                            <span className="attachment-file-error">{attachment.errorMessage}</span>
                          )}
                        </div>
                      </div>
                      <div className="attachment-pending-actions">
                        {attachment.status === 'pending' && (
                          <span className="attachment-status-chip attachment-status-chip--pending">
                            <Loader2 size={14} className="attachment-spinner" />
                            Processing…
                          </span>
                        )}
                        {attachment.status === 'error' && (
                          <span className="attachment-status-chip attachment-status-chip--error">
                            <CircleAlert size={14} />
                            Failed
                          </span>
                        )}
                        <button
                          type="button"
                          className="link-button attachment-remove"
                          onClick={() => handleRemoveAttachment(attachment.id)}
                          aria-label={`Remove ${attachment.name}`}
                        >
                          Remove
                        </button>
                      </div>
                    </li>
                  );
                })}
              </ul>
            )}

            {hasErroredAttachments && (
              <p className="attachment-warning">
                Files marked as failed will be skipped. Remove them and try again.
              </p>
            )}
          </div>
        </div>

        <div className="modal-footer">
          {errorMessage && <span className="form-error">{errorMessage}</span>}
          <div className="modal-actions">
            <button className="ghost-button" type="button" onClick={onClose} disabled={submitting}>
              Cancel
            </button>
            <button className="primary-button" type="submit" disabled={submitting}>
              {submitting ? 'Creating…' : 'Create Issue'}
            </button>
          </div>
        </div>
      </form>
    </Modal>
  );
}

interface MetricsDialogProps {
  stats: DashboardStats;
  issues: Issue[];
  processor: ProcessorSettings;
  agentSettings: AgentSettings;
  onClose: () => void;
  onOpenIssue: (issueId: string) => void;
}

function MetricsDialog({
  stats,
  issues,
  processor,
  agentSettings,
  onClose,
  onOpenIssue,
}: MetricsDialogProps) {
  return (
    <Modal onClose={onClose} labelledBy="metrics-dialog-title" panelClassName="modal-panel--wide">
      <div className="modal-header">
        <div>
          <p className="modal-eyebrow">Dashboard</p>
          <h2 id="metrics-dialog-title" className="modal-title">
            Metrics
          </h2>
        </div>
        <button className="modal-close" type="button" aria-label="Close metrics" onClick={onClose}>
          <X size={18} />
        </button>
      </div>

      <div className="modal-body modal-body--dashboard">
        <Dashboard
          stats={stats}
          issues={issues}
          processor={processor}
          agentSettings={agentSettings}
          onOpenIssues={onClose}
          onOpenIssue={onOpenIssue}
          onOpenAutomationSettings={onClose}
        />
      </div>
    </Modal>
  );
}

interface SettingsDialogProps {
  apiBaseUrl: string;
  processor: ProcessorSettings;
  agent: AgentSettings;
  display: DisplaySettings;
  onProcessorChange: (settings: ProcessorSettings) => void;
  onAgentChange: (settings: AgentSettings) => void;
  onDisplayChange: (settings: DisplaySettings) => void;
  onClose: () => void;
  issuesProcessed?: number;
  issuesRemaining?: number | string;
}

function SettingsDialog({
  apiBaseUrl,
  processor,
  agent,
  display,
  onProcessorChange,
  onAgentChange,
  onDisplayChange,
  onClose,
  issuesProcessed,
  issuesRemaining,
}: SettingsDialogProps) {
  return (
    <Modal onClose={onClose} labelledBy="settings-dialog-title" panelClassName="modal-panel--wide">
      <div className="modal-header">
        <div>
          <p className="modal-eyebrow">Configuration</p>
          <h2 id="settings-dialog-title" className="modal-title">
            Settings
          </h2>
        </div>
        <button className="modal-close" type="button" aria-label="Close settings" onClick={onClose}>
          <X size={18} />
        </button>
      </div>

      <div className="modal-body modal-body--settings">
        <SettingsPage
          apiBaseUrl={apiBaseUrl}
          processor={processor}
          agent={agent}
          display={display}
          onProcessorChange={onProcessorChange}
          onAgentChange={onAgentChange}
          onDisplayChange={onDisplayChange}
          issuesProcessed={issuesProcessed}
          issuesRemaining={issuesRemaining}
        />
      </div>
    </Modal>
  );
}

export default App;
