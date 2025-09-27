import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { CircleAlert, KanbanSquare, LayoutDashboard } from 'lucide-react';
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
  IssueStatus,
  NavKey,
  Priority,
  ProcessorSettings,
  SampleData,
} from './data/sampleData';
import './styles/app.css';

const API_BASE_INPUT = (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? '/api';
const API_BASE_URL = API_BASE_INPUT.endsWith('/') ? API_BASE_INPUT.slice(0, -1) : API_BASE_INPUT;
const ISSUE_FETCH_LIMIT = 200;
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

function buildApiUrl(path: string): string {
  return `${API_BASE_URL}${path}`;
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
  const tags = Array.isArray(raw.metadata?.tags)
    ? raw.metadata?.tags.filter((tag) => Boolean(tag)).map((tag) => String(tag))
    : [];
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
    assignee,
    priority: normalizePriority(raw.priority),
    createdAt,
    status: normalizeStatus(raw.status),
    app: labels.app ?? raw.app_id ?? 'unknown',
    tags,
    resolvedAt,
  };
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

function App() {
  const isMountedRef = useRef(true);
  const [activeNav, setActiveNav] = useState<NavKey>('dashboard');
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [processorSettings, setProcessorSettings] = useState<ProcessorSettings>(SampleData.processor);
  const [agentSettings, setAgentSettings] = useState(AppSettings.agent);
  const [displaySettings, setDisplaySettings] = useState(AppSettings.display);
  const [rateLimits, setRateLimits] = useState(AppSettings.rateLimits);
  const [allowedAgents, setAllowedAgents] = useState<ActiveAgentOption[]>([]);
  const [issues, setIssues] = useState<Issue[]>([]);
  const [dashboardStats, setDashboardStats] = useState<DashboardStats>(DEFAULT_STATS);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);

  useEffect(() => {
    return () => {
      isMountedRef.current = false;
    };
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
    void fetchAllData();
  }, [fetchAllData]);

  const createIssue = useCallback(async () => {
    if (loading) {
      return;
    }

    try {
      const now = new Date();
      const response = await fetch(buildApiUrl('/issues'), {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          title: `UI created issue at ${now.toLocaleTimeString()}`,
          description: 'Issue created from the web UI quick action.',
          type: 'bug',
          priority: 'medium',
          app_id: 'web-ui',
          reporter_name: 'UI Operator',
          reporter_email: 'ui-operator@local.test',
        }),
      });

      if (!response.ok) {
        throw new Error(`Issue creation failed with status ${response.status}`);
      }

      await fetchAllData();
    } catch (error) {
      console.error('Failed to create issue', error);
      if (isMountedRef.current) {
        setLoadError('Failed to create a new issue via the API.');
      }
    }
  }, [fetchAllData, loading]);

  const handleCreateIssue = useCallback(() => {
    void createIssue();
  }, [createIssue]);

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
        return <IssuesBoard issues={issues} />;
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
  ]);

  return (
    <div className={`app-shell ${displaySettings.theme}`}>
      <Sidebar
        collapsed={sidebarCollapsed}
        items={navItems}
        activeItem={activeNav}
        onSelect={setActiveNav}
        onToggle={() => setSidebarCollapsed((state) => !state)}
      />
      <div className="main-panel">
        <Header
          processor={processorSettings}
          agents={allowedAgents.length > 0 ? allowedAgents : SampleData.activeAgents}
          onToggleActive={handleToggleActive}
          onCreateIssue={handleCreateIssue}
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
  );
}

export default App;
