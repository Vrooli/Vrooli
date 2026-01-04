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
  Code,
  Copy,
  Check,
  ArrowLeft,
  ExternalLink,
} from 'lucide-react';
import { useObservability, useRefreshDiagnostics, useRefreshCache, useMetrics } from '@/domains/observability';
import type { ComponentStatus, BrowserComponent, SessionsComponent, RecordingComponent, CleanupComponent, MetricsComponent, MetricsResponse, MetricData, MetricValue, ConfigOption, ConfigTier } from '@/domains/observability';
import { SettingSection } from '../shared';

type ViewMode = 'dashboard' | 'json' | 'metrics';

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
  const hasData = recording.script_version || (stats && stats.total > 0);

  return (
    <div className="space-y-3 text-sm">
      {/* Active recordings count */}
      <div>
        <span className="text-gray-500">Active Recordings:</span>
        <span className="ml-2 text-surface">{recording.active_count}</span>
      </div>

      {/* Show data if we have it, otherwise show helpful empty state */}
      {hasData ? (
        <>
          {recording.script_version && (
            <div>
              <span className="text-gray-500">Script Version:</span>
              <span className="ml-2 text-surface">{recording.script_version}</span>
            </div>
          )}
          {stats && stats.total > 0 && (
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
              {stats.methods && (
                <div className="mt-2 grid grid-cols-2 gap-2 text-xs">
                  <div className="text-gray-500">Injection Methods:</div>
                  <div className="text-gray-400">
                    {Object.entries(stats.methods)
                      .filter(([, count]) => count > 0)
                      .map(([method, count]) => `${method}: ${count}`)
                      .join(', ') || 'None'}
                  </div>
                </div>
              )}
            </div>
          )}
        </>
      ) : (
        <div className="p-3 bg-gray-800/50 border border-gray-700 rounded text-gray-400 text-center">
          <Video size={24} className="mx-auto mb-2 text-gray-500" />
          <p>No recording sessions active</p>
          <p className="text-xs text-gray-500 mt-1">Start a recording to see injection statistics</p>
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

function MetricsDetails({ metrics, onViewMetrics }: { metrics: MetricsComponent; onViewMetrics?: () => void }) {
  return (
    <div className="space-y-3">
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
      {metrics.enabled && onViewMetrics && (
        <button
          onClick={onViewMetrics}
          className="flex items-center gap-2 px-3 py-2 text-sm bg-gray-700 hover:bg-gray-600 text-gray-300 hover:text-surface rounded-lg transition-colors"
        >
          <BarChart3 size={14} />
          View Metrics Data
        </button>
      )}
    </div>
  );
}

// JSON Viewer Component
interface JsonViewerProps {
  data: unknown;
  onBack: () => void;
  title?: string;
}

function JsonViewer({ data, onBack, title = 'JSON Data' }: JsonViewerProps) {
  const [copied, setCopied] = useState(false);
  const jsonString = useMemo(() => JSON.stringify(data, null, 2), [data]);

  const handleCopy = useCallback(() => {
    navigator.clipboard.writeText(jsonString);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }, [jsonString]);

  const handleDownload = useCallback(() => {
    const blob = new Blob([jsonString], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${title.toLowerCase().replace(/\s+/g, '-')}-${new Date().toISOString().slice(0, 19).replace(/:/g, '-')}.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  }, [jsonString, title]);

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <button
            onClick={onBack}
            className="flex items-center gap-1 text-gray-400 hover:text-surface transition-colors"
          >
            <ArrowLeft size={18} />
          </button>
          <div className="flex items-center gap-2">
            <Code size={20} className="text-flow-accent" />
            <h2 className="text-lg font-semibold text-surface">{title}</h2>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={handleCopy}
            className="flex items-center gap-2 px-3 py-2 text-sm text-gray-400 hover:text-surface bg-gray-700 hover:bg-gray-600 rounded-lg transition-colors"
            title="Copy to clipboard"
          >
            {copied ? <Check size={16} className="text-emerald-400" /> : <Copy size={16} />}
            {copied ? 'Copied!' : 'Copy'}
          </button>
          <button
            onClick={handleDownload}
            className="flex items-center gap-2 px-3 py-2 text-sm text-gray-400 hover:text-surface bg-gray-700 hover:bg-gray-600 rounded-lg transition-colors"
            title="Download JSON"
          >
            <Download size={16} />
            Download
          </button>
        </div>
      </div>

      {/* JSON Content */}
      <div className="bg-gray-900 border border-gray-700 rounded-lg overflow-hidden">
        <pre className="p-4 text-sm font-mono text-gray-300 overflow-auto max-h-[70vh] whitespace-pre-wrap break-words">
          {jsonString}
        </pre>
      </div>
    </div>
  );
}

// Configuration Panel Component
interface ConfigurationPanelProps {
  config: {
    summary: string;
    modified_count: number;
    total_count?: number;
    by_tier?: { essential: number; advanced: number; internal: number };
    modified_options?: Array<{ env_var: string; tier: ConfigTier; description?: string; current_value: string; default_value?: string }>;
    all_options?: { essential: ConfigOption[]; advanced: ConfigOption[]; internal: ConfigOption[] };
  };
}

const TIER_COLORS: Record<ConfigTier, { bg: string; text: string; label: string }> = {
  essential: { bg: 'bg-emerald-500/20', text: 'text-emerald-300', label: 'Essential' },
  advanced: { bg: 'bg-amber-500/20', text: 'text-amber-300', label: 'Advanced' },
  internal: { bg: 'bg-gray-500/20', text: 'text-gray-400', label: 'Internal' },
};

function ConfigOptionRow({ option }: { option: ConfigOption }) {
  return (
    <div className={`flex flex-col sm:flex-row sm:items-center justify-between text-sm rounded px-3 py-2 gap-2 ${option.is_modified ? 'bg-gray-800 border border-amber-500/30' : 'bg-gray-800/50'}`}>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="font-mono text-xs text-gray-300 truncate">{option.env_var}</span>
          {option.is_modified && (
            <span className="text-xs px-1.5 py-0.5 rounded bg-amber-500/20 text-amber-300">modified</span>
          )}
        </div>
        <p className="text-xs text-gray-500 mt-0.5 truncate">{option.description}</p>
      </div>
      <div className="flex items-center gap-2 flex-shrink-0">
        <span className="text-xs text-gray-400 font-mono">{option.current_value || '(empty)'}</span>
        {option.is_modified && option.default_value !== option.current_value && (
          <span className="text-xs text-gray-600" title={`Default: ${option.default_value}`}>
            ← {option.default_value || '(empty)'}
          </span>
        )}
      </div>
    </div>
  );
}

function ConfigTierSection({ tier, options, defaultOpen = false }: { tier: ConfigTier; options: ConfigOption[]; defaultOpen?: boolean }) {
  const [isExpanded, setIsExpanded] = useState(defaultOpen);
  const modifiedCount = options.filter(o => o.is_modified).length;
  const tierStyle = TIER_COLORS[tier];

  return (
    <div className="border border-gray-700 rounded-lg overflow-hidden">
      <button
        type="button"
        onClick={() => setIsExpanded(!isExpanded)}
        className="w-full flex items-center justify-between px-4 py-3 bg-gray-800/50 hover:bg-gray-800 transition-colors text-left"
      >
        <div className="flex items-center gap-3">
          <span className={`text-xs px-2 py-0.5 rounded ${tierStyle.bg} ${tierStyle.text}`}>
            {tierStyle.label}
          </span>
          <span className="text-sm text-surface">{options.length} options</span>
          {modifiedCount > 0 && (
            <span className="text-xs text-amber-400">({modifiedCount} modified)</span>
          )}
        </div>
        {isExpanded ? <ChevronDown size={16} className="text-gray-400" /> : <ChevronRight size={16} className="text-gray-400" />}
      </button>
      {isExpanded && (
        <div className="p-3 bg-gray-900/50 border-t border-gray-700 space-y-2">
          {options.map((option) => (
            <ConfigOptionRow key={option.env_var} option={option} />
          ))}
        </div>
      )}
    </div>
  );
}

function ConfigurationPanel({ config }: ConfigurationPanelProps) {
  const [showOnlyModified, setShowOnlyModified] = useState(true);

  return (
    <SettingSection title="Configuration" tooltip="Current configuration status and all options" defaultOpen={false}>
      <div className="space-y-4">
        {/* Summary */}
        <div className="flex items-center justify-between gap-4">
          <div className="flex items-center gap-2 text-sm">
            <Settings2 size={16} className="text-gray-400" />
            <span className="text-surface">{config.summary}</span>
          </div>
          {config.all_options && (
            <button
              onClick={() => setShowOnlyModified(!showOnlyModified)}
              className="text-xs px-2 py-1 rounded bg-gray-700 hover:bg-gray-600 text-gray-300 transition-colors"
            >
              {showOnlyModified ? 'Show All Options' : 'Show Modified Only'}
            </button>
          )}
        </div>

        {/* Modified-only view */}
        {showOnlyModified && (
          <div className="space-y-2">
            {config.modified_count === 0 ? (
              <div className="p-3 bg-gray-800/50 border border-gray-700 rounded text-gray-400 text-center text-sm">
                <CheckCircle2 size={20} className="mx-auto mb-2 text-emerald-500" />
                <p>All options at defaults</p>
                <p className="text-xs text-gray-500 mt-1">Click "Show All Options" to browse available settings</p>
              </div>
            ) : config.modified_options ? (
              config.modified_options.map((opt) => (
                <div key={opt.env_var} className="flex items-center justify-between text-sm bg-gray-800 border border-amber-500/30 rounded px-3 py-2">
                  <div>
                    <span className="font-mono text-xs text-gray-300">{opt.env_var}</span>
                    {opt.description && (
                      <p className="text-xs text-gray-500 mt-0.5">{opt.description}</p>
                    )}
                  </div>
                  <div className="flex items-center gap-2">
                    <span className={`text-xs px-1.5 py-0.5 rounded ${TIER_COLORS[opt.tier].bg} ${TIER_COLORS[opt.tier].text}`}>
                      {opt.tier}
                    </span>
                    <span className="text-gray-400 font-mono text-xs">{opt.current_value}</span>
                    {opt.default_value && opt.default_value !== opt.current_value && (
                      <span className="text-xs text-gray-600">← {opt.default_value}</span>
                    )}
                  </div>
                </div>
              ))
            ) : null}
          </div>
        )}

        {/* All options view */}
        {!showOnlyModified && config.all_options && (
          <div className="space-y-3">
            <ConfigTierSection tier="essential" options={config.all_options.essential} defaultOpen={true} />
            <ConfigTierSection tier="advanced" options={config.all_options.advanced} />
            <ConfigTierSection tier="internal" options={config.all_options.internal} />
          </div>
        )}
      </div>
    </SettingSection>
  );
}

// Metrics Viewer Component
interface MetricsViewerProps {
  data: MetricsResponse;
  onBack: () => void;
}

// Check if a metric has any meaningful data
function hasMetricData(metric: MetricData): boolean {
  if (metric.values.length === 0) return false;
  // Check if all values are 0 (no activity)
  return metric.values.some((v: MetricValue) => v.value !== 0);
}

// Get a summary value for a metric (for display when collapsed)
function getMetricSummaryValue(metric: MetricData): string {
  if (metric.values.length === 0) return 'No data';

  // For counters, sum all values or get the total
  if (metric.type === 'counter') {
    const total = metric.values.find((v: MetricValue) => v.labels._suffix === '_total');
    if (total) return total.value.toString();
    const sum = metric.values.reduce((acc: number, v: MetricValue) => acc + v.value, 0);
    return sum.toString();
  }

  // For gauges, get the current value
  if (metric.type === 'gauge') {
    const value = metric.values.find((v: MetricValue) => v.labels._suffix === '_value');
    if (value) return value.value.toString();
    // Sum all labeled values for multi-label gauges
    const sum = metric.values.reduce((acc: number, v: MetricValue) => acc + v.value, 0);
    return sum.toString();
  }

  // For histograms, show count and sum
  if (metric.type === 'histogram') {
    const count = metric.values.find((v: MetricValue) => v.labels._suffix === '_count');
    const sum = metric.values.find((v: MetricValue) => v.labels._suffix === '_sum');
    if (count && sum) {
      const avg = count.value > 0 ? (sum.value / count.value).toFixed(1) : '0';
      return `${count.value} samples, avg ${avg}`;
    }
    return `${metric.values.length} buckets`;
  }

  return `${metric.values.length} values`;
}

function MetricsViewer({ data, onBack }: MetricsViewerProps) {
  const [showJson, setShowJson] = useState(false);
  const [showInactive, setShowInactive] = useState(false);

  // Separate metrics into active (has data) and inactive (empty/all zeros)
  const { activeMetrics, inactiveMetrics } = useMemo(() => {
    const active: Array<[string, MetricData]> = [];
    const inactive: Array<[string, MetricData]> = [];

    for (const [name, metric] of Object.entries(data.metrics)) {
      if (hasMetricData(metric)) {
        active.push([name, metric]);
      } else {
        inactive.push([name, metric]);
      }
    }

    return { activeMetrics: active, inactiveMetrics: inactive };
  }, [data.metrics]);

  if (showJson) {
    return <JsonViewer data={data} onBack={() => setShowJson(false)} title="Metrics JSON" />;
  }

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <button
            onClick={onBack}
            className="flex items-center gap-1 text-gray-400 hover:text-surface transition-colors"
          >
            <ArrowLeft size={18} />
          </button>
          <div className="flex items-center gap-2">
            <BarChart3 size={20} className="text-flow-accent" />
            <h2 className="text-lg font-semibold text-surface">Prometheus Metrics</h2>
          </div>
        </div>
        <button
          onClick={() => setShowJson(true)}
          className="flex items-center gap-2 px-3 py-2 text-sm text-gray-400 hover:text-surface bg-gray-700 hover:bg-gray-600 rounded-lg transition-colors"
        >
          <Code size={16} />
          View JSON
        </button>
      </div>

      {/* Summary */}
      <div className="bg-gray-800/50 border border-gray-700 rounded-lg p-4">
        <div className="flex flex-wrap items-center gap-4 text-sm text-gray-400">
          <span><strong className="text-surface">{activeMetrics.length}</strong> active metrics</span>
          <span><strong className="text-gray-500">{inactiveMetrics.length}</strong> inactive</span>
          <span>Updated: {new Date(data.summary.timestamp).toLocaleTimeString()}</span>
          {data.summary.config.port && (
            <span className="flex items-center gap-1">
              <ExternalLink size={12} />
              Port {data.summary.config.port}
            </span>
          )}
        </div>
      </div>

      {/* Active Metrics */}
      {activeMetrics.length > 0 ? (
        <div className="space-y-4">
          {activeMetrics.map(([name, metric]) => (
            <div key={name} className="bg-gray-800/50 border border-gray-700 rounded-lg overflow-hidden">
              <div className="px-4 py-3 border-b border-gray-700">
                <div className="flex items-center justify-between">
                  <span className="font-mono text-sm text-surface">{name}</span>
                  <div className="flex items-center gap-2">
                    <span className="text-xs font-medium text-emerald-400">{getMetricSummaryValue(metric)}</span>
                    <span className="text-xs px-2 py-0.5 bg-gray-700 text-gray-400 rounded">{metric.type}</span>
                  </div>
                </div>
                {metric.help && (
                  <p className="text-xs text-gray-500 mt-1">{metric.help}</p>
                )}
              </div>
              <div className="p-4">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="text-left text-gray-500">
                      <th className="pb-2 font-normal">Labels</th>
                      <th className="pb-2 font-normal text-right">Value</th>
                    </tr>
                  </thead>
                  <tbody className="text-gray-300">
                    {metric.values.slice(0, 10).map((v: MetricValue, idx: number) => (
                      <tr key={idx} className="border-t border-gray-700/50">
                        <td className="py-2 pr-4">
                          <span className="font-mono text-xs">
                            {Object.entries(v.labels)
                              .filter(([k]) => k !== '_suffix')
                              .map(([k, val]) => `${k}="${val}"`)
                              .join(', ') || (v.labels._suffix ? v.labels._suffix.replace(/^_/, '') : 'value')}
                          </span>
                        </td>
                        <td className="py-2 text-right font-mono">
                          {typeof v.value === 'number' && !Number.isInteger(v.value)
                            ? v.value.toFixed(4)
                            : v.value}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
                {metric.values.length > 10 && (
                  <p className="text-xs text-gray-500 mt-2">
                    Showing 10 of {metric.values.length} values
                  </p>
                )}
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className="bg-gray-800/50 border border-gray-700 rounded-lg p-8 text-center">
          <BarChart3 size={32} className="mx-auto text-gray-600 mb-3" />
          <p className="text-gray-400">No metrics with recorded data yet</p>
          <p className="text-xs text-gray-500 mt-1">Metrics will appear here once activity is recorded</p>
        </div>
      )}

      {/* Inactive Metrics (Collapsible) */}
      {inactiveMetrics.length > 0 && (
        <div className="border border-gray-700 rounded-lg overflow-hidden">
          <button
            onClick={() => setShowInactive(!showInactive)}
            className="w-full flex items-center justify-between px-4 py-3 bg-gray-800/30 hover:bg-gray-800/50 transition-colors text-left"
          >
            <span className="text-sm text-gray-400">
              {inactiveMetrics.length} inactive metric{inactiveMetrics.length !== 1 ? 's' : ''} (no data recorded)
            </span>
            {showInactive ? <ChevronDown size={16} className="text-gray-500" /> : <ChevronRight size={16} className="text-gray-500" />}
          </button>
          {showInactive && (
            <div className="p-4 bg-gray-900/30 border-t border-gray-700">
              <div className="space-y-2">
                {inactiveMetrics.map(([name, metric]) => (
                  <div key={name} className="flex items-center justify-between py-2 px-3 bg-gray-800/30 rounded text-sm">
                    <div>
                      <span className="font-mono text-xs text-gray-400">{name}</span>
                      {metric.help && (
                        <p className="text-xs text-gray-600 mt-0.5">{metric.help}</p>
                      )}
                    </div>
                    <span className="text-xs px-2 py-0.5 bg-gray-700/50 text-gray-500 rounded">{metric.type || 'unknown'}</span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export function DiagnosticsTab() {
  const [viewMode, setViewMode] = useState<ViewMode>('dashboard');

  const { data, isLoading, isFetching, error, refetch, dataUpdatedAt } = useObservability({
    depth: 'standard',
    refetchInterval: 30000,
  });

  const { data: metricsData, isLoading: isLoadingMetrics, refetch: refetchMetrics } = useMetrics({
    enabled: viewMode === 'metrics',
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

  const handleViewJson = useCallback(() => {
    setViewMode('json');
  }, []);

  const handleViewMetrics = useCallback(async () => {
    setViewMode('metrics');
    await refetchMetrics();
  }, [refetchMetrics]);

  const handleBackToDashboard = useCallback(() => {
    setViewMode('dashboard');
  }, []);

  const overallStatus = useMemo((): StatusColorKey => {
    if (isLoading) return 'loading';
    if (error) return 'error';
    return data?.status || 'error';
  }, [isLoading, error, data?.status]);

  const lastUpdated = useMemo(() => {
    if (!dataUpdatedAt) return 'Never';
    return formatRelativeTime(new Date(dataUpdatedAt).toISOString());
  }, [dataUpdatedAt]);

  // Render JSON View
  if (viewMode === 'json' && data) {
    return <JsonViewer data={data} onBack={handleBackToDashboard} title="Diagnostics JSON" />;
  }

  // Render Metrics View
  if (viewMode === 'metrics') {
    if (isLoadingMetrics) {
      return (
        <div className="flex items-center justify-center h-64">
          <Loader2 size={32} className="animate-spin text-gray-400" />
        </div>
      );
    }
    if (metricsData) {
      return <MetricsViewer data={metricsData} onBack={handleBackToDashboard} />;
    }
    return (
      <div className="space-y-4">
        <button onClick={handleBackToDashboard} className="flex items-center gap-2 text-gray-400 hover:text-surface">
          <ArrowLeft size={18} />
          Back to Dashboard
        </button>
        <div className="bg-red-500/10 border border-red-500/30 rounded-lg p-4">
          <p className="text-red-300">Failed to load metrics data</p>
        </div>
      </div>
    );
  }

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
          onClick={handleViewJson}
          disabled={!data || isLoading}
          className="flex items-center gap-2 px-3 py-2 text-sm text-subtle hover:text-surface hover:bg-gray-700 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          title="View diagnostics as JSON"
        >
          <Code size={16} />
          <span className="hidden sm:inline">View JSON</span>
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
              details={<MetricsDetails metrics={data.components.metrics} onViewMetrics={handleViewMetrics} />}
            />
          </div>
        </SettingSection>
      )}

      {/* Configuration */}
      {data?.config && (
        <ConfigurationPanel config={data.config} />
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
                    <StatusBadge status={
                      diagnosticsResult.results.recording.ready
                        ? 'healthy'
                        : diagnosticsResult.results.recording.issues.some(i => i.severity === 'error')
                          ? 'error'
                          : 'degraded'
                    } />
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
