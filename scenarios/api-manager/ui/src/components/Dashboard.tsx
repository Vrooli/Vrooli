import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { 
  Shield,
  TrendingUp,
  AlertCircle,
  CheckCircle,
  Activity,
  Server,
  Zap,
  GitBranch,
  ArrowUpRight,
  RefreshCw
} from 'lucide-react'
import clsx from 'clsx'
import { format } from 'date-fns'
import { apiService } from '@/services/api'
import { StatCard } from './common/StatCard'
import { Card } from './common/Card'
import { Badge } from './common/Badge'

export default function Dashboard() {
  const { data: summary, isLoading: loadingSummary, refetch: refetchSummary } = useQuery({
    queryKey: ['healthSummary'],
    queryFn: apiService.getHealthSummary,
    refetchInterval: 30000,
  })

  const { data: scenarios, isLoading: loadingScenarios } = useQuery({
    queryKey: ['scenarios'],
    queryFn: apiService.getScenarios,
    refetchInterval: 60000,
  })

  const { data: recentScans, isLoading: loadingScans } = useQuery({
    queryKey: ['recentScans'],
    queryFn: apiService.getRecentScans,
  })

  const { data: alerts } = useQuery({
    queryKey: ['healthAlerts'],
    queryFn: apiService.getHealthAlerts,
    refetchInterval: 30000,
  })

  const stats = [
    {
      title: 'System Health',
      value: summary?.system_health_score || 0,
      unit: '%',
      icon: Activity,
      trend: summary?.health_trend || 'stable',
      color: summary?.system_health_score >= 80 ? 'success' : 
              summary?.system_health_score >= 60 ? 'warning' : 'danger',
    },
    {
      title: 'Active Scenarios',
      value: summary?.scenarios?.active || 0,
      icon: Server,
      trend: 'up',
      color: 'primary',
    },
    {
      title: 'Vulnerabilities',
      value: summary?.vulnerabilities?.total || 0,
      icon: Shield,
      detail: `${summary?.vulnerabilities?.critical || 0} critical`,
      trend: summary?.vulnerabilities?.trend || 'stable',
      color: summary?.vulnerabilities?.critical > 0 ? 'danger' : 'warning',
    },
    {
      title: 'API Endpoints',
      value: summary?.endpoints?.total || 0,
      icon: GitBranch,
      detail: `${summary?.endpoints?.monitored || 0} monitored`,
      color: 'primary',
    },
  ]

  const getSeverityColor = (severity: string) => {
    switch (severity.toLowerCase()) {
      case 'critical': return 'danger'
      case 'high': return 'warning'
      case 'medium': return 'warning'
      case 'low': return 'info'
      default: return 'default'
    }
  }

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'healthy': return <CheckCircle className="h-4 w-4 text-success-500" />
      case 'degraded': return <AlertCircle className="h-4 w-4 text-warning-500" />
      case 'unhealthy': return <AlertCircle className="h-4 w-4 text-danger-500" />
      default: return null
    }
  }

  return (
    <div className="space-y-6 animate-in">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-dark-900">API Security Dashboard</h1>
          <p className="text-dark-500 mt-1">Monitor and manage your API ecosystem</p>
        </div>
        <button
          onClick={() => refetchSummary()}
          className="flex items-center gap-2 rounded-lg bg-white px-4 py-2 text-sm font-medium text-dark-700 shadow-sm border border-dark-200 hover:bg-dark-50 transition-colors"
        >
          <RefreshCw className="h-4 w-4" />
          Refresh
        </button>
      </div>

      {/* Stats Grid */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {stats.map((stat) => (
          <StatCard key={stat.title} {...stat} loading={loadingSummary} />
        ))}
      </div>

      {/* Main Content Grid */}
      <div className="grid gap-6 lg:grid-cols-3">
        {/* Scenarios List */}
        <Card className="lg:col-span-2">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-dark-900">Active Scenarios</h2>
            <Link
              to="/scenarios"
              className="text-sm text-primary-600 hover:text-primary-700 font-medium flex items-center gap-1"
            >
              View all
              <ArrowUpRight className="h-3 w-3" />
            </Link>
          </div>
          
          {loadingScenarios ? (
            <div className="space-y-3">
              {[1, 2, 3].map(i => (
                <div key={i} className="h-20 bg-dark-100 rounded-lg animate-pulse" />
              ))}
            </div>
          ) : (
            <div className="space-y-3">
              {scenarios?.slice(0, 5).map((scenario: any) => (
                <Link
                  key={scenario.name}
                  to={`/scenario/${scenario.name}`}
                  className="flex items-center justify-between p-4 rounded-lg bg-dark-50 hover:bg-dark-100 transition-colors group"
                >
                  <div className="flex items-center gap-3">
                    {getStatusIcon(scenario.status)}
                    <div>
                      <p className="font-medium text-dark-900 group-hover:text-primary-600">
                        {scenario.name}
                      </p>
                      <p className="text-sm text-dark-500">
                        {scenario.endpoint_count || 0} endpoints • Last scan: {
                          scenario.last_scan 
                            ? format(new Date(scenario.last_scan), 'MMM d, HH:mm')
                            : 'Never'
                        }
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    {scenario.vulnerability_count > 0 && (
                      <Badge variant={scenario.critical_count > 0 ? 'danger' : 'warning'}>
                        {scenario.vulnerability_count} issues
                      </Badge>
                    )}
                    <ArrowUpRight className="h-4 w-4 text-dark-400 group-hover:text-primary-600" />
                  </div>
                </Link>
              ))}
            </div>
          )}
        </Card>

        {/* Alerts Panel */}
        <Card>
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-dark-900">Active Alerts</h2>
            <span className="text-sm text-dark-500">{alerts?.count || 0} total</span>
          </div>
          
          {alerts?.alerts?.length > 0 ? (
            <div className="space-y-3">
              {alerts.alerts.slice(0, 5).map((alert: any, idx: number) => (
                <div
                  key={idx}
                  className={clsx(
                    'p-3 rounded-lg border-l-4',
                    alert.level === 'critical' && 'bg-danger-50 border-danger-500',
                    alert.level === 'warning' && 'bg-warning-50 border-warning-500',
                    alert.level === 'info' && 'bg-primary-50 border-primary-500'
                  )}
                >
                  <div className="flex items-start gap-2">
                    <AlertCircle className={clsx(
                      'h-4 w-4 mt-0.5',
                      alert.level === 'critical' && 'text-danger-600',
                      alert.level === 'warning' && 'text-warning-600',
                      alert.level === 'info' && 'text-primary-600'
                    )} />
                    <div className="flex-1">
                      <p className="text-sm font-medium text-dark-900">{alert.title}</p>
                      <p className="text-xs text-dark-600 mt-0.5">{alert.description}</p>
                      <p className="text-xs text-dark-500 mt-1">
                        {format(new Date(alert.created_at), 'MMM d, HH:mm')}
                      </p>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-8">
              <CheckCircle className="h-12 w-12 text-success-500 mx-auto mb-3" />
              <p className="text-sm text-dark-600">No active alerts</p>
              <p className="text-xs text-dark-500 mt-1">System is operating normally</p>
            </div>
          )}
        </Card>
      </div>

      {/* Recent Scans */}
      <Card>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-dark-900">Recent Security Scans</h2>
          <Link
            to="/vulnerabilities"
            className="text-sm text-primary-600 hover:text-primary-700 font-medium flex items-center gap-1"
          >
            Run scan
            <Zap className="h-3 w-3" />
          </Link>
        </div>
        
        {loadingScans ? (
          <div className="h-32 bg-dark-100 rounded-lg animate-pulse" />
        ) : recentScans?.length > 0 ? (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-dark-200">
                  <th className="text-left text-xs font-medium text-dark-500 uppercase tracking-wider pb-2">
                    Scenario
                  </th>
                  <th className="text-left text-xs font-medium text-dark-500 uppercase tracking-wider pb-2">
                    Type
                  </th>
                  <th className="text-left text-xs font-medium text-dark-500 uppercase tracking-wider pb-2">
                    Findings
                  </th>
                  <th className="text-left text-xs font-medium text-dark-500 uppercase tracking-wider pb-2">
                    Status
                  </th>
                  <th className="text-left text-xs font-medium text-dark-500 uppercase tracking-wider pb-2">
                    Time
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-dark-100">
                {recentScans.map((scan: any) => (
                  <tr key={scan.id} className="hover:bg-dark-50">
                    <td className="py-3 text-sm font-medium text-dark-900">
                      {scan.scenario_name}
                    </td>
                    <td className="py-3 text-sm text-dark-600">
                      {scan.scan_type}
                    </td>
                    <td className="py-3">
                      <div className="flex items-center gap-2">
                        {scan.vulnerabilities?.critical > 0 && (
                          <Badge variant="danger" size="sm">
                            {scan.vulnerabilities.critical} critical
                          </Badge>
                        )}
                        {scan.vulnerabilities?.high > 0 && (
                          <Badge variant="warning" size="sm">
                            {scan.vulnerabilities.high} high
                          </Badge>
                        )}
                        {scan.vulnerabilities?.total === 0 && (
                          <Badge variant="success" size="sm">
                            Clean
                          </Badge>
                        )}
                      </div>
                    </td>
                    <td className="py-3">
                      <Badge variant={scan.status === 'completed' ? 'success' : 'default'}>
                        {scan.status}
                      </Badge>
                    </td>
                    <td className="py-3 text-sm text-dark-500">
                      {format(new Date(scan.completed_at), 'MMM d, HH:mm')}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="text-center py-8">
            <Shield className="h-12 w-12 text-dark-300 mx-auto mb-3" />
            <p className="text-sm text-dark-600">No recent scans</p>
            <p className="text-xs text-dark-500 mt-1">Run a security scan to get started</p>
          </div>
        )}
      </Card>
    </div>
  )
}