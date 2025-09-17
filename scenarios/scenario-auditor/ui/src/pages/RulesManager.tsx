import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Shield, Plus, Settings, Filter } from 'lucide-react'
import { apiService } from '../services/api'

export default function RulesManager() {
  const [selectedCategory, setSelectedCategory] = useState<string>('')

  const { data: rulesData, isLoading } = useQuery({
    queryKey: ['rules', selectedCategory],
    queryFn: () => apiService.getRules(selectedCategory || undefined),
  })

  if (isLoading) {
    return (
      <div className="px-6">
        <div className="animate-pulse">
          <div className="h-8 bg-gray-200 rounded w-1/4 mb-6"></div>
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            {[...Array(6)].map((_, i) => (
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

  const rules = rulesData?.rules || {}
  const categories = rulesData?.categories || {}

  return (
    <div className="px-6">
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900" data-testid="rules-title">
            Rules Manager
          </h1>
          <p className="mt-1 text-sm text-gray-500">
            Manage and configure auditing rules
          </p>
        </div>
        <button 
          className="btn-primary"
          data-testid="create-rule-btn"
        >
          <Plus className="h-4 w-4 mr-2" />
          Create Rule
        </button>
      </div>

      {/* Filters */}
      <div className="mb-6 flex items-center space-x-4">
        <select
          value={selectedCategory}
          onChange={(e) => setSelectedCategory(e.target.value)}
          className="rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
          data-testid="category-filter"
        >
          <option value="">All Categories</option>
          {Object.values(categories).map((category: any) => (
            <option key={category.id} value={category.id}>
              {category.name}
            </option>
          ))}
        </select>
      </div>

      {/* Rules Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {Object.values(rules).map((rule: any) => (
          <div key={rule.id} className="card" data-testid={`rule-${rule.id}`}>
            <div className="card-body">
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <h3 className="text-lg font-medium text-gray-900">{rule.name}</h3>
                  <p className="mt-1 text-sm text-gray-500">{rule.description}</p>
                </div>
                <div className="ml-4">
                  <button className="text-gray-400 hover:text-gray-600">
                    <Settings className="h-5 w-5" />
                  </button>
                </div>
              </div>
              
              <div className="mt-4 flex items-center justify-between">
                <div className="flex items-center space-x-2">
                  <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium category-${rule.category}`}>
                    {rule.category}
                  </span>
                  <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium severity-${rule.severity}`}>
                    {rule.severity}
                  </span>
                </div>
                <div className="flex items-center">
                  <input
                    type="checkbox"
                    checked={rule.enabled}
                    className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                    data-testid={`rule-toggle-${rule.id}`}
                  />
                </div>
              </div>
            </div>
          </div>
        ))}
      </div>

      {Object.keys(rules).length === 0 && (
        <div className="text-center py-12">
          <Shield className="mx-auto h-12 w-12 text-gray-400" />
          <h3 className="mt-2 text-sm font-medium text-gray-900">No rules found</h3>
          <p className="mt-1 text-sm text-gray-500">
            {selectedCategory ? 'No rules in this category.' : 'Get started by creating your first rule.'}
          </p>
        </div>
      )}
    </div>
  )
}