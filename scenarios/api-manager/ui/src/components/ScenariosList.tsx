import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { 
  Shield,
  Activity,
  AlertCircle,
  CheckCircle,
  ArrowUpRight,
  Search,
  Filter
} from 'lucide-react'
import { useState } from 'react'
import clsx from 'clsx'
import { format } from 'date-fns'
import { apiService } from '@/services/api'
import { Card } from './common/Card'
import { Badge } from './common/Badge'

export default function ScenariosList() {
  const [searchTerm, setSearchTerm] = useState('')
  const [statusFilter, setStatusFilter] = useState('all')

  const { data: scenarios, isLoading, error } = useQuery({
    queryKey: ['scenarios'],
    queryFn: () => apiService.getScenarios(),
    refetchInterval: 30000,
  })

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'healthy': return <CheckCircle className="h-4 w-4 text-success-500" />
      case 'degraded': return <AlertCircle className="h-4 w-4 text-warning-500" />
      case 'unhealthy': return <AlertCircle className="h-4 w-4 text-danger-500" />
      default: return <Activity className="h-4 w-4 text-dark-400" />
    }
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'healthy': return 'success'
      case 'degraded': return 'warning'
      case 'unhealthy': return 'danger'
      default: return 'default'
    }
  }

  // Filter scenarios based on search and status
  const filteredScenarios = scenarios?.filter((scenario: any) => {
    const matchesSearch = scenario.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         scenario.description?.toLowerCase().includes(searchTerm.toLowerCase())
    const matchesStatus = statusFilter === 'all' || scenario.status === statusFilter
    return matchesSearch && matchesStatus
  }) || []

  const statusOptions = [
    { value: 'all', label: 'All Statuses' },
    { value: 'healthy', label: 'Healthy' },
    { value: 'degraded', label: 'Degraded' },
    { value: 'unhealthy', label: 'Unhealthy' },
    { value: 'available', label: 'Available' },
    { value: 'running', label: 'Running' },
  ]

  if (error) {
    return (
      <div className="space-y-6">
        <div className="text-center py-12">
          <AlertCircle className="h-12 w-12 text-danger-500 mx-auto mb-4" />
          <h3 className="text-lg font-medium text-dark-900 mb-2">Failed to load scenarios</h3>
          <p className="text-dark-600">{error.message}</p>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6 animate-in">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-dark-900">All Scenarios</h1>
        <p className="text-dark-500 mt-1">
          {filteredScenarios.length} of {scenarios?.length || 0} scenarios
        </p>
      </div>

      {/* Filters */}
      <Card>
        <div className="flex flex-col sm:flex-row gap-4">
          {/* Search */}
          <div className="flex-1">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-dark-400" />
              <input
                type="text"
                placeholder="Search scenarios..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="w-full pl-10 pr-4 py-2 border border-dark-200 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
              />
            </div>
          </div>

          {/* Status Filter */}
          <div className="sm:w-48">
            <div className="relative">
              <Filter className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-dark-400" />
              <select
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value)}
                className="w-full pl-10 pr-4 py-2 border border-dark-200 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500 bg-white"
              >
                {statusOptions.map(option => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
            </div>
          </div>
        </div>
      </Card>

      {/* Scenarios Grid */}
      {isLoading ? (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3, 4, 5, 6].map(i => (
            <Card key={i}>
              <div className="space-y-3">
                <div className="h-4 bg-dark-200 rounded animate-pulse" />
                <div className="h-3 bg-dark-200 rounded animate-pulse w-3/4" />
                <div className="h-3 bg-dark-200 rounded animate-pulse w-1/2" />
              </div>
            </Card>
          ))}
        </div>
      ) : filteredScenarios.length > 0 ? (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {filteredScenarios.map((scenario: any) => (
            <Link
              key={scenario.name}
              to={`/scenario/${scenario.name}`}
              className="group"
            >
              <Card className="h-full transition-all hover:shadow-lg hover:scale-[1.02]">
                <div className="flex items-start justify-between mb-3">
                  <div className="flex items-center gap-2">
                    {getStatusIcon(scenario.status)}
                    <Badge variant={getStatusColor(scenario.status) as any}>
                      {scenario.status}
                    </Badge>
                  </div>
                  <ArrowUpRight className="h-4 w-4 text-dark-400 group-hover:text-primary-600 transition-colors" />
                </div>

                <h3 className="font-semibold text-dark-900 group-hover:text-primary-600 transition-colors mb-2">
                  {scenario.name}
                </h3>
                
                <p className="text-sm text-dark-600 mb-4 line-clamp-2">
                  {scenario.description || 'No description available'}
                </p>

                <div className="space-y-2 text-xs text-dark-500">
                  <div className="flex justify-between">
                    <span>Endpoints:</span>
                    <span className="font-medium">{scenario.endpoint_count || 0}</span>
                  </div>
                  <div className="flex justify-between">
                    <span>Vulnerabilities:</span>
                    <span className={clsx(
                      'font-medium',
                      scenario.vulnerability_count > 0 ? 'text-danger-600' : 'text-success-600'
                    )}>
                      {scenario.vulnerability_count || 0}
                    </span>
                  </div>
                  {scenario.last_scan && (
                    <div className="flex justify-between">
                      <span>Last scan:</span>
                      <span className="font-medium">
                        {format(new Date(scenario.last_scan), 'MMM d')}
                      </span>
                    </div>
                  )}
                </div>

                {scenario.tags && scenario.tags.length > 0 && (
                  <div className="mt-3 flex flex-wrap gap-1">
                    {scenario.tags.slice(0, 3).map((tag: string) => (
                      <span
                        key={tag}
                        className="inline-block px-2 py-1 text-xs bg-dark-100 text-dark-600 rounded"
                      >
                        {tag}
                      </span>
                    ))}
                    {scenario.tags.length > 3 && (
                      <span className="text-xs text-dark-400">+{scenario.tags.length - 3}</span>
                    )}
                  </div>
                )}
              </Card>
            </Link>
          ))}
        </div>
      ) : (
        <div className="text-center py-12">
          <Shield className="h-12 w-12 text-dark-300 mx-auto mb-4" />
          <h3 className="text-lg font-medium text-dark-900 mb-2">No scenarios found</h3>
          <p className="text-dark-600">
            {searchTerm || statusFilter !== 'all' 
              ? 'Try adjusting your search or filter criteria'
              : 'No scenarios are currently available'
            }
          </p>
        </div>
      )}
    </div>
  )
}