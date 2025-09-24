import { ReactNode, useState, useEffect, useRef } from 'react';
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
import { appService, resourceService, systemService } from '@/services/api';
import './Layout.css';

interface LayoutProps {
  children: ReactNode;
  isConnected: boolean;
}


export default function Layout({ children, isConnected }: LayoutProps) {
  const location = useLocation();
  const [appCount, setAppCount] = useState(0);
  const [resourceCount, setResourceCount] = useState(0);
  const [uptime, setUptime] = useState('00:00:00');
  const [isFetching, setIsFetching] = useState(false);
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(false);
  const [isMobile, setIsMobile] = useState(false);
  const mobileViewportRef = useRef(false);
  const desktopCollapsedRef = useRef(false);

  // Fetch counts and system info
  useEffect(() => {
    const fetchData = async () => {
      // Skip if already fetching to prevent request stacking
      if (isFetching) {
        console.log('Skipping fetch - already in progress');
        return;
      }
      
      setIsFetching(true);
      try {
        // Fetch app count
        const apps = await appService.getApps();
        setAppCount(apps.length);
        
        // Fetch resource count
        const resources = await resourceService.getResources();
        setResourceCount(resources.length);
        
        // Fetch system info for uptime
        const systemInfo = await systemService.getSystemInfo();
        if (systemInfo && systemInfo.orchestrator_running && systemInfo.uptime) {
          setUptime(systemInfo.uptime);
        }
      } catch (error) {
        console.error('Failed to fetch layout data:', error);
      } finally {
        setIsFetching(false);
      }
    };
    
    fetchData();
    const interval = setInterval(fetchData, 60000); // Update every 60 seconds (reduced from 10s to prevent CPU overload)
    
    return () => clearInterval(interval);
  }, [isFetching]);

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

  const menuItems: Array<{ path: string; label: string; Icon: LucideIcon }> = [
    { path: '/apps', label: 'APPLICATIONS', Icon: LayoutDashboard },
    { path: '/logs', label: 'SYSTEM LOGS', Icon: ScrollText },
    { path: '/resources', label: 'RESOURCES', Icon: Server },
  ];

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

  const isPreviewRoute = /^\/apps\/[^/]+\/preview/.test(location.pathname);

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
              <div className="logo-text">VROOLI</div>
              <div className="logo-subtitle">SYSTEMS MONITOR</div>
            </div>
          </div>
          <div className="status-bar">
            <div className="status-item">
              <span className="status-label">SYSTEM</span>
              <span className={clsx('status-value', isConnected ? 'online' : 'offline')}>
                {isConnected ? 'ONLINE' : 'OFFLINE'}
              </span>
            </div>
            <div className="status-item">
              <span className="status-label">APPS</span>
              <span className="status-value">{appCount}</span>
            </div>
            <div className="status-item">
              <span className="status-label">RESOURCES</span>
              <span className="status-value">{resourceCount}</span>
            </div>
            <div className="status-item">
              <span className="status-label">UPTIME</span>
              <span className="status-value">{uptime}</span>
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
                    to={item.path}
                    className={({ isActive }) => clsx('menu-item', { active: isActive })}
                    title={item.label}
                    onClick={handleNavClick}
                  >
                    <item.Icon aria-hidden className="menu-icon" />
                    <span className="menu-label">{item.label}</span>
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
