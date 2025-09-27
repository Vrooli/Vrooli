import { ReactNode, useEffect, useMemo, useRef, useState } from 'react';
import { NavLink, useLocation } from 'react-router-dom';
import clsx from 'clsx';
import type { LucideIcon } from 'lucide-react';
import {
  Activity,
  LayoutDashboard,
  PanelLeftClose,
  PanelLeftOpen,
  Power,
  RefreshCw,
  ScrollText,
  Server,
} from 'lucide-react';
import { appService, resourceService, healthService } from '@/services/api';
import { locateAppByIdentifier } from '@/utils/appPreview';
import { useAppsStore } from '@/state/appsStore';
import './Layout.css';

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

interface LayoutProps {
  children: ReactNode;
  isConnected: boolean;
}


export default function Layout({ children, isConnected }: LayoutProps) {
  const apps = useAppsStore(state => state.apps);
  const location = useLocation();
  const [appCount, setAppCount] = useState(0);
  const [resourceCount, setResourceCount] = useState(0);
  const [uptimeSeconds, setUptimeSeconds] = useState<number | null>(null);
  const [healthStatus, setHealthStatus] = useState<string | null>(null);
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(false);
  const [isMobile, setIsMobile] = useState(false);
  const [appsSearch, setAppsSearch] = useState('');
  const mobileViewportRef = useRef(false);
  const desktopCollapsedRef = useRef(false);
  const pollingRef = useRef(false);

  // Fetch counts and system info
  useEffect(() => {
    let isMounted = true;

    const fetchData = async () => {
      if (pollingRef.current) {
        return;
      }

      pollingRef.current = true;
      try {
        const [appsResult, resourcesResult, healthResult] = await Promise.allSettled([
          appService.getApps(),
          resourceService.getResources(),
          healthService.checkHealth(),
        ]);

        if (!isMounted) {
          return;
        }

        if (appsResult.status === 'fulfilled') {
          setAppCount(appsResult.value.length);
        }

        if (resourcesResult.status === 'fulfilled') {
          setResourceCount(resourcesResult.value.length);
        }

        if (appsResult.status === 'rejected') {
          console.error('Failed to fetch app list for layout metrics:', appsResult.reason);
        }

        if (resourcesResult.status === 'rejected') {
          console.error('Failed to fetch resource list for layout metrics:', resourcesResult.reason);
        }

        if (healthResult.status === 'fulfilled' && healthResult.value) {
          const normalizedStatus = typeof healthResult.value.status === 'string'
            ? healthResult.value.status.toLowerCase()
            : null;
          setHealthStatus(normalizedStatus);

          const healthUptimeSeconds = typeof healthResult.value.metrics?.uptime_seconds === 'number'
            ? healthResult.value.metrics.uptime_seconds
            : null;

          if (normalizedStatus === 'unhealthy') {
            setUptimeSeconds(null);
          }

          if (healthUptimeSeconds !== null && normalizedStatus !== 'unhealthy') {
            setUptimeSeconds(healthUptimeSeconds);
          }
        } else if (healthResult.status === 'rejected') {
          console.error('Health check failed:', healthResult.reason);
          setHealthStatus('unhealthy');
          setUptimeSeconds(null);
        }

      } catch (error) {
        if (isMounted) {
          console.error('Failed to fetch layout data:', error);
          setHealthStatus('unhealthy');
          setUptimeSeconds(null);
        }
      } finally {
        pollingRef.current = false;
      }
    };

    fetchData();
    const interval = setInterval(fetchData, 60000); // Update every 60 seconds (reduced from 10s to prevent CPU overload)

    return () => {
      isMounted = false;
      clearInterval(interval);
    };
  }, []);

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

  useEffect(() => {
    if (location.pathname === '/apps' || location.pathname.startsWith('/apps/')) {
      setAppsSearch(location.search || '');
    }
  }, [location.pathname, location.search]);

  // Handle quick actions
  const handleRestartAll = async () => {
    if (confirm('Restart all applications?')) {
      await appService.controlAllApps('restart');
    }
  };

  const handleStopAll = async () => {
    if (confirm('Stop all applications?')) {
      await appService.controlAllApps('stop');
    }
  };

  const handleHealthCheck = async () => {
    alert('Health check initiated');
  };

  const previewRouteMatch = location.pathname.match(/^\/apps\/([^/]+)\/preview/);
  const isPreviewRoute = Boolean(previewRouteMatch);
  const previewAppIdentifier = previewRouteMatch ? decodeURIComponent(previewRouteMatch[1]) : null;

  const previewAppName = useMemo(() => {
    if (!previewAppIdentifier) {
      return null;
    }

    const matched = locateAppByIdentifier(apps, previewAppIdentifier);
    if (matched?.scenario_name) {
      return matched.scenario_name;
    }
    if (matched?.id) {
      return matched.id;
    }

    return previewAppIdentifier;
  }, [apps, previewAppIdentifier]);

  const headerTitle = isPreviewRoute ? (previewAppName ?? 'Loading…') : 'Vrooli';

  const menuItems: Array<{ path: string; label: string; Icon: LucideIcon; count?: number }> = useMemo(() => ([
    { path: '/apps', label: 'APPLICATIONS', Icon: LayoutDashboard, count: appCount },
    { path: '/logs', label: 'SYSTEM LOGS', Icon: ScrollText },
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
              <div
                className="header-title"
                title={isPreviewRoute ? previewAppName ?? undefined : 'Vrooli'}
              >
                {headerTitle}
              </div>
            </div>
          </div>
          <div className="status-bar">
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

        <main className={clsx('main-content', { 'main-content--preview': isPreviewRoute })}>
          {children}
        </main>
      </div>
    </>
  );
}
