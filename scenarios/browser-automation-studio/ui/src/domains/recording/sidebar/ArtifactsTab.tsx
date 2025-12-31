/**
 * ArtifactsTab Component
 *
 * Unified artifacts viewer for execution mode. Combines screenshots, logs,
 * console output, network requests, and DOM snapshots into a single tabbed interface.
 *
 * Sub-types:
 * - Screenshots: Captured screenshots from execution steps
 * - Execution Logs: Step progress logs (started, completed, failed)
 * - Console: Browser console output (log, warn, error)
 * - Network: Network requests and responses
 * - DOM Snapshots: HTML snapshots at various points
 */

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { format } from 'date-fns';
import {
  Image,
  FileText,
  Terminal,
  Network,
  Code2,
  Loader2,
  Filter,
  AlertCircle,
  CheckCircle,
  Info,
  AlertTriangle,
  ArrowUpRight,
  ArrowDownLeft,
  XCircle,
} from 'lucide-react';
import clsx from 'clsx';
import type { Screenshot, LogEntry, Execution } from '@/domains/executions';
import type { ArtifactSubType } from './types';
import { DEFAULT_ARTIFACT_SUBTYPE } from './types';

// ============================================================================
// Types
// ============================================================================

/** Console log entry from browser */
export interface ConsoleLogEntry {
  id: string;
  level: 'debug' | 'info' | 'warn' | 'error' | 'log';
  text: string;
  timestamp: Date;
  stack?: string;
  location?: string;
}

/** Network event entry */
export interface NetworkEventEntry {
  id: string;
  type: 'request' | 'response' | 'failure';
  url: string;
  method?: string;
  resourceType?: string;
  status?: number;
  ok?: boolean;
  failure?: string;
  timestamp: Date;
  durationMs?: number;
}

/** DOM snapshot entry */
export interface DomSnapshotEntry {
  id: string;
  stepIndex: number;
  stepName: string;
  timestamp: Date;
  storageUrl?: string;
  sizeBytes?: number;
}

export interface ArtifactsTabProps {
  /** Captured screenshots */
  screenshots: Screenshot[];
  /** Index of currently selected screenshot */
  selectedScreenshotIndex?: number;
  /** Callback when a screenshot is selected */
  onSelectScreenshot?: (index: number) => void;

  /** Execution progress logs */
  executionLogs: LogEntry[];
  /** Current log filter */
  logFilter?: 'all' | 'error' | 'warning' | 'info' | 'success';
  /** Callback when log filter changes */
  onLogFilterChange?: (filter: 'all' | 'error' | 'warning' | 'info' | 'success') => void;

  /** Browser console logs */
  consoleLogs: ConsoleLogEntry[];

  /** Network events */
  networkEvents: NetworkEventEntry[];

  /** DOM snapshots */
  domSnapshots: DomSnapshotEntry[];

  /** Current execution status */
  executionStatus?: Execution['status'];

  /** Currently selected sub-type */
  activeSubType?: ArtifactSubType;
  /** Callback when sub-type changes */
  onSubTypeChange?: (subType: ArtifactSubType) => void;

  /** Additional CSS classes */
  className?: string;
}

// ============================================================================
// Sub-type Configuration
// ============================================================================

interface SubTypeConfig {
  id: ArtifactSubType;
  label: string;
  icon: typeof Image;
  shortLabel: string;
}

const SUB_TYPE_CONFIGS: SubTypeConfig[] = [
  { id: 'screenshots', label: 'Screenshots', shortLabel: 'Shots', icon: Image },
  { id: 'execution-logs', label: 'Execution', shortLabel: 'Exec', icon: FileText },
  { id: 'console', label: 'Console', shortLabel: 'Console', icon: Terminal },
  { id: 'network', label: 'Network', shortLabel: 'Network', icon: Network },
  { id: 'dom-snapshots', label: 'DOM', shortLabel: 'DOM', icon: Code2 },
];

// ============================================================================
// Helper Components
// ============================================================================

