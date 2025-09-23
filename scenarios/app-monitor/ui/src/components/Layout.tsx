import { ReactNode, useState, useEffect } from 'react';
import { NavLink } from 'react-router-dom';
import clsx from 'clsx';
import { appService, resourceService, systemService } from '@/services/api';
import './Layout.css';

interface LayoutProps {
  children: ReactNode;
  isConnected: boolean;
}


export default function Layout({ children, isConnected }: LayoutProps) {
  const [appCount, setAppCount] = useState(0);
  const [resourceCount, setResourceCount] = useState(0);
  const [uptime, setUptime] = useState('00:00:00');
  const [isFetching, setIsFetching] = useState(false);

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

  const menuItems = [
    { path: '/apps', label: 'APPLICATIONS', icon: '▶' },
    { path: '/logs', label: 'SYSTEM LOGS', icon: '▶' },
    { path: '/resources', label: 'RESOURCES', icon: '▶' },
  ];

  return (
    <>
      <header className="main-header">
        <div className="header-grid">
          <div className="logo-section">
            <div className="logo-text">VROOLI</div>
            <div className="logo-subtitle">SYSTEMS MONITOR</div>
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

      <div className="dashboard">
        <aside className="sidebar">
          <div className="menu-section">
            <h3>NAVIGATION</h3>
            <ul className="menu-list">
              {menuItems.map(item => (
                <li key={item.path}>
                  <NavLink
                    to={item.path}
                    className={({ isActive }) => clsx('menu-item', { active: isActive })}
                  >
                    <span className="menu-icon">{item.icon}</span>
                    {item.label}
                  </NavLink>
                </li>
              ))}
            </ul>
          </div>
          
          <div className="quick-actions">
            <h3>QUICK ACTIONS</h3>
            <button className="action-btn" onClick={handleRestartAll}>
              RESTART ALL
            </button>
            <button className="action-btn" onClick={handleStopAll}>
              STOP ALL
            </button>
            <button className="action-btn" onClick={handleHealthCheck}>
              HEALTH CHECK
            </button>
          </div>
        </aside>

        <main className="main-content">
          {children}
        </main>
      </div>
    </>
  );
}
