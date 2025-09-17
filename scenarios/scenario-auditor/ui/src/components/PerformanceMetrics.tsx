import { useState } from 'react'
import { useQuery, useMutation } from '@tanstack/react-query'
import { 
  Zap, 
  TrendingUp,
  Clock,
  Gauge,
  BarChart3,
  Activity,
  RefreshCw,
  Play,
  AlertCircle
} from 'lucide-react'
import {
  LineChart,
  Line,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend
} from 'recharts'
import { format } from 'date-fns'
import { apiService } from '../services/api'
import { Card } from './common/Card'
import { Badge } from './common/Badge'
import clsx from 'clsx'

export default function PerformanceMetrics() {
  const [selectedScenario, setSelectedScenario] = useState<string>('')
  const [baselineConfig, setBaselineConfig] = useState({
    duration_seconds: 60,
    load_level: 'light' as 'light' | 'moderate' | 'heavy',
  })

  const { data: scenarios } = useQuery({
    queryKey: ['scenarios'],
    queryFn: apiService.getScenarios,
  })

  const { data: metrics, refetch: refetchMetrics } = useQuery({
    queryKey: ['performanceMetrics', selectedScenario],
    queryFn: () => apiService.getPerformanceMetrics(selectedScenario),
    enabled: !!selectedScenario,
  })

  const { data: performanceAlerts } = useQuery({
    queryKey: ['performanceAlerts'],
    queryFn: apiService.getPerformanceAlerts,
    refetchInterval: 60000,
  })

  const baselineMutation = useMutation({
    mutationFn: (scenario: string) => 
      apiService.createPerformanceBaseline(scenario, baselineConfig),
    onSuccess: () => {
      refetchMetrics()
    },
  })

  // Mock performance data
  const responseTimeData = [
    { time: '00:00', p50: 45, p95: 120, p99: 180 },
    { time: '04:00', p50: 48, p95: 125, p99: 190 },
    { time: '08:00', p50: 52, p95: 135, p99: 210 },
    { time: '12:00', p50: 58, p95: 145, p99: 230 },
    { time: '16:00', p50: 55, p95: 140, p99: 220 },
    { time: '20:00', p50: 50, p95: 130, p99: 200 },
  ]

  const throughputData = [
    { endpoint: '/api/v1/health', requests: 1250 },
    { endpoint: '/api/v1/scenarios', requests: 980 },
    { endpoint: '/api/v1/scan', requests: 450 },
    { endpoint: '/api/v1/vulnerabilities', requests: 780 },
    { endpoint: '/api/v1/fix/config', requests: 120 },
  ]

  const performanceScore = 87

  return (
    <div className="space-y-6 animate-in">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-dark-900">Performance Metrics</h1>
          <p className="text-dark-500 mt-1">Monitor API performance and create baselines</p>
        </div>
        <button
          onClick={() => refetchMetrics()}
          className="flex items-center gap-2 rounded-lg bg-white px-4 py-2 text-sm font-medium text-dark-700 shadow-sm border border-dark-200 hover:bg-dark-50 transition-colors"
        >
          <RefreshCw className="h-4 w-4" />
          Refresh
        </button>
      </div>

      {/* Performance Score Overview */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card padding="sm">
          <div className="flex items-center justify-between mb-2">
            <Gauge className="h-5 w-5 text-primary-500" />
            <Badge variant={performanceScore >= 80 ? 'success' : performanceScore >= 60 ? 'warning' : 'danger'} size="sm">
              {performanceScore >= 80 ? 'Excellent' : performanceScore >= 60 ? 'Good' : 'Poor'}
            </Badge>
          </div>
          <p className="text-3xl font-bold text-dark-900">{performanceScore}</p>
          <p className="text-sm text-dark-600">Performance Score</p>
        </Card>
        
        <Card padding="sm">
          <div className="flex items-center justify-between mb-2">
            <Clock className="h-5 w-5 text-blue-500" />
          </div>
          <p className="text-3xl font-bold text-dark-900">52<span className="text-lg text-dark-500">ms</span></p>
          <p className="text-sm text-dark-600">Avg Response Time</p>
        </Card>
        
        <Card padding="sm">
          <div className="flex items-center justify-between mb-2">
            <TrendingUp className="h-5 w-5 text-success-500" />
          </div>
          <p className="text-3xl font-bold text-dark-900">1.2k<span className="text-lg text-dark-500">/s</span></p>
          <p className="text-sm text-dark-600">Throughput</p>
        </Card>
        
        <Card padding="sm">
          <div className="flex items-center justify-between mb-2">
            <Activity className="h-5 w-5 text-warning-500" />
          </div>
          <p className="text-3xl font-bold text-dark-900">99.9<span className="text-lg text-dark-500">%</span></p>
          <p className="text-sm text-dark-600">Uptime</p>
        </Card>
      </div>

      {/* Baseline Configuration */}
      <Card>
        <h2 className="text-lg font-semibold text-dark-900 mb-4">Performance Baseline</h2>
        
        <div className="flex flex-col sm:flex-row gap-4">
          <div className="flex-1">
            <label className="block text-sm font-medium text-dark-700 mb-2">
              Scenario
            </label>
            <select
              value={selectedScenario}
              onChange={(e) => setSelectedScenario(e.target.value)}
              className="w-full rounded-lg border border-dark-300 bg-white px-4 py-2 text-dark-900 focus:border-primary-500 focus:ring-2 focus:ring-primary-500/20"
            >
              <option value="">Select a scenario</option>
              {scenarios?.map((scenario) => (
                <option key={scenario.name} value={scenario.name}>
                  {scenario.name}
                </option>
              ))}
            </select>
          </div>
          
          <div>
            <label className="block text-sm font-medium text-dark-700 mb-2">
              Duration
            </label>
            <select
              value={baselineConfig.duration_seconds}
              onChange={(e) => setBaselineConfig(prev => ({ 
                ...prev, 
                duration_seconds: parseInt(e.target.value) 
              }))}
              className="rounded-lg border border-dark-300 bg-white px-4 py-2 text-dark-900 focus:border-primary-500 focus:ring-2 focus:ring-primary-500/20"
            >
              <option value="60">1 minute</option>
              <option value="300">5 minutes</option>
              <option value="600">10 minutes</option>
              <option value="1800">30 minutes</option>
            </select>
          </div>
          
          <div>
            <label className="block text-sm font-medium text-dark-700 mb-2">
              Load Level
            </label>
            <div className="flex gap-2">
              {(['light', 'moderate', 'heavy'] as const).map((level) => (
                <button
                  key={level}
                  onClick={() => setBaselineConfig(prev => ({ ...prev, load_level: level }))}
                  className={clsx(
                    'px-4 py-2 rounded-lg text-sm font-medium capitalize transition-colors',
                    baselineConfig.load_level === level
                      ? 'bg-primary-500 text-white'
                      : 'bg-dark-100 text-dark-700 hover:bg-dark-200'
                  )}
                >
                  {level}
                </button>
              ))}
            </div>
          </div>
          
          <div className="flex items-end">
            <button
              onClick={() => selectedScenario && baselineMutation.mutate(selectedScenario)}
              disabled={!selectedScenario || baselineMutation.isPending}
              className="flex items-center gap-2 rounded-lg bg-primary-500 px-4 py-2 text-sm font-medium text-white hover:bg-primary-600 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {baselineMutation.isPending ? (
                <>
                  <RefreshCw className="h-4 w-4 animate-spin" />
                  Creating...
                </>
              ) : (
                <>
                  <Play className="h-4 w-4" />
                  Create Baseline
                </>
              )}
            </button>
          </div>
        </div>
        
        {baselineMutation.isSuccess && (
          <div className="mt-4 flex items-center gap-2 rounded-lg bg-success-50 p-3 text-success-700">
            <Zap className="h-4 w-4" />
            <span className="text-sm font-medium">Performance baseline created successfully</span>
          </div>
        )}
      </Card>

      {/* Charts */}
      <div className="grid gap-6 lg:grid-cols-2">
        {/* Response Time Chart */}
        <Card>
          <h3 className="text-lg font-semibold text-dark-900 mb-4">Response Time Percentiles</h3>
          <ResponsiveContainer width="100%" height={250}>
            <LineChart data={responseTimeData}>
              <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
              <XAxis dataKey="time" stroke="#6b7280" fontSize={12} />
              <YAxis stroke="#6b7280" fontSize={12} label={{ value: 'ms', angle: -90, position: 'insideLeft' }} />
              <Tooltip
                contentStyle={{
                  backgroundColor: 'white',
                  border: '1px solid #e5e7eb',
                  borderRadius: '8px'
                }}
              />
              <Legend />
              <Line type="monotone" dataKey="p50" stroke="#22c55e" strokeWidth={2} name="P50" />
              <Line type="monotone" dataKey="p95" stroke="#f59e0b" strokeWidth={2} name="P95" />
              <Line type="monotone" dataKey="p99" stroke="#ef4444" strokeWidth={2} name="P99" />
            </LineChart>
          </ResponsiveContainer>
        </Card>

        {/* Throughput Chart */}
        <Card>
          <h3 className="text-lg font-semibold text-dark-900 mb-4">Endpoint Throughput</h3>
          <ResponsiveContainer width="100%" height={250}>
            <BarChart data={throughputData}>
              <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
              <XAxis dataKey="endpoint" stroke="#6b7280" fontSize={10} angle={-45} textAnchor="end" height={80} />
              <YAxis stroke="#6b7280" fontSize={12} label={{ value: 'Requests', angle: -90, position: 'insideLeft' }} />
              <Tooltip
                contentStyle={{
                  backgroundColor: 'white',
                  border: '1px solid #e5e7eb',
                  borderRadius: '8px'
                }}
              />
              <Bar dataKey="requests" fill="#0ea5e9" radius={[8, 8, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </Card>
      </div>

      {/* Performance Alerts */}
      <Card>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-dark-900">Performance Alerts</h2>
          <Badge variant="default">{performanceAlerts?.count || 0} active</Badge>
        </div>
        
        {performanceAlerts && performanceAlerts.alerts.length > 0 ? (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-dark-200">
                  <th className="text-left text-xs font-medium text-dark-500 uppercase tracking-wider pb-2">
                    Alert
                  </th>
                  <th className="text-left text-xs font-medium text-dark-500 uppercase tracking-wider pb-2">
                    Scenario
                  </th>
                  <th className="text-left text-xs font-medium text-dark-500 uppercase tracking-wider pb-2">
                    Severity
                  </th>
                  <th className="text-left text-xs font-medium text-dark-500 uppercase tracking-wider pb-2">
                    Threshold
                  </th>
                  <th className="text-left text-xs font-medium text-dark-500 uppercase tracking-wider pb-2">
                    Time
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-dark-100">
                {performanceAlerts.alerts.map((alert: any, idx: number) => (
                  <tr key={idx} className="hover:bg-dark-50">
                    <td className="py-3">
                      <div className="flex items-center gap-2">
                        <AlertCircle className={clsx(
                          'h-4 w-4',
                          alert.severity === 'high' && 'text-danger-500',
                          alert.severity === 'medium' && 'text-warning-500',
                          alert.severity === 'low' && 'text-blue-500'
                        )} />
                        <span className="text-sm font-medium text-dark-900">{alert.title}</span>
                      </div>
                    </td>
                    <td className="py-3 text-sm text-dark-600">
                      {alert.scenario}
                    </td>
                    <td className="py-3">
                      <Badge 
                        variant={
                          alert.severity === 'high' ? 'danger' :
                          alert.severity === 'medium' ? 'warning' :
                          'info'
                        }
                        size="sm"
                      >
                        {alert.severity}
                      </Badge>
                    </td>
                    <td className="py-3 text-sm text-dark-600">
                      {alert.threshold}
                    </td>
                    <td className="py-3 text-sm text-dark-500">
                      {alert.created_at && !isNaN(new Date(alert.created_at).getTime())
                        ? format(new Date(alert.created_at), 'MMM d, HH:mm')
                        : 'N/A'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="text-center py-8">
            <Zap className="h-12 w-12 text-success-500 mx-auto mb-3" />
            <p className="text-sm text-dark-600">No performance alerts</p>
            <p className="text-xs text-dark-500 mt-1">System is performing within expected parameters</p>
          </div>
        )}
      </Card>
    </div>
  )
}