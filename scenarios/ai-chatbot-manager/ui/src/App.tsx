import { useEffect, useMemo, useState } from 'react';
import {
  BrowserRouter as Router,
  Routes,
  Route,
  Link,
  useLocation,
  useNavigate,
} from 'react-router-dom';
import type { LucideIcon } from 'lucide-react';
import {
  Activity,
  AlertTriangle,
  Bot,
  LayoutDashboard,
  Loader2,
  MessageSquare,
  PanelLeftOpen,
  PanelLeftClose,
  PlusCircle,
  Sparkles,
} from 'lucide-react';

import Dashboard from './components/Dashboard';
import ChatbotList from './components/ChatbotList';
import ChatbotEditor from './components/ChatbotEditor';
import Analytics from './components/Analytics';
import TestChat from './components/TestChat';
import apiClient from './utils/api';
import type { ApiConnectionState, ApiHealthResponse } from './types';
import './App.css';

const NAV_ITEMS: NavItem[] = [
  {
    path: '/',
    label: 'Overview',
    icon: LayoutDashboard,
  },
  {
    path: '/chatbots',
    label: 'Chatbots',
    icon: Bot,
  },
  {
    path: '/chatbots/new',
    label: 'Create Chatbot',
    icon: PlusCircle,
  },
];

interface NavItem {
  path: string;
  label: string;
  icon: LucideIcon;
}

const App = () => (
  <Router>
    <AppShell />
  </Router>
);

function AppShell() {
  const location = useLocation();
  const navigate = useNavigate();
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [apiState, setApiState] = useState<ApiConnectionState>({
    status: 'checking',
    latencyMs: null,
  });

  useEffect(() => {
    let isMounted = true;

    const checkApiHealth = async () => {
      try {
        const startedAt = performance.now();
        const response = await apiClient.get('/health');
        const latencyMs = Math.round(performance.now() - startedAt);

        if (!response.ok) {
          const errorMessage = `API responded with status ${response.status}`;
          if (isMounted) {
            setApiState({ status: 'error', latencyMs, message: errorMessage, lastChecked: new Date().toISOString() });
          }
          return;
        }

        const payload = (await response.json()) as ApiHealthResponse;
        const normalizedStatus: ApiConnectionState['status'] =
          payload.status === 'healthy' ? 'healthy' : payload.status === 'degraded' ? 'degraded' : 'error';

        const message =
          normalizedStatus === 'healthy'
            ? 'API operational'
            : payload.status === 'degraded'
            ? 'API reporting degraded dependencies'
            : 'API reported an unhealthy state';

        if (isMounted) {
          setApiState({
            status: normalizedStatus,
            latencyMs,
            message,
            lastChecked: payload.timestamp,
          });
        }
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Failed to reach API';
        if (isMounted) {
          setApiState({ status: 'error', latencyMs: null, message, lastChecked: new Date().toISOString() });
        }
      }
    };

    checkApiHealth();
    const interval = window.setInterval(checkApiHealth, 30_000);

    return () => {
      isMounted = false;
      window.clearInterval(interval);
    };
  }, []);

  const routeTitle = useMemo(() => deriveRouteTitle(location.pathname), [location.pathname]);

  if (apiState.status === 'checking') {
    return (
      <FullScreenState
        title="Connecting to AI Chatbot Manager"
        subtitle="Verifying API availability and loading workspace"
        icon={Loader2}
        iconClassName="spin"
      />
    );
  }

  if (apiState.status === 'error') {
    return (
      <FullScreenState
        title="Unable to reach the API"
        subtitle={apiState.message || 'The AI Chatbot Manager API is not responding.'}
        icon={AlertTriangle}
        actionLabel="Retry connection"
        onAction={() => window.location.reload()}
      />
    );
  }

  return (
    <div className="app-shell">
      <Sidebar
        collapsed={sidebarCollapsed}
        currentPath={location.pathname}
        onNavigate={(path) => navigate(path)}
        onToggle={() => setSidebarCollapsed((prev) => !prev)}
        apiState={apiState}
      />
      <div className="main-panel">
        <header className="app-topbar">
          <button
            className="sidebar-toggle"
            onClick={() => setSidebarCollapsed((prev) => !prev)}
            aria-label={sidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          >
            {sidebarCollapsed ? <PanelLeftOpen size={18} /> : <PanelLeftClose size={18} />}
          </button>
          <div className="topbar-heading">
            <h1>{routeTitle.title}</h1>
            <p>{routeTitle.subtitle}</p>
          </div>
          <div className="topbar-actions">
            <ApiStatusIndicator state={apiState} />
            <button className="primary-action" onClick={() => navigate('/chatbots/new')}>
              <Sparkles size={16} />
              <span>Launch New Chatbot</span>
            </button>
          </div>
        </header>
        <main className="app-content">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/chatbots" element={<ChatbotList />} />
            <Route path="/chatbots/new" element={<ChatbotEditor />} />
            <Route path="/chatbots/:id/edit" element={<ChatbotEditor />} />
            <Route path="/chatbots/:id/analytics" element={<Analytics />} />
            <Route path="/chatbots/:id/test" element={<TestChat />} />
          </Routes>
        </main>
      </div>
    </div>
  );
}

interface SidebarProps {
  collapsed: boolean;
  currentPath: string;
  onNavigate: (path: string) => void;
  onToggle: () => void;
  apiState: ApiConnectionState;
}

function Sidebar({ collapsed, currentPath, onNavigate, onToggle, apiState }: SidebarProps) {
  return (
    <aside className={`sidebar ${collapsed ? 'is-collapsed' : ''}`}>
      <div className="sidebar-brand" onClick={!collapsed ? undefined : onToggle}>
        <div className="brand-icon">
          <Bot size={22} />
        </div>
        {!collapsed && (
          <div className="brand-copy">
            <span className="brand-title">AI Chatbot Manager</span>
            <span className="brand-subtitle">Conversation intelligence at scale</span>
          </div>
        )}
      </div>
      <nav className="sidebar-nav">
        {NAV_ITEMS.map((item) => {
          const isActive = item.path === '/'
            ? currentPath === item.path
            : currentPath.startsWith(item.path);

          return (
            <button
              key={item.path}
              className={`nav-link ${isActive ? 'is-active' : ''}`}
              onClick={() => onNavigate(item.path)}
              title={collapsed ? item.label : undefined}
            >
              <item.icon size={18} />
              {!collapsed && <span>{item.label}</span>}
            </button>
          );
        })}
      </nav>
      <div className="sidebar-footer">
        <ApiStatusIndicator state={apiState} collapsed={collapsed} />
        <div className="sidebar-meta">
          <MessageSquare size={14} />
          {!collapsed && <span>Real-time chat orchestration</span>}
        </div>
      </div>
      <button className="sidebar-collapse" onClick={onToggle} aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}>
        {collapsed ? <PanelLeftOpen size={16} /> : <PanelLeftClose size={16} />}
      </button>
    </aside>
  );
}

