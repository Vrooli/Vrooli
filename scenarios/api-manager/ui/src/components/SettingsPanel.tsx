import { useState } from 'react'
import { 
  Settings,
  Save,
  RefreshCw,
  Database,
  Globe,
  Shield,
  Bell,
  Clock,
  Check
} from 'lucide-react'
import clsx from 'clsx'
import { Card } from './common/Card'
import { Badge } from './common/Badge'

export default function SettingsPanel() {
  const [settings, setSettings] = useState({
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