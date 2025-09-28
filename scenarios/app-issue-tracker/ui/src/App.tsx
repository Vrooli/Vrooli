import type { FormEvent, MouseEvent as ReactMouseEvent, ReactNode } from 'react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  CalendarClock,
  CircleAlert,
  ChevronDown,
  ExternalLink,
  FileCode,
  FileDown,
  FileText,
  Hash,
  Image as ImageIcon,
  KanbanSquare,
  LayoutDashboard,
  Loader2,
  Mail,
  Paperclip,
  Tag,
  User,
  X,
} from 'lucide-react';
import { Sidebar } from './components/Sidebar';
import { Header } from './components/Header';
import { Dashboard } from './pages/Dashboard';
import { IssuesBoard } from './pages/IssuesBoard';
import { SettingsPage } from './pages/Settings';
import {
  AppSettings,
  DashboardStats,
  ActiveAgentOption,
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
const VALID_STATUSES: IssueStatus[] = ['open', 'investigating', 'in-progress', 'fixed', 'closed', 'failed'];

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
  fixedToday?: number;
}

interface ApiAgent {
  id: string;
  name?: string;
  display_name?: string;
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
  const codePlaceholderPrefix = '__CODESEG__';

  let output = escapeHtml(source);

  output = output.replace(/`([^`]+)`/g, (_match, code: string) => {
    const placeholder = `${codePlaceholderPrefix}${codeSegments.length}__`;
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

  output = output.replace(/__CODESEG__(\d+)__/g, (_match, index: string) => {
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
  fixedToday: 0,
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

  const openStatuses: IssueStatus[] = ['open', 'investigating'];
  const todayKey = new Date().toISOString().slice(0, 10);

  const totalsFromIssues = {
    total: issues.length,
    open: issues.filter((issue) => openStatuses.includes(issue.status)).length,
    inProgress: issues.filter((issue) => issue.status === 'in-progress').length,
    fixedToday: issues.filter(
      (issue) => issue.status === 'fixed' && getDateKey(issue.resolvedAt) === todayKey,
    ).length,
  };

  return {
    totalIssues: apiStats?.totalIssues ?? totalsFromIssues.total,
    openIssues: apiStats?.openIssues ?? totalsFromIssues.open,
    inProgress: apiStats?.inProgress ?? totalsFromIssues.inProgress,
    fixedToday: apiStats?.fixedToday ?? totalsFromIssues.fixedToday,
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

const navItems = [
  {
    key: 'dashboard' as NavKey,
    label: 'Dashboard',
    icon: LayoutDashboard,
  },
  {
    key: 'issues' as NavKey,
    label: 'Issues',
    icon: KanbanSquare,
  },
  {
    key: 'settings' as NavKey,
    label: 'Settings',
    icon: CircleAlert,
  },
];

function normalizePathname(pathname: string): string {
  if (pathname === '/') {
    return '/';
  }
  const trimmed = pathname.replace(/\/+$/, '');
  return trimmed.length > 0 ? trimmed : '/';
}

function pathToNavKey(pathname: string): NavKey {
  const normalized = normalizePathname(pathname);
  switch (normalized) {
    case '/issues':
      return 'issues';
    case '/settings':
      return 'settings';
    case '/':
    default:
      return 'dashboard';
  }
}

function navKeyToPath(nav: NavKey): string {
  switch (nav) {
    case 'issues':
      return '/issues';
    case 'settings':
      return '/settings';
    case 'dashboard':
    default:
      return '/';
  }
}

function getInitialNavKey(): NavKey {
  if (typeof window === 'undefined') {
    return 'dashboard';
  }
  return pathToNavKey(window.location.pathname);
}

function App() {
  const isMountedRef = useRef(true);
  const [activeNav, setActiveNav] = useState<NavKey>(() => getInitialNavKey());
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [processorSettings, setProcessorSettings] = useState<ProcessorSettings>(SampleData.processor);
  const [agentSettings, setAgentSettings] = useState(AppSettings.agent);
  const [displaySettings, setDisplaySettings] = useState(AppSettings.display);
  const [rateLimits, setRateLimits] = useState(AppSettings.rateLimits);
  const [allowedAgents, setAllowedAgents] = useState<ActiveAgentOption[]>([]);
  const [selectedAgentId, setSelectedAgentId] = useState<string>(SampleData.activeAgents[0]?.id ?? '');
  const [issues, setIssues] = useState<Issue[]>([]);
  const [createIssueOpen, setCreateIssueOpen] = useState(false);
  const [issueDetailOpen, setIssueDetailOpen] = useState(false);
  const [dashboardStats, setDashboardStats] = useState<DashboardStats>(DEFAULT_STATS);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [focusedIssueId, setFocusedIssueId] = useState<string | null>(() => getIssueIdFromLocation());

  const selectedIssue = useMemo(() => {
    return issues.find((issue) => issue.id === focusedIssueId) ?? null;
  }, [issues, focusedIssueId]);

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
      setActiveNav(pathToNavKey(window.location.pathname));
    };

    window.addEventListener('popstate', handlePopState);
    return () => {
      window.removeEventListener('popstate', handlePopState);
    };
  }, []);

  useEffect(() => {
    if (typeof window === 'undefined') {
      return;
    }
    setActiveNav(pathToNavKey(window.location.pathname));
  }, []);

  const fetchAllData = useCallback(async () => {
    if (isMountedRef.current) {
      setLoading(true);
      setLoadError(null);
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
            fixedToday: typeof stats.fixed_today === 'number' ? stats.fixed_today : undefined,
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
      const response = await fetch(buildApiUrl('/agents'));
      if (response.ok) {
        const payload = await response.json();
        const agents: ApiAgent[] = Array.isArray(payload?.data?.agents) ? payload.data.agents : [];
        if (isMountedRef.current) {
          if (agents.length > 0) {
            const options = agents.map((agent) => ({
              id: agent.id,
              label: agent.display_name ?? agent.name ?? agent.id,
            }));
            setAllowedAgents(options);
          } else {
            setAllowedAgents([]);
          }
        }
      }
    } catch (error) {
      console.warn('Failed to load agents', error);
    }

    if (isMountedRef.current) {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const activeOptions = allowedAgents.length > 0 ? allowedAgents : SampleData.activeAgents;
    setSelectedAgentId((prev) => {
      if (prev && activeOptions.some((option) => option.id === prev)) {
        return prev;
      }
      return activeOptions[0]?.id ?? '';
    });
  }, [allowedAgents]);

  useEffect(() => {
    void fetchAllData();
  }, [fetchAllData]);

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

  useEffect(() => {
    if (!focusedIssueId) {
      return;
    }

    if (activeNav !== 'issues') {
      setActiveNav('issues');
    }

    if (typeof window === 'undefined') {
      return;
    }

    const normalizedPath = normalizePathname(window.location.pathname);
    if (normalizedPath !== '/issues') {
      const url = new URL(window.location.href);
      url.pathname = '/issues';
      const nextSearch = url.searchParams.toString();
      const nextUrl = `${url.pathname}${nextSearch ? `?${nextSearch}` : ''}${url.hash}`;
      window.history.replaceState({}, '', nextUrl);
    }
  }, [focusedIssueId, activeNav]);

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

  const handleCreateIssue = useCallback(() => {
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
      setActiveNav('issues');
      if (newIssueId) {
        setFocusedIssueId(newIssueId);
        setIssueDetailOpen(true);
      }
    },
    [createIssue],
  );

  const handleIssueDetailClose = useCallback(() => {
    setIssueDetailOpen(false);
    setFocusedIssueId(null);
  }, []);

  const handleNavSelect = useCallback(
    (nextNav: NavKey) => {
      setActiveNav(nextNav);
      if (nextNav !== 'issues') {
        setFocusedIssueId(null);
        setIssueDetailOpen(false);
      }

      if (typeof window === 'undefined') {
        return;
      }

      const url = new URL(window.location.href);
      const targetPath = navKeyToPath(nextNav);
      const normalizedCurrent = normalizePathname(url.pathname);
      const normalizedTarget = normalizePathname(targetPath);

      if (normalizedCurrent !== normalizedTarget) {
        url.pathname = targetPath;
      }

      if (nextNav !== 'issues') {
        url.searchParams.delete('issue');
      }

      const nextSearch = url.searchParams.toString();
      const nextUrl = `${normalizePathname(url.pathname)}${nextSearch ? `?${nextSearch}` : ''}${url.hash}`;
      const currentUrl = `${normalizePathname(window.location.pathname)}${window.location.search}${window.location.hash}`;

      if (nextUrl !== currentUrl) {
        window.history.pushState({}, '', nextUrl);
      }
    },
    [],
  );

  const handleToggleActive = () => {
    setProcessorSettings((prev) => ({
      ...prev,
      active: !prev.active,
    }));
  };

  const processorSetter = (settings: ProcessorSettings) => {
    setProcessorSettings(settings);
  };

  const renderedPage = useMemo(() => {
    switch (activeNav) {
      case 'dashboard':
        return (
          <Dashboard
            stats={dashboardStats}
            issues={issues}
            processor={processorSettings}
            agentSettings={agentSettings}
          />
        );
      case 'issues':
        return (
          <IssuesBoard
            issues={issues}
            focusedIssueId={focusedIssueId}
            onIssueSelect={handleIssueSelect}
          />
        );
      case 'settings':
        return (
          <SettingsPage
            processor={processorSettings}
            agent={agentSettings}
            display={displaySettings}
            rateLimits={rateLimits}
            activeAgents={allowedAgents.length > 0 ? allowedAgents : SampleData.activeAgents}
            onProcessorChange={processorSetter}
            onAgentChange={setAgentSettings}
            onDisplayChange={setDisplaySettings}
            onRateLimitChange={setRateLimits}
            onAgentsUpdate={setAllowedAgents}
          />
        );
      default:
        return null;
    }
  }, [
    activeNav,
    dashboardStats,
    issues,
    processorSettings,
    agentSettings,
    displaySettings,
    rateLimits,
    allowedAgents,
    focusedIssueId,
    handleIssueSelect,
  ]);

  const activeAgentOptions = allowedAgents.length > 0 ? allowedAgents : SampleData.activeAgents;
  const activeAgentId =
    selectedAgentId && activeAgentOptions.some((option) => option.id === selectedAgentId)
      ? selectedAgentId
      : activeAgentOptions[0]?.id ?? '';

  const showIssueDetailModal = issueDetailOpen && selectedIssue;

  return (
    <>
      <div className={`app-shell ${displaySettings.theme}`}>
        <Sidebar
          collapsed={sidebarCollapsed}
          items={navItems}
          activeItem={activeNav}
          onSelect={handleNavSelect}
          onToggle={() => setSidebarCollapsed((state) => !state)}
        />
        <div className="main-panel">
          <Header
            processor={processorSettings}
            agents={activeAgentOptions}
            selectedAgentId={activeAgentId}
            onToggleActive={handleToggleActive}
            onCreateIssue={handleCreateIssue}
            onSelectAgent={setSelectedAgentId}
          />
          <main className="page-container">
            {loading && <div className="data-banner loading">Loading latest data…</div>}
            {loadError && (
              <div className="data-banner error">
                <span>{loadError}</span>
                <button type="button" onClick={() => void fetchAllData()}>
                  Retry
                </button>
              </div>
            )}
            {renderedPage}
          </main>
        </div>
      </div>

      {createIssueOpen && (
        <CreateIssueModal onClose={handleCreateIssueClose} onSubmit={handleSubmitNewIssue} />
      )}

      {showIssueDetailModal && selectedIssue && (
        <IssueDetailsModal issue={selectedIssue} onClose={handleIssueDetailClose} />
      )}
    </>
  );
}

interface ModalProps {
  onClose: () => void;
  labelledBy?: string;
  children: ReactNode;
}

function Modal({ onClose, labelledBy, children }: ModalProps) {
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

  return (
    <div className="modal-backdrop" onMouseDown={handleBackdropMouseDown}>
      <div className="modal-panel" role="dialog" aria-modal="true" aria-labelledby={labelledBy}>
        {children}
      </div>
    </div>
  );
}

interface IssueDetailsModalProps {
  issue: Issue;
  onClose: () => void;
}

function IssueDetailsModal({ issue, onClose }: IssueDetailsModalProps) {
  const createdText = formatDateTime(issue.createdAt);
  const updatedText = formatDateTime(issue.updatedAt);
  const resolvedText = formatDateTime(issue.resolvedAt);
  const createdHint = formatRelativeTime(issue.createdAt);
  const updatedHint = formatRelativeTime(issue.updatedAt);
  const resolvedHint = formatRelativeTime(issue.resolvedAt);
  const description = issue.description?.trim();
  const notes = issue.notes?.trim();

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

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!title.trim()) {
      setErrorMessage('Title is required.');
      return;
    }

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
      });
      onClose();
    } catch (error) {
      console.error('Failed to create issue', error);
      setErrorMessage(error instanceof Error ? error.message : 'Failed to create issue.');
    } finally {
      setSubmitting(false);
    }
  };

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

export default App;
