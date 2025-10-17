import { useQuery } from '@tanstack/react-query'
import {
  AlertTriangle,
  BarChart3,
  CheckCircle,
  Clock,
  FileText,
  Shield,
  TrendingUp,
  Zap
} from 'lucide-react'
import { apiService } from '../services/api'

interface DashboardOverview {
  total_scenarios: number
  scenarios_scanned: number
  total_violations: number
  critical_violations: number
  auto_fixable: number
  average_score: number
  rules_enabled: number
  rules_total: number
}

interface DashboardScanSummary {
  total_violations: number
  score: number
}

interface DashboardScan {
  id: string
  scenario_name: string
  summary: DashboardScanSummary
  status: string
  end_time: string
}

interface DashboardResponse {
  overview?: DashboardOverview
  recent_scans?: DashboardScan[]
  recommendations?: string[]
}

export default function ScanDashboard() {
  const { data: dashboardData, isLoading, error } = useQuery<DashboardResponse>({
    queryKey: ['dashboard'],
    queryFn: () => apiService.getDashboard(),
    refetchInterval: 30000, // Refresh every 30 seconds
  })

  if (isLoading) {
    return (
      <div className="px-6">
        <div className="animate-pulse">
          <div className="h-8 bg-gray-200 rounded w-1/4 mb-6"></div>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
            {[...Array(4)].map((_, i) => (
              <div key={i} className="bg-white p-6 rounded-lg shadow">
                <div className="h-4 bg-gray-200 rounded w-3/4 mb-2"></div>
                <div className="h-8 bg-gray-200 rounded w-1/2"></div>
              </div>
            ))}
          </div>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="px-6">
        <div className="bg-red-50 border border-red-200 rounded-md p-4">
          <div className="flex">
            <AlertTriangle className="h-5 w-5 text-red-400" />
            <div className="ml-3">
              <h3 className="text-sm font-medium text-red-800">
                Failed to load dashboard data
              </h3>
              <div className="mt-2 text-sm text-red-700">
                {error instanceof Error ? error.message : 'Unknown error occurred'}
              </div>
            </div>
          </div>
        </div>
      </div>
    )
  }

  const overview: DashboardOverview = dashboardData?.overview || {
    total_scenarios: 0,
    scenarios_scanned: 0,
    total_violations: 0,
    critical_violations: 0,
    auto_fixable: 0,
    average_score: 0,
    rules_enabled: 0,
    rules_total: 0,
  }

  const recentScans: DashboardScan[] = dashboardData?.recent_scans ?? []
  const recommendations: string[] = dashboardData?.recommendations ?? []
  const scannedPercent = overview.total_scenarios > 0
    ? Math.round((overview.scenarios_scanned / overview.total_scenarios) * 100)
    : 0

  return (
    <div className="px-6">
      {/* Header */}
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-900" data-testid="dashboard-title">
          Standards Dashboard
        </h1>
        <p className="mt-1 text-sm text-gray-500">
          Overview of scenario compliance and quality metrics
        </p>
      </div>

      {/* Quick Actions */}
      <div className="mb-6 flex space-x-4">
        <button
          className="btn-primary"
          data-testid="scan-all-btn"
          onClick={() => {
            // TODO: Implement scan all functionality
            console.log('Scan all scenarios')
          }}
        >
          <Zap className="h-4 w-4 mr-2" />
          Scan All Scenarios
        </button>
        <button
          className="btn-secondary"
          data-testid="scan-current-btn"
          onClick={() => {
            // TODO: Implement scan current functionality  
            console.log('Scan current scenario')
          }}
        >
          <Shield className="h-4 w-4 mr-2" />
          Scan Current
        </button>
      </div>

      {/* Overview Stats */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        <div className="stat-card" data-testid="scenarios-stat">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <FileText className="h-8 w-8 text-blue-600" />
            </div>
            <div className="ml-5 w-0 flex-1">
              <dl>
                <dt className="text-sm font-medium text-gray-500 truncate">
                  Scenarios
                </dt>
                <dd className="text-lg font-medium text-gray-900">
                  {overview.scenarios_scanned} / {overview.total_scenarios}
                </dd>
              </dl>
            </div>
          </div>
          <div className="mt-4">
            <div className="text-sm text-gray-500">
              {scannedPercent}% scanned
            </div>
          </div>
        </div>

        <div className="stat-card" data-testid="violations-stat">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <AlertTriangle className="h-8 w-8 text-orange-600" />
            </div>
            <div className="ml-5 w-0 flex-1">
              <dl>
                <dt className="text-sm font-medium text-gray-500 truncate">
                  Violations
                </dt>
                <dd className="text-lg font-medium text-gray-900">
                  {overview.total_violations}
                </dd>
              </dl>
            </div>
          </div>
          <div className="mt-4">
            <div className="text-sm text-red-600">
              {overview.critical_violations} critical
            </div>
          </div>
        </div>

        <div className="stat-card" data-testid="score-stat">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <BarChart3 className="h-8 w-8 text-green-600" />
            </div>
            <div className="ml-5 w-0 flex-1">
              <dl>
                <dt className="text-sm font-medium text-gray-500 truncate">
                  Average Score
                </dt>
                <dd className="text-lg font-medium text-gray-900">
                  {overview.average_score.toFixed(1)}/100
                </dd>
              </dl>
            </div>
          </div>
          <div className="mt-4">
            <div className={`text-sm ${overview.average_score >= 90 ? 'text-green-600' :
                overview.average_score >= 70 ? 'text-yellow-600' : 'text-red-600'
              }`}>
              {overview.average_score >= 90 ? 'Excellent' :
                overview.average_score >= 70 ? 'Good' : 'Needs Improvement'}
            </div>
          </div>
        </div>

        <div className="stat-card" data-testid="rules-stat">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <Shield className="h-8 w-8 text-purple-600" />
            </div>
            <div className="ml-5 w-0 flex-1">
              <dl>
                <dt className="text-sm font-medium text-gray-500 truncate">
                  Active Rules
                </dt>
                <dd className="text-lg font-medium text-gray-900">
                  {overview.rules_enabled} / {overview.rules_total}
                </dd>
              </dl>
            </div>
          </div>
          <div className="mt-4">
            <div className="text-sm text-gray-500">
              {overview.auto_fixable} auto-fixable violations
            </div>
          </div>
        </div>
      </div>

      {/* Recent Scans */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        <div className="card" data-testid="recent-scans">
          <div className="card-header">
            <h2 className="text-lg font-medium text-gray-900">Recent Scans</h2>
          </div>
          <div className="card-body">
            {recentScans.length > 0 ? (
              <div className="space-y-4">
                {recentScans.slice(0, 5).map((scan: DashboardScan) => (
                  <div key={scan.id} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                    <div>
                      <div className="font-medium text-gray-900">{scan.scenario_name}</div>
                      <div className="text-sm text-gray-500">
                        {scan.summary.total_violations} violations • Score: {scan.summary.score.toFixed(1)}
                      </div>
                    </div>
                    <div className="text-right">
                      <div className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${scan.status === 'completed' ? 'bg-green-100 text-green-800' : 'bg-yellow-100 text-yellow-800'
                        }`}>
                        {scan.status}
                      </div>
                      <div className="text-xs text-gray-500 mt-1">
                        {new Date(scan.end_time).toLocaleTimeString()}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="text-center py-6">
                <Clock className="mx-auto h-12 w-12 text-gray-400" />
                <h3 className="mt-2 text-sm font-medium text-gray-900">No recent scans</h3>
                <p className="mt-1 text-sm text-gray-500">Start by scanning a scenario to see results here.</p>
              </div>
            )}
          </div>
        </div>

        {/* Recommendations */}
        <div className="card" data-testid="recommendations">
          <div className="card-header">
            <h2 className="text-lg font-medium text-gray-900">Recommendations</h2>
          </div>
          <div className="card-body">
            {recommendations.length > 0 ? (
              <div className="space-y-3">
                {recommendations.map((recommendation: string, index: number) => (
                  <div key={index} className="flex items-start">
                    <TrendingUp className="h-5 w-5 text-blue-500 mt-0.5 mr-3 flex-shrink-0" />
                    <p className="text-sm text-gray-700">{recommendation}</p>
                  </div>
                ))}
              </div>
            ) : (
              <div className="text-center py-6">
                <CheckCircle className="mx-auto h-12 w-12 text-gray-400" />
                <h3 className="mt-2 text-sm font-medium text-gray-900">All good!</h3>
                <p className="mt-1 text-sm text-gray-500">No recommendations at this time.</p>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
