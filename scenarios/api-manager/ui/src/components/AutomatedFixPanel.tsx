import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { 
  Shield,
  AlertTriangle,
  Lock,
  Unlock,
  CheckCircle,
  XCircle,
  RefreshCw,
  Zap,
  History,
  FileCode,
  AlertOctagon,
  Settings
} from 'lucide-react'
import clsx from 'clsx'
import { format } from 'date-fns'
import { apiService } from '@/services/api'
import { Card } from './common/Card'
import { Badge } from './common/Badge'

export default function AutomatedFixPanel() {
  const [showEnableDialog, setShowEnableDialog] = useState(false)
  const [enableConfig, setEnableConfig] = useState({
    allowed_categories: ['Resource Leak', 'Error Handling'],
    max_confidence: 'high' as 'low' | 'medium' | 'high',
    require_approval: true,
    backup_enabled: true,
    rollback_window: 24,
  })
  const [confirmationChecked, setConfirmationChecked] = useState(false)
  
  const queryClient = useQueryClient()

  const { data: fixConfig, isLoading, refetch } = useQuery({
    queryKey: ['fixConfig'],
    queryFn: apiService.getFixConfig,
    refetchInterval: 30000,
  })

  const { data: fixHistory } = useQuery({
    queryKey: ['fixHistory'],
    queryFn: apiService.getFixHistory,
  })

  const enableMutation = useMutation({
    mutationFn: () => apiService.enableAutomatedFixes({
      ...enableConfig,
      confirmation_understood: true,
    }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['fixConfig'] })
      setShowEnableDialog(false)
      setConfirmationChecked(false)
    },
  })

  const disableMutation = useMutation({
    mutationFn: apiService.disableAutomatedFixes,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['fixConfig'] })
    },
  })

  const rollbackMutation = useMutation({
    mutationFn: (fixId: string) => apiService.rollbackFix(fixId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['fixHistory'] })
    },
  })

  const categories = [
    'Resource Leak',
    'Error Handling',
    'SQL Injection',
    'XSS',
    'Path Traversal',
    'Hardcoded Credentials',
    'Weak Cryptography',
  ]

  return (
    <div className="space-y-6 animate-in">
      {/* Header with Critical Safety Notice */}
      <div>
        <h1 className="text-2xl font-bold text-dark-900">Automated Fix Management</h1>
        <p className="text-dark-500 mt-1">Configure and monitor automated vulnerability fixes</p>
      </div>

      {/* Safety Status Card */}
      <Card className={clsx(
        'border-2',
        fixConfig?.enabled ? 'border-warning-500 bg-warning-50/30' : 'border-success-500 bg-success-50/30'
      )}>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            {fixConfig?.enabled ? (
              <div className="flex h-12 w-12 items-center justify-center rounded-full bg-warning-500 text-white">
                <Unlock className="h-6 w-6" />
              </div>
            ) : (
              <div className="flex h-12 w-12 items-center justify-center rounded-full bg-success-500 text-white">
                <Lock className="h-6 w-6" />
              </div>
            )}
            <div>
              <h2 className="text-lg font-bold text-dark-900">
                Automated Fixes are {fixConfig?.enabled ? 'ENABLED' : 'DISABLED'}
              </h2>
              <p className="text-sm text-dark-600">
                {fixConfig?.enabled 
                  ? '⚠️ System can automatically apply code fixes based on configuration'
                  : '✅ System is in safe mode - no automatic fixes will be applied'}
              </p>
            </div>
          </div>
          
          <div className="flex gap-2">
            {fixConfig?.enabled ? (
              <button
                onClick={() => disableMutation.mutate()}
                disabled={disableMutation.isPending}
                className="flex items-center gap-2 rounded-lg bg-danger-500 px-4 py-2 text-sm font-medium text-white hover:bg-danger-600 transition-colors"
              >
                {disableMutation.isPending ? (
                  <RefreshCw className="h-4 w-4 animate-spin" />
                ) : (
                  <Lock className="h-4 w-4" />
                )}
                Disable Now
              </button>
            ) : (
              <button
                onClick={() => setShowEnableDialog(true)}
                className="flex items-center gap-2 rounded-lg bg-dark-700 px-4 py-2 text-sm font-medium text-white hover:bg-dark-800 transition-colors"
              >
                <Settings className="h-4 w-4" />
                Configure & Enable
              </button>
            )}
            <button
              onClick={() => refetch()}
              className="flex items-center justify-center rounded-lg bg-dark-100 p-2 text-dark-700 hover:bg-dark-200 transition-colors"
            >
              <RefreshCw className="h-4 w-4" />
            </button>
          </div>
        </div>
      </Card>

      {/* Current Configuration */}
      <div className="grid gap-6 lg:grid-cols-2">
        <Card>
          <h3 className="text-lg font-semibold text-dark-900 mb-4">Current Configuration</h3>
          
          {isLoading ? (
            <div className="space-y-3">
              {[1, 2, 3, 4].map(i => (
                <div key={i} className="h-12 bg-dark-100 rounded animate-pulse" />
              ))}
            </div>
          ) : fixConfig ? (
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium text-dark-700">Status</span>
                <Badge variant={fixConfig.enabled ? 'warning' : 'success'}>
                  {fixConfig.enabled ? 'Enabled' : 'Disabled'}
                </Badge>
              </div>
              
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium text-dark-700">Maximum Confidence Level</span>
                <Badge variant="default">{fixConfig.max_confidence}</Badge>
              </div>
              
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium text-dark-700">Manual Approval Required</span>
                {fixConfig.require_approval ? (
                  <CheckCircle className="h-5 w-5 text-success-500" />
                ) : (
                  <XCircle className="h-5 w-5 text-danger-500" />
                )}
              </div>
              
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium text-dark-700">Automatic Backups</span>
                {fixConfig.backup_enabled ? (
                  <CheckCircle className="h-5 w-5 text-success-500" />
                ) : (
                  <XCircle className="h-5 w-5 text-danger-500" />
                )}
              </div>
              
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium text-dark-700">Rollback Window</span>
                <span className="text-dark-900">{fixConfig.rollback_window} hours</span>
              </div>
              
              <div>
                <p className="text-sm font-medium text-dark-700 mb-2">Allowed Categories</p>
                <div className="flex flex-wrap gap-2">
                  {fixConfig.allowed_categories.map(cat => (
                    <Badge key={cat} variant="primary" size="sm">{cat}</Badge>
                  ))}
                </div>
              </div>
            </div>
          ) : null}
        </Card>

        {/* Safety Controls */}
        <Card>
          <h3 className="text-lg font-semibold text-dark-900 mb-4">Safety Controls</h3>
          
          <div className="space-y-3">
            <div className="flex items-start gap-3 p-3 rounded-lg bg-dark-50">
              <Shield className="h-5 w-5 text-primary-500 mt-0.5" />
              <div className="flex-1">
                <p className="text-sm font-medium text-dark-900">Multi-Layer Validation</p>
                <p className="text-xs text-dark-600 mt-0.5">
                  Each fix goes through category, confidence, and approval checks
                </p>
              </div>
            </div>
            
            <div className="flex items-start gap-3 p-3 rounded-lg bg-dark-50">
              <History className="h-5 w-5 text-primary-500 mt-0.5" />
              <div className="flex-1">
                <p className="text-sm font-medium text-dark-900">Automatic Backups</p>
                <p className="text-xs text-dark-600 mt-0.5">
                  Files are backed up before any modification
                </p>
              </div>
            </div>
            
            <div className="flex items-start gap-3 p-3 rounded-lg bg-dark-50">
              <RefreshCw className="h-5 w-5 text-primary-500 mt-0.5" />
              <div className="flex-1">
                <p className="text-sm font-medium text-dark-900">24-Hour Rollback</p>
                <p className="text-xs text-dark-600 mt-0.5">
                  Any fix can be rolled back within 24 hours
                </p>
              </div>
            </div>
            
            <div className="flex items-start gap-3 p-3 rounded-lg bg-dark-50">
              <AlertOctagon className="h-5 w-5 text-primary-500 mt-0.5" />
              <div className="flex-1">
                <p className="text-sm font-medium text-dark-900">Explicit Confirmation</p>
                <p className="text-xs text-dark-600 mt-0.5">
                  Enabling requires acknowledgment of risks
                </p>
              </div>
            </div>
          </div>
        </Card>
      </div>

      {/* Fix History */}
      <Card>
        <h3 className="text-lg font-semibold text-dark-900 mb-4">Fix History</h3>
        
        {fixHistory && fixHistory.length > 0 ? (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-dark-200">
                  <th className="text-left text-xs font-medium text-dark-500 uppercase tracking-wider pb-2">
                    Fix ID
                  </th>
                  <th className="text-left text-xs font-medium text-dark-500 uppercase tracking-wider pb-2">
                    Scenario
                  </th>
                  <th className="text-left text-xs font-medium text-dark-500 uppercase tracking-wider pb-2">
                    Category
                  </th>
                  <th className="text-left text-xs font-medium text-dark-500 uppercase tracking-wider pb-2">
                    Confidence
                  </th>
                  <th className="text-left text-xs font-medium text-dark-500 uppercase tracking-wider pb-2">
                    Status
                  </th>
                  <th className="text-left text-xs font-medium text-dark-500 uppercase tracking-wider pb-2">
                    Applied
                  </th>
                  <th className="text-left text-xs font-medium text-dark-500 uppercase tracking-wider pb-2">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-dark-100">
                {fixHistory.map((fix) => (
                  <tr key={fix.id} className="hover:bg-dark-50">
                    <td className="py-3 text-sm font-mono text-dark-600">
                      {fix.id.substring(0, 8)}...
                    </td>
                    <td className="py-3 text-sm text-dark-900">
                      {fix.scenario_name}
                    </td>
                    <td className="py-3">
                      <Badge variant="default" size="sm">{fix.category}</Badge>
                    </td>
                    <td className="py-3">
                      <Badge 
                        variant={
                          fix.confidence === 'high' ? 'success' :
                          fix.confidence === 'medium' ? 'warning' :
                          'danger'
                        }
                        size="sm"
                      >
                        {fix.confidence}
                      </Badge>
                    </td>
                    <td className="py-3">
                      <Badge
                        variant={
                          fix.status === 'applied' ? 'success' :
                          fix.status === 'rolled_back' ? 'warning' :
                          'default'
                        }
                        size="sm"
                      >
                        {fix.status}
                      </Badge>
                    </td>
                    <td className="py-3 text-sm text-dark-500">
                      {fix.applied_at && !isNaN(new Date(fix.applied_at).getTime())
                        ? format(new Date(fix.applied_at), 'MMM d, HH:mm')
                        : 'N/A'}
                    </td>
                    <td className="py-3">
                      {fix.status === 'applied' && (
                        <button
                          onClick={() => rollbackMutation.mutate(fix.id)}
                          disabled={rollbackMutation.isPending}
                          className="text-sm text-danger-600 hover:text-danger-700 font-medium"
                        >
                          Rollback
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="text-center py-8">
            <FileCode className="h-12 w-12 text-dark-300 mx-auto mb-3" />
            <p className="text-sm text-dark-600">No fixes have been applied yet</p>
            <p className="text-xs text-dark-500 mt-1">Automated fixes are currently disabled</p>
          </div>
        )}
      </Card>

      {/* Enable Dialog */}
      {showEnableDialog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
          <div className="w-full max-w-2xl bg-white rounded-xl shadow-xl p-6 m-4">
            <div className="flex items-center gap-3 mb-4">
              <AlertTriangle className="h-8 w-8 text-warning-500" />
              <h2 className="text-xl font-bold text-dark-900">Enable Automated Fixes</h2>
            </div>
            
            <div className="mb-6 p-4 rounded-lg bg-warning-50 border border-warning-300">
              <p className="text-sm font-medium text-warning-900 mb-2">⚠️ Critical Safety Warning</p>
              <p className="text-sm text-warning-800">
                Enabling automated fixes allows the system to modify your code automatically. 
                While safety controls are in place, you should understand the risks:
              </p>
              <ul className="mt-2 space-y-1 text-sm text-warning-700 list-disc list-inside">
                <li>Code may be modified without your direct review</li>
                <li>Fixes may introduce unexpected behavior</li>
                <li>Always test thoroughly after fixes are applied</li>
                <li>Keep backups of critical code</li>
              </ul>
            </div>
            
            <div className="space-y-4 mb-6">
              <div>
                <label className="block text-sm font-medium text-dark-700 mb-2">
                  Allowed Categories
                </label>
                <div className="grid grid-cols-2 gap-2">
                  {categories.map(cat => (
                    <label key={cat} className="flex items-center gap-2">
                      <input
                        type="checkbox"
                        checked={enableConfig.allowed_categories.includes(cat)}
                        onChange={(e) => {
                          if (e.target.checked) {
                            setEnableConfig(prev => ({
                              ...prev,
                              allowed_categories: [...prev.allowed_categories, cat]
                            }))
                          } else {
                            setEnableConfig(prev => ({
                              ...prev,
                              allowed_categories: prev.allowed_categories.filter(c => c !== cat)
                            }))
                          }
                        }}
                        className="rounded text-primary-500 focus:ring-primary-500"
                      />
                      <span className="text-sm text-dark-700">{cat}</span>
                    </label>
                  ))}
                </div>
              </div>
              
              <div>
                <label className="block text-sm font-medium text-dark-700 mb-2">
                  Maximum Confidence Level
                </label>
                <select
                  value={enableConfig.max_confidence}
                  onChange={(e) => setEnableConfig(prev => ({ 
                    ...prev, 
                    max_confidence: e.target.value as any 
                  }))}
                  className="w-full rounded-lg border border-dark-300 bg-white px-4 py-2 text-dark-900"
                >
                  <option value="low">Low</option>
                  <option value="medium">Medium</option>
                  <option value="high">High (Recommended)</option>
                </select>
              </div>
              
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={enableConfig.require_approval}
                  onChange={(e) => setEnableConfig(prev => ({ 
                    ...prev, 
                    require_approval: e.target.checked 
                  }))}
                  className="rounded text-primary-500"
                />
                <span className="text-sm text-dark-700">Require manual approval for each fix</span>
              </label>
              
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={enableConfig.backup_enabled}
                  onChange={(e) => setEnableConfig(prev => ({ 
                    ...prev, 
                    backup_enabled: e.target.checked 
                  }))}
                  className="rounded text-primary-500"
                />
                <span className="text-sm text-dark-700">Create automatic backups before fixes</span>
              </label>
            </div>
            
            <div className="mb-6 p-4 rounded-lg bg-danger-50 border border-danger-300">
              <label className="flex items-start gap-2">
                <input
                  type="checkbox"
                  checked={confirmationChecked}
                  onChange={(e) => setConfirmationChecked(e.target.checked)}
                  className="rounded text-danger-500 mt-1"
                />
                <span className="text-sm text-danger-900">
                  I understand the risks and confirm that I want to enable automated fixes 
                  with the above configuration. I have backups of my code and will test 
                  thoroughly after any fixes are applied.
                </span>
              </label>
            </div>
            
            <div className="flex gap-3 justify-end">
              <button
                onClick={() => {
                  setShowEnableDialog(false)
                  setConfirmationChecked(false)
                }}
                className="px-4 py-2 text-sm font-medium text-dark-700 hover:bg-dark-100 rounded-lg transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={() => enableMutation.mutate()}
                disabled={!confirmationChecked || enableConfig.allowed_categories.length === 0 || enableMutation.isPending}
                className="flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-danger-500 hover:bg-danger-600 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                {enableMutation.isPending ? (
                  <>
                    <RefreshCw className="h-4 w-4 animate-spin" />
                    Enabling...
                  </>
                ) : (
                  <>
                    <Unlock className="h-4 w-4" />
                    Enable Automated Fixes
                  </>
                )}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}