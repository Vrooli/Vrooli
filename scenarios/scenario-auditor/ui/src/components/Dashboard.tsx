import type { HealthAlert, SecurityScan } from '@/types/api'
import { useQuery } from '@tanstack/react-query'
import clsx from 'clsx'
import { format } from 'date-fns'
import {
  Activity,
  AlertCircle,
  ArrowUpRight,
  CheckCircle,
  GitBranch,
  RefreshCw,
  Server,
  Shield,
  Zap
} from 'lucide-react'
import { Link } from 'react-router-dom'
import { apiService } from '../services/api'
import { Badge } from './common/Badge'
import { Card } from './common/Card'
import type { StatCardProps } from './common/StatCard'
import { StatCard } from './common/StatCard'
import Tooltip from './Tooltip'

export default function Dashboard() {
  const { data: summary, isLoading: loadingSummary, refetch: refetchSummary } = useQuery({
    queryKey: ['healthSummary', 'v4'], // Changed key to clear cache
    queryFn: () => apiService.getHealthSummary(), // Fixed: wrap in arrow function to preserve 'this'
    refetchInterval: 30000,
    staleTime: 0, // Always refetch
    retry: 1, // Reduce retries for faster debugging
  })

  const { data: scenarios, isLoading: loadingScenarios } = useQuery({
    queryKey: ['scenarios'],
    queryFn: () => apiService.getScenarios(), // Fixed: wrap in arrow function
    refetchInterval: 60000,
  })

  const { data: recentScans, isLoading: loadingScans } = useQuery({
    queryKey: ['recentScans'],
    queryFn: () => apiService.getRecentScans(), // Fixed: wrap in arrow function
  })

  const { data: alerts } = useQuery({
    queryKey: ['healthAlerts'],
    queryFn: () => apiService.getHealthAlerts(), // Fixed: wrap in arrow function
    refetchInterval: 30000,
  })

  const { data: testCoverage } = useQuery({
    queryKey: ['testCoverage'],
    queryFn: () => apiService.getTestCoverage(),
    refetchInterval: 60000,
  })

  const hasScans = summary?.scan_status?.has_scans ?? false
  const scanWarningMessage = "No security scans performed yet. Run a scan to assess system health and detect vulnerabilities."
  const alertItems = alerts?.alerts ?? []

  const healthScore = summary?.system_health_score
  const scenariosMetric = typeof summary?.scenarios === 'object' ? summary?.scenarios : undefined
  const vulnerabilitiesMetric = typeof summary?.vulnerabilities === 'object' ? summary?.vulnerabilities : undefined

  const stats: StatCardProps[] = [
    {
      title: 'System Health',
      value: healthScore !== null && healthScore !== undefined ?
        Math.round(healthScore * 10) / 10 : null,
      unit: healthScore !== null && healthScore !== undefined ? '%' : undefined,
      icon: Activity,
      trend: hasScans ? (summary?.health_trend || 'stable') : undefined,
      color: !hasScans ? 'warning' :
        (healthScore ?? 0) >= 80 ? 'success' :
          (healthScore ?? 0) >= 60 ? 'warning' : 'danger',
      hasWarning: !hasScans,
      warningMessage: !hasScans ? scanWarningMessage : undefined,
    },
    {
      title: 'Scenarios',
      value: scenariosMetric?.total ?? (typeof summary?.scenarios === 'number' ? summary.scenarios : 0),
      icon: Server,
      trend: scenariosMetric?.trend,
      color: 'primary',
    },
    {
      title: 'Vulnerabilities',
      value: hasScans ? (
        typeof summary?.vulnerabilities === 'number'
          ? summary.vulnerabilities
          : vulnerabilitiesMetric?.total ?? 0
      ) : null,
      icon: Shield,
      detail: hasScans ? `${summary?.vulnerabilities_detail?.critical || 0} critical` : 'No scans performed',
      trend: hasScans ? (vulnerabilitiesMetric?.trend || 'stable') : undefined,
      color: !hasScans ? 'warning' :
        (summary?.vulnerabilities_detail?.critical || 0) > 0 ? 'danger' : 'success',
      hasWarning: !hasScans,
      warningMessage: !hasScans ? scanWarningMessage : undefined,
    },
    {
      title: 'API Endpoints',
      value: summary?.endpoints?.total ?? 0,
      icon: GitBranch,
      detail: `${summary?.endpoints?.monitored ?? 0} monitored`,
      color: 'primary',
    },
    {
      title: 'Test Coverage',
      value: testCoverage?.coverage_metrics?.coverage_percentage ?
        Math.round(testCoverage.coverage_metrics.coverage_percentage) : 0,
      unit: '%',
      icon: CheckCircle,
      detail: testCoverage ?
        `${testCoverage.coverage_metrics?.rules_with_tests || 0}/${testCoverage.coverage_metrics?.total_rules || 0} rules` :
        'Loading...',
      color: testCoverage?.coverage_metrics?.coverage_percentage >= 80 ? 'success' :
        testCoverage?.coverage_metrics?.coverage_percentage >= 60 ? 'warning' : 'danger',
    },
  ]

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
          <h1 className="text-2xl font-bold text-dark-900">Standards Enforcement Dashboard</h1>
          <p className="text-dark-500 mt-1">Monitor standards compliance and quality assurance</p>
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
            <h2 className="text-lg font-semibold text-dark-900">Scenarios</h2>
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
                          scenario.last_scan && !isNaN(new Date(scenario.last_scan).getTime())
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

          {alertItems.length > 0 ? (
            <div className="space-y-3">
              {alertItems.slice(0, 5).map((alert) => {
                const extendedAlert = alert as HealthAlert & {
                  scenarios?: string[]
                  total_count?: number
                  message?: string
                  action?: string
                }
                const alertScenarios = Array.isArray(extendedAlert.scenarios) ? extendedAlert.scenarios : []
                const totalScenarioCount = typeof extendedAlert.total_count === 'number'
                  ? extendedAlert.total_count
                  : alertScenarios.length

                return (
                  <div
                    key={extendedAlert.id}
                    className={clsx(
                      'p-3 rounded-lg border-l-4',
                      extendedAlert.level === 'critical' && 'bg-danger-50 border-danger-500',
                      extendedAlert.level === 'warning' && 'bg-warning-50 border-warning-500',
                      extendedAlert.level === 'info' && 'bg-primary-50 border-primary-500'
                    )}
                  >
                    <div className="flex items-start gap-2">
                      <AlertCircle className={clsx(
                        'h-4 w-4 mt-0.5',
                        extendedAlert.level === 'critical' && 'text-danger-600',
                        extendedAlert.level === 'warning' && 'text-warning-600',
                        extendedAlert.level === 'info' && 'text-primary-600'
                      )} />
                      <div className="flex-1">
                        <div className="flex items-center gap-2">
                          <p className="text-sm font-medium text-dark-900">{extendedAlert.title}</p>
                          <Tooltip
                            content={
                              extendedAlert.title.includes('Stale Scenarios')
                                ? "Scenarios that haven't been scanned for security vulnerabilities in the last 48 hours. Regular scanning helps identify new vulnerabilities early."
                                : extendedAlert.title.includes('Critical Vulnerabilities')
                                  ? "High-severity security issues that could be exploited by attackers. These should be addressed immediately."
                                  : extendedAlert.title.includes('Automated Fixes')
                                    ? "The system can automatically apply fixes for certain types of vulnerabilities when this feature is enabled."
                                    : extendedAlert.message || extendedAlert.description
                            }
                            position="right"
                          />
                        </div>
                        <p className="text-xs text-dark-600 mt-0.5">{extendedAlert.message || extendedAlert.description}</p>

                        {/* Show stale scenario links */}
                        {alertScenarios.length > 0 && (
                          <div className="mt-2 flex flex-wrap gap-1">
                            {alertScenarios.map((scenario) => (
                              <Link
                                key={scenario}
                                to={`/scenario/${scenario}`}
                                className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs bg-white border border-dark-200 text-dark-700 hover:bg-dark-50 hover:border-dark-300 transition-colors"
                              >
                                {scenario}
                                <ArrowUpRight className="h-3 w-3" />
                              </Link>
                            ))}
                            {totalScenarioCount > alertScenarios.length && (
                              <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs bg-dark-100 text-dark-600">
                                +{totalScenarioCount - alertScenarios.length} more
                              </span>
                            )}
                          </div>
                        )}

                        {extendedAlert.action && (
                          <p className="text-xs text-dark-500 mt-1 italic">{extendedAlert.action}</p>
                        )}
                        {extendedAlert.created_at && (
                          <p className="text-xs text-dark-500 mt-1">
                            {format(new Date(extendedAlert.created_at), 'MMM d, HH:mm')}
                          </p>
                        )}
                      </div>
                    </div>
                  </div>
                )
              })}
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

      {/* Test Coverage Details */}
      {testCoverage && (
        <Card className="lg:col-span-2">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-dark-900">Test Coverage Details</h2>
            <Link
              to="/rules"
              className="text-sm text-primary-600 hover:text-primary-700 font-medium flex items-center gap-1"
            >
              View All Rules
              <ArrowUpRight className="h-3 w-3" />
            </Link>
          </div>

          <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
            <div className="text-center">
              <p className="text-2xl font-bold text-dark-900">
                {testCoverage.coverage_metrics?.total_test_cases || 0}
              </p>
              <p className="text-xs text-dark-600">Total Test Cases</p>
            </div>
            <div className="text-center">
              <p className="text-2xl font-bold text-success-600">
                {testCoverage.test_results?.passing || 0}
              </p>
              <p className="text-xs text-dark-600">Passing Tests</p>
            </div>
            <div className="text-center">
              <p className="text-2xl font-bold text-danger-600">
                {testCoverage.test_results?.failing || 0}
              </p>
              <p className="text-xs text-dark-600">Failing Tests</p>
            </div>
            <div className="text-center">
              <p className="text-2xl font-bold text-primary-600">
                {testCoverage.coverage_metrics?.average_tests_per_rule?.toFixed(1) || 0}
              </p>
              <p className="text-xs text-dark-600">Avg Tests/Rule</p>
            </div>
          </div>

          {testCoverage.category_breakdown && (
            <div className="mt-4 pt-4 border-t border-dark-200">
              <p className="text-sm font-medium text-dark-700 mb-2">Coverage by Category</p>
              <div className="space-y-2">
                {Object.entries(testCoverage.category_breakdown).map(([category, stats]: [string, any]) => (
                  <div key={category} className="flex items-center justify-between text-sm">
                    <span className="text-dark-600 capitalize">{category}</span>
                    <div className="flex items-center gap-2">
                      <span className="text-dark-900 font-medium">
                        {stats.with_tests}/{stats.total} rules
                      </span>
                      <span className={clsx(
                        'text-xs px-2 py-0.5 rounded-full',
                        (stats.with_tests / stats.total) >= 0.8 ? 'bg-success-100 text-success-700' :
                          (stats.with_tests / stats.total) >= 0.6 ? 'bg-warning-100 text-warning-700' :
                            'bg-danger-100 text-danger-700'
                      )}>
                        {Math.round((stats.with_tests / stats.total) * 100)}%
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </Card>
      )}

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
        ) : (recentScans?.length ?? 0) > 0 ? (
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
                {(recentScans as SecurityScan[]).map((scan) => (
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
                      {scan.completed_at && !isNaN(new Date(scan.completed_at).getTime())
                        ? format(new Date(scan.completed_at), 'MMM d, HH:mm')
                        : 'N/A'}
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
