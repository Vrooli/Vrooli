import { useQuery } from '@tanstack/react-query'
import { 
  Activity,
  Heart,
  TrendingUp,
  TrendingDown,
  AlertTriangle,
  CheckCircle,
  XCircle,
  RefreshCw,
  Clock
} from 'lucide-react'
import clsx from 'clsx'
import { format } from 'date-fns'
import {
  LineChart,
  Line,
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend
} from 'recharts'
import { apiService } from '@/services/api'
import { Card } from './common/Card'
import { Badge } from './common/Badge'

export default function HealthMonitor() {
  const { data: summary, isLoading, refetch } = useQuery({
    queryKey: ['healthSummary'],
    queryFn: apiService.getHealthSummary,
    refetchInterval: 30000,
  })

  const { data: alerts } = useQuery({
    queryKey: ['healthAlerts'],
    queryFn: apiService.getHealthAlerts,
    refetchInterval: 30000,
  })

  const { data: scenarios } = useQuery({
    queryKey: ['scenarios'],
    queryFn: apiService.getScenarios,
  })

  // Mock health history data for the chart
  const healthHistory = [
    { time: '00:00', score: 95, scenarios: 12 },
    { time: '04:00', score: 92, scenarios: 12 },
    { time: '08:00', score: 88, scenarios: 13 },
    { time: '12:00', score: 85, scenarios: 13 },
    { time: '16:00', score: 89, scenarios: 14 },
    { time: '20:00', score: summary?.system_health_score || 91, scenarios: summary?.scenarios?.active || 14 },
  ]

  const getHealthColor = (score: number) => {
    if (score >= 90) return 'text-success-600'
    if (score >= 70) return 'text-warning-600'
    return 'text-danger-600'
  }

  const getHealthBg = (score: number) => {
    if (score >= 90) return 'from-success-500 to-success-600'
    if (score >= 70) return 'from-warning-500 to-warning-600'
    return 'from-danger-500 to-danger-600'
  }

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'healthy': return <CheckCircle className="h-5 w-5 text-success-500" />
      case 'degraded': return <AlertTriangle className="h-5 w-5 text-warning-500" />
      case 'unhealthy': return <XCircle className="h-5 w-5 text-danger-500" />
      default: return null
    }
  }

  return (
    <div className="space-y-6 animate-in">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-dark-900">Health Monitor</h1>
          <p className="text-dark-500 mt-1">Real-time system health and performance metrics</p>
        </div>
        <button
          onClick={() => refetch()}
          className="flex items-center gap-2 rounded-lg bg-white px-4 py-2 text-sm font-medium text-dark-700 shadow-sm border border-dark-200 hover:bg-dark-50 transition-colors"
        >
          <RefreshCw className="h-4 w-4" />
          Refresh
        </button>
      </div>

      {/* Main Health Score */}
      <Card>
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-lg font-semibold text-dark-900">System Health Score</h2>
          <Badge variant={summary?.system_health_score >= 90 ? 'success' : summary?.system_health_score >= 70 ? 'warning' : 'danger'}>
            {summary?.status || 'Unknown'}
          </Badge>
        </div>
        
        <div className="grid gap-6 lg:grid-cols-3">
          {/* Score Display */}
          <div className="flex flex-col items-center justify-center">
            <div className={clsx(
              'relative flex h-32 w-32 items-center justify-center rounded-full bg-gradient-to-br text-white',
              getHealthBg(summary?.system_health_score || 0)
            )}>
              <span className="text-4xl font-bold">
                {summary?.system_health_score !== null && summary?.system_health_score !== undefined ? 
                  Math.round(summary.system_health_score * 10) / 10 : 0}
              </span>
              <span className="absolute -bottom-2 text-xs font-medium bg-white text-dark-700 px-2 py-1 rounded-full shadow-sm">
                / 100
              </span>
            </div>
            <p className="mt-4 text-sm text-dark-600">Overall System Health</p>
          </div>
          
          {/* Health Chart */}
          <div className="lg:col-span-2">
            <ResponsiveContainer width="100%" height={150}>
              <AreaChart data={healthHistory}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                <XAxis dataKey="time" stroke="#6b7280" fontSize={12} />
                <YAxis stroke="#6b7280" fontSize={12} />
                <Tooltip 
                  contentStyle={{ 
                    backgroundColor: 'white',
                    border: '1px solid #e5e7eb',
                    borderRadius: '8px'
                  }}
                />
                <Area 
                  type="monotone" 
                  dataKey="score" 
                  stroke="#0ea5e9" 
                  fill="#0ea5e9" 
                  fillOpacity={0.2}
                  strokeWidth={2}
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>
      </Card>

      {/* Health Breakdown */}
      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-4">
        <Card padding="sm">
          <div className="flex items-center justify-between mb-2">
            <Heart className="h-5 w-5 text-success-500" />
            <span className="text-xs text-dark-500">Healthy</span>
          </div>
          <p className="text-2xl font-bold text-dark-900">{summary?.scenarios?.healthy || 0}</p>
          <p className="text-sm text-dark-600">Scenarios</p>
        </Card>
        
        <Card padding="sm" className="border-warning-200 bg-warning-50/30">
          <div className="flex items-center justify-between mb-2">
            <AlertTriangle className="h-5 w-5 text-warning-500" />
            <span className="text-xs text-dark-500">Degraded</span>
          </div>
          <p className="text-2xl font-bold text-warning-700">{summary?.scenarios?.degraded || 0}</p>
          <p className="text-sm text-warning-600">Scenarios</p>
        </Card>
        
        <Card padding="sm" className="border-danger-200 bg-danger-50/30">
          <div className="flex items-center justify-between mb-2">
            <XCircle className="h-5 w-5 text-danger-500" />
            <span className="text-xs text-dark-500">Unhealthy</span>
          </div>
          <p className="text-2xl font-bold text-danger-700">{summary?.scenarios?.unhealthy || 0}</p>
          <p className="text-sm text-danger-600">Scenarios</p>
        </Card>
        
        <Card padding="sm">
          <div className="flex items-center justify-between mb-2">
            <Activity className="h-5 w-5 text-primary-500" />
            <span className="text-xs text-dark-500">Total</span>
          </div>
          <p className="text-2xl font-bold text-dark-900">{summary?.scenarios?.total || 0}</p>
          <p className="text-sm text-dark-600">Scenarios</p>
        </Card>
      </div>

      {/* Scenarios Health Grid */}
      <div className="grid gap-6 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <h2 className="text-lg font-semibold text-dark-900 mb-4">Scenario Health Status</h2>
          
          {isLoading ? (
            <div className="space-y-3">
              {[1, 2, 3].map(i => (
                <div key={i} className="h-16 bg-dark-100 rounded-lg animate-pulse" />
              ))}
            </div>
          ) : scenarios && scenarios.length > 0 ? (
            <div className="space-y-3">
              {scenarios.map((scenario) => (
                <div
                  key={scenario.name}
                  className="flex items-center justify-between p-3 rounded-lg bg-dark-50 hover:bg-dark-100 transition-colors"
                >
                  <div className="flex items-center gap-3">
                    {getStatusIcon(scenario.status)}
                    <div>
                      <p className="font-medium text-dark-900">{scenario.name}</p>
                      <p className="text-xs text-dark-500">
                        {scenario.endpoint_count} endpoints • {scenario.vulnerability_count} issues
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <div className="text-right">
                      <p className={clsx('text-lg font-semibold', getHealthColor(85))}>
                        85%
                      </p>
                      <p className="text-xs text-dark-500">health</p>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-center text-dark-500 py-8">No scenarios found</p>
          )}
        </Card>

        {/* Active Alerts */}
        <Card>
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-dark-900">Health Alerts</h2>
            <Badge variant="default" size="sm">{alerts?.count || 0}</Badge>
          </div>
          
          {alerts && alerts.alerts.length > 0 ? (
            <div className="space-y-3">
              {alerts.alerts.slice(0, 5).map((alert, idx) => (
                <div
                  key={idx}
                  className={clsx(
                    'p-3 rounded-lg',
                    alert.level === 'critical' && 'bg-danger-50 border-l-4 border-danger-500',
                    alert.level === 'warning' && 'bg-warning-50 border-l-4 border-warning-500',
                    alert.level === 'info' && 'bg-primary-50 border-l-4 border-primary-500'
                  )}
                >
                  <p className="text-sm font-medium text-dark-900">{alert.title}</p>
                  <p className="text-xs text-dark-600 mt-1">{alert.description}</p>
                  <div className="flex items-center gap-2 mt-2">
                    <Clock className="h-3 w-3 text-dark-400" />
                    <p className="text-xs text-dark-500">
                      {format(new Date(alert.created_at), 'HH:mm')}
                    </p>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-8">
              <CheckCircle className="h-12 w-12 text-success-500 mx-auto mb-3" />
              <p className="text-sm text-dark-600">No active health alerts</p>
            </div>
          )}
        </Card>
      </div>

      {/* Vulnerability Breakdown */}
      <Card>
        <h2 className="text-lg font-semibold text-dark-900 mb-4">Vulnerability Overview</h2>
        
        <div className="grid gap-4 sm:grid-cols-4">
          <div className={clsx(
            'text-center p-4 rounded-lg',
            summary?.vulnerabilities?.critical > 0 ? 'bg-danger-50' : 'bg-dark-50'
          )}>
            <p className="text-3xl font-bold text-danger-700">
              {summary?.vulnerabilities?.critical || 0}
            </p>
            <p className="text-sm text-danger-600 font-medium">Critical</p>
          </div>
          
          <div className={clsx(
            'text-center p-4 rounded-lg',
            summary?.vulnerabilities?.high > 0 ? 'bg-warning-50' : 'bg-dark-50'
          )}>
            <p className="text-3xl font-bold text-warning-700">
              {summary?.vulnerabilities?.high || 0}
            </p>
            <p className="text-sm text-warning-600 font-medium">High</p>
          </div>
          
          <div className="text-center p-4 rounded-lg bg-dark-50">
            <p className="text-3xl font-bold text-yellow-700">
              {summary?.vulnerabilities?.medium || 0}
            </p>
            <p className="text-sm text-yellow-600 font-medium">Medium</p>
          </div>
          
          <div className="text-center p-4 rounded-lg bg-dark-50">
            <p className="text-3xl font-bold text-blue-700">
              {summary?.vulnerabilities?.low || 0}
            </p>
            <p className="text-sm text-blue-600 font-medium">Low</p>
          </div>
        </div>
        
        <div className="mt-4 pt-4 border-t border-dark-200 flex items-center justify-between">
          <p className="text-sm text-dark-600">
            Total vulnerabilities: <span className="font-semibold text-dark-900">
              {summary?.vulnerabilities?.total || 0}
            </span>
          </p>
          {summary?.vulnerabilities?.trend && (
            <div className="flex items-center gap-1">
              {summary.vulnerabilities.trend === 'up' ? (
                <TrendingUp className="h-4 w-4 text-danger-500" />
              ) : summary.vulnerabilities.trend === 'down' ? (
                <TrendingDown className="h-4 w-4 text-success-500" />
              ) : null}
              <span className="text-sm text-dark-600">
                {summary.vulnerabilities.trend === 'up' ? 'Increasing' :
                 summary.vulnerabilities.trend === 'down' ? 'Decreasing' : 'Stable'}
              </span>
            </div>
          )}
        </div>
      </Card>
    </div>
  )
}