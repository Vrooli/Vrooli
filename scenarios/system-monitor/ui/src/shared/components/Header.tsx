import { Moon, Pause, Play, Settings, Sun, Terminal } from 'lucide-react';
import { NavLink, useLocation, useNavigate } from 'react-router-dom';
import { StatusIndicator } from './StatusIndicator';
import { AgentDropdown } from './AgentDropdown';
import { useTheme } from '../theme/ThemeProvider';
import type { InvestigationAgentState } from '../../types';
import type { SystemHealthStatus } from '../../features/monitoring/hooks/useSystemMonitor';
import { TIME_RANGE_OPTIONS, useTimeRange } from '../time/TimeRangeContext';

const NAV_ITEMS = [
  { path: '/', label: 'Dashboard' },
  { path: '/forensics', label: 'Forensics' },
  { path: '/logs', label: 'Logs' },
  { path: '/capacity', label: 'Capacity' },
  { path: '/scripts', label: 'Scripts' },
] as const;

interface HeaderProps {
  unreadErrorCount: number;
  agents: InvestigationAgentState[];
  onStopAgent: (agentId: string) => Promise<void>;
  stoppingAgentIds: ReadonlySet<string>;
  agentErrors: Record<string, string>;
  onRefreshAgents?: () => void;
  onToggleTerminal: () => void;
  onOpenSettings: () => void;
  healthStatus: SystemHealthStatus | null;
  healthError: string | null;
  onToggleMonitoring: () => Promise<void>;
  onRefreshHealth: () => Promise<void>;
  isLoadingHealth: boolean;
}

export const Header = ({
  unreadErrorCount,
  agents,
  onStopAgent,
  stoppingAgentIds,
  agentErrors,
  onRefreshAgents,
  onToggleTerminal,
  onOpenSettings,
  healthStatus,
  healthError,
  onToggleMonitoring,
  onRefreshHealth,
  isLoadingHealth
}: HeaderProps) => {
  const { theme, toggleTheme } = useTheme();
  const { range, setRange, paused, setPaused } = useTimeRange();
  const location = useLocation();
  const navigate = useNavigate();
  const currentPath = location.pathname === '/' ? '/' : `/${location.pathname.split('/')[1]}`;

  return (
    <header className="app-header">
      <div className="app-header-inner">
        <h1
          className="system-monitor-title icon-text"
          data-sm-style="sm-style-a02ac2f8c1"
        >
          <span className="system-monitor-title-text">System Monitor</span>
        </h1>

        <nav className="app-nav flex-row-center gap-sm" aria-label="Primary navigation" data-sm-style="sm-style-c938f99d82">
          {NAV_ITEMS.map(item => (
            <NavLink key={item.path} to={item.path} end={item.path === '/'} className="app-nav-link">
              {item.label}
            </NavLink>
          ))}
        </nav>

        <label className="app-mobile-nav">
          <span className="sr-only">Navigate to</span>
          <select aria-label="Navigate to" value={currentPath} onChange={event => { void navigate(event.target.value); }}>
            {NAV_ITEMS.map(item => <option key={item.path} value={item.path}>{item.label}</option>)}
          </select>
        </label>

        <div className="flex-row-center">
          {/* Instrument-scope: what this app is currently doing. */}
          <div className="header-group">
            <AgentDropdown
              agents={agents}
              stoppingAgentIds={stoppingAgentIds}
              agentErrors={agentErrors}
              onStopAgent={onStopAgent}
              onRefreshAgents={onRefreshAgents}
            />

            <StatusIndicator
              healthStatus={healthStatus}
              healthError={healthError}
              onToggleMonitoring={onToggleMonitoring}
              onRefreshHealth={onRefreshHealth}
              isLoading={isLoadingHealth}
            />
          </div>

          {/* View-scope: these change WHAT THE PAGE SHOWS. */}
          <div className="header-group">
            <label className="history-window-control">
              <span className="sr-only">Shared time range</span>
              <select
                aria-label="Shared time range"
                value={range.key}
                onChange={event => { setRange(event.target.value); }}
              >
                {TIME_RANGE_OPTIONS.map(option => (
                  <option key={option.key} value={option.key}>{option.label}</option>
                ))}
              </select>
            </label>

            <button
              className="header-button icon-button"
              onClick={() => { setPaused(!paused); }}
              type="button"
              title={paused ? 'Resume live updates' : 'Pause live updates'}
              aria-label={paused ? 'Resume live updates' : 'Pause live updates'}
            >
              {paused ? <Play size={16} /> : <Pause size={16} />}
            </button>
          </div>

          {/* App-scope: these change THE APP, not the reading. */}
          <div className="header-group">
            <button
              className="theme-toggle"
              onClick={toggleTheme}
              type="button"
              title={`Switch to ${theme === 'dark' ? 'light' : 'dark'} mode`}
              aria-label={`Switch to ${theme === 'dark' ? 'light' : 'dark'} mode`}
            >
              {theme === 'dark' ? <Sun size={16} /> : <Moon size={16} />}
            </button>

            <button
              className="header-button icon-button"
              onClick={onOpenSettings}
              type="button"
              title="Open system settings"
              aria-label="Open system settings"
            >
              <Settings size={16} />
            </button>

            {/* The unread count is rendered as a bare digit in the badge, which
                assistive tech announces as a stray number. It belongs in the
                accessible name, where it says what it actually counts. */}
            <button
              className="header-button icon-button"
              onClick={onToggleTerminal}
              type="button"
              title="Toggle system output"
              aria-label={
                unreadErrorCount > 0
                  ? `Toggle system output, ${unreadErrorCount} unread ${unreadErrorCount === 1 ? 'error' : 'errors'}`
                  : 'Toggle system output'
              }
              data-sm-style="sm-style-821233d621"
            >
              <Terminal size={16} />
              {unreadErrorCount > 0 && (
                <span className="icon-button-badge" aria-hidden="true">
                  {unreadErrorCount}
                </span>
              )}
            </button>
          </div>
        </div>
      </div>
    </header>
  );
};
