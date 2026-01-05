/**
 * Component Details
 *
 * Detail view components for each system component (Browser, Sessions, Recording, Cleanup, Metrics).
 */

import { useState } from 'react';
import {
  ChevronDown,
  ChevronRight,
  Video,
  Trash2,
  Loader2,
  BarChart3,
  Bug,
} from 'lucide-react';
import type {
  BrowserComponent,
  SessionsComponent,
  RecordingComponent,
  CleanupComponent,
  MetricsComponent,
  SessionInfo,
} from '@/domains/observability';
import { useRecordingDebug, type RecordingDebugResponse } from '@/domains/observability/hooks/useRecordingDebug';
import { formatRelativeTime } from './utils';

// ============================================================================
// Browser Details
// ============================================================================

export function BrowserDetails({ browser }: { browser: BrowserComponent }) {
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

// ============================================================================
// Sessions Details
// ============================================================================

interface SessionsDetailsProps {
  sessions: SessionsComponent;
  sessionList?: SessionInfo[];
  isLoadingList?: boolean;
  onRefreshList?: () => void;
}

export function SessionsDetails({ sessions, sessionList, isLoadingList, onRefreshList }: SessionsDetailsProps) {
  const [showList, setShowList] = useState(false);
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

      {/* Session List Toggle */}
      {sessions.total > 0 && (
        <div className="pt-2 border-t border-gray-700">
          <button
            onClick={() => {
              if (!showList && onRefreshList) {
                onRefreshList();
              }
              setShowList(!showList);
            }}
            className="flex items-center gap-2 text-xs text-gray-400 hover:text-surface transition-colors"
          >
            {showList ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
            <span>{showList ? 'Hide' : 'Show'} session details</span>
            {isLoadingList && <Loader2 size={12} className="animate-spin ml-2" />}
          </button>

          {showList && sessionList && (
            <div className="mt-3 space-y-2">
              {sessionList.map((session) => (
                <div
                  key={session.id}
                  className={`p-2 rounded text-xs ${
                    session.is_recording
                      ? 'bg-rose-500/10 border border-rose-500/30'
                      : session.is_idle
                        ? 'bg-gray-800/50 border border-gray-700'
                        : 'bg-emerald-500/10 border border-emerald-500/30'
                  }`}
                >
                  <div className="flex items-center justify-between mb-1">
                    <div className="flex items-center gap-2">
                      <span className="font-mono text-gray-300">{session.id.slice(0, 8)}</span>
                      <span
                        className={`px-1.5 py-0.5 rounded text-[10px] font-medium ${
                          session.phase === 'recording'
                            ? 'bg-rose-500/20 text-rose-300'
                            : session.phase === 'executing'
                              ? 'bg-amber-500/20 text-amber-300'
                              : session.phase === 'ready'
                                ? 'bg-emerald-500/20 text-emerald-300'
                                : 'bg-gray-700 text-gray-400'
                        }`}
                      >
                        {session.phase}
                      </span>
                      {session.is_recording && (
                        <span className="flex items-center gap-1 text-rose-400">
                          <Video size={10} />
                          Recording
                        </span>
                      )}
                    </div>
                    <span className="text-gray-500">{session.page_count} page{session.page_count !== 1 ? 's' : ''}</span>
                  </div>
                  {session.current_url && (
                    <div className="text-gray-500 truncate mt-1" title={session.current_url}>
                      {session.current_url.length > 50
                        ? session.current_url.slice(0, 50) + '...'
                        : session.current_url}
                    </div>
                  )}
                  <div className="flex items-center gap-3 mt-1 text-gray-500">
                    <span>Last used: {formatRelativeTime(session.last_used_at)}</span>
                    <span>{session.instruction_count} instructions</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// ============================================================================
// Debug Info Display (helper for RecordingDetails)
// ============================================================================

function DebugInfoDisplay({ debug }: { debug: RecordingDebugResponse }) {
  const script = debug.browser_script;
  const diag = debug.diagnostics;

  // Collect issues
  const issues: { message: string; severity: 'error' | 'warning'; detail?: string }[] = [];
  if (diag.service_worker_blocking) {
    issues.push({
      message: 'Service Worker intercepting events!',
      severity: 'error',
      detail: 'The site has a service worker that is intercepting fetch requests. Events are being sent but never reach the server. This is a known limitation with PWAs and sites using aggressive caching.',
    });
  }
  if (diag.script_not_loaded) issues.push({ message: 'Script not loaded on page', severity: 'error' });
  if (diag.script_not_ready) issues.push({ message: 'Script loaded but not ready', severity: 'error' });
  if (diag.script_not_in_main) issues.push({ message: 'Script not in MAIN context (isolated)', severity: 'error' });
  if (diag.script_inactive) issues.push({ message: 'Script is inactive (isActive=false)', severity: 'warning' });
  if (diag.no_handlers) issues.push({ message: 'No event handlers registered', severity: 'error' });
  if (diag.no_event_handler) issues.push({ message: 'Server event handler not set', severity: 'error' });
  if (diag.events_being_dropped) issues.push({ message: 'Events are being dropped', severity: 'warning' });

  return (
    <div className="space-y-3 text-xs">
      {/* Issues */}
      {issues.length > 0 && (
        <div className="space-y-1">
          {issues.map((issue, i) => (
            <div
              key={i}
              className={`p-2 rounded ${
                issue.severity === 'error'
                  ? 'bg-red-500/10 border border-red-500/30 text-red-300'
                  : 'bg-amber-500/10 border border-amber-500/30 text-amber-300'
              }`}
            >
              <div className="flex items-start gap-2">
                <span className="flex-shrink-0">{issue.severity === 'error' ? '✕' : '⚠'}</span>
                <span className="font-medium">{issue.message}</span>
              </div>
              {issue.detail && (
                <div className="mt-1 ml-5 text-gray-400 text-xs">{issue.detail}</div>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Browser Script State */}
      {script && (
        <div className="space-y-2">
          <div className="text-gray-400 font-medium">Browser Script</div>
          <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-gray-300">
            <div>Loaded: <span className={script.loaded ? 'text-emerald-400' : 'text-red-400'}>{script.loaded ? 'Yes' : 'No'}</span></div>
            <div>Ready: <span className={script.ready ? 'text-emerald-400' : 'text-gray-500'}>{script.ready ? 'Yes' : 'No'}</span></div>
            <div>Active: <span className={script.isActive ? 'text-emerald-400' : 'text-red-400'}>
              {script.isActive === null ? 'Unknown' : script.isActive ? 'Yes' : 'No'}
            </span></div>
            <div>Context: <span className={script.inMainContext ? 'text-emerald-400' : 'text-red-400'}>
              {script.inMainContext ? 'MAIN' : 'ISOLATED'}
            </span></div>
            <div>Handlers: <span className={script.handlersCount && script.handlersCount > 0 ? 'text-emerald-400' : 'text-red-400'}>
              {script.handlersCount ?? 0}
            </span></div>
            <div>Version: <span className="text-gray-400">{script.version || 'Unknown'}</span></div>
            <div className="col-span-2">
              Service Worker: <span className={script.serviceWorkerActive ? 'text-amber-400' : 'text-gray-400'}>
                {script.serviceWorkerActive ? 'Active ⚠' : 'None'}
              </span>
              {script.serviceWorkerActive && script.serviceWorkerUrl && (
                <div className="text-gray-500 text-xs mt-0.5 truncate" title={script.serviceWorkerUrl}>
                  {script.serviceWorkerUrl}
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Browser-side Telemetry */}
      {script && script.loaded && (
        <div className="space-y-2">
          <div className="text-gray-400 font-medium">Browser Telemetry</div>
          <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-gray-300">
            <div>Events Detected: <span className="text-surface">{script.eventsDetected ?? 0}</span></div>
            <div>Events Captured: <span className="text-surface">{script.eventsCaptured ?? 0}</span></div>
            <div>Events Sent: <span className="text-surface">{script.eventsSent ?? 0}</span></div>
            <div>Send Failures: <span className={script.eventsSendFailed && script.eventsSendFailed > 0 ? 'text-red-400' : 'text-surface'}>
              {script.eventsSendFailed ?? 0}
            </span></div>
          </div>
          {script.lastError && (
            <div className="p-2 bg-red-500/10 border border-red-500/30 rounded text-red-300">
              Last Error: {script.lastError}
            </div>
          )}
        </div>
      )}

      {/* Server State */}
      <div className="space-y-2">
        <div className="text-gray-400 font-medium">Server State</div>
        <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-gray-300">
          <div>Recording: <span className={debug.server.is_recording ? 'text-emerald-400' : 'text-gray-500'}>
            {debug.server.is_recording ? 'Active' : 'Inactive'}
          </span></div>
          <div>Handler Set: <span className={debug.server.has_event_handler ? 'text-emerald-400' : 'text-red-400'}>
            {debug.server.has_event_handler ? 'Yes' : 'No'}
          </span></div>
          <div className="col-span-2">Phase: <span className="text-gray-400">{debug.server.phase}</span></div>
        </div>
      </div>

      {/* No issues message */}
      {issues.length === 0 && (
        <div className="p-2 bg-emerald-500/10 border border-emerald-500/30 rounded text-emerald-300 text-center">
          ✓ No obvious issues detected. Script appears healthy.
        </div>
      )}
    </div>
  );
}

// ============================================================================
// Recording Details
// ============================================================================

export function RecordingDetails({ recording }: { recording: RecordingComponent }) {
  const stats = recording.injection_stats;
  const routeStats = recording.route_handler_stats;
  const successRate = stats && stats.total > 0 ? Math.round((stats.successful / stats.total) * 100) : 100;
  const hasData = recording.script_version || (stats && stats.total > 0);
  const hasEventFlow = routeStats && routeStats.eventsReceived > 0;

  // Calculate event flow health
  const eventsDropped = routeStats?.eventsDroppedNoHandler ?? 0;
  const eventsWithErrors = routeStats?.eventsWithErrors ?? 0;
  const eventsReceived = routeStats?.eventsReceived ?? 0;
  const eventsProcessed = routeStats?.eventsProcessed ?? 0;
  const eventFlowHealthy = eventsDropped === 0 && eventsWithErrors === 0;
  const hasEventHandler = recording.has_event_handler;

  // Debug state
  const { fetchDebug, isLoading: isDebugLoading, data: debugData, error: debugError, reset: resetDebug } = useRecordingDebug();
  const [showDebug, setShowDebug] = useState(false);

  const handleDebugClick = async () => {
    if (!recording.active_session_id) return;

    if (showDebug) {
      setShowDebug(false);
      resetDebug();
      return;
    }

    try {
      await fetchDebug(recording.active_session_id);
      setShowDebug(true);
    } catch {
      // Error is handled by the hook
    }
  };

  return (
    <div className="space-y-4 text-sm">
      {/* Active recordings count and handler status */}
      <div className="flex items-center justify-between">
        <div>
          <span className="text-gray-500">Active Recordings:</span>
          <span className="ml-2 text-surface">{recording.active_count}</span>
        </div>
        <div className="flex items-center gap-2">
          {recording.active_count > 0 && (
            <>
              <span className="text-gray-500 text-xs">Event Handler:</span>
              <span className={`px-2 py-0.5 rounded text-xs font-medium ${
                hasEventHandler
                  ? 'bg-emerald-500/20 text-emerald-300'
                  : 'bg-red-500/20 text-red-300'
              }`}>
                {hasEventHandler ? 'Connected' : 'Disconnected'}
              </span>
            </>
          )}
          {recording.active_session_id && (
            <button
              onClick={handleDebugClick}
              disabled={isDebugLoading}
              className="flex items-center gap-1 px-2 py-1 text-xs bg-blue-500/20 hover:bg-blue-500/30 text-blue-300 rounded transition-colors disabled:opacity-50"
              title="Show browser script debug info"
            >
              {isDebugLoading ? (
                <Loader2 size={12} className="animate-spin" />
              ) : (
                <Bug size={12} />
              )}
              {showDebug ? 'Hide Debug' : 'Debug'}
            </button>
          )}
        </div>
      </div>

      {/* Event Flow Stats - Only show when recording is active */}
      {recording.active_count > 0 && routeStats && (
        <div className="p-3 bg-gray-800/50 border border-gray-700 rounded space-y-3">
          <div className="flex items-center justify-between">
            <span className="text-gray-400 font-medium text-xs uppercase tracking-wide">Event Flow</span>
            {hasEventFlow && (
              <span className={`flex items-center gap-1.5 text-xs ${
                eventFlowHealthy ? 'text-emerald-400' : 'text-amber-400'
              }`}>
                <span className={`w-2 h-2 rounded-full ${
                  eventFlowHealthy ? 'bg-emerald-400' : 'bg-amber-400'
                }`} />
                {eventFlowHealthy ? 'Healthy' : 'Issues Detected'}
              </span>
            )}
          </div>

          {/* Event pipeline visualization */}
          <div className="flex items-center gap-2 text-xs">
            <div className="flex-1 text-center p-2 bg-gray-900 rounded">
              <div className="text-lg font-semibold text-surface">{eventsReceived}</div>
              <div className="text-gray-500">Received</div>
            </div>
            <div className="text-gray-600">→</div>
            <div className="flex-1 text-center p-2 bg-gray-900 rounded">
              <div className={`text-lg font-semibold ${eventsProcessed > 0 ? 'text-emerald-400' : 'text-surface'}`}>
                {eventsProcessed}
              </div>
              <div className="text-gray-500">Processed</div>
            </div>
            {(eventsDropped > 0 || eventsWithErrors > 0) && (
              <>
                <div className="text-gray-600">|</div>
                <div className="flex-1 text-center p-2 bg-red-500/10 border border-red-500/30 rounded">
                  <div className="text-lg font-semibold text-red-400">
                    {eventsDropped + eventsWithErrors}
                  </div>
                  <div className="text-red-400/70">Lost</div>
                </div>
              </>
            )}
          </div>

          {/* Warning if events are being dropped */}
          {eventsDropped > 0 && (
            <div className="flex items-start gap-2 p-2 bg-red-500/10 border border-red-500/30 rounded text-xs">
              <span className="text-red-400 flex-shrink-0 mt-0.5">⚠</span>
              <div>
                <span className="text-red-300 font-medium">{eventsDropped} events dropped</span>
                <span className="text-red-400/80 ml-1">- No handler was set. Events are being lost!</span>
              </div>
            </div>
          )}

          {/* Warning if handler errors */}
          {eventsWithErrors > 0 && (
            <div className="flex items-start gap-2 p-2 bg-amber-500/10 border border-amber-500/30 rounded text-xs">
              <span className="text-amber-400 flex-shrink-0 mt-0.5">⚠</span>
              <div>
                <span className="text-amber-300 font-medium">{eventsWithErrors} handler errors</span>
                <span className="text-amber-400/80 ml-1">- Some events failed during processing</span>
              </div>
            </div>
          )}

          {/* Last event info */}
          {routeStats.lastEventAt && (
            <div className="flex items-center justify-between text-xs text-gray-500 pt-2 border-t border-gray-700">
              <span>Last event: <span className="text-gray-400">{routeStats.lastEventType || 'unknown'}</span></span>
              <span>{formatRelativeTime(routeStats.lastEventAt)}</span>
            </div>
          )}

          {/* No events yet state */}
          {!hasEventFlow && (
            <div className="text-center py-2 text-gray-500 text-xs">
              <p>Waiting for events...</p>
              <p className="text-gray-600 mt-1">Click or type in the browser to generate events</p>
            </div>
          )}
        </div>
      )}

      {/* Debug Info Panel */}
      {showDebug && (debugData || debugError) && (
        <div className="p-3 bg-gray-900 border border-blue-500/30 rounded space-y-3">
          <div className="flex items-center justify-between">
            <span className="text-blue-400 font-medium text-xs uppercase tracking-wide flex items-center gap-1.5">
              <Bug size={12} />
              Browser Script Debug
            </span>
            <button
              onClick={() => recording.active_session_id && fetchDebug(recording.active_session_id)}
              disabled={isDebugLoading}
              className="text-xs text-blue-400 hover:text-blue-300"
            >
              {isDebugLoading ? 'Refreshing...' : 'Refresh'}
            </button>
          </div>

          {debugError && (
            <div className="p-2 bg-red-500/10 border border-red-500/30 rounded text-xs text-red-300">
              Error: {debugError.message}
            </div>
          )}

          {debugData && (
            <DebugInfoDisplay debug={debugData} />
          )}
        </div>
      )}

      {/* Script injection stats */}
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
      ) : recording.active_count === 0 ? (
        <div className="p-3 bg-gray-800/50 border border-gray-700 rounded text-gray-400 text-center">
          <Video size={24} className="mx-auto mb-2 text-gray-500" />
          <p>No recording sessions active</p>
          <p className="text-xs text-gray-500 mt-1">Start a recording to see event flow statistics</p>
        </div>
      ) : null}
    </div>
  );
}

// ============================================================================
// Cleanup Details
// ============================================================================

interface CleanupDetailsProps {
  cleanup: CleanupComponent;
  onCleanNow?: () => void;
  isCleaningUp?: boolean;
  cleanupResult?: { cleaned_up: number } | null;
}

export function CleanupDetails({ cleanup, onCleanNow, isCleaningUp, cleanupResult }: CleanupDetailsProps) {
  return (
    <div className="space-y-3">
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

      {/* Clean Now button and result */}
      <div className="flex items-center gap-3 pt-2 border-t border-gray-700">
        {onCleanNow && (
          <button
            onClick={onCleanNow}
            disabled={isCleaningUp || cleanup.is_running}
            className="flex items-center gap-2 px-3 py-1.5 text-sm bg-gray-700 hover:bg-gray-600 text-gray-300 hover:text-surface rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isCleaningUp ? (
              <>
                <Loader2 size={14} className="animate-spin" />
                Cleaning...
              </>
            ) : (
              <>
                <Trash2 size={14} />
                Clean Now
              </>
            )}
          </button>
        )}
        {cleanupResult && (
          <span className="text-xs text-gray-400">
            {cleanupResult.cleaned_up === 0
              ? 'No idle sessions to clean'
              : `Cleaned ${cleanupResult.cleaned_up} idle session${cleanupResult.cleaned_up !== 1 ? 's' : ''}`}
          </span>
        )}
      </div>
    </div>
  );
}

// ============================================================================
// Metrics Details
// ============================================================================

export function MetricsDetails({ metrics, onViewMetrics }: { metrics: MetricsComponent; onViewMetrics?: () => void }) {
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
