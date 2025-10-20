import { ReactNode, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { CSSProperties } from 'react';
import { createPortal } from 'react-dom';
import { NavLink, useLocation, useNavigate } from 'react-router-dom';
import clsx from 'clsx';
import type { LucideIcon } from 'lucide-react';
import {
  Activity,
  LayoutDashboard,
  PanelLeftClose,
  PanelLeftOpen,
  Power,
  RefreshCw,
  Server,
} from 'lucide-react';
import { useSystemStatus } from '@/state/systemStatusStore';
import {
  collectAppIdentifiers,
  locateAppByIdentifier,
  normalizeIdentifier,
  resolveAppIdentifier,
} from '@/utils/appPreview';
import { useAppsStore } from '@/state/appsStore';
import type { App } from '@/types';
import HistoryMenu from './HistoryMenu';
import './Layout.css';
import { useShellOverlayStore } from '@/state/shellOverlayStore';
import { buildAlphabetizedApps, buildRecentApps } from '@/utils/appCollections';
import { useScenarioActions } from '@/hooks/useScenarioActions';

const OFFLINE_STATES = new Set(['unhealthy', 'offline', 'critical']);

const formatSecondsToDuration = (seconds: number): string => {
  if (!Number.isFinite(seconds) || Number.isNaN(seconds)) {
    return '--:--:--';
  }

  const totalSeconds = Math.max(0, Math.floor(seconds));
  const days = Math.floor(totalSeconds / 86400);
  const hours = Math.floor((totalSeconds % 86400) / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const secs = totalSeconds % 60;

  if (days > 0) {
    return `${days}d ${hours.toString().padStart(2, '0')}h ${minutes.toString().padStart(2, '0')}m`;
  }

  return [hours, minutes, secs]
    .map(value => value.toString().padStart(2, '0'))
    .join(':');
};

const HISTORY_LIMIT = 16;
const STATUS_POPOVER_OFFSET = 10;
const STATUS_POPOVER_MARGIN = 12;

interface LayoutProps {
  children: ReactNode;
  isConnected: boolean;
}


export default function Layout({ children, isConnected }: LayoutProps) {
  const apps = useAppsStore(state => state.apps);
  const location = useLocation();
  const navigate = useNavigate();
  const [uptimeSeconds, setUptimeSeconds] = useState<number | null>(null);
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(true);
  const [isMobile, setIsMobile] = useState(false);
  const [appsSearch, setAppsSearch] = useState('');
  const mobileViewportRef = useRef(false);
  const desktopCollapsedRef = useRef(true);
  const isBrowser = typeof document !== 'undefined';
  const statusButtonRef = useRef<HTMLButtonElement | null>(null);
  const statusPopoverRef = useRef<HTMLDivElement | null>(null);
  const [statusPopoverCoords, setStatusPopoverCoords] = useState<{ top: number; left: number } | null>(null);
  const [isStatusPopoverOpen, setIsStatusPopoverOpen] = useState(false);
  const activeOverlay = useShellOverlayStore(state => state.activeView);
  const openOverlay = useShellOverlayStore(state => state.openView);
  const closeOverlayView = useShellOverlayStore(state => state.closeView);

  useEffect(() => {
    if (activeOverlay !== null) {
      setIsSidebarCollapsed(true);
      setIsStatusPopoverOpen(false);
      setStatusPopoverCoords(null);
    }
  }, [activeOverlay]);

  const updateStatusPopoverPosition = useCallback(() => {
    if (!isBrowser) {
      return;
    }

    const button = statusButtonRef.current;
    const popover = statusPopoverRef.current;
    if (!button || !popover) {
      return;
    }

    const buttonRect = button.getBoundingClientRect();
    const popoverRect = popover.getBoundingClientRect();
    const popoverWidth = popoverRect.width || popover.offsetWidth;
    const viewportWidth = window.innerWidth;

    const idealLeft = buttonRect.left;
    const minLeft = STATUS_POPOVER_MARGIN;
    const maxLeft = Math.max(minLeft, viewportWidth - popoverWidth - STATUS_POPOVER_MARGIN);
    const clampedLeft = Math.min(Math.max(idealLeft, minLeft), maxLeft);

    const top = Math.round(buttonRect.bottom + STATUS_POPOVER_OFFSET);
    const left = Math.round(clampedLeft);

    setStatusPopoverCoords({ top, left });
  }, [isBrowser]);

  const statusPopoverStyle = useMemo<CSSProperties>(() => {
    if (!statusPopoverCoords) {
      return {
        top: '-9999px',
        left: '-9999px',
        visibility: 'hidden',
      };
    }

    return {
      top: `${statusPopoverCoords.top}px`,
      left: `${statusPopoverCoords.left}px`,
      visibility: 'visible',
    };
  }, [statusPopoverCoords]);
  const recordRouteDebug = useCallback((event: string, detail?: Record<string, unknown>) => {
    try {
      const payload = {
        event,
        timestamp: Date.now(),
        detail: {
          pathname: location.pathname,
          search: location.search,
          ...(detail ?? {}),
        },
        userAgent: typeof navigator !== 'undefined' ? navigator.userAgent : undefined,
      };
      const body = JSON.stringify(payload);
      if (typeof navigator !== 'undefined' && typeof navigator.sendBeacon === 'function') {
        const blob = new Blob([body], { type: 'application/json' });
        navigator.sendBeacon('/__debug/client-event', blob);
      } else {
        void fetch('/__debug/client-event', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body,
          keepalive: true,
        });
      }
    } catch (error) {
      // Debug logging is best-effort
    }
  }, [location.pathname, location.search]);

  useEffect(() => {
    recordRouteDebug('route-change', { key: location.key });
  }, [location.key, recordRouteDebug]);

  useEffect(() => {
    recordRouteDebug('layout-mount');
    return () => {
      recordRouteDebug('layout-unmount');
    };
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const previewRouteInfo = useMemo(() => {
    const match = location.pathname.match(/^\/apps\/([^/]+)\/preview/);
    const identifier = match ? decodeURIComponent(match[1]) : null;
    return {
      match,
      identifier,
      isPreviewRoute: Boolean(match),
    } as const;
  }, [location.pathname]);

  const previewApp = useMemo(() => {
    if (!previewRouteInfo.identifier) {
      return null;
    }

    return locateAppByIdentifier(apps, previewRouteInfo.identifier) ?? null;
  }, [apps, previewRouteInfo.identifier]);

  const previewAppIdentifiers = useMemo(() => {
    if (previewApp) {
      return collectAppIdentifiers(previewApp);
    }

    const normalized = normalizeIdentifier(previewRouteInfo.identifier);
    return normalized ? [normalized] : [];
  }, [previewApp, previewRouteInfo.identifier]);

  const previewAppName = useMemo(() => {
    if (!previewRouteInfo.identifier) {
      return null;
    }

    if (previewApp?.scenario_name) {
      return previewApp.scenario_name;
    }

    if (previewApp?.name) {
      return previewApp.name;
    }

    if (previewApp?.id) {
      return previewApp.id;
    }

    return previewRouteInfo.identifier;
  }, [previewApp, previewRouteInfo.identifier]);

  const pathname = location.pathname;
  const isAppsListRoute = pathname === '/apps';
  const isResourcesRoute = pathname === '/resources' || pathname.startsWith('/resources/');

  const recentApps = useMemo(
    () => buildRecentApps(apps, { excludeIdentifiers: previewAppIdentifiers, limit: HISTORY_LIMIT }),
    [apps, previewAppIdentifiers],
  );

  const alphabetizedApps = useMemo(() => buildAlphabetizedApps(apps), [apps]);

  const shouldShowHistory = (previewRouteInfo.isPreviewRoute || isAppsListRoute)
    && (recentApps.length > 0 || alphabetizedApps.length > 0);

  useEffect(() => {
    const handleResize = () => {
      const mobile = window.innerWidth <= 768;
      setIsMobile(mobile);

      setIsSidebarCollapsed(prev => {
        if (mobile && !mobileViewportRef.current) {
          mobileViewportRef.current = true;
          desktopCollapsedRef.current = prev;
          return true;
        }

        if (!mobile && mobileViewportRef.current) {
          mobileViewportRef.current = false;
          return desktopCollapsedRef.current;
        }

        return prev;
      });
    };

    handleResize();
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  const shouldShowOverlay = isMobile && !isSidebarCollapsed;

  useEffect(() => {
    if (!shouldShowOverlay) {
      return;
    }

    const originalOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';

    return () => {
      document.body.style.overflow = originalOverflow;
    };
  }, [shouldShowOverlay]);

  useEffect(() => {
    if (!isMobile && isStatusPopoverOpen) {
      setIsStatusPopoverOpen(false);
      setStatusPopoverCoords(null);
    }
  }, [isMobile, isStatusPopoverOpen]);

  useEffect(() => {
    if (!isStatusPopoverOpen) {
      return;
    }

    const handlePointer = (event: MouseEvent | TouchEvent) => {
      const target = event.target as Node | null;
      if (!target) {
        return;
      }

      if (statusButtonRef.current?.contains(target)) {
        return;
      }

      if (statusPopoverRef.current?.contains(target)) {
        return;
      }

      closeStatusPopover();
    };

    const handleKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault();
        closeStatusPopover();
      }
    };

    document.addEventListener('mousedown', handlePointer);
    document.addEventListener('touchstart', handlePointer, { passive: true });
    document.addEventListener('keydown', handleKey);

    return () => {
      document.removeEventListener('mousedown', handlePointer);
      document.removeEventListener('touchstart', handlePointer);
      document.removeEventListener('keydown', handleKey);
    };
  }, [isStatusPopoverOpen]);

  useEffect(() => {
    if (!isStatusPopoverOpen) {
      setStatusPopoverCoords(null);
      return;
    }

    const frame = window.requestAnimationFrame(() => {
      updateStatusPopoverPosition();
    });

    const handleResizeOrScroll = () => {
      updateStatusPopoverPosition();
    };

    window.addEventListener('resize', handleResizeOrScroll);
    window.addEventListener('scroll', handleResizeOrScroll, true);

    return () => {
      window.cancelAnimationFrame(frame);
      window.removeEventListener('resize', handleResizeOrScroll);
      window.removeEventListener('scroll', handleResizeOrScroll, true);
    };
  }, [isStatusPopoverOpen, updateStatusPopoverPosition]);

  useEffect(() => {
    setIsStatusPopoverOpen(false);
    setStatusPopoverCoords(null);
  }, [previewRouteInfo.identifier]);

  const toggleSidebar = () => {
    setIsSidebarCollapsed(prev => {
      const next = !prev;
      if (!isMobile) {
        desktopCollapsedRef.current = next;
      }
      return next;
    });
  };

  const closeSidebar = () => {
    setIsSidebarCollapsed(() => {
      if (!isMobile) {
        desktopCollapsedRef.current = true;
      }
      return true;
    });
  };

  const handleNavClick = () => {
    if (isMobile) {
      closeSidebar();
    }
  };

  const { restartAll: handleRestartAll, stopAll: handleStopAll, triggerHealthCheck: handleHealthCheck } = useScenarioActions();
  const { status: healthStatus, uptimeSeconds: remoteUptime, appCount, resourceCount } = useSystemStatus();

  useEffect(() => {
    setUptimeSeconds(remoteUptime);
  }, [remoteUptime]);

  const handleStatusButtonClick = () => {
    if (isMobile) {
      openOverlay('actions');
      return;
    }

    setIsStatusPopoverOpen(prev => {
      const next = !prev;
      if (next) {
        setTimeout(() => {
          updateStatusPopoverPosition();
        }, 0);
      } else {
        setStatusPopoverCoords(null);
      }
      return next;
    });
  };

  const closeStatusPopover = () => {
    setIsStatusPopoverOpen(false);
    setStatusPopoverCoords(null);
    statusButtonRef.current?.focus();
  };

  useEffect(() => {
    if (location.pathname === '/apps' || location.pathname.startsWith('/apps/')) {
      setAppsSearch(location.search || '');
    }
  }, [location.pathname, location.search]);

  // Handle quick actions
  const handleHistorySelect = useCallback((app: App) => {
    const identifier = resolveAppIdentifier(app);
    if (!identifier) {
      return;
    }

    recordRouteDebug('navigate-event', {
      source: 'history-menu',
      targetPath: `/apps/${encodeURIComponent(identifier)}/preview`,
      search: location.search || undefined,
    });
    navigate({
      pathname: `/apps/${encodeURIComponent(identifier)}/preview`,
      search: location.search || undefined,
    });
  }, [location.search, navigate, recordRouteDebug]);

  const headerTitle = useMemo(() => {
    if (previewRouteInfo.isPreviewRoute) {
      return previewAppName ?? 'Loading…';
    }

    if (isAppsListRoute) {
      return 'Apps';
    }

    if (isResourcesRoute) {
      return 'Resources';
    }

    return 'Vrooli';
  }, [isAppsListRoute, isResourcesRoute, previewAppName, previewRouteInfo.isPreviewRoute]);

  const menuItems: Array<{ path: string; label: string; Icon: LucideIcon; count?: number }> = useMemo(() => ([
    { path: '/apps', label: 'APPLICATIONS', Icon: LayoutDashboard, count: appCount },
    { path: '/resources', label: 'RESOURCES', Icon: Server, count: resourceCount },
  ]), [appCount, resourceCount]);

  const quickActions: Array<{
    label: string;
    onClick: () => void | Promise<void>;
    Icon: LucideIcon;
    title: string;
  }> = [
    {
      label: 'RESTART ALL',
      onClick: handleRestartAll,
      Icon: RefreshCw,
      title: 'Restart all applications',
    },
    {
      label: 'STOP ALL',
      onClick: handleStopAll,
      Icon: Power,
      title: 'Stop all applications',
    },
    {
      label: 'HEALTH CHECK',
      onClick: handleHealthCheck,
      Icon: Activity,
      title: 'Trigger a system health check',
    },
  ];

  const isSystemOnline = healthStatus
    ? !OFFLINE_STATES.has(healthStatus)
    : isConnected;
  const uptimeDisplay = useMemo(() => (
    uptimeSeconds !== null
      ? formatSecondsToDuration(uptimeSeconds)
      : '--:--:--'
  ), [uptimeSeconds]);

  const statusText = isSystemOnline ? uptimeDisplay : 'Offline';
  const shouldTick = isSystemOnline && uptimeSeconds !== null;

  useEffect(() => {
    if (!shouldTick) {
      return;
    }

    const interval = window.setInterval(() => {
      setUptimeSeconds(prev => (prev !== null ? prev + 1 : prev));
    }, 1000);

    return () => {
      window.clearInterval(interval);
    };
  }, [shouldTick]);

  return (
    <>
      <header className="main-header">
        <div className="header-grid">
          <div className="logo-section">
            <button
              type="button"
              className="sidebar-toggle"
              aria-label={isSidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
              aria-expanded={!isSidebarCollapsed}
              onClick={toggleSidebar}
            >
              {isSidebarCollapsed ? (
                <PanelLeftOpen aria-hidden className="toggle-icon" />
              ) : (
                <PanelLeftClose aria-hidden className="toggle-icon" />
              )}
            </button>
            <div className="logo-copy">
              <div className="header-title-wrapper">
                <div
                  className="header-title"
                  title={headerTitle}
                >
                  {headerTitle}
                </div>
                <HistoryMenu
                  recentApps={recentApps}
                  allApps={alphabetizedApps}
                  shouldShowHistory={shouldShowHistory}
                  onSelect={handleHistorySelect}
                />
              </div>
            </div>
          </div>
          <div className="status-bar">
            {isMobile ? (
              <div className="status-chip-wrapper" role="status" aria-live="polite">
                <button
                  type="button"
                  ref={statusButtonRef}
                className={clsx(
                  'status-chip',
                  'status-chip--button',
                  { offline: !isSystemOnline, 'status-chip--active': isStatusPopoverOpen },
                )}
                aria-haspopup="dialog"
                aria-expanded={isStatusPopoverOpen}
                aria-label={isSystemOnline ? 'System status online. View uptime' : 'System status offline. View details'}
                onClick={handleStatusButtonClick}
                >
                  <span
                    className={clsx('status-indicator', isSystemOnline ? 'online' : 'offline')}
                    aria-hidden
                  />
                </button>
                {isBrowser && isStatusPopoverOpen && createPortal(
                  <div
                    className="status-popover"
                    role="dialog"
                    aria-label="System status details"
                    ref={statusPopoverRef}
                    style={statusPopoverStyle}
                  >
                    <div className="status-popover__row">
                      <span className="status-popover__label">Status</span>
                      <span className="status-popover__value">{isSystemOnline ? 'Online' : 'Offline'}</span>
                    </div>
                    <div className="status-popover__row">
                      <span className="status-popover__label">Uptime</span>
                      <span className="status-popover__value">{uptimeDisplay}</span>
                    </div>
                  </div>,
                  document.body,
                )}
              </div>
            ) : (
              <div
                className={clsx('status-chip', { offline: !isSystemOnline })}
                role="status"
                aria-live="polite"
              >
                <span
                  className={clsx('status-indicator', isSystemOnline ? 'online' : 'offline')}
                  aria-hidden
                />
                <span className="status-text">{statusText}</span>
              </div>
            )}
          </div>
        </div>
      </header>

      <div className={clsx('dashboard', { 'sidebar-open-mobile': shouldShowOverlay })}>
        {shouldShowOverlay && (
          <button
            type="button"
            className="sidebar-backdrop"
            onClick={closeSidebar}
            aria-label="Close navigation overlay"
          />
        )}
        <aside className={clsx('sidebar', { collapsed: isSidebarCollapsed, 'mobile-open': shouldShowOverlay })}>
          {isMobile && (
            <button
              type="button"
              className="sidebar-close"
              aria-label="Close sidebar"
              onClick={closeSidebar}
            >
              <PanelLeftClose aria-hidden className="toggle-icon" />
            </button>
          )}
          <div className="menu-section">
            <h3>NAVIGATION</h3>
            <ul className="menu-list">
              {menuItems.map(item => (
                <li key={item.path}>
                  <NavLink
                    to={item.path === '/apps' && appsSearch
                      ? { pathname: item.path, search: appsSearch }
                      : item.path}
                    className={({ isActive }) => clsx('menu-item', { active: isActive })}
                    title={item.label}
                    onClick={handleNavClick}
                  >
                    <item.Icon aria-hidden className="menu-icon" />
                    <span className="menu-label">
                      {item.label}
                      {typeof item.count === 'number' && (
                        <span className="menu-count">({item.count})</span>
                      )}
                    </span>
                  </NavLink>
                </li>
              ))}
            </ul>
          </div>
          
          <div className="quick-actions">
            <h3>QUICK ACTIONS</h3>
            {quickActions.map(action => (
              <button
                key={action.label}
                className="action-btn"
                onClick={() => {
                  closeOverlayView();
                  void action.onClick();
                }}
                title={action.title}
                type="button"
              >
                <action.Icon aria-hidden className="action-icon" />
                <span className="action-label">{action.label}</span>
              </button>
            ))}
          </div>
        </aside>

        <main className={clsx('main-content', { 'main-content--preview': previewRouteInfo.isPreviewRoute })}>
          {children}
        </main>
      </div>
    </>
  );
}
