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
} from 'lucide-react';
import type {
  BrowserComponent,
  SessionsComponent,
  RecordingComponent,
  CleanupComponent,
  MetricsComponent,
  SessionInfo,
} from '@/domains/observability';
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
// Recording Details
// ============================================================================

export function RecordingDetails({ recording }: { recording: RecordingComponent }) {
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