/** Get icon and color for log level */
function getLogLevelConfig(level: LogEntry['level']) {
  switch (level) {
    case 'error':
      return { icon: AlertCircle, color: 'text-red-400', bgColor: 'bg-red-500/10' };
    case 'warning':
      return { icon: AlertTriangle, color: 'text-yellow-400', bgColor: 'bg-yellow-500/10' };
    case 'success':
      return { icon: CheckCircle, color: 'text-green-400', bgColor: 'bg-green-500/10' };
    default:
      return { icon: Info, color: 'text-gray-400', bgColor: 'bg-gray-500/10' };
  }
}

/** Get icon and color for console level */
function getConsoleLevelConfig(level: ConsoleLogEntry['level']) {
  switch (level) {
    case 'error':
      return { icon: XCircle, color: 'text-red-400', bgColor: 'bg-red-500/10' };
    case 'warn':
      return { icon: AlertTriangle, color: 'text-yellow-400', bgColor: 'bg-yellow-500/10' };
    case 'debug':
      return { icon: Terminal, color: 'text-purple-400', bgColor: 'bg-purple-500/10' };
    default:
      return { icon: Info, color: 'text-blue-400', bgColor: 'bg-blue-500/10' };
  }
}

/** Get status color for network request */
function getNetworkStatusConfig(status?: number, ok?: boolean, failure?: string) {
  if (failure) {
    return { color: 'text-red-400', bgColor: 'bg-red-500/10' };
  }
  if (!status) {
    return { color: 'text-gray-400', bgColor: 'bg-gray-500/10' };
  }
  if (ok || (status >= 200 && status < 300)) {
    return { color: 'text-green-400', bgColor: 'bg-green-500/10' };
  }
  if (status >= 300 && status < 400) {
    return { color: 'text-blue-400', bgColor: 'bg-blue-500/10' };
  }
  if (status >= 400 && status < 500) {
    return { color: 'text-yellow-400', bgColor: 'bg-yellow-500/10' };
  }
  return { color: 'text-red-400', bgColor: 'bg-red-500/10' };
}

// ============================================================================
// Sub-Views
// ============================================================================

/** Screenshots sub-view */
function ScreenshotsView({
  screenshots,
  selectedIndex = 0,
  onSelectScreenshot,
  executionStatus,
}: {
  screenshots: Screenshot[];
  selectedIndex?: number;
  onSelectScreenshot?: (index: number) => void;
  executionStatus?: Execution['status'];
}) {
  const thumbnailContainerRef = useRef<HTMLDivElement>(null);
  const selectedThumbnailRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (selectedThumbnailRef.current && thumbnailContainerRef.current) {
      selectedThumbnailRef.current.scrollIntoView({
        behavior: 'smooth',
        block: 'nearest',
        inline: 'center',
      });
    }
  }, [selectedIndex]);

  const selectedScreenshot = screenshots[selectedIndex];
  const isLoading = executionStatus === 'pending' || executionStatus === 'running';

  if (screenshots.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center p-6 text-center">
        <div>
          {isLoading ? (
            <>
              <Loader2 size={32} className="mx-auto mb-3 text-flow-accent animate-spin" />
              <div className="text-sm text-gray-400 mb-1">Waiting for screenshots...</div>
              <div className="text-xs text-gray-500">Screenshots will appear as the workflow executes</div>
            </>
          ) : (
            <>
              <Image size={32} className="mx-auto mb-3 text-gray-600" />
              <div className="text-sm text-gray-400 mb-1">No screenshots captured</div>
              <div className="text-xs text-gray-500">This execution did not capture any screenshots</div>
            </>
          )}
        </div>
      </div>
    );
  }

  return (
    <>
      <div className="flex-1 p-3 overflow-auto">
        {selectedScreenshot && (
          <div className="rounded-xl border border-gray-700 overflow-hidden bg-gray-900">
            <div className="bg-gray-800/80 px-3 py-2 flex items-center justify-between text-xs border-b border-gray-700">
              <span className="truncate font-medium text-gray-300">{selectedScreenshot.stepName}</span>
              <span className="text-gray-500 flex-shrink-0 ml-2">
                {format(selectedScreenshot.timestamp, 'HH:mm:ss.SSS')}
              </span>
            </div>
            <img src={selectedScreenshot.url} alt={selectedScreenshot.stepName} className="block w-full" loading="lazy" />
          </div>
        )}
      </div>

      {screenshots.length > 1 && (
        <div className="border-t border-gray-700 p-2">
          <div ref={thumbnailContainerRef} className="flex gap-2 overflow-x-auto pb-1">
            {screenshots.map((screenshot, index) => {
              const isSelected = index === selectedIndex;
              return (
                <button
                  key={screenshot.id}
                  ref={isSelected ? selectedThumbnailRef : undefined}
                  type="button"
                  onClick={() => onSelectScreenshot?.(index)}
                  className={clsx(
                    'flex-shrink-0 w-16 h-12 rounded-lg overflow-hidden border-2 transition-all duration-150',
                    isSelected ? 'border-flow-accent shadow-lg shadow-flow-accent/20' : 'border-gray-700 hover:border-gray-600'
                  )}
                  title={`${screenshot.stepName} - ${format(screenshot.timestamp, 'HH:mm:ss')}`}
                >
                  <img src={screenshot.url} alt={screenshot.stepName} loading="lazy" className="w-full h-full object-cover" />
                </button>
              );
            })}
          </div>
          <div className="text-xs text-gray-500 text-center mt-1">
            {selectedIndex + 1} of {screenshots.length}
          </div>
        </div>
      )}
    </>
  );
}

/** Execution logs sub-view */
function ExecutionLogsView({
  logs,
  filter = 'all',
  onFilterChange,
  executionStatus,
}: {
  logs: LogEntry[];
  filter?: 'all' | 'error' | 'warning' | 'info' | 'success';
  onFilterChange?: (filter: 'all' | 'error' | 'warning' | 'info' | 'success') => void;
  executionStatus?: Execution['status'];
}) {
  const logsContainerRef = useRef<HTMLDivElement>(null);
  const [autoScroll, setAutoScroll] = useState(true);
  const prevLogsLengthRef = useRef(logs.length);

  const filteredLogs = useMemo(() => {
    if (filter === 'all') return logs;
    return logs.filter((log) => log.level === filter);
  }, [logs, filter]);

  const counts = useMemo(() => {
    const result: Record<string, number> = { all: logs.length, error: 0, warning: 0, info: 0, success: 0 };
    for (const log of logs) {
      if (log.level in result) {
        result[log.level]++;
      }
    }
    return result;
  }, [logs]);

  useEffect(() => {
    if (autoScroll && logs.length > prevLogsLengthRef.current && logsContainerRef.current) {
      logsContainerRef.current.scrollTop = logsContainerRef.current.scrollHeight;
    }
    prevLogsLengthRef.current = logs.length;
  }, [logs.length, autoScroll]);

  const handleScroll = useCallback(() => {
    if (!logsContainerRef.current) return;
    const { scrollTop, scrollHeight, clientHeight } = logsContainerRef.current;
    const isAtBottom = scrollHeight - scrollTop - clientHeight < 50;
    setAutoScroll(isAtBottom);
  }, []);

  const isLoading = executionStatus === 'pending' || executionStatus === 'running';

  if (logs.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center p-6 text-center">
        <div>
          {isLoading ? (
            <>
              <Loader2 size={32} className="mx-auto mb-3 text-flow-accent animate-spin" />
              <div className="text-sm text-gray-400 mb-1">Waiting for logs...</div>
              <div className="text-xs text-gray-500">Logs will appear as the workflow executes</div>
            </>
          ) : (
            <>
              <FileText size={32} className="mx-auto mb-3 text-gray-600" />
              <div className="text-sm text-gray-400 mb-1">No logs recorded</div>
              <div className="text-xs text-gray-500">This execution did not produce any log entries</div>
            </>
          )}
        </div>
      </div>
    );
  }

  const filters: Array<'all' | 'error' | 'warning' | 'info'> = ['all', 'error', 'warning', 'info'];

  return (
    <>
      <div className="border-b border-gray-700 px-3 py-2 flex items-center gap-1 flex-shrink-0">
        <Filter size={14} className="text-gray-500 mr-1" />
        {filters.map((f) => (
          <button
            key={f}
            type="button"
            onClick={() => onFilterChange?.(f)}
            className={clsx(
              'px-2 py-1 text-xs rounded-md transition-colors flex items-center gap-1',
              filter === f ? 'bg-flow-accent/20 text-flow-accent' : 'text-gray-400 hover:text-gray-300 hover:bg-gray-800'
            )}
          >
            {f === 'all' ? 'All' : f.charAt(0).toUpperCase() + f.slice(1)}
            {counts[f] > 0 && (
              <span className={clsx('text-[10px] px-1 rounded', filter === f ? 'bg-flow-accent/30' : 'bg-gray-700')}>
                {counts[f]}
              </span>
            )}
          </button>
        ))}
      </div>

      <div ref={logsContainerRef} onScroll={handleScroll} className="flex-1 overflow-auto p-3 font-mono text-xs">
        {filteredLogs.length === 0 ? (
          <div className="text-center text-gray-500 py-4">No {filter} logs</div>
        ) : (
          <div className="space-y-1">
            {filteredLogs.map((log) => {
              const config = getLogLevelConfig(log.level);
              const Icon = config.icon;
              return (
                <div key={log.id} className={clsx('flex items-start gap-2 px-2 py-1.5 rounded-md', config.bgColor)}>
                  <Icon size={14} className={clsx('flex-shrink-0 mt-0.5', config.color)} />
                  <div className="flex-1 min-w-0">
                    <div className={clsx('break-words', config.color)}>{log.message}</div>
                    <div className="text-gray-600 mt-0.5">{format(log.timestamp, 'HH:mm:ss.SSS')}</div>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </>
  );
}

/** Console logs sub-view */
function ConsoleLogsView({
  consoleLogs,
  executionStatus,
}: {
  consoleLogs: ConsoleLogEntry[];
  executionStatus?: Execution['status'];
}) {
  const isLoading = executionStatus === 'pending' || executionStatus === 'running';

  if (consoleLogs.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center p-6 text-center">
        <div>
          {isLoading ? (
            <>
              <Loader2 size={32} className="mx-auto mb-3 text-flow-accent animate-spin" />
              <div className="text-sm text-gray-400 mb-1">Waiting for console output...</div>
              <div className="text-xs text-gray-500">Console logs will appear as the workflow executes</div>
            </>
          ) : (
            <>
              <Terminal size={32} className="mx-auto mb-3 text-gray-600" />
              <div className="text-sm text-gray-400 mb-1">No console output</div>
              <div className="text-xs text-gray-500">This execution did not produce any console logs</div>
            </>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className="flex-1 overflow-auto p-3 font-mono text-xs">
      <div className="space-y-1">
        {consoleLogs.map((log) => {
          const config = getConsoleLevelConfig(log.level);
          const Icon = config.icon;
          return (
            <div key={log.id} className={clsx('flex items-start gap-2 px-2 py-1.5 rounded-md', config.bgColor)}>
              <Icon size={14} className={clsx('flex-shrink-0 mt-0.5', config.color)} />
              <div className="flex-1 min-w-0">
                <div className={clsx('break-words whitespace-pre-wrap', config.color)}>{log.text}</div>
                {log.location && <div className="text-gray-600 mt-0.5 text-[10px]">{log.location}</div>}
                <div className="text-gray-600 mt-0.5">{format(log.timestamp, 'HH:mm:ss.SSS')}</div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

/** Network events sub-view */
function NetworkView({
  networkEvents,
  executionStatus,
}: {
  networkEvents: NetworkEventEntry[];
  executionStatus?: Execution['status'];
}) {
  const isLoading = executionStatus === 'pending' || executionStatus === 'running';

  if (networkEvents.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center p-6 text-center">
        <div>
          {isLoading ? (
            <>
              <Loader2 size={32} className="mx-auto mb-3 text-flow-accent animate-spin" />
              <div className="text-sm text-gray-400 mb-1">Waiting for network activity...</div>
              <div className="text-xs text-gray-500">Network requests will appear as the workflow executes</div>
            </>
          ) : (
            <>
              <Network size={32} className="mx-auto mb-3 text-gray-600" />
              <div className="text-sm text-gray-400 mb-1">No network activity</div>
              <div className="text-xs text-gray-500">This execution did not capture any network requests</div>
            </>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className="flex-1 overflow-auto p-3 font-mono text-xs">
      <div className="space-y-1">
        {networkEvents.map((event) => {
          const statusConfig = getNetworkStatusConfig(event.status, event.ok, event.failure);
          const isRequest = event.type === 'request';
          const isFailure = event.type === 'failure';

          return (
            <div key={event.id} className={clsx('flex items-start gap-2 px-2 py-1.5 rounded-md', statusConfig.bgColor)}>
              {isRequest ? (
                <ArrowUpRight size={14} className="flex-shrink-0 mt-0.5 text-blue-400" />
              ) : isFailure ? (
                <XCircle size={14} className="flex-shrink-0 mt-0.5 text-red-400" />
              ) : (
                <ArrowDownLeft size={14} className={clsx('flex-shrink-0 mt-0.5', statusConfig.color)} />
              )}
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  {event.method && (
                    <span className="px-1.5 py-0.5 bg-gray-700 rounded text-gray-300 font-medium">{event.method}</span>
                  )}
                  {event.status && (
                    <span className={clsx('font-medium', statusConfig.color)}>{event.status}</span>
                  )}
                  {event.resourceType && (
                    <span className="text-gray-500">{event.resourceType}</span>
                  )}
                </div>
                <div className="text-gray-300 break-all mt-1">{event.url}</div>
                {event.failure && <div className="text-red-400 mt-1">{event.failure}</div>}
                <div className="text-gray-600 mt-0.5 flex items-center gap-2">
                  <span>{format(event.timestamp, 'HH:mm:ss.SSS')}</span>
                  {event.durationMs != null && <span>{event.durationMs}ms</span>}
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

/** DOM snapshots sub-view */
function DomSnapshotsView({
  domSnapshots,
  executionStatus,
}: {
  domSnapshots: DomSnapshotEntry[];
  executionStatus?: Execution['status'];
}) {
  const isLoading = executionStatus === 'pending' || executionStatus === 'running';

  if (domSnapshots.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center p-6 text-center">
        <div>
          {isLoading ? (
            <>
              <Loader2 size={32} className="mx-auto mb-3 text-flow-accent animate-spin" />
              <div className="text-sm text-gray-400 mb-1">Waiting for DOM snapshots...</div>
              <div className="text-xs text-gray-500">DOM snapshots will appear as the workflow executes</div>
            </>
          ) : (
            <>
              <Code2 size={32} className="mx-auto mb-3 text-gray-600" />
              <div className="text-sm text-gray-400 mb-1">No DOM snapshots</div>
              <div className="text-xs text-gray-500">This execution did not capture any DOM snapshots</div>
            </>
          )}
        </div>
      </div>
    );
  }

  const formatSize = (bytes?: number) => {
    if (!bytes) return null;
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  return (
    <div className="flex-1 overflow-auto p-3">
      <div className="space-y-2">
        {domSnapshots.map((snapshot) => (
          <div
            key={snapshot.id}
            className="flex items-center justify-between px-3 py-2 rounded-lg bg-gray-800/50 border border-gray-700 hover:bg-gray-800 transition-colors"
          >
            <div className="flex items-center gap-3 min-w-0">
              <Code2 size={16} className="flex-shrink-0 text-gray-400" />
              <div className="min-w-0">
                <div className="text-sm text-gray-300 truncate">{snapshot.stepName}</div>
                <div className="text-xs text-gray-500 flex items-center gap-2">
                  <span>{format(snapshot.timestamp, 'HH:mm:ss.SSS')}</span>
                  {snapshot.sizeBytes && <span>{formatSize(snapshot.sizeBytes)}</span>}
                </div>
              </div>
            </div>
            {snapshot.storageUrl && (
              <a
                href={snapshot.storageUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="px-2 py-1 text-xs text-flow-accent hover:bg-flow-accent/20 rounded transition-colors"
              >
                View
              </a>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}

// ============================================================================
// Main Component
// ============================================================================

export function ArtifactsTab({
  screenshots,
  selectedScreenshotIndex,
  onSelectScreenshot,
  executionLogs,
  logFilter,
  onLogFilterChange,
  consoleLogs,
  networkEvents,
  domSnapshots,
  executionStatus,
  activeSubType = DEFAULT_ARTIFACT_SUBTYPE,
  onSubTypeChange,
  className,
}: ArtifactsTabProps) {
  // Calculate counts for badges
  const counts = useMemo(() => ({
    screenshots: screenshots.length,
    'execution-logs': executionLogs.length,
    console: consoleLogs.length,
    network: networkEvents.length,
    'dom-snapshots': domSnapshots.length,
  }), [screenshots.length, executionLogs.length, consoleLogs.length, networkEvents.length, domSnapshots.length]);

  return (
    <div className={clsx('flex flex-col h-full', className)}>
      {/* Segmented control */}
      <div className="border-b border-gray-700 px-2 py-2 flex-shrink-0">
        <div className="flex gap-1 overflow-x-auto">
          {SUB_TYPE_CONFIGS.map((config) => {
            const isActive = activeSubType === config.id;
            const count = counts[config.id];
            const Icon = config.icon;

            return (
              <button
                key={config.id}
                type="button"
                onClick={() => onSubTypeChange?.(config.id)}
                className={clsx(
                  'flex items-center gap-1.5 px-2.5 py-1.5 text-xs rounded-md transition-colors whitespace-nowrap',
                  isActive
                    ? 'bg-flow-accent/20 text-flow-accent'
                    : 'text-gray-400 hover:text-gray-300 hover:bg-gray-800'
                )}
                title={config.label}
              >
                <Icon size={14} />
                <span className="hidden sm:inline">{config.shortLabel}</span>
                {count > 0 && (
                  <span
                    className={clsx(
                      'text-[10px] px-1 rounded min-w-[16px] text-center',
                      isActive ? 'bg-flow-accent/30' : 'bg-gray-700'
                    )}
                  >
                    {count > 99 ? '99+' : count}
                  </span>
                )}
              </button>
            );
          })}
        </div>
      </div>

      {/* Content area */}
      <div className="flex-1 flex flex-col min-h-0">
        {activeSubType === 'screenshots' && (
          <ScreenshotsView
            screenshots={screenshots}
            selectedIndex={selectedScreenshotIndex}
            onSelectScreenshot={onSelectScreenshot}
            executionStatus={executionStatus}
          />
        )}

        {activeSubType === 'execution-logs' && (
          <ExecutionLogsView
            logs={executionLogs}
            filter={logFilter}
            onFilterChange={onLogFilterChange}
            executionStatus={executionStatus}
          />
        )}

        {activeSubType === 'console' && (
          <ConsoleLogsView
            consoleLogs={consoleLogs}
            executionStatus={executionStatus}
          />
        )}

        {activeSubType === 'network' && (
          <NetworkView
            networkEvents={networkEvents}
            executionStatus={executionStatus}
          />
        )}

        {activeSubType === 'dom-snapshots' && (
          <DomSnapshotsView
            domSnapshots={domSnapshots}
            executionStatus={executionStatus}
          />
        )}
      </div>
    </div>
  );
}

export default ArtifactsTab;
