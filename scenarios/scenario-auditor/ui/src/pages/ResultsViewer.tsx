import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { FileText, Search, Filter, AlertTriangle, CheckCircle } from 'lucide-react'
import { apiService } from '../services/api'

export default function ResultsViewer() {
  const [searchTerm, setSearchTerm] = useState('')
  const [selectedSeverity, setSelectedSeverity] = useState('')

  const { data: resultsData, isLoading } = useQuery({
    queryKey: ['results'],
    queryFn: () => apiService.getResults(),
  })

  if (isLoading) {
    return (
      <div className="px-6">
        <div className="animate-pulse">
          <div className="h-8 bg-gray-200 rounded w-1/4 mb-6"></div>
          <div className="space-y-4">
            {[...Array(5)].map((_, i) => (
              <div key={i} className="bg-white p-6 rounded-lg shadow">
                <div className="h-4 bg-gray-200 rounded w-3/4 mb-2"></div>
                <div className="h-3 bg-gray-200 rounded w-1/2"></div>
              </div>
            ))}
          </div>
        </div>
      </div>
    )
  }

  const results = resultsData?.results || []

  return (
    <div className="px-6">
      {/* Header */}
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-900" data-testid="results-title">
          Scan Results
        </h1>
        <p className="mt-1 text-sm text-gray-500">
          View and analyze standards violations
        </p>
      </div>

      {/* Search and Filters */}
      <div className="mb-6 flex flex-col sm:flex-row gap-4">
        <div className="flex-1">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-gray-400" />
            <input
              type="text"
              placeholder="Search scenarios or violations..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="pl-10 w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
              data-testid="search-input"
            />
          </div>
        </div>
        <select
          value={selectedSeverity}
          onChange={(e) => setSelectedSeverity(e.target.value)}
          className="rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
          data-testid="severity-filter"
        >
          <option value="">All Severities</option>
          <option value="critical">Critical</option>
          <option value="high">High</option>
          <option value="medium">Medium</option>
          <option value="low">Low</option>
        </select>
      </div>

      {/* Results List */}
      <div className="space-y-6">
        {results.length > 0 ? (
          results.map((result: any) => (
            <div key={result.id} className="card" data-testid={`result-${result.id}`}>
              <div className="card-body">
                <div className="flex items-center justify-between mb-4">
                  <div>
                    <h3 className="text-lg font-medium text-gray-900">{result.scenario_name}</h3>
                    <p className="text-sm text-gray-500">
                      Scanned {new Date(result.end_time).toLocaleString()}
                    </p>
                  </div>
                  <div className="text-right">
                    <div className="text-2xl font-bold text-gray-900">
                      {result.summary.score.toFixed(1)}
                    </div>
                    <div className="text-sm text-gray-500">Score</div>
                  </div>
                </div>

                <div className="grid grid-cols-2 sm:grid-cols-4 gap-4 mb-4">
                  <div className="text-center">
                    <div className="text-lg font-semibold text-gray-900">
                      {result.summary.total_violations}
                    </div>
                    <div className="text-xs text-gray-500">Total Violations</div>
                  </div>
                  <div className="text-center">
                    <div className="text-lg font-semibold text-red-600">
                      {result.summary.by_severity?.critical || 0}
                    </div>
                    <div className="text-xs text-gray-500">Critical</div>
                  </div>
                  <div className="text-center">
                    <div className="text-lg font-semibold text-orange-600">
                      {result.summary.by_severity?.high || 0}
                    </div>
                    <div className="text-xs text-gray-500">High</div>
                  </div>
                  <div className="text-center">
                    <div className="text-lg font-semibold text-green-600">
                      {result.summary.auto_fixable}
                    </div>
                    <div className="text-xs text-gray-500">Auto-fixable</div>
                  </div>
                </div>

                {result.violations && result.violations.length > 0 && (
                  <div className="mt-4">
                    <h4 className="text-sm font-medium text-gray-900 mb-2">Violations</h4>
                    <div className="space-y-2">
                      {result.violations.slice(0, 3).map((violation: any) => (
                        <div key={violation.id} className="flex items-start p-3 bg-gray-50 rounded-lg">
                          <AlertTriangle className="h-4 w-4 text-orange-500 mt-0.5 mr-3 flex-shrink-0" />
                          <div className="flex-1">
                            <div className="text-sm font-medium text-gray-900">{violation.message}</div>
                            <div className="text-xs text-gray-500">
                              {violation.file_path} • {violation.category} • {violation.severity}
                            </div>
                          </div>
                        </div>
                      ))}
                      {result.violations.length > 3 && (
                        <div className="text-center">
                          <button className="text-sm text-blue-600 hover:text-blue-800">
                            View {result.violations.length - 3} more violations
                          </button>
                        </div>
                      )}
                    </div>
                  </div>
                )}
              </div>
            </div>
          ))
        ) : (
          <div className="text-center py-12">
            <FileText className="mx-auto h-12 w-12 text-gray-400" />
            <h3 className="mt-2 text-sm font-medium text-gray-900">No scan results</h3>
            <p className="mt-1 text-sm text-gray-500">
              Run a scan to see results here.
            </p>
          </div>
        )}
      </div>
    </div>
  )
}