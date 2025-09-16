import { useState, useEffect } from 'react'
import { Routes, Route, Navigate, useNavigate, useLocation } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { 
  Shield, 
  Activity, 
  AlertTriangle,
  Settings,
  BarChart3,
  Terminal,
  Zap,
  Menu,
  X,
  ChevronLeft,
  ChevronRight,
  Server
} from 'lucide-react'
import clsx from 'clsx'
import Dashboard from './components/Dashboard'
import ScenariosList from './components/ScenariosList'
import ScenarioDetail from './components/ScenarioDetail'
import VulnerabilityScanner from './components/VulnerabilityScanner'
import HealthMonitor from './components/HealthMonitor'
import PerformanceMetrics from './components/PerformanceMetrics'
import AutomatedFixPanel from './components/AutomatedFixPanel'
import SettingsPanel from './components/SettingsPanel'
import { apiService } from './services/api'

export default function App() {
  const navigate = useNavigate()
  const location = useLocation()
  
  // Load sidebar state from localStorage or default to true
  const [sidebarOpen, setSidebarOpen] = useState(() => {
    const saved = localStorage.getItem('sidebarOpen')
    return saved !== null ? JSON.parse(saved) : true
  })
  
  // Sidebar collapsed state (different from mobile open/close)
  const [sidebarCollapsed, setSidebarCollapsed] = useState(() => {
    const saved = localStorage.getItem('sidebarCollapsed')
    return saved !== null ? JSON.parse(saved) : false
  })
  
  const { 
    data: systemStatus, 
    error: systemStatusError, 
    isLoading: systemStatusLoading,
    isError: systemStatusIsError,
    isFetching: systemStatusIsFetching 
  } = useQuery({
    queryKey: ['systemStatus'],
    queryFn: () => apiService.getSystemStatus(),
    refetchInterval: 30000,
    retry: 3,
  })


  const navigation = [
    { id: 'dashboard', name: 'Dashboard', icon: BarChart3, path: '/' },
    { id: 'scenarios', name: 'Scenarios', icon: Server, path: '/scenarios' },
    { id: 'vulnerabilities', name: 'Security Scanner', icon: Shield, path: '/vulnerabilities' },
    { id: 'health', name: 'Health Monitor', icon: Activity, path: '/health' },
    { id: 'performance', name: 'Performance', icon: Zap, path: '/performance' },
    { id: 'automated-fixes', name: 'Automated Fixes', icon: Terminal, path: '/fixes' },
    { id: 'settings', name: 'Settings', icon: Settings, path: '/settings' },
  ]
  
  // Get current view based on location
  const currentView = navigation.find(n => n.path === location.pathname)?.id || 'dashboard'

  // Save sidebar state to localStorage when it changes
  useEffect(() => {
    localStorage.setItem('sidebarOpen', JSON.stringify(sidebarOpen))
  }, [sidebarOpen])
  
  useEffect(() => {
    localStorage.setItem('sidebarCollapsed', JSON.stringify(sidebarCollapsed))
  }, [sidebarCollapsed])

  useEffect(() => {
    const handleResize = () => {
      if (window.innerWidth < 768) {
        setSidebarOpen(false)
      }
    }
    window.addEventListener('resize', handleResize)
    handleResize()
    return () => window.removeEventListener('resize', handleResize)
  }, [])

  return (
    <div className="flex h-screen bg-gradient-to-br from-dark-50 via-white to-primary-50">
      {/* Sidebar */}
      <aside className={clsx(
        'fixed inset-y-0 z-50 flex flex-col transition-all duration-300 md:relative md:translate-x-0',
        sidebarOpen ? 'translate-x-0' : '-translate-x-full',
        sidebarCollapsed ? 'w-20' : 'w-72'
      )}>
        <div className={clsx(
          "flex h-full flex-col gap-y-5 overflow-y-auto bg-white/95 backdrop-blur-xl pb-4 shadow-xl border-r border-dark-200",
          sidebarCollapsed ? "px-3" : "px-6"
        )}>
          {/* Logo and Collapse Button */}
          <div className="flex h-16 items-center justify-between">
            {!sidebarCollapsed ? (
              <>
                <div className="flex items-center gap-2">
                  <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-gradient-to-br from-primary-500 to-primary-600 text-white">
                    <Shield className="h-6 w-6" />
                  </div>
                  <div>
                    <h2 className="text-lg font-bold text-dark-900">API Manager</h2>
                    <p className="text-xs text-dark-500">Security & Intelligence</p>
                  </div>
                </div>
                
                {/* Desktop collapse toggle - expanded state */}
                <button
                  className="hidden md:flex items-center justify-center h-8 w-8 rounded-lg hover:bg-dark-100 transition-colors"
                  onClick={() => setSidebarCollapsed(true)}
                  title="Collapse sidebar"
                >
                  <ChevronLeft className="h-4 w-4 text-dark-600" />
                </button>
                
                {/* Mobile close button */}
                <button
                  className="md:hidden"
                  onClick={() => setSidebarOpen(false)}
                >
                  <X className="h-5 w-5 text-dark-500" />
                </button>
              </>
            ) : (
              <>
                {/* Collapsed state - just the logo and expand button */}
                <div className="flex w-full items-center justify-between">
                  <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-gradient-to-br from-primary-500 to-primary-600 text-white">
                    <Shield className="h-6 w-6" />
                  </div>
                  
                  {/* Desktop expand toggle - collapsed state */}
                  <button
                    className="hidden md:flex items-center justify-center h-8 w-8 rounded-lg hover:bg-dark-100 transition-colors"
                    onClick={() => setSidebarCollapsed(false)}
                    title="Expand sidebar"
                  >
                    <ChevronRight className="h-4 w-4 text-dark-600" />
                  </button>
                </div>
                
                {/* Mobile close button - still needed when collapsed on mobile */}
                <button
                  className="md:hidden absolute top-4 right-3"
                  onClick={() => setSidebarOpen(false)}
                >
                  <X className="h-5 w-5 text-dark-500" />
                </button>
              </>
            )}
          </div>

          {/* System Status */}
          {!sidebarCollapsed ? (
            <div className="rounded-lg bg-gradient-to-r from-primary-500 to-primary-600 p-4 text-white">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-xs font-medium opacity-90">System Status</p>
                  <p className="text-2xl font-bold">
                    {systemStatus?.health_score || 0}%
                  </p>
                </div>
                <div className={clsx(
                  'flex h-12 w-12 items-center justify-center rounded-full',
                  systemStatus?.status === 'healthy' ? 'bg-success-500/20' :
                  systemStatus?.status === 'degraded' ? 'bg-warning-500/20' :
                  'bg-danger-500/20'
                )}>
                  {systemStatus?.status === 'healthy' ? (
                    <Activity className="h-6 w-6 text-white" />
                  ) : (
                    <AlertTriangle className="h-6 w-6 text-white" />
                  )}
                </div>
              </div>
              <div className="mt-3 space-y-1">
                <div className="flex justify-between text-xs">
                  <span className="opacity-90">Scenarios</span>
                  <span className="font-semibold">{systemStatus?.scenarios || 0}</span>
                </div>
                <div className="flex justify-between text-xs">
                  <span className="opacity-90">Open Vulnerabilities</span>
                  <span className="font-semibold">{systemStatus?.vulnerabilities || 0}</span>
                </div>
              </div>
            </div>
          ) : (
            <div className="flex justify-center">
              <div className={clsx(
                'flex h-10 w-10 items-center justify-center rounded-full',
                systemStatus?.status === 'healthy' ? 'bg-success-500' :
                systemStatus?.status === 'degraded' ? 'bg-warning-500' :
                'bg-danger-500'
              )} title={`System: ${systemStatus?.status || 'unknown'}`}>
                {systemStatus?.status === 'healthy' ? (
                  <Activity className="h-5 w-5 text-white" />
                ) : (
                  <AlertTriangle className="h-5 w-5 text-white" />
                )}
              </div>
            </div>
          )}

          {/* Navigation */}
          <nav className={clsx(
            "flex-1",
            sidebarCollapsed ? "space-y-2 flex flex-col items-center" : "space-y-1"
          )}>
            {navigation.map((item) => {
              const Icon = item.icon
              const isActive = currentView === item.id
              
              return (
                <button
                  key={item.id}
                  onClick={() => {
                    navigate(item.path)
                    // Close mobile sidebar after navigation
                    if (window.innerWidth < 768) {
                      setSidebarOpen(false)
                    }
                  }}
                  className={clsx(
                    'group flex items-center rounded-lg text-sm font-medium transition-all relative',
                    isActive
                      ? 'bg-primary-500 text-white shadow-lg shadow-primary-500/25'
                      : 'text-dark-700 hover:bg-dark-100 hover:text-dark-900',
                    sidebarCollapsed 
                      ? 'justify-center p-3 h-12 w-12 flex-shrink-0' 
                      : 'gap-x-3 px-3 py-2.5 w-full'
                  )}
                  title={sidebarCollapsed ? item.name : undefined}
                >
                  <Icon className={clsx(
                    'h-5 w-5 flex-shrink-0',
                    isActive ? 'text-white' : 'text-dark-400 group-hover:text-dark-600'
                  )} />
{!sidebarCollapsed && (
                    <>
                      <span>{item.name}</span>
                      {item.id === 'automated-fixes' && (
                        <span className="ml-auto inline-flex items-center rounded-full bg-danger-100 px-2 py-0.5 text-xs font-medium text-danger-700">
                          INACTIVE
                        </span>
                      )}
                    </>
                  )}
                  {sidebarCollapsed && item.id === 'automated-fixes' && (
                    <div className="absolute -top-1 -right-1 h-3 w-3 rounded-full bg-danger-500" title="Automated fixes inactive" />
                  )}
                </button>
              )
            })}
          </nav>

          {/* Footer */}
          {!sidebarCollapsed && (
            <div className="border-t border-dark-200 pt-4">
              <div className="rounded-lg bg-dark-50 p-3">
                <p className="text-xs text-dark-500">API Manager v2.0</p>
                <p className="text-xs text-dark-400 mt-1">© 2024 Vrooli</p>
              </div>
            </div>
          )}
        </div>
      </aside>

      {/* Mobile sidebar backdrop */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/50 backdrop-blur-sm md:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      {/* Main content */}
      <div className="flex flex-1 flex-col">
        {/* Top bar */}
        <header className="sticky top-0 z-30 flex h-16 items-center gap-4 bg-white/80 backdrop-blur-xl border-b border-dark-200 px-4 sm:px-6">
          {/* Mobile menu button or Desktop expand button when collapsed */}
          <button
            className={clsx(
              "flex items-center justify-center h-9 w-9 rounded-lg hover:bg-dark-100 transition-colors",
              sidebarCollapsed ? "block" : "md:hidden"
            )}
            onClick={() => {
              if (window.innerWidth < 768) {
                setSidebarOpen(true)
              } else if (sidebarCollapsed) {
                setSidebarCollapsed(false)
              }
            }}
            title={sidebarCollapsed ? "Expand sidebar" : "Open menu"}
          >
            <Menu className="h-6 w-6 text-dark-600" />
          </button>
          
          <div className="flex flex-1 items-center justify-between">
            <h1 className="text-lg font-semibold text-dark-900">
              {navigation.find(n => n.id === currentView)?.name || 'Dashboard'}
            </h1>
            
            <div className="flex items-center gap-4">
              {/* Quick stats */}
              <div className="hidden sm:flex items-center gap-4">
                <div className="flex items-center gap-2">
                  <div className={clsx(
                    'h-2 w-2 rounded-full animate-pulse',
                    systemStatus?.status === 'healthy' ? 'bg-success-500' :
                    systemStatus?.status === 'degraded' ? 'bg-warning-500' :
                    systemStatus?.status === 'unhealthy' ? 'bg-danger-500' :
                    'bg-dark-400'
                  )}></div>
                  <span className="text-sm text-dark-600">
                    {systemStatusIsFetching ? 'Loading...' :
                     systemStatusIsError ? `Error: ${systemStatusError?.message || 'Unknown error'}` :
                     systemStatus?.status === 'healthy' ? 'System Online' :
                     systemStatus?.status === 'degraded' ? 'System Degraded' :
                     systemStatus?.status === 'unhealthy' ? 'System Unhealthy' :
                     'System Status Unknown'}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </header>

        {/* Page content */}
        <main className="flex-1 overflow-auto p-4 sm:p-6">
          <div className="mx-auto max-w-7xl">
            <Routes>
              <Route path="/" element={<Dashboard />} />
              <Route path="/scenarios" element={<ScenariosList />} />
              <Route path="/scenario/:id" element={<ScenarioDetail />} />
              <Route path="/vulnerabilities" element={<VulnerabilityScanner />} />
              <Route path="/health" element={<HealthMonitor />} />
              <Route path="/performance" element={<PerformanceMetrics />} />
              <Route path="/fixes" element={<AutomatedFixPanel />} />
              <Route path="/settings" element={<SettingsPanel />} />
              <Route path="*" element={<Navigate to="/" replace />} />
            </Routes>
          </div>
        </main>
      </div>
    </div>
  )
}