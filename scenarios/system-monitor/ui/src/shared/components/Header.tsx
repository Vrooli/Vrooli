import { Moon, Settings, Sun, Terminal } from 'lucide-react';
import { StatusIndicator } from './StatusIndicator';
import { AgentDropdown } from './AgentDropdown';
import { useTheme } from '../theme/ThemeProvider';
import type { InvestigationAgentState } from '../../types';
import type { SystemHealthStatus } from '../../features/monitoring/hooks/useSystemMonitor';

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

  return (
    <header className="app-header">
      <div className="app-header-inner">
        <h1
          className="system-monitor-title icon-text"
          style={{ margin: 0, fontSize: 'var(--text-xl)' }}
        >
          <span className="system-monitor-title-text">System Monitor</span>
        </h1>

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
            style={{ position: 'relative' }}
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
