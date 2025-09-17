import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Settings, Save, RefreshCw } from 'lucide-react'
import { apiService } from '../services/api'

export default function Preferences() {
  const queryClient = useQueryClient()
  const [hasChanges, setHasChanges] = useState(false)

  const { data: preferences, isLoading } = useQuery({
    queryKey: ['preferences'],
    queryFn: () => apiService.getPreferences(),
  })

  const { data: rulesData } = useQuery({
    queryKey: ['rules'],
    queryFn: () => apiService.getRules(),
  })

  const updateMutation = useMutation({
    mutationFn: (prefs: any) => apiService.updatePreferences(prefs),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['preferences'] })
      setHasChanges(false)
    },
  })

  const handleCategoryToggle = (categoryId: string, enabled: boolean) => {
    setHasChanges(true)
    // TODO: Update local state
  }

  const handleSeverityToggle = (severity: string, enabled: boolean) => {
    setHasChanges(true)
    // TODO: Update local state
  }

  const handleSave = () => {
    // TODO: Implement save functionality
    console.log('Saving preferences...')
  }

  if (isLoading) {
    return (
      <div className="px-6">
        <div className="animate-pulse">
          <div className="h-8 bg-gray-200 rounded w-1/4 mb-6"></div>
          <div className="space-y-6">
            {[...Array(3)].map((_, i) => (
              <div key={i} className="bg-white p-6 rounded-lg shadow">
                <div className="h-6 bg-gray-200 rounded w-1/3 mb-4"></div>
                <div className="space-y-2">
                  {[...Array(4)].map((_, j) => (
                    <div key={j} className="h-4 bg-gray-200 rounded w-3/4"></div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    )
  }

  const categories = rulesData?.categories || {}
  const severities = ['critical', 'high', 'medium', 'low']

  return (
    <div className="px-6">
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900" data-testid="preferences-title">
            Preferences
          </h1>
          <p className="mt-1 text-sm text-gray-500">
            Configure rule preferences and scanning options
          </p>
        </div>
        {hasChanges && (
          <button 
            onClick={handleSave}
            disabled={updateMutation.isPending}
            className="btn-primary"
            data-testid="save-preferences-btn"
          >
            {updateMutation.isPending ? (
              <RefreshCw className="h-4 w-4 mr-2 animate-spin" />
            ) : (
              <Save className="h-4 w-4 mr-2" />
            )}
            Save Changes
          </button>
        )}
      </div>

      <div className="space-y-6">
        {/* Rule Categories */}
        <div className="card" data-testid="categories-section">
          <div className="card-header">
            <h2 className="text-lg font-medium text-gray-900">Rule Categories</h2>
            <p className="mt-1 text-sm text-gray-500">
              Enable or disable entire categories of rules
            </p>
          </div>
          <div className="card-body">
            <div className="space-y-4">
              {Object.values(categories).map((category: any) => (
                <div key={category.id} className="flex items-center justify-between">
                  <div className="flex items-center">
                    <div 
                      className="w-4 h-4 rounded mr-3"
                      style={{ backgroundColor: category.color }}
                    ></div>
                    <div>
                      <div className="font-medium text-gray-900">{category.name}</div>
                      <div className="text-sm text-gray-500">{category.description}</div>
                    </div>
                  </div>
                  <label className="relative inline-flex items-center cursor-pointer">
                    <input
                      type="checkbox"
                      checked={preferences?.categories?.[category.id] !== false}
                      onChange={(e) => handleCategoryToggle(category.id, e.target.checked)}
                      className="sr-only peer"
                      data-testid={`category-toggle-${category.id}`}
                    />
                    <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
                  </label>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Severity Levels */}
        <div className="card" data-testid="severities-section">
          <div className="card-header">
            <h2 className="text-lg font-medium text-gray-900">Severity Levels</h2>
            <p className="mt-1 text-sm text-gray-500">
              Choose which severity levels to include in scans
            </p>
          </div>
          <div className="card-body">
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
              {severities.map((severity) => (
                <div key={severity} className="flex items-center">
                  <input
                    id={`severity-${severity}`}
                    type="checkbox"
                    checked={preferences?.severities?.[severity] !== false}
                    onChange={(e) => handleSeverityToggle(severity, e.target.checked)}
                    className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                    data-testid={`severity-toggle-${severity}`}
                  />
                  <label 
                    htmlFor={`severity-${severity}`}
                    className={`ml-2 text-sm font-medium capitalize severity-${severity} px-2 py-1 rounded`}
                  >
                    {severity}
                  </label>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* General Settings */}
        <div className="card" data-testid="general-settings">
          <div className="card-header">
            <h2 className="text-lg font-medium text-gray-900">General Settings</h2>
          </div>
          <div className="card-body">
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <div className="font-medium text-gray-900">Auto-fix violations</div>
                  <div className="text-sm text-gray-500">
                    Automatically apply fixes for violations when possible
                  </div>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={preferences?.auto_fix || false}
                    className="sr-only peer"
                    data-testid="auto-fix-toggle"
                  />
                  <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
                </label>
              </div>

              <div className="flex items-center justify-between">
                <div>
                  <div className="font-medium text-gray-900">Notifications</div>
                  <div className="text-sm text-gray-500">
                    Receive notifications about scan results and violations
                  </div>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={preferences?.notifications !== false}
                    className="sr-only peer"
                    data-testid="notifications-toggle"
                  />
                  <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
                </label>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}