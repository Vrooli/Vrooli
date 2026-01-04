/**
 * DiagnosticsTab Component
 *
 * System health monitoring and diagnostics panel.
 * Provides visibility into all subsystems of the browser automation studio.
 */

import { useState, useCallback, useMemo } from 'react';
import {
  Activity,
  RefreshCw,
  Download,
  ChevronDown,
  ChevronRight,
  AlertTriangle,
  CheckCircle2,
  AlertCircle,
  Loader2,
  Clock,
  Wifi,
  Server,
  Monitor,
  Video,
  Trash2,
  BarChart3,
  Settings2,
  Zap,
  Play,
} from 'lucide-react';
import { useObservability, useRefreshDiagnostics, useRefreshCache } from '@/domains/observability';
import type { ComponentStatus, BrowserComponent, SessionsComponent, RecordingComponent, CleanupComponent, MetricsComponent } from '@/domains/observability';
import { SettingSection } from '../shared';

// Status indicator colors
// Note: API returns 'ok' | 'degraded' | 'error' for overall status,
// and 'healthy' | 'degraded' | 'error' for component status
type StatusColorKey = ComponentStatus | 'ok' | 'loading';
const STATUS_COLORS: Record<StatusColorKey, { bg: string; text: string; icon: typeof CheckCircle2 }> = {
  ok: { bg: 'bg-emerald-500/20', text: 'text-emerald-400', icon: CheckCircle2 },
  healthy: { bg: 'bg-emerald-500/20', text: 'text-emerald-400', icon: CheckCircle2 },
  degraded: { bg: 'bg-amber-500/20', text: 'text-amber-400', icon: AlertTriangle },
  error: { bg: 'bg-red-500/20', text: 'text-red-400', icon: AlertCircle },
  loading: { bg: 'bg-gray-500/20', text: 'text-gray-400', icon: Loader2 },
};

