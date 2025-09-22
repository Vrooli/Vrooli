import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import clsx from 'clsx'
import {
  AlertTriangle,
  Bell,
  Check,
  Clock,
  Database,
  Globe,
  RefreshCw,
  Save,
  Settings,
  Shield,
  Sliders
} from 'lucide-react'
import { useEffect, useState } from 'react'
import { apiService } from '../services/api'
import { Badge } from './common/Badge'
import { Card } from './common/Card'

type SettingsState = {
  general: {
    api_timeout: number
    max_retries: number
    debug_mode: boolean
    log_level: string
  }
  scanning: {
    auto_scan: boolean
    scan_interval: number
    scan_depth: string
    parallel_scans: number
  }
  notifications: {
    email_enabled: boolean
    email_address: string
    alert_critical: boolean
    alert_high: boolean
    alert_medium: boolean
    alert_low: boolean
  }
  database: {
    connection_pool: number
    query_timeout: number
    auto_backup: boolean
    backup_interval: number
  }
}

export default function SettingsPanel() {
  const [settings, setSettings] = useState<SettingsState>({
    general: {
      api_timeout: 30,
      max_retries: 3,
      debug_mode: false,
      log_level: 'info',
    },
    scanning: {
      auto_scan: true,
      scan_interval: 3600,
      scan_depth: 'medium',
      parallel_scans: 2,
    },
    notifications: {
      email_enabled: true,
      email_address: 'admin@example.com',
      alert_critical: true,
      alert_high: true,
      alert_medium: false,
      alert_low: false,
    },
    database: {
      connection_pool: 10,
      query_timeout: 5000,
      auto_backup: true,
      backup_interval: 86400,
    },
  })

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

  const [saved, setSaved] = useState(false)

  const handleSave = () => {
    // In a real app, this would save to the backend
    setSaved(true)
    setTimeout(() => setSaved(false), 3000)
  }

  return (
    <div className="space-y-6 animate-in">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-dark-900">Settings</h1>
          <p className="text-dark-500 mt-1">Configure API Manager preferences and behavior</p>
        </div>
        <button
          onClick={handleSave}
          className={clsx(
            'flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium transition-all',
            saved
              ? 'bg-success-500 text-white'
              : 'bg-primary-500 text-white hover:bg-primary-600'
          )}
        >
          {saved ? (
            <>
              <Check className="h-4 w-4" />
              Saved
            </>
          ) : (
            <>
              <Save className="h-4 w-4" />
              Save Changes
            </>
          )}
        </button>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        {/* General Settings */}
        <Card>
          <div className="flex items-center gap-2 mb-4">
            <Settings className="h-5 w-5 text-primary-500" />
            <h2 className="text-lg font-semibold text-dark-900">General Settings</h2>
          </div>

          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-dark-700 mb-2">
                API Timeout (seconds)
              </label>
              <input
                type="number"
                value={settings.general.api_timeout}
                onChange={(e) => setSettings(prev => ({
                  ...prev,
                  general: { ...prev.general, api_timeout: parseInt(e.target.value) }
                }))}
                className="w-full rounded-lg border border-dark-300 bg-white px-4 py-2 text-dark-900 focus:border-primary-500 focus:ring-2 focus:ring-primary-500/20"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-dark-700 mb-2">
                Max Retries
              </label>
              <input
                type="number"
                value={settings.general.max_retries}
                onChange={(e) => setSettings(prev => ({
                  ...prev,
                  general: { ...prev.general, max_retries: parseInt(e.target.value) }
                }))}
                className="w-full rounded-lg border border-dark-300 bg-white px-4 py-2 text-dark-900 focus:border-primary-500 focus:ring-2 focus:ring-primary-500/20"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-dark-700 mb-2">
                Log Level
              </label>
              <select
                value={settings.general.log_level}
                onChange={(e) => setSettings(prev => ({
                  ...prev,
                  general: { ...prev.general, log_level: e.target.value }
                }))}
                className="w-full rounded-lg border border-dark-300 bg-white px-4 py-2 text-dark-900 focus:border-primary-500 focus:ring-2 focus:ring-primary-500/20"
              >
                <option value="error">Error</option>
                <option value="warning">Warning</option>
                <option value="info">Info</option>
                <option value="debug">Debug</option>
              </select>
            </div>

            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={settings.general.debug_mode}
                onChange={(e) => setSettings(prev => ({
                  ...prev,
                  general: { ...prev.general, debug_mode: e.target.checked }
                }))}
                className="rounded text-primary-500 focus:ring-primary-500"
              />
              <span className="text-sm text-dark-700">Enable Debug Mode</span>
            </label>
          </div>
        </Card>

        {/* Scanning Settings */}
        <Card>
          <div className="flex items-center gap-2 mb-4">
            <Shield className="h-5 w-5 text-primary-500" />
            <h2 className="text-lg font-semibold text-dark-900">Scanning Settings</h2>
          </div>

          <div className="space-y-4">
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={settings.scanning.auto_scan}
                onChange={(e) => setSettings(prev => ({
                  ...prev,
                  scanning: { ...prev.scanning, auto_scan: e.target.checked }
                }))}
                className="rounded text-primary-500 focus:ring-primary-500"
              />
              <span className="text-sm text-dark-700">Enable Automatic Scanning</span>
            </label>

            <div>
              <label className="block text-sm font-medium text-dark-700 mb-2">
                Scan Interval (seconds)
              </label>
              <input
                type="number"
                value={settings.scanning.scan_interval}
                onChange={(e) => setSettings(prev => ({
                  ...prev,
                  scanning: { ...prev.scanning, scan_interval: parseInt(e.target.value) }
                }))}
                disabled={!settings.scanning.auto_scan}
                className="w-full rounded-lg border border-dark-300 bg-white px-4 py-2 text-dark-900 focus:border-primary-500 focus:ring-2 focus:ring-primary-500/20 disabled:opacity-50"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-dark-700 mb-2">
                Scan Depth
              </label>
              <select
                value={settings.scanning.scan_depth}
                onChange={(e) => setSettings(prev => ({
                  ...prev,
                  scanning: { ...prev.scanning, scan_depth: e.target.value }
                }))}
                className="w-full rounded-lg border border-dark-300 bg-white px-4 py-2 text-dark-900 focus:border-primary-500 focus:ring-2 focus:ring-primary-500/20"
              >
                <option value="shallow">Shallow</option>
                <option value="medium">Medium</option>
                <option value="deep">Deep</option>
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium text-dark-700 mb-2">
                Parallel Scans
              </label>
              <input
                type="number"
                value={settings.scanning.parallel_scans}
                onChange={(e) => setSettings(prev => ({
                  ...prev,
                  scanning: { ...prev.scanning, parallel_scans: parseInt(e.target.value) }
                }))}
                min="1"
                max="10"
                className="w-full rounded-lg border border-dark-300 bg-white px-4 py-2 text-dark-900 focus:border-primary-500 focus:ring-2 focus:ring-primary-500/20"
              />
            </div>
          </div>
        </Card>

        {/* Notification Settings */}
        <Card>
          <div className="flex items-center gap-2 mb-4">
            <Bell className="h-5 w-5 text-primary-500" />
            <h2 className="text-lg font-semibold text-dark-900">Notification Settings</h2>
          </div>

          <div className="space-y-4">
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={settings.notifications.email_enabled}
                onChange={(e) => setSettings(prev => ({
                  ...prev,
                  notifications: { ...prev.notifications, email_enabled: e.target.checked }
                }))}
                className="rounded text-primary-500 focus:ring-primary-500"
              />
              <span className="text-sm text-dark-700">Enable Email Notifications</span>
            </label>

            <div>
              <label className="block text-sm font-medium text-dark-700 mb-2">
                Email Address
              </label>
              <input
                type="email"
                value={settings.notifications.email_address}
                onChange={(e) => setSettings(prev => ({
                  ...prev,
                  notifications: { ...prev.notifications, email_address: e.target.value }
                }))}
                disabled={!settings.notifications.email_enabled}
                className="w-full rounded-lg border border-dark-300 bg-white px-4 py-2 text-dark-900 focus:border-primary-500 focus:ring-2 focus:ring-primary-500/20 disabled:opacity-50"
              />
            </div>

            <div>
              <p className="text-sm font-medium text-dark-700 mb-2">Alert Levels</p>
              <div className="space-y-2">
                <label className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={settings.notifications.alert_critical}
                    onChange={(e) => setSettings(prev => ({
                      ...prev,
                      notifications: { ...prev.notifications, alert_critical: e.target.checked }
                    }))}
                    className="rounded text-danger-500 focus:ring-danger-500"
                  />
                  <span className="text-sm text-dark-700">Critical</span>
                  <Badge variant="danger" size="sm">High Priority</Badge>
                </label>

                <label className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={settings.notifications.alert_high}
                    onChange={(e) => setSettings(prev => ({
                      ...prev,
                      notifications: { ...prev.notifications, alert_high: e.target.checked }
                    }))}
                    className="rounded text-warning-500 focus:ring-warning-500"
                  />
                  <span className="text-sm text-dark-700">High</span>
                  <Badge variant="warning" size="sm">Important</Badge>
                </label>

                <label className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={settings.notifications.alert_medium}
                    onChange={(e) => setSettings(prev => ({
                      ...prev,
                      notifications: { ...prev.notifications, alert_medium: e.target.checked }
                    }))}
                    className="rounded text-yellow-500 focus:ring-yellow-500"
                  />
                  <span className="text-sm text-dark-700">Medium</span>
                </label>

                <label className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={settings.notifications.alert_low}
                    onChange={(e) => setSettings(prev => ({
                      ...prev,
                      notifications: { ...prev.notifications, alert_low: e.target.checked }
                    }))}
                    className="rounded text-blue-500 focus:ring-blue-500"
                  />
                  <span className="text-sm text-dark-700">Low</span>
                </label>
              </div>
            </div>
          </div>
        </Card>

        {/* Database Settings */}
        <Card>
          <div className="flex items-center gap-2 mb-4">
            <Database className="h-5 w-5 text-primary-500" />
            <h2 className="text-lg font-semibold text-dark-900">Database Settings</h2>
          </div>

          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-dark-700 mb-2">
                Connection Pool Size
              </label>
              <input
                type="number"
                value={settings.database.connection_pool}
                onChange={(e) => setSettings(prev => ({
                  ...prev,
                  database: { ...prev.database, connection_pool: parseInt(e.target.value) }
                }))}
                min="1"
                max="100"
                className="w-full rounded-lg border border-dark-300 bg-white px-4 py-2 text-dark-900 focus:border-primary-500 focus:ring-2 focus:ring-primary-500/20"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-dark-700 mb-2">
                Query Timeout (ms)
              </label>
              <input
                type="number"
                value={settings.database.query_timeout}
                onChange={(e) => setSettings(prev => ({
                  ...prev,
                  database: { ...prev.database, query_timeout: parseInt(e.target.value) }
                }))}
                className="w-full rounded-lg border border-dark-300 bg-white px-4 py-2 text-dark-900 focus:border-primary-500 focus:ring-2 focus:ring-primary-500/20"
              />
            </div>

            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={settings.database.auto_backup}
                onChange={(e) => setSettings(prev => ({
                  ...prev,
                  database: { ...prev.database, auto_backup: e.target.checked }
                }))}
                className="rounded text-primary-500 focus:ring-primary-500"
              />
              <span className="text-sm text-dark-700">Enable Automatic Backups</span>
            </label>

            <div>
              <label className="block text-sm font-medium text-dark-700 mb-2">
                Backup Interval (seconds)
              </label>
              <input
                type="number"
                value={settings.database.backup_interval}
                onChange={(e) => setSettings(prev => ({
                  ...prev,
                  database: { ...prev.database, backup_interval: parseInt(e.target.value) }
                }))}
                disabled={!settings.database.auto_backup}
                className="w-full rounded-lg border border-dark-300 bg-white px-4 py-2 text-dark-900 focus:border-primary-500 focus:ring-2 focus:ring-primary-500/20 disabled:opacity-50"
              />
            </div>
          </div>
        </Card>
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

      {/* System Info */}
      <Card>
        <div className="flex items-center gap-2 mb-4">
          <Globe className="h-5 w-5 text-primary-500" />
          <h2 className="text-lg font-semibold text-dark-900">System Information</h2>
        </div>

        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div>
            <p className="text-sm font-medium text-dark-700">Version</p>
            <p className="text-dark-900">v2.0.0</p>
          </div>
          <div>
            <p className="text-sm font-medium text-dark-700">Environment</p>
            <Badge variant="primary">Production</Badge>
          </div>
          <div>
            <p className="text-sm font-medium text-dark-700">Database</p>
            <p className="text-dark-900">PostgreSQL 14</p>
          </div>
          <div>
            <p className="text-sm font-medium text-dark-700">License</p>
            <p className="text-dark-900">MIT</p>
          </div>
        </div>
      </Card>
    </div>
  )
}
