import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  AlertTriangle,
  Clock,
  RefreshCw,
  Save,
  Sliders
} from 'lucide-react'
import { useEffect, useState } from 'react'
import { apiService } from '../services/api'
import { Card } from './common/Card'

export default function SettingsPanel() {

  const queryClient = useQueryClient()
  const [preferencesState, setPreferencesState] = useState<any | null>(null)
  const [hasPreferenceChanges, setHasPreferenceChanges] = useState(false)
  const [preferencesError, setPreferencesError] = useState<string | null>(null)

  const { data: preferencesData, isLoading: preferencesLoading } = useQuery({
    queryKey: ['preferences'],
    queryFn: () => apiService.getPreferences(),
  })

  const { data: rulesData } = useQuery({
    queryKey: ['rules'],
    queryFn: () => apiService.getRules(),
  })

  useEffect(() => {
    if (!preferencesData) return
    if (!hasPreferenceChanges) {
      setPreferencesState(preferencesData)
    } else if (preferencesState === null) {
      setPreferencesState(preferencesData)
    }
  }, [preferencesData, hasPreferenceChanges, preferencesState])

  const updateMutation = useMutation({
    mutationFn: (prefs: any) => apiService.updatePreferences(prefs),
    onSuccess: (data) => {
      setPreferencesError(null)
      setHasPreferenceChanges(false)
      if (data && typeof data === 'object' && !Array.isArray(data) && (('categories' in data) || ('severities' in data))) {
        setPreferencesState(data)
      }
      queryClient.invalidateQueries({ queryKey: ['preferences'] })
    },
    onError: (error: unknown) => {
      console.error('Failed to update preferences:', error)
      setPreferencesError(error instanceof Error ? error.message : 'Failed to update preferences')
    },
  })

  const categories = (rulesData?.categories || {}) as Record<string, any>
  const severities = ['critical', 'high', 'medium', 'low'] as const

  const handleCategoryToggle = (categoryId: string, enabled: boolean) => {
    if (!preferencesState) return
    setPreferencesError(null)
    setPreferencesState((prev: any) => {
      if (!prev) return prev
      const nextCategories = { ...(prev.categories || {}) }
      if (enabled) {
        delete nextCategories[categoryId]
      } else {
        nextCategories[categoryId] = false
      }
      return {
        ...prev,
        categories: nextCategories,
      }
    })
    setHasPreferenceChanges(true)
  }

  const handleSeverityToggle = (severity: string, enabled: boolean) => {
    if (!preferencesState) return
    setPreferencesError(null)
    setPreferencesState((prev: any) => {
      if (!prev) return prev
      const nextSeverities = { ...(prev.severities || {}) }
      if (enabled) {
        delete nextSeverities[severity]
      } else {
        nextSeverities[severity] = false
      }
      return {
        ...prev,
        severities: nextSeverities,
      }
    })
    setHasPreferenceChanges(true)
  }

  const handlePreferenceToggle = (key: 'auto_fix' | 'notifications', enabled: boolean) => {
    if (!preferencesState) return
    setPreferencesError(null)
    setPreferencesState((prev: any) => {
      if (!prev) return prev
      return {
        ...prev,
        [key]: enabled,
      }
    })
    setHasPreferenceChanges(true)
  }

  const handleSavePreferences = () => {
    if (!preferencesState || updateMutation.isPending) return
    setPreferencesError(null)
    updateMutation.mutate(preferencesState)
  }

  const isPreferenceLoading = preferencesLoading && !preferencesState

  return (
    <div className="space-y-6 animate-in">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-dark-900">Settings</h1>
        <p className="text-dark-500 mt-1">Configure Scenario Auditor rule preferences and automation behavior</p>
      </div>

      <div className="space-y-4">
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h2 className="text-xl font-semibold text-dark-900" data-testid="preferences-title">Rule Preferences</h2>
            <p className="text-sm text-dark-500">Tune enforcement coverage and automation defaults.</p>
          </div>
          {hasPreferenceChanges && (
            <button
              onClick={handleSavePreferences}
              disabled={updateMutation.isPending}
              className="flex items-center gap-2 rounded-lg bg-primary-500 px-4 py-2 text-sm font-medium text-white hover:bg-primary-600 disabled:opacity-60"
              data-testid="save-preferences-btn"
            >
              {updateMutation.isPending ? (
                <>
                  <RefreshCw className="h-4 w-4 animate-spin" />
                  Saving…
                </>
              ) : (
                <>
                  <Save className="h-4 w-4" />
                  Save Rule Preferences
                </>
              )}
            </button>
          )}
        </div>
        {preferencesError && (
          <p className="text-sm text-danger-600" role="alert">{preferencesError}</p>
        )}
        {isPreferenceLoading ? (
          <div className="grid gap-6 lg:grid-cols-2">
            {[0, 1, 2].map((index) => (
              <Card
                key={`preferences-skeleton-${index}`}
                className={index === 2 ? 'lg:col-span-2' : ''}
              >
                <div className="space-y-3 animate-pulse">
                  <div className="h-5 w-1/3 rounded bg-dark-200" />
                  <div className="space-y-2">
                    {[0, 1, 2, 3].map(item => (
                      <div key={item} className="h-4 w-3/4 rounded bg-dark-100" />
                    ))}
                  </div>
                </div>
              </Card>
            ))}
          </div>
        ) : (
          <div className="grid gap-6 lg:grid-cols-2">
            <Card data-testid="categories-section">
              <div className="flex items-center gap-2 mb-4">
                <Sliders className="h-5 w-5 text-primary-500" />
                <h3 className="text-lg font-semibold text-dark-900">Rule Categories</h3>
              </div>
              <p className="text-sm text-dark-500 mb-4">Enable or disable entire categories of rules.</p>
              <div className="space-y-4">
                {Object.entries(categories).length > 0 ? (
                  Object.entries(categories).map(([categoryId, category]) => {
                    const cat = category as any
                    const isEnabled = preferencesState?.categories?.[categoryId] !== false
                    return (
                      <div key={categoryId} className="flex items-center justify-between">
                        <div className="flex items-center">
                          <div
                            className="mr-3 h-4 w-4 rounded"
                            style={{ backgroundColor: cat.color }}
                          />
                          <div>
                            <div className="font-medium text-dark-900">{cat.name || categoryId}</div>
                            {cat.description && (
                              <div className="text-sm text-dark-500">{cat.description}</div>
                            )}
                          </div>
                        </div>
                        <label className="relative inline-flex items-center cursor-pointer">
                          <input
                            type="checkbox"
                            checked={isEnabled}
                            onChange={(e) => handleCategoryToggle(categoryId, e.target.checked)}
                            className="sr-only peer"
                            data-testid={`category-toggle-${categoryId}`}
                            disabled={updateMutation.isPending}
                          />
                          <div className="w-11 h-6 bg-dark-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-primary-200 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-dark-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-500"></div>
                        </label>
                      </div>
                    )
                  })
                ) : (
                  <p className="text-sm text-dark-500">No rule categories found.</p>
                )}
              </div>
            </Card>
            <Card data-testid="severities-section">
              <div className="flex items-center gap-2 mb-4">
                <AlertTriangle className="h-5 w-5 text-primary-500" />
                <h3 className="text-lg font-semibold text-dark-900">Severity Filters</h3>
              </div>
              <p className="text-sm text-dark-500 mb-4">Choose which severity levels to include in scans.</p>
              <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
                {severities.map((severity) => {
                  const label = severity as string
                  const isEnabled = preferencesState?.severities?.[label] !== false
                  return (
                    <div key={label} className="flex items-center">
                      <input
                        id={`severity-${label}`}
                        type="checkbox"
                        checked={isEnabled}
                        onChange={(e) => handleSeverityToggle(label, e.target.checked)}
                        className="h-4 w-4 text-primary-500 focus:ring-primary-500 border-dark-300 rounded"
                        data-testid={`severity-toggle-${label}`}
                        disabled={updateMutation.isPending}
                      />
                      <label
                        htmlFor={`severity-${label}`}
                        className={`ml-2 text-sm font-medium capitalize severity-${label} px-2 py-1 rounded`}
                      >
                        {label}
                      </label>
                    </div>
                  )
                })}
              </div>
            </Card>
            <Card className="lg:col-span-2" data-testid="general-settings">
              <div className="flex items-center gap-2 mb-4">
                <Clock className="h-5 w-5 text-primary-500" />
                <h3 className="text-lg font-semibold text-dark-900">Rule Automation</h3>
              </div>
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <div>
                    <div className="font-medium text-dark-900">Auto-fix violations</div>
                    <div className="text-sm text-dark-500">Automatically apply fixes for violations when possible.</div>
                  </div>
                  <label className="relative inline-flex items-center cursor-pointer">
                    <input
                      type="checkbox"
                      checked={Boolean(preferencesState?.auto_fix)}
                      onChange={(e) => handlePreferenceToggle('auto_fix', e.target.checked)}
                      className="sr-only peer"
                      data-testid="auto-fix-toggle"
                      disabled={updateMutation.isPending}
                    />
                    <div className="w-11 h-6 bg-dark-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-primary-200 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-dark-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-500"></div>
                  </label>
                </div>

                <div className="flex items-center justify-between">
                  <div>
                    <div className="font-medium text-dark-900">Notifications</div>
                    <div className="text-sm text-dark-500">Receive alerts about scan results and violations.</div>
                  </div>
                  <label className="relative inline-flex items-center cursor-pointer">
                    <input
                      type="checkbox"
                      checked={preferencesState?.notifications !== false}
                      onChange={(e) => handlePreferenceToggle('notifications', e.target.checked)}
                      className="sr-only peer"
                      data-testid="notifications-toggle"
                      disabled={updateMutation.isPending}
                    />
                    <div className="w-11 h-6 bg-dark-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-primary-200 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-dark-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-500"></div>
                  </label>
                </div>
              </div>
            </Card>
          </div>
        )}
      </div>

    </div>
  )
}