// Format uptime in human-readable format
function formatUptime(ms: number): string {
  const seconds = Math.floor(ms / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  if (days > 0) {
    return `${days}d ${hours % 24}h`;
  }
  if (hours > 0) {
    return `${hours}h ${minutes % 60}m`;
  }
  if (minutes > 0) {
    return `${minutes}m ${seconds % 60}s`;
  }
  return `${seconds}s`;
}

// Format relative time
function formatRelativeTime(dateString: string | undefined): string {
  if (!dateString) return 'Never';
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSeconds = Math.floor(diffMs / 1000);

  if (diffSeconds < 60) return `${diffSeconds}s ago`;
  const diffMinutes = Math.floor(diffSeconds / 60);
  if (diffMinutes < 60) return `${diffMinutes}m ago`;
  const diffHours = Math.floor(diffMinutes / 60);
  if (diffHours < 24) return `${diffHours}h ago`;
  return date.toLocaleDateString();
}

interface StatusBadgeProps {
  status: StatusColorKey;
  label?: string;
}

function StatusBadge({ status, label }: StatusBadgeProps) {
  const config = STATUS_COLORS[status] ?? STATUS_COLORS.error;
  const Icon = config.icon;

  return (
    <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium ${config.bg} ${config.text}`}>
      <Icon size={12} className={status === 'loading' ? 'animate-spin' : ''} />
      {label || status.charAt(0).toUpperCase() + status.slice(1)}
    </span>
  );
}

interface ComponentRowProps {
  icon: React.ReactNode;
  title: string;
  status: ComponentStatus;
  summary: string;
  details?: React.ReactNode;
  hint?: string;
}

function ComponentRow({ icon, title, status, summary, details, hint }: ComponentRowProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const hasDetails = !!details;

  return (
    <div className="border border-gray-700 rounded-lg overflow-hidden">
      <button
        type="button"
        onClick={() => hasDetails && setIsExpanded(!isExpanded)}
        className={`w-full flex items-center gap-4 px-4 py-3 bg-gray-800/50 text-left ${hasDetails ? 'hover:bg-gray-800 cursor-pointer' : 'cursor-default'} transition-colors`}
        disabled={!hasDetails}
      >
        <div className="flex items-center gap-3 flex-1 min-w-0">
          <div className="text-gray-400 flex-shrink-0">{icon}</div>
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <span className="font-medium text-surface truncate">{title}</span>
              <StatusBadge status={status} />
            </div>
            <p className="text-xs text-gray-400 truncate mt-0.5">{summary}</p>
          </div>
        </div>
        {hasDetails && (
          <div className="text-gray-400 flex-shrink-0">
            {isExpanded ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
          </div>
        )}
      </button>
      {isExpanded && details && (
        <div className="p-4 bg-gray-900/50 border-t border-gray-700">
          {hint && (
            <div className="mb-3 p-2 bg-amber-500/10 border border-amber-500/30 rounded text-xs text-amber-300">
              <strong>Hint:</strong> {hint}
            </div>
          )}
          {details}
        </div>
      )}
    </div>
  );
}

function BrowserDetails({ browser }: { browser: BrowserComponent }) {
  return (
    <div className="grid grid-cols-2 gap-3 text-sm">
      <div>
        <span className="text-gray-500">Version:</span>
        <span className="ml-2 text-surface">{browser.version || 'Unknown'}</span>
      </div>
      <div>
        <span className="text-gray-500">Connected:</span>
        <span className="ml-2 text-surface">{browser.connected ? 'Yes' : 'No'}</span>
      </div>
      <div>
        <span className="text-gray-500">Provider:</span>
        <span className="ml-2 text-surface">{browser.provider || 'Default'}</span>
      </div>
      {browser.capabilities && (
        <div className="col-span-2">
          <span className="text-gray-500">Capabilities:</span>
          <div className="mt-1 flex flex-wrap gap-2">
            {browser.capabilities.evaluate_isolated && (
              <span className="px-2 py-0.5 bg-gray-800 text-gray-300 text-xs rounded">evaluateIsolated</span>
            )}
            {browser.capabilities.expose_binding_isolated && (
              <span className="px-2 py-0.5 bg-gray-800 text-gray-300 text-xs rounded">exposeBindingIsolated</span>
            )}
            {browser.capabilities.has_anti_detection && (
              <span className="px-2 py-0.5 bg-emerald-500/20 text-emerald-300 text-xs rounded">Anti-Detection</span>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

function SessionsDetails({ sessions }: { sessions: SessionsComponent }) {
  const utilizationPercent = sessions.capacity > 0 ? Math.round((sessions.total / sessions.capacity) * 100) : 0;

  return (
    <div className="space-y-3">
      <div className="grid grid-cols-3 gap-3 text-sm">
        <div className="text-center p-2 bg-gray-800 rounded">
          <div className="text-lg font-semibold text-surface">{sessions.active}</div>
          <div className="text-xs text-gray-400">Active</div>
        </div>
        <div className="text-center p-2 bg-gray-800 rounded">
          <div className="text-lg font-semibold text-surface">{sessions.idle}</div>
          <div className="text-xs text-gray-400">Idle</div>
        </div>
        <div className="text-center p-2 bg-gray-800 rounded">
          <div className="text-lg font-semibold text-surface">{sessions.active_recordings}</div>
          <div className="text-xs text-gray-400">Recording</div>
        </div>
      </div>
      <div>
        <div className="flex justify-between text-xs text-gray-400 mb-1">
          <span>Capacity</span>
          <span>{sessions.total}/{sessions.capacity} ({utilizationPercent}%)</span>
        </div>
        <div className="h-2 bg-gray-700 rounded-full overflow-hidden">
          <div
            className={`h-full transition-all ${
              utilizationPercent >= 90 ? 'bg-red-500' : utilizationPercent >= 70 ? 'bg-amber-500' : 'bg-emerald-500'
            }`}
            style={{ width: `${utilizationPercent}%` }}
          />
        </div>
      </div>
      <div className="text-xs text-gray-400">
        Idle timeout: {Math.round(sessions.idle_timeout_ms / 1000 / 60)}m
      </div>
    </div>
  );
}

function RecordingDetails({ recording }: { recording: RecordingComponent }) {
  const stats = recording.injection_stats;
  const successRate = stats && stats.total > 0 ? Math.round((stats.successful / stats.total) * 100) : 100;

  return (
    <div className="space-y-3 text-sm">
      {recording.script_version && (
        <div>
          <span className="text-gray-500">Script Version:</span>
          <span className="ml-2 text-surface">{recording.script_version}</span>
        </div>
      )}
      {stats && (
        <div>
          <div className="flex justify-between text-xs text-gray-400 mb-1">
            <span>Injection Success Rate</span>
            <span>{stats.successful}/{stats.total} ({successRate}%)</span>
          </div>
          <div className="h-2 bg-gray-700 rounded-full overflow-hidden">
            <div
              className={`h-full transition-all ${
                successRate >= 95 ? 'bg-emerald-500' : successRate >= 80 ? 'bg-amber-500' : 'bg-red-500'
              }`}
              style={{ width: `${successRate}%` }}
            />
          </div>
        </div>
      )}
    </div>
  );
}

function CleanupDetails({ cleanup }: { cleanup: CleanupComponent }) {
  return (
    <div className="grid grid-cols-2 gap-3 text-sm">
      <div>
        <span className="text-gray-500">Status:</span>
        <span className={`ml-2 ${cleanup.is_running ? 'text-amber-400' : 'text-surface'}`}>
          {cleanup.is_running ? 'Running' : 'Idle'}
        </span>
      </div>
      <div>
        <span className="text-gray-500">Interval:</span>
        <span className="ml-2 text-surface">{Math.round(cleanup.interval_ms / 1000)}s</span>
      </div>
      <div>
        <span className="text-gray-500">Last Run:</span>
        <span className="ml-2 text-surface">{formatRelativeTime(cleanup.last_run_at)}</span>
      </div>
      {cleanup.next_run_in_ms !== undefined && (
        <div>
          <span className="text-gray-500">Next Run:</span>
          <span className="ml-2 text-surface">{Math.round(cleanup.next_run_in_ms / 1000)}s</span>
        </div>
      )}
    </div>
  );
}

function MetricsDetails({ metrics }: { metrics: MetricsComponent }) {
  return (
    <div className="grid grid-cols-2 gap-3 text-sm">
      <div>
        <span className="text-gray-500">Enabled:</span>
        <span className={`ml-2 ${metrics.enabled ? 'text-emerald-400' : 'text-gray-400'}`}>
          {metrics.enabled ? 'Yes' : 'No'}
        </span>
      </div>
      {metrics.enabled && (
        <>
          <div>
            <span className="text-gray-500">Port:</span>
            <span className="ml-2 text-surface">{metrics.port}</span>
          </div>
          <div className="col-span-2">
            <span className="text-gray-500">Endpoint:</span>
            <span className="ml-2 text-surface font-mono text-xs">{metrics.endpoint}</span>
          </div>
        </>
      )}
    </div>
  );
}

export function DiagnosticsTab() {
  const { data, isLoading, isFetching, error, refetch, dataUpdatedAt } = useObservability({
    depth: 'standard',
    refetchInterval: 30000,
  });

  const { runDiagnostics, isRunning: isDiagnosticsRunning, result: diagnosticsResult } = useRefreshDiagnostics();
  const { refresh: refreshCache, isRefreshing: isRefreshingCache } = useRefreshCache();

  const handleRefresh = useCallback(async () => {
    refreshCache();
    await refetch();
  }, [refreshCache, refetch]);

  const handleRunDiagnostics = useCallback(async () => {
    await runDiagnostics({ type: 'all', options: { level: 'full' } });
  }, [runDiagnostics]);

  const handleExportJson = useCallback(() => {
    if (!data) return;

    const json = JSON.stringify(data, null, 2);
    const blob = new Blob([json], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `diagnostics-${new Date().toISOString().slice(0, 19).replace(/:/g, '-')}.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  }, [data]);

  const overallStatus = useMemo((): StatusColorKey => {
    if (isLoading) return 'loading';
    if (error) return 'error';
    return data?.status || 'error';
  }, [isLoading, error, data?.status]);

  const lastUpdated = useMemo(() => {
    if (!dataUpdatedAt) return 'Never';
    return formatRelativeTime(new Date(dataUpdatedAt).toISOString());
  }, [dataUpdatedAt]);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <Activity size={24} className="text-flow-accent" />
          <div>
            <h2 className="text-lg font-semibold text-surface">System Diagnostics</h2>
            <p className="text-sm text-gray-400">Monitor system health and troubleshoot issues</p>
          </div>
        </div>
        <button
          onClick={handleExportJson}
          disabled={!data || isLoading}
          className="flex items-center gap-2 px-3 py-2 text-sm text-subtle hover:text-surface hover:bg-gray-700 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          title="Export diagnostics as JSON"
        >
          <Download size={16} />
          <span className="hidden sm:inline">JSON</span>
        </button>
      </div>

      {/* Error Banner */}
      {error && (
        <div className="bg-red-500/10 border border-red-500/30 rounded-lg p-4">
          <div className="flex items-start gap-3">
            <AlertCircle size={20} className="text-red-400 flex-shrink-0 mt-0.5" />
            <div>
              <p className="text-sm text-red-200 font-medium">Failed to load diagnostics</p>
              <p className="text-xs text-red-300/80 mt-1">{error.message}</p>
            </div>
          </div>
        </div>
      )}

      {/* System Health Card */}
      <div className="bg-gray-800/50 border border-gray-700 rounded-lg p-4">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div className="flex items-center gap-4">
            <StatusBadge status={overallStatus} label={isLoading ? 'Loading...' : undefined} />
            <div className="flex items-center gap-4 text-sm text-gray-400">
              {data?.ready !== undefined && (
                <span className="flex items-center gap-1">
                  <Zap size={14} className={data.ready ? 'text-emerald-400' : 'text-amber-400'} />
                  {data.ready ? 'Ready' : 'Not Ready'}
                </span>
              )}
              {data?.uptime_ms !== undefined && (
                <span className="flex items-center gap-1">
                  <Clock size={14} />
                  {formatUptime(data.uptime_ms)}
                </span>
              )}
              {data?.version && (
                <span className="hidden sm:inline text-gray-500">v{data.version}</span>
              )}
            </div>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-xs text-gray-500">Updated {lastUpdated}</span>
            <button
              onClick={handleRefresh}
              disabled={isFetching || isRefreshingCache}
              className="flex items-center gap-1.5 px-3 py-1.5 text-sm text-gray-400 hover:text-surface bg-gray-700 hover:bg-gray-600 rounded-lg transition-colors disabled:opacity-50"
            >
              <RefreshCw size={14} className={isFetching || isRefreshingCache ? 'animate-spin' : ''} />
              Refresh
            </button>
          </div>
        </div>

        {/* Quick Summary */}
        {data?.summary && (
          <div className="grid grid-cols-3 gap-4 mt-4 pt-4 border-t border-gray-700">
            <div className="text-center">
              <div className="text-2xl font-semibold text-surface">{data.summary.sessions}</div>
              <div className="text-xs text-gray-400">Sessions</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-semibold text-surface">{data.summary.recordings}</div>
              <div className="text-xs text-gray-400">Recordings</div>
            </div>
            <div className="text-center">
              <div className={`text-2xl font-semibold ${data.summary.browser_connected ? 'text-emerald-400' : 'text-red-400'}`}>
                {data.summary.browser_connected ? <Wifi size={24} className="mx-auto" /> : <AlertCircle size={24} className="mx-auto" />}
              </div>
              <div className="text-xs text-gray-400">Browser</div>
            </div>
          </div>
        )}
      </div>

      {/* Components */}
      {data?.components && (
        <SettingSection title="Components" tooltip="Health status of individual system components">
          <div className="space-y-3">
            <ComponentRow
              icon={<Monitor size={18} />}
              title="Browser"
              status={data.components.browser.status}
              summary={data.components.browser.connected ? `Chromium ${data.components.browser.version || ''}` : 'Disconnected'}
              details={<BrowserDetails browser={data.components.browser} />}
              hint={data.components.browser.hint}
            />
            <ComponentRow
              icon={<Server size={18} />}
              title="Sessions"
              status={data.components.sessions.status}
              summary={`${data.components.sessions.total}/${data.components.sessions.capacity} sessions, ${data.components.sessions.active_recordings} recording`}
              details={<SessionsDetails sessions={data.components.sessions} />}
              hint={data.components.sessions.hint}
            />
            <ComponentRow
              icon={<Video size={18} />}
              title="Recording"
              status={data.components.recording.status}
              summary={data.components.recording.message || 'Recording subsystem healthy'}
              details={<RecordingDetails recording={data.components.recording} />}
              hint={data.components.recording.hint}
            />
            <ComponentRow
              icon={<Trash2 size={18} />}
              title="Cleanup"
              status="healthy"
              summary={`Interval: ${Math.round(data.components.cleanup.interval_ms / 1000)}s, Last: ${formatRelativeTime(data.components.cleanup.last_run_at)}`}
              details={<CleanupDetails cleanup={data.components.cleanup} />}
            />
            <ComponentRow
              icon={<BarChart3 size={18} />}
              title="Metrics"
              status={data.components.metrics.enabled ? 'healthy' : 'degraded'}
              summary={data.components.metrics.enabled ? `Port ${data.components.metrics.port}` : 'Disabled'}
              details={<MetricsDetails metrics={data.components.metrics} />}
            />
          </div>
        </SettingSection>
      )}

      {/* Configuration */}
      {data?.config && (
        <SettingSection title="Configuration" tooltip="Current configuration status and modified options" defaultOpen={false}>
          <div className="space-y-3">
            <div className="flex items-center gap-2 text-sm">
              <Settings2 size={16} className="text-gray-400" />
              <span className="text-surface">{data.config.summary}</span>
            </div>
            {data.config.modified_count > 0 && data.config.modified_options && (
              <div className="mt-3 space-y-2">
                <p className="text-xs text-gray-400 uppercase tracking-wider">Modified Options ({data.config.modified_count})</p>
                <div className="space-y-1">
                  {data.config.modified_options.map((opt) => (
                    <div key={opt.env_var} className="flex items-center justify-between text-sm bg-gray-800 rounded px-3 py-2">
                      <span className="font-mono text-xs text-gray-300">{opt.env_var}</span>
                      <div className="flex items-center gap-2">
                        <span className={`text-xs px-1.5 py-0.5 rounded ${
                          opt.tier === 'essential' ? 'bg-emerald-500/20 text-emerald-300' :
                          opt.tier === 'advanced' ? 'bg-amber-500/20 text-amber-300' :
                          'bg-red-500/20 text-red-300'
                        }`}>
                          {opt.tier}
                        </span>
                        <span className="text-gray-400">{String(opt.current_value)}</span>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        </SettingSection>
      )}

      {/* Deep Diagnostics */}
      <SettingSection title="Deep Diagnostics" tooltip="Run comprehensive diagnostic scans" defaultOpen={false}>
        <div className="space-y-4">
          <p className="text-sm text-gray-400">
            Run full diagnostic scans to check script injection, event flow, and other subsystems.
            These scans may take up to 5 seconds.
          </p>
          <div className="flex flex-wrap gap-3">
            <button
              onClick={handleRunDiagnostics}
              disabled={isDiagnosticsRunning}
              className="flex items-center gap-2 px-4 py-2 bg-flow-accent text-white rounded-lg hover:bg-blue-600 transition-colors disabled:opacity-50"
            >
              {isDiagnosticsRunning ? (
                <>
                  <Loader2 size={16} className="animate-spin" />
                  Running...
                </>
              ) : (
                <>
                  <Play size={16} />
                  Run Full Scan
                </>
              )}
            </button>
          </div>

          {/* Diagnostics Results */}
          {diagnosticsResult && (
            <div className="mt-4 p-4 bg-gray-800 rounded-lg">
              <div className="flex items-center justify-between mb-3">
                <span className="text-sm font-medium text-surface">Last Scan Results</span>
                <span className="text-xs text-gray-400">{diagnosticsResult.duration_ms}ms</span>
              </div>
              {diagnosticsResult.results.recording && (
                <div className="space-y-2">
                  <div className="flex items-center gap-2">
                    <StatusBadge status={diagnosticsResult.results.recording.ready ? 'healthy' : 'error'} />
                    <span className="text-sm text-gray-300">Recording Diagnostics</span>
                  </div>
                  {diagnosticsResult.results.recording.issues.length > 0 ? (
                    <div className="space-y-2 mt-2">
                      {diagnosticsResult.results.recording.issues.map((issue, idx) => (
                        <div
                          key={idx}
                          className={`p-3 rounded border text-sm ${
                            issue.severity === 'error' ? 'bg-red-500/10 border-red-500/30 text-red-300' :
                            issue.severity === 'warning' ? 'bg-amber-500/10 border-amber-500/30 text-amber-300' :
                            'bg-gray-700 border-gray-600 text-gray-300'
                          }`}
                        >
                          <div className="flex items-start gap-2">
                            {issue.severity === 'error' ? <AlertCircle size={14} className="flex-shrink-0 mt-0.5" /> :
                             issue.severity === 'warning' ? <AlertTriangle size={14} className="flex-shrink-0 mt-0.5" /> :
                             <CheckCircle2 size={14} className="flex-shrink-0 mt-0.5" />}
                            <div>
                              <span className="font-medium">[{issue.category}]</span> {issue.message}
                              {issue.suggestion && (
                                <p className="text-xs mt-1 opacity-80">Suggestion: {issue.suggestion}</p>
                              )}
                            </div>
                          </div>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <p className="text-sm text-emerald-400">No issues found</p>
                  )}
                </div>
              )}
            </div>
          )}
        </div>
      </SettingSection>
    </div>
  );
}

export default DiagnosticsTab;