interface ApiStatusIndicatorProps {
  state: ApiConnectionState;
  collapsed?: boolean;
}

function ApiStatusIndicator({ state, collapsed }: ApiStatusIndicatorProps) {
  const meta = useMemo(() => getStatusPresentation(state), [state]);

  return (
    <div className={`api-status ${meta.variant} ${collapsed ? 'is-collapsed' : ''}`} title={collapsed ? meta.description : undefined}>
      <span className="api-status-dot" />
      {!collapsed && (
        <div className="api-status-copy">
          <span className="api-status-label">API {meta.label}</span>
          <span className="api-status-description">{meta.description}</span>
        </div>
      )}
      {state.latencyMs !== null && !collapsed && (
        <span className="api-status-latency">{state.latencyMs} ms</span>
      )}
    </div>
  );
}

function getStatusPresentation(state: ApiConnectionState) {
  switch (state.status) {
    case 'healthy':
      return {
        label: 'Connected',
        description: state.message ?? 'Operational and responsive',
        variant: 'is-healthy',
      };
    case 'degraded':
      return {
        label: 'Degraded',
        description: state.message ?? 'Some dependencies require attention',
        variant: 'is-degraded',
      };
    default:
      return {
        label: 'Unknown',
        description: state.message ?? 'Status unavailable',
        variant: 'is-unknown',
      };
  }
}

interface FullScreenStateProps {
  title: string;
  subtitle?: string;
  icon: LucideIcon;
  iconClassName?: string;
  actionLabel?: string;
  onAction?: () => void;
}

function FullScreenState({ title, subtitle, icon: Icon, iconClassName, actionLabel, onAction }: FullScreenStateProps) {
  return (
    <div className="fullscreen-state">
      <div className="state-card">
        <div className={`state-icon ${iconClassName ?? ''}`}>
          <Icon size={32} />
        </div>
        <h2>{title}</h2>
        {subtitle && <p>{subtitle}</p>}
        {actionLabel && onAction && (
          <button className="primary-action" onClick={onAction}>
            <Activity size={16} />
            <span>{actionLabel}</span>
          </button>
        )}
      </div>
    </div>
  );
}

function deriveRouteTitle(pathname: string) {
  if (pathname === '/') {
    return {
      title: 'Executive Overview',
      subtitle: 'Monitor performance across every customer touchpoint.',
    };
  }

  if (pathname.startsWith('/chatbots/new')) {
    return {
      title: 'Create New Chatbot',
      subtitle: 'Define tone, knowledge, and launch-ready configuration.',
    };
  }

  if (pathname.match(/\/chatbots\/.+\/edit/)) {
    return {
      title: 'Refine Chatbot Experience',
      subtitle: 'Iterate on copy, escalation flows, and handoffs in minutes.',
    };
  }

  if (pathname.match(/\/chatbots\/.+\/analytics/)) {
    return {
      title: 'Conversion Analytics',
      subtitle: 'Understand engagement, intent, and revenue impact.',
    };
  }

  if (pathname.match(/\/chatbots\/.+\/test/)) {
    return {
      title: 'Live Conversation Sandbox',
      subtitle: 'Stress-test prompts and escalate flows before going live.',
    };
  }

  if (pathname.startsWith('/chatbots')) {
    return {
      title: 'Chatbot Portfolio',
      subtitle: 'Manage deployments, compliance, and performance SLAs.',
    };
  }

  return {
    title: 'AI Chatbot Manager',
    subtitle: 'Design, launch, and scale high-impact AI concierge experiences.',
  };
}

export default App;
