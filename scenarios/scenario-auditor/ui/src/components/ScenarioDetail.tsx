import { useParams, useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { 
  ArrowLeft, 
  Shield, 
  Activity,
  GitBranch,
  Clock,
  AlertCircle,
  CheckCircle
} from 'lucide-react'
import { format } from 'date-fns'
import { apiService } from '../services/api'
import { Card } from './common/Card'
import { Badge } from './common/Badge'

export default function ScenarioDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const { data: scenario, isLoading } = useQuery({
    queryKey: ['scenario', id],
    queryFn: () => apiService.getScenario(id!),
    enabled: !!id,
  })

  const { data: health } = useQuery({
    queryKey: ['scenarioHealth', id],
    queryFn: () => apiService.getScenarioHealth(id!),
    enabled: !!id,
    refetchInterval: 30000,
  })

  const { data: vulnerabilities } = useQuery({
    queryKey: ['vulnerabilities', id],
    queryFn: () => apiService.getVulnerabilities(id!),
    enabled: !!id,
  })

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="h-32 bg-dark-100 rounded-lg animate-pulse" />
        <div className="grid gap-6 lg:grid-cols-3">
          <div className="lg:col-span-2 h-64 bg-dark-100 rounded-lg animate-pulse" />
          <div className="h-64 bg-dark-100 rounded-lg animate-pulse" />
        </div>
      </div>
    )
  }

  if (!scenario) {
    return (
      <div className="text-center py-12">
        <AlertCircle className="h-12 w-12 text-danger-500 mx-auto mb-3" />
        <p className="text-lg font-medium text-dark-900">Scenario not found</p>
        <button
          onClick={() => navigate('/')}
          className="mt-4 text-primary-600 hover:text-primary-700 font-medium"
        >
          Return to Dashboard
        </button>
      </div>
    )
  }

  return (
    <div className="space-y-6 animate-in">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <button
            onClick={() => navigate('/')}
            className="flex items-center justify-center rounded-lg bg-dark-100 p-2 text-dark-700 hover:bg-dark-200 transition-colors"
          >
            <ArrowLeft className="h-5 w-5" />
          </button>
          <div>
            <h1 className="text-2xl font-bold text-dark-900">{scenario.name}</h1>
            <p className="text-dark-500 mt-1">{scenario.description}</p>
          </div>
        </div>
        <Badge 
          variant={
            scenario.status === 'healthy' ? 'success' :
            scenario.status === 'degraded' ? 'warning' :
            'danger'
          }
        >
          {scenario.status}
        </Badge>
      </div>

      {/* Stats */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card padding="sm" className="text-center">
          <div className="flex items-center justify-center mb-2">
            <Activity className="h-8 w-8 text-primary-500" />
          </div>
          <p className="text-2xl font-bold text-dark-900">{health?.score || 0}%</p>
          <p className="text-sm text-dark-500">Health Score</p>
        </Card>
        
        <Card padding="sm" className="text-center">
          <div className="flex items-center justify-center mb-2">
            <GitBranch className="h-8 w-8 text-primary-500" />
          </div>
          <p className="text-2xl font-bold text-dark-900">{scenario.endpoint_count}</p>
          <p className="text-sm text-dark-500">Endpoints</p>
        </Card>
        
        <Card padding="sm" className="text-center">
          <div className="flex items-center justify-center mb-2">
            <Shield className="h-8 w-8 text-warning-500" />
          </div>
          <p className="text-2xl font-bold text-dark-900">{scenario.vulnerability_count}</p>
          <p className="text-sm text-dark-500">Vulnerabilities</p>
        </Card>
        
        <Card padding="sm" className="text-center">
          <div className="flex items-center justify-center mb-2">
            <Clock className="h-8 w-8 text-primary-500" />
          </div>
          <p className="text-lg font-bold text-dark-900">
            {scenario.last_scan && !isNaN(new Date(scenario.last_scan).getTime())
              ? format(new Date(scenario.last_scan), 'MMM d')
              : 'Never'}
          </p>
          <p className="text-sm text-dark-500">Last Scan</p>
        </Card>
      </div>

      {/* Main Content */}
      <div className="grid gap-6 lg:grid-cols-3">
        {/* Health Details */}
        <Card className="lg:col-span-2">
          <h2 className="text-lg font-semibold text-dark-900 mb-4">Health Details</h2>
          
          {health ? (
            <div className="space-y-4">
              <div className="grid gap-4 sm:grid-cols-2">
                <div>
                  <p className="text-sm font-medium text-dark-700">Status</p>
                  <div className="flex items-center gap-2 mt-1">
                    {health.status === 'healthy' ? (
                      <CheckCircle className="h-4 w-4 text-success-500" />
                    ) : (
                      <AlertCircle className="h-4 w-4 text-warning-500" />
                    )}
                    <span className="text-dark-900 capitalize">{health.status}</span>
                  </div>
                </div>
                
                <div>
                  <p className="text-sm font-medium text-dark-700">Uptime</p>
                  <p className="text-dark-900 mt-1">{health.uptime || 'N/A'}</p>
                </div>
                
                <div>
                  <p className="text-sm font-medium text-dark-700">Response Time</p>
                  <p className="text-dark-900 mt-1">{health.response_time || 'N/A'}ms</p>
                </div>
                
                <div>
                  <p className="text-sm font-medium text-dark-700">Error Rate</p>
                  <p className="text-dark-900 mt-1">{health.error_rate || '0'}%</p>
                </div>
              </div>
              
              {health.checks && (
                <div>
                  <p className="text-sm font-medium text-dark-700 mb-2">Health Checks</p>
                  <div className="space-y-2">
                    {Object.entries(health.checks).map(([check, checkResult]) => (
                      <div key={check} className="flex items-center justify-between p-2 bg-dark-50 rounded-lg">
                        <span className="text-sm text-dark-700 capitalize">
                          {check.replace(/_/g, ' ')}
                        </span>
                        {checkResult.status === 'passing' ? (
                          <CheckCircle className="h-4 w-4 text-success-500" />
                        ) : (
                          <AlertCircle className="h-4 w-4 text-danger-500" />
                        )}
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          ) : (
            <p className="text-dark-500">Loading health data...</p>
          )}
        </Card>

        {/* Vulnerabilities Summary */}
        <Card>
          <h2 className="text-lg font-semibold text-dark-900 mb-4">Security Summary</h2>
          
          {vulnerabilities && vulnerabilities.length > 0 ? (
            <div className="space-y-3">
              <div className="space-y-2">
                {['critical', 'high', 'medium', 'low'].map((severity) => {
                  const count = vulnerabilities.filter(v => v.severity === severity).length
                  if (count === 0) return null
                  
                  return (
                    <div key={severity} className="flex items-center justify-between">
                      <span className="text-sm text-dark-700 capitalize">{severity}</span>
                      <Badge 
                        variant={
                          severity === 'critical' ? 'danger' :
                          severity === 'high' ? 'warning' :
                          'default'
                        }
                        size="sm"
                      >
                        {count}
                      </Badge>
                    </div>
                  )
                })}
              </div>
              
              <div className="pt-3 border-t border-dark-200">
                <p className="text-sm font-medium text-dark-700 mb-2">Recent Vulnerabilities</p>
                <div className="space-y-2">
                  {vulnerabilities.slice(0, 3).map((vuln) => (
                    <div key={vuln.id} className="text-sm">
                      <p className="font-medium text-dark-900">{vuln.title}</p>
                      <p className="text-xs text-dark-500 mt-0.5">
                        {vuln.type} • {vuln.discovered_at && !isNaN(new Date(vuln.discovered_at).getTime())
                          ? format(new Date(vuln.discovered_at), 'MMM d')
                          : 'Unknown date'}
                      </p>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          ) : (
            <div className="text-center py-8">
              <CheckCircle className="h-12 w-12 text-success-500 mx-auto mb-3" />
              <p className="text-sm text-dark-700">No vulnerabilities detected</p>
            </div>
          )}
        </Card>
      </div>

      {/* Metadata */}
      {scenario.metadata && Object.keys(scenario.metadata).length > 0 && (
        <Card>
          <h2 className="text-lg font-semibold text-dark-900 mb-4">Metadata</h2>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {Object.entries(scenario.metadata).map(([key, value]) => (
              <div key={key}>
                <p className="text-sm font-medium text-dark-700 capitalize">
                  {key.replace(/_/g, ' ')}
                </p>
                <p className="text-dark-900 mt-1">{String(value)}</p>
              </div>
            ))}
          </div>
        </Card>
      )}
    </div>
  )
}