import { Moon, Pause, Play, Settings, Sun, Terminal } from 'lucide-react';
import { NavLink } from 'react-router-dom';
import { StatusIndicator } from './StatusIndicator';
import { AgentDropdown } from './AgentDropdown';
import { useTheme } from '../theme/ThemeProvider';
import type { InvestigationAgentState } from '../../types';
import type { SystemHealthStatus } from '../../features/monitoring/hooks/useSystemMonitor';
import { TIME_RANGE_OPTIONS, useTimeRange } from '../time/TimeRangeContext';

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

  return (
    <header className="app-header">
      <div className="app-header-inner">
        <h1
          className="system-monitor-title icon-text"
          data-sm-style="sm-style-a02ac2f8c1"
        >
          <span className="system-monitor-title-text">System Monitor</span>
        </h1>

        <nav className="app-nav flex-row-center gap-sm" data-sm-style="sm-style-c938f99d82">
          <NavLink to="/" end className="app-nav-link">
            Dashboard
          </NavLink>
          <NavLink to="/forensics" className="app-nav-link">
            Forensics
          </NavLink>
          <NavLink to="/logs" className="app-nav-link">
            Logs
          </NavLink>
          <NavLink to="/capacity" className="app-nav-link">
            Capacity
          </NavLink>
          <NavLink to="/scripts" className="app-nav-link">
            Scripts
          </NavLink>
        </nav>

        <div className="flex-row-center gap-sm">
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

          <button
            className="theme-toggle"
            onClick={toggleTheme}
            type="button"
            title={`Switch to ${theme === 'dark' ? 'light' : 'dark'} mode`}
          >
            {theme === 'dark' ? <Sun size={16} /> : <Moon size={16} />}
          </button>

          <button
            className="header-button icon-button"
            onClick={onOpenSettings}
            type="button"
            title="Open system settings"
          >
            <Settings size={16} />
          </button>

          <button
            className="header-button icon-button"
            onClick={onToggleTerminal}
            type="button"
            title="Toggle system output"
            data-sm-style="sm-style-821233d621"
          >
            <Terminal size={16} />
            {unreadErrorCount > 0 && (
              <span className="icon-button-badge">
                {unreadErrorCount}
              </span>
            )}
          </button>
        </div>
      </div>
    </header>
  );
};
