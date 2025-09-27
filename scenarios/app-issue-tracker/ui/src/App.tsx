import { useMemo, useState } from 'react';
import { CircleAlert, KanbanSquare, LayoutDashboard } from 'lucide-react';
import { Sidebar } from './components/Sidebar';
import { Header } from './components/Header';
import { Dashboard } from './pages/Dashboard';
import { IssuesBoard } from './pages/IssuesBoard';
import { SettingsPage } from './pages/Settings';
import { AppSettings, Issue, NavKey, ProcessorSettings, SampleData } from './data/sampleData';
import './styles/app.css';

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
  const [activeNav, setActiveNav] = useState<NavKey>('dashboard');
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [processorSettings, setProcessorSettings] = useState<ProcessorSettings>(SampleData.processor);
  const [agentSettings, setAgentSettings] = useState(AppSettings.agent);
  const [displaySettings, setDisplaySettings] = useState(AppSettings.display);
  const [rateLimits, setRateLimits] = useState(AppSettings.rateLimits);
  const [allowedAgents, setAllowedAgents] = useState(SampleData.activeAgents);
  const [issues, setIssues] = useState<Issue[]>(SampleData.issues);

  const dashboardStats = useMemo(() => SampleData.stats, []);

  const handleToggleActive = () => {
    setProcessorSettings((prev) => ({
      ...prev,
      active: !prev.active,
    }));
  };

  const handleCreateIssue = () => {
    const newIssue: Issue = {
      id: `ISSUE-${issues.length + 101}`,
      title: 'New frontend regression',
      assignee: 'Unassigned',
      priority: 'High',
      app: 'vrooli-core',
      createdAt: new Date().toISOString(),
      status: 'pending',
      summary: 'Placeholder issue created from UI. Replace with real workflow integration.',
      tags: ['frontend', 'regression'],
    };
    setIssues((prev) => [newIssue, ...prev]);
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
            activeAgents={allowedAgents}
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
          agents={allowedAgents}
          onToggleActive={handleToggleActive}
          onCreateIssue={handleCreateIssue}
        />
        <main className="page-container">{renderedPage}</main>
      </div>
    </div>
  );
}

export default App;
