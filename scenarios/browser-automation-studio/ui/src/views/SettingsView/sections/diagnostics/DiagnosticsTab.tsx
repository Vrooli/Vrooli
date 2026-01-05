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
  AlertTriangle,
  AlertCircle,
  Loader2,
  Clock,
  Wifi,
  Server,
  Monitor,
  Video,
  Trash2,
  BarChart3,
  Zap,
  Play,
  Code,
  TestTube2,
} from 'lucide-react';
import {
  useObservability,
  useRefreshDiagnostics,
  useRefreshCache,
  useMetrics,
  useRunCleanup,
  useSessionList,
  useConfigUpdate,
  usePipelineTest,
} from '@/domains/observability';
import { SettingSection } from '../shared';

// Local imports from split modules
import { formatUptime, formatRelativeTime, type StatusColorKey } from './utils';
import { StatusBadge } from './StatusBadge';
import { ComponentRow } from './ComponentRow';
import {
  BrowserDetails,
  SessionsDetails,
  RecordingDetails,
  CleanupDetails,
  MetricsDetails,
} from './ComponentDetails';
import { JsonViewer } from './JsonViewer';
import { ConfigurationPanel } from './ConfigurationPanel';
import { MetricsViewer } from './MetricsViewer';
import { DiagnosticResultsCard } from './DiagnosticResults';
import { PipelineTestResults } from './PipelineTestResults';

type ViewMode = 'dashboard' | 'json' | 'metrics';

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
  const { runCleanup, isRunning: isCleaningUp, result: cleanupResult } = useRunCleanup();
  const { data: sessionListData, isFetching: isFetchingSessionList, refetch: refetchSessionList } = useSessionList({
    enabled: false, // Only fetch when explicitly requested
  });
  const { updateConfig, resetConfig, isUpdating: isConfigUpdating } = useConfigUpdate({
    onSuccess: () => {
      // Refetch observability data to get updated config values
      void refetch();
    },
  });
  const { runTest: runPipelineTest, isRunning: isPipelineTestRunning, result: pipelineTestResult } = usePipelineTest();

  const handleRefresh = useCallback(async () => {
    refreshCache();
    await refetch();
  }, [refreshCache, refetch]);

  const handleRunDiagnostics = useCallback(async () => {
    await runDiagnostics({ type: 'all', options: { level: 'full' } });
  }, [runDiagnostics]);

  const handleRunPipelineTest = useCallback(async () => {
    // The pipeline test is fully autonomous - it creates a session if needed
    await runPipelineTest();
  }, [runPipelineTest]);

  const handleCleanNow = useCallback(async () => {
    await runCleanup();
    await refetch(); // Refresh to get updated session counts
  }, [runCleanup, refetch]);

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

  const handleConfigUpdate = useCallback(async (envVar: string, value: string) => {
    return updateConfig(envVar, value);
  }, [updateConfig]);

  const handleConfigReset = useCallback(async (envVar: string) => {
    await resetConfig(envVar);
  }, [resetConfig]);

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
              details={
                <SessionsDetails
                  sessions={data.components.sessions}
                  sessionList={sessionListData?.sessions}
                  isLoadingList={isFetchingSessionList}
                  onRefreshList={() => void refetchSessionList()}
                />
              }
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
              hint="Automatically removes browser sessions that have been idle longer than the configured timeout to free up system resources."
              details={
                <CleanupDetails
                  cleanup={data.components.cleanup}
                  onCleanNow={handleCleanNow}
                  isCleaningUp={isCleaningUp}
                  cleanupResult={cleanupResult}
                />
              }
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
        <ConfigurationPanel
          config={data.config}
          onUpdate={handleConfigUpdate}
          onReset={handleConfigReset}
          isUpdating={isConfigUpdating}
        />
      )}

      {/* Deep Diagnostics */}
      <SettingSection title="Deep Diagnostics" tooltip="Run comprehensive diagnostic scans" defaultOpen={false}>
        <div className="space-y-4">
          {/* Description */}
          <div className="space-y-2">
            <p className="text-sm text-gray-400">
              Deep diagnostics run comprehensive tests on system subsystems to verify they're working correctly.
            </p>
            <div className="text-xs text-gray-500 space-y-1">
              <p><strong>What's tested:</strong></p>
              <ul className="list-disc list-inside ml-2 space-y-0.5">
                <li>Recording script injection into browser pages</li>
                <li>Event flow from browser to server (console events, clicks)</li>
                <li>Browser context and session state</li>
              </ul>
            </div>
          </div>

          {/* Prerequisites notice */}
          {data?.summary && data.summary.sessions === 0 && (
            <div className="flex items-start gap-3 p-3 bg-amber-500/10 border border-amber-500/30 rounded-lg">
              <AlertTriangle size={18} className="text-amber-400 flex-shrink-0 mt-0.5" />
              <div className="text-sm">
                <p className="text-amber-300 font-medium">No active sessions</p>
                <p className="text-amber-200/80 text-xs mt-1">
                  Recording diagnostics require at least one browser session with a page loaded.
                  Start a recording or navigate to a URL in the browser panel to enable full diagnostics.
                </p>
              </div>
            </div>
          )}

          {/* Action buttons */}
          <div className="flex flex-wrap items-center gap-3">
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
            <button
              onClick={handleRunPipelineTest}
              disabled={isPipelineTestRunning}
              className="flex items-center gap-2 px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              title="Run automated pipeline test (creates session automatically if needed)"
            >
              {isPipelineTestRunning ? (
                <>
                  <Loader2 size={16} className="animate-spin" />
                  Testing Pipeline...
                </>
              ) : (
                <>
                  <TestTube2 size={16} />
                  Test Pipeline
                </>
              )}
            </button>
            {data?.summary && (
              <span className="text-xs text-gray-500">
                {data.summary.sessions} session{data.summary.sessions !== 1 ? 's' : ''} available
                {data.summary.recordings > 0 && ` (${data.summary.recordings} recording)`}
              </span>
            )}
          </div>

          {/* Pipeline Test Description */}
          <div className="p-3 bg-purple-500/10 border border-purple-500/20 rounded-lg">
            <div className="flex items-start gap-2">
              <TestTube2 size={16} className="text-purple-400 flex-shrink-0 mt-0.5" />
              <div className="text-sm">
                <p className="text-purple-300 font-medium">Fully Autonomous Pipeline Test</p>
                <p className="text-purple-200/70 text-xs mt-1">
                  Tests the entire recording pipeline by automatically creating a browser session,
                  navigating to an internal test page, simulating real user interactions, and verifying events flow correctly.
                  <strong className="text-purple-300"> No active session or manual clicking required!</strong>
                </p>
              </div>
            </div>
          </div>

          {/* Pipeline Test Results */}
          {pipelineTestResult && (
            <PipelineTestResults result={pipelineTestResult} />
          )}

          {/* Diagnostics Results */}
          {diagnosticsResult && (
            <DiagnosticResultsCard result={diagnosticsResult} />
          )}
        </div>
      </SettingSection>
    </div>
  );
}

export default DiagnosticsTab;
