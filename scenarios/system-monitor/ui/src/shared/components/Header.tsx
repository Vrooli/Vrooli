import { Menu, Moon, Pause, Play, Settings, Sun, Terminal, X } from 'lucide-react';
import { NavLink, useLocation, useNavigate } from 'react-router-dom';
import { useEffect, useRef, useState } from 'react';
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
  const [mobileNavOpen, setMobileNavOpen] = useState(false);
  const mobileNavTriggerRef = useRef<HTMLButtonElement>(null);
  const mobileNavPanelRef = useRef<HTMLElement>(null);
  const currentPath = location.pathname === '/' ? '/' : `/${location.pathname.split('/')[1]}`;

  useEffect(() => {
    if (!mobileNavOpen) {
      return undefined;
    }

    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';

    const panel = mobileNavPanelRef.current;
    const focusableSelector = 'a[href], button:not([disabled])';
    const firstFocusable = panel?.querySelector<HTMLElement>(focusableSelector);
    firstFocusable?.focus();

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault();
        setMobileNavOpen(false);
        return;
      }
      if (event.key !== 'Tab' || !panel) {
        return;
      }

      const focusable = Array.from(panel.querySelectorAll<HTMLElement>(focusableSelector));
      if (focusable.length === 0) {
        event.preventDefault();
        return;
      }
      const first = focusable.at(0);
      const last = focusable.at(-1);
      if (!first || !last) {
        return;
      }
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => {
      document.body.style.overflow = previousOverflow;
      document.removeEventListener('keydown', handleKeyDown);
      mobileNavTriggerRef.current?.focus();
    };
  }, [mobileNavOpen]);

  const closeMobileNav = () => { setMobileNavOpen(false); };
  const navigateFromMobileNav = (path: string) => {
    closeMobileNav();
    void navigate(path);
  };

  return (
    <header className="app-header">
      <div className="app-header-inner">
        <div className="app-header-brand">
          <h1
            className="system-monitor-title icon-text"
            data-sm-style="sm-style-a02ac2f8c1"
          >
            <span className="system-monitor-title-text">System Monitor</span>
          </h1>

          <button
            ref={mobileNavTriggerRef}
            className="app-mobile-nav-trigger"
            type="button"
            aria-label={mobileNavOpen ? 'Close navigation' : 'Open navigation'}
            aria-expanded={mobileNavOpen}
            aria-controls="system-monitor-mobile-navigation"
            onClick={() => { setMobileNavOpen(prev => !prev); }}
          >
            {mobileNavOpen ? <X size={20} aria-hidden="true" /> : <Menu size={20} aria-hidden="true" />}
            <span>Menu</span>
          </button>
        </div>

        <nav className="app-nav flex-row-center gap-sm" aria-label="Primary navigation" data-sm-style="sm-style-c938f99d82">
          {NAV_ITEMS.map(item => (
            <NavLink key={item.path} to={item.path} end={item.path === '/'} className="app-nav-link">
              {item.label}
            </NavLink>
          ))}
        </nav>

        {mobileNavOpen && (
          <>
            <button
              className="app-mobile-nav-scrim"
              type="button"
              aria-label="Close navigation"
              onClick={closeMobileNav}
            />
            <aside
              ref={mobileNavPanelRef}
              id="system-monitor-mobile-navigation"
              className="app-mobile-nav-panel"
              aria-label="Primary navigation"
            >
              <div className="app-mobile-nav-panel-header">
                <span className="app-mobile-nav-panel-title">Navigate</span>
                <button className="header-button icon-button" type="button" onClick={closeMobileNav} aria-label="Close navigation">
                  <X size={18} aria-hidden="true" />
                </button>
              </div>
              <nav className="app-mobile-nav-links" aria-label="Primary navigation links">
                {NAV_ITEMS.map(item => (
                  <NavLink
                    key={item.path}
                    to={item.path}
                    end={item.path === '/'}
                    className="app-mobile-nav-link"
                    onClick={() => { navigateFromMobileNav(item.path); }}
                  >
                    <span>{item.label}</span>
                    {currentPath === item.path && <span className="app-mobile-nav-current">Current</span>}
                  </NavLink>
                ))}
              </nav>
            </aside>
          </>
        )}

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
