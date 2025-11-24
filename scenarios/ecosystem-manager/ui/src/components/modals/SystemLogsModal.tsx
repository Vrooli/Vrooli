/**
 * System Logs Modal
 * Modernized multi-tab view for API logs, execution history, and Auto Steer performance.
 */

import { useEffect, useMemo, useRef, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Activity, ChevronsDown, Clock3, History, LineChart, RefreshCw } from 'lucide-react';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '../ui/dialog';
import { Button } from '../ui/button';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../ui/select';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../ui/tabs';
import { Input } from '../ui/input';
import { ExecutionDetailCard } from '../executions/ExecutionDetailCard';
import { useSystemLogs } from '@/hooks/useSystemLogs';
import { useAutoSteerProfiles } from '@/hooks/useAutoSteer';
import { api } from '@/lib/api';
import { queryKeys } from '@/lib/queryKeys';
import type { ExecutionHistory, LogEntry, ProfilePerformance } from '@/types/api';

interface SystemLogsModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

type LogLevel = 'all' | 'info' | 'warning' | 'error';
type ExecutionStatus = 'all' | 'running' | 'completed' | 'failed' | 'rate_limited';

export function SystemLogsModal({ open, onOpenChange }: SystemLogsModalProps) {
  const [activeTab, setActiveTab] = useState<'logs' | 'executions' | 'performance'>('logs');
  const [logLevel, setLogLevel] = useState<LogLevel>('all');
  const [autoScroll, setAutoScroll] = useState(true);
  const [executionStatus, setExecutionStatus] = useState<ExecutionStatus>('all');
  const [executionSearch, setExecutionSearch] = useState('');
  const [profileFilter, setProfileFilter] = useState<string>('all');
  const [scenarioFilter, setScenarioFilter] = useState('');
  const [selectedExecutionId, setSelectedExecutionId] = useState<string | null>(null);
  const [selectedPerformanceId, setSelectedPerformanceId] = useState<string | null>(null);

  const logsEndRef = useRef<HTMLDivElement>(null);
  const logsContainerRef = useRef<HTMLDivElement>(null);

  const { data: logs = [], isLoading: isLoadingLogs, refetch: refetchLogs } = useSystemLogs({
    limit: 500,
    level: logLevel,
    refetchInterval: open && activeTab === 'logs' ? 5000 : undefined,
  });

  const { data: executions = [], isLoading: isLoadingExecutions, refetch: refetchExecutions } =
    useQuery({
      queryKey: queryKeys.executions.list(),
      queryFn: () => api.getAllExecutionHistory(),
      enabled: open && activeTab === 'executions',
      staleTime: 10000,
    });

  const { data: performanceHistory = [], isLoading: isLoadingPerformance, refetch: refetchPerformance } =
    useQuery({
      queryKey: queryKeys.autoSteer.history(
        profileFilter === 'all' ? undefined : profileFilter,
        scenarioFilter || undefined,
      ),
      queryFn: () =>
        api.getAutoSteerHistory({
          profile_id: profileFilter === 'all' ? undefined : profileFilter,
          scenario: scenarioFilter || undefined,
        }),
      enabled: open && activeTab === 'performance',
      staleTime: 15000,
    });

  const { data: profiles = [] } = useAutoSteerProfiles();

  const profileNameMap = useMemo(
    () => Object.fromEntries((profiles ?? []).map((p) => [p.id, p.name])),
    [profiles],
  );

  useEffect(() => {
    if (activeTab !== 'logs') return;
    if (autoScroll && logsEndRef.current) {
      logsEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [logs, autoScroll, activeTab]);

  const handleScroll = () => {
    if (!logsContainerRef.current) return;
    const { scrollTop, scrollHeight, clientHeight } = logsContainerRef.current;
    const isAtBottom = Math.abs(scrollHeight - clientHeight - scrollTop) < 10;
    if (!isAtBottom && autoScroll) {
      setAutoScroll(false);
    } else if (isAtBottom && !autoScroll) {
      setAutoScroll(true);
    }
  };

  const formatDateTime = (value?: string) => {
    if (!value) return '--';
    try {
      return new Date(value).toLocaleString();
    } catch {
      return value;
    }
  };

  const formatDurationMs = (ms?: number) => {
    if (!ms || ms <= 0) return '--';
    const totalSeconds = Math.floor(ms / 1000);
    const minutes = Math.floor(totalSeconds / 60);
    const hours = Math.floor(minutes / 60);
    if (hours > 0) {
      return `${hours}h ${minutes % 60}m`;
    }
    if (minutes > 0) {
      return `${minutes}m ${totalSeconds % 60}s`;
    }
    return `${totalSeconds}s`;
  };

  const formatExecutionDuration = (execution: ExecutionHistory) => {
    if (execution.duration) {
      return execution.duration;
    }
    if (execution.end_time) {
      const start = new Date(execution.start_time).getTime();
      const end = new Date(execution.end_time).getTime();
      return formatDurationMs(end - start);
    }
    return '--';
  };

  const formatSteerInfo = (execution?: ExecutionHistory | null) => {
    if (!execution) return '—';
    const mode = execution.steer_mode || execution.steering_source || '';
    const phase = execution.steer_phase_index ? ` • phase ${execution.steer_phase_index}` : '';
    return mode ? `${mode}${phase}` : '—';
  };

  const getLevelStyles = (level: LogEntry['level']) => {
    switch (level) {
      case 'error':
        return {
          container: 'border-red-500/40 bg-red-500/10 shadow-[0_0_0_1px_rgba(248,113,113,0.25)]',
          accent: 'bg-red-500/70',
          badge: 'bg-red-500/20 text-red-100 border-red-500/50',
          timestamp: 'text-red-100/80',
        };
      case 'warning':
        return {
          container: 'border-amber-400/50 bg-amber-400/10 shadow-[0_0_0_1px_rgba(251,191,36,0.2)]',
          accent: 'bg-amber-400/80',
          badge: 'bg-amber-500/20 text-amber-100 border-amber-500/50',
          timestamp: 'text-amber-100/80',
        };
      case 'info':
        return {
          container: 'border-blue-400/40 bg-blue-500/10 shadow-[0_0_0_1px_rgba(96,165,250,0.2)]',
          accent: 'bg-blue-400/80',
          badge: 'bg-blue-500/20 text-blue-100 border-blue-500/50',
          timestamp: 'text-blue-100/80',
        };
      default:
        return {
          container: 'border-slate-500/40 bg-slate-700/20',
          accent: 'bg-slate-400/70',
          badge: 'bg-slate-500/20 text-slate-200 border-slate-500/50',
          timestamp: 'text-slate-200/80',
        };
    }
  };

  const getStatusTone = (status: ExecutionHistory['status']) => {
    switch (status) {
      case 'completed':
        return 'bg-emerald-500/20 text-emerald-100 border-emerald-400/60';
      case 'running':
        return 'bg-amber-400/20 text-amber-100 border-amber-300/60';
      case 'failed':
        return 'bg-red-500/20 text-red-100 border-red-400/60';
      case 'rate_limited':
        return 'bg-orange-500/20 text-orange-100 border-orange-400/60';
      default:
        return 'bg-slate-500/20 text-slate-200 border-slate-500/40';
    }
  };

  const computeImprovement = (perf: ProfilePerformance) => {
    const start = perf.start_metrics?.operational_targets_percentage ?? 0;
    const end = perf.end_metrics?.operational_targets_percentage ?? start;
    return end - start;
  };

  const filteredExecutions = useMemo(() => {
    const search = executionSearch.trim().toLowerCase();
    return (executions ?? [])
      .filter((exec) => (executionStatus === 'all' ? true : exec.status === executionStatus))
      .filter((exec) => {
        if (!search) return true;
        return (
          exec.id?.toLowerCase().includes(search) ||
          exec.task_id?.toLowerCase().includes(search)
        );
      })
      .sort(
        (a, b) =>
          new Date(b.start_time).getTime() - new Date(a.start_time).getTime(),
      );
  }, [executions, executionSearch, executionStatus]);

  useEffect(() => {
    if (filteredExecutions.length === 0) {
      if (selectedExecutionId !== null) {
        setSelectedExecutionId(null);
      }
      return;
    }

    if (!selectedExecutionId) {
      setSelectedExecutionId(filteredExecutions[0]?.id ?? null);
      return;
    }

    if (!filteredExecutions.some((exec) => exec.id === selectedExecutionId)) {
      setSelectedExecutionId(filteredExecutions[0]?.id ?? null);
    }
  }, [filteredExecutions, selectedExecutionId]);

  const selectedExecution =
    filteredExecutions.find((exec) => exec.id === selectedExecutionId) ?? null;

  const { data: selectedExecutionPrompt, isFetching: isFetchingPrompt } = useQuery({
    queryKey:
      selectedExecution && selectedExecution.task_id && selectedExecution.id
        ? queryKeys.executions.prompt(selectedExecution.task_id, selectedExecution.id)
        : ['executions', 'prompt', 'inactive'],
    queryFn: () => api.getExecutionPrompt(selectedExecution!.task_id, selectedExecution!.id),
    enabled:
      !!selectedExecution &&
      !!selectedExecution.task_id &&
      !!selectedExecution.id &&
      open &&
      activeTab === 'executions',
    staleTime: 15000,
  });

  const { data: selectedExecutionOutput, isFetching: isFetchingSelectedOutput } = useQuery({
    queryKey:
      selectedExecution && selectedExecution.task_id && selectedExecution.id
        ? queryKeys.executions.output(selectedExecution.task_id, selectedExecution.id)
        : ['executions', 'output', 'inactive'],
    queryFn: () => api.getExecutionOutput(selectedExecution!.task_id, selectedExecution!.id),
    enabled:
      !!selectedExecution &&
      !!selectedExecution.task_id &&
      !!selectedExecution.id &&
      open &&
      activeTab === 'executions',
    staleTime: 15000,
  });

  const selectedPromptText =
    (selectedExecutionPrompt as any)?.content ??
    (selectedExecutionPrompt as any)?.prompt ??
    '';
  const selectedOutputText =
    (selectedExecutionOutput as any)?.output ??
    (selectedExecutionOutput as any)?.content ??
    '';

  const filteredPerformance = useMemo(() => {
    const scenarioTerm = scenarioFilter.trim().toLowerCase();
    return (performanceHistory ?? []).filter((perf) => {
      const matchesProfile = profileFilter === 'all' ? true : perf.profile_id === profileFilter;
      const matchesScenario = scenarioTerm
        ? perf.scenario_name?.toLowerCase().includes(scenarioTerm)
        : true;
      return matchesProfile && matchesScenario;
    });
  }, [performanceHistory, profileFilter, scenarioFilter]);

  useEffect(() => {
    if (filteredPerformance.length > 0 && !selectedPerformanceId) {
      const firstId =
        filteredPerformance[0].execution_id ||
        (filteredPerformance[0] as any).id ||
        null;
      if (firstId) {
        setSelectedPerformanceId(String(firstId));
      }
    } else if (filteredPerformance.length === 0 && selectedPerformanceId) {
      setSelectedPerformanceId(null);
    }
  }, [filteredPerformance, selectedPerformanceId]);

  const selectedPerformance =
    filteredPerformance.find(
      (perf) =>
        String(perf.execution_id ?? (perf as any).id ?? '') ===
        String(selectedPerformanceId ?? ''),
    ) ?? null;

  const averageImprovement =
    filteredPerformance.length === 0
      ? 0
      : filteredPerformance.reduce((sum, perf) => sum + computeImprovement(perf), 0) /
        filteredPerformance.length;

  const ratingData = filteredPerformance.reduce(
    (acc, perf) => {
      if (perf.user_feedback?.rating) {
        acc.total += perf.user_feedback.rating;
        acc.count += 1;
      }
      return acc;
    },
    { total: 0, count: 0 },
  );
  const averageRating = ratingData.count === 0 ? null : ratingData.total / ratingData.count;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-6xl h-[82vh] flex flex-col bg-slate-950/95 border border-white/10 text-white shadow-2xl">
        <DialogHeader className="flex flex-row items-center space-y-0 pb-2">
          <DialogTitle>System Logs & History</DialogTitle>
        </DialogHeader>

        <Tabs
          value={activeTab}
          onValueChange={(value) =>
            setActiveTab(value as 'logs' | 'executions' | 'performance')
          }
          className="flex-1 flex flex-col"
        >
          <div className="flex items-center justify-between border-b border-white/10 pb-3">
            <TabsList className="bg-slate-900/70 border border-white/5">
              <TabsTrigger value="logs" className="gap-2">
                <Activity className="h-4 w-4" />
                API Logs
              </TabsTrigger>
              <TabsTrigger value="executions" className="gap-2">
                <History className="h-4 w-4" />
                Execution History
              </TabsTrigger>
              <TabsTrigger value="performance" className="gap-2">
                <LineChart className="h-4 w-4" />
                Auto Steer Performance
              </TabsTrigger>
            </TabsList>

            <div className="flex items-center gap-2">
              {activeTab === 'logs' && (
                <>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setAutoScroll(!autoScroll)}
                    className={autoScroll ? 'bg-green-500/10 border-green-500/30' : ''}
                  >
                    <ChevronsDown className="h-4 w-4 mr-2" />
                    Auto-scroll {autoScroll ? 'ON' : 'OFF'}
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => refetchLogs()}
                    disabled={isLoadingLogs}
                  >
                    <RefreshCw className={`h-4 w-4 mr-2 ${isLoadingLogs ? 'animate-spin' : ''}`} />
                    Refresh
                  </Button>
                </>
              )}

              {activeTab === 'executions' && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => refetchExecutions()}
                  disabled={isLoadingExecutions}
                >
                  <RefreshCw className={`h-4 w-4 mr-2 ${isLoadingExecutions ? 'animate-spin' : ''}`} />
                  Refresh
                </Button>
              )}

              {activeTab === 'performance' && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => refetchPerformance()}
                  disabled={isLoadingPerformance}
                >
                  <RefreshCw className={`h-4 w-4 mr-2 ${isLoadingPerformance ? 'animate-spin' : ''}`} />
                  Refresh
                </Button>
              )}
            </div>
          </div>

          <TabsContent value="logs" className="flex-1 flex flex-col">
            <div className="flex items-center justify-between gap-4 pb-4">
              <div className="flex items-center gap-3">
                <label className="text-sm text-slate-400">Level:</label>
                <Select value={logLevel} onValueChange={(value: LogLevel) => setLogLevel(value)}>
                  <SelectTrigger className="w-32">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All</SelectItem>
                    <SelectItem value="info">Info</SelectItem>
                    <SelectItem value="warning">Warning</SelectItem>
                    <SelectItem value="error">Error</SelectItem>
                  </SelectContent>
                </Select>

                <div className="text-xs text-slate-500 ml-2">
                  {logs.length} {logs.length === 1 ? 'entry' : 'entries'}
                </div>
              </div>
            </div>

            <div className="flex-1 rounded-lg border border-white/5 bg-slate-900/50 p-2">
              <div
                ref={logsContainerRef}
                onScroll={handleScroll}
                className="flex-1 max-h-[60vh] overflow-y-auto space-y-2 pr-2"
              >
                {isLoadingLogs && logs.length === 0 ? (
                  <div className="flex items-center justify-center h-full text-slate-500">
                    Loading logs...
                  </div>
                ) : logs.length === 0 ? (
                  <div className="flex items-center justify-center h-full text-slate-500">
                    No logs found
                  </div>
                ) : (
                  logs.map((log, index) => {
                    const levelStyles = getLevelStyles(log.level);
                    return (
                      <div
                        key={`${log.timestamp}-${index}`}
                        className={`relative flex gap-3 p-3 rounded-lg border overflow-hidden ${levelStyles.container}`}
                      >
                        <div className={`absolute inset-y-0 left-0 w-1 ${levelStyles.accent}`} />

                        <div className={`flex-shrink-0 text-xs font-mono w-40 ${levelStyles.timestamp}`}>
                          {formatDateTime(log.timestamp)}
                        </div>

                        <div className="flex-shrink-0">
                          <span
                            className={`text-xs font-semibold px-2 py-1 rounded border uppercase ${levelStyles.badge}`}
                          >
                            {log.level}
                          </span>
                        </div>

                        <div className="flex-1 text-sm break-words text-slate-100">
                          {log.message}
                          {log.context && Object.keys(log.context).length > 0 && (
                            <pre className="mt-2 text-xs text-slate-300 overflow-x-auto bg-white/5 rounded p-2">
                              {JSON.stringify(log.context, null, 2)}
                            </pre>
                          )}
                        </div>
                      </div>
                    );
                  })
                )}
                <div ref={logsEndRef} />
              </div>
            </div>
          </TabsContent>

          <TabsContent value="executions" className="flex-1 flex flex-col gap-4">
            <div className="flex flex-wrap items-center gap-3">
              <Input
                placeholder="Search by task or execution id..."
                value={executionSearch}
                onChange={(e) => setExecutionSearch(e.target.value)}
                className="w-full md:w-72"
              />
              <Select value={executionStatus} onValueChange={(value: ExecutionStatus) => setExecutionStatus(value)}>
                <SelectTrigger className="w-44">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All statuses</SelectItem>
                  <SelectItem value="running">Running</SelectItem>
                  <SelectItem value="completed">Completed</SelectItem>
                  <SelectItem value="failed">Failed</SelectItem>
                  <SelectItem value="rate_limited">Rate limited</SelectItem>
                </SelectContent>
              </Select>
              <div className="text-xs text-slate-500">
                {filteredExecutions.length} {filteredExecutions.length === 1 ? 'execution' : 'executions'}
              </div>
            </div>

            <div className="grid grid-cols-1 xl:grid-cols-[minmax(0,1.05fr)_minmax(0,1fr)] gap-4 flex-1">
              <div className="overflow-hidden rounded-lg border border-white/5 bg-slate-900/50">
                <div className="max-h-[58vh] overflow-y-auto">
                  <table className="w-full text-sm">
                    <thead className="bg-white/5 text-slate-200 sticky top-0 backdrop-blur">
                      <tr>
                        <th className="text-left px-4 py-3 font-medium">Execution</th>
                        <th className="text-left px-4 py-3 font-medium">Task</th>
                        <th className="text-left px-4 py-3 font-medium">Steer</th>
                        <th className="text-left px-4 py-3 font-medium">Status</th>
                        <th className="text-left px-4 py-3 font-medium">Started</th>
                        <th className="text-left px-4 py-3 font-medium">Duration</th>
                        <th className="text-left px-4 py-3 font-medium">Exit</th>
                      </tr>
                    </thead>
                    <tbody>
                      {isLoadingExecutions && filteredExecutions.length === 0 && (
                        <tr>
                          <td colSpan={7} className="px-4 py-6 text-center text-slate-500">
                            Loading execution history...
                          </td>
                        </tr>
                      )}
                      {!isLoadingExecutions && filteredExecutions.length === 0 && (
                        <tr>
                          <td colSpan={7} className="px-4 py-6 text-center text-slate-500">
                            No executions found
                          </td>
                        </tr>
                      )}
                      {filteredExecutions.map((exec) => {
                        const isSelected = exec.id === selectedExecutionId;
                        return (
                          <tr
                            key={exec.id}
                            onClick={() => setSelectedExecutionId(exec.id)}
                            className={`border-t border-white/5 cursor-pointer transition-colors ${
                              isSelected ? 'bg-white/10' : 'hover:bg-white/5'
                            }`}
                          >
                            <td className="px-4 py-2 font-mono text-xs text-slate-200">{exec.id}</td>
                            <td className="px-4 py-2 font-mono text-xs text-slate-300">
                              {exec.task_title || exec.task_id}
                            </td>
                            <td className="px-4 py-2 text-slate-200 text-xs">{formatSteerInfo(exec)}</td>
                            <td className="px-4 py-3">
                              <span className={`text-xs px-2 py-1 rounded border ${getStatusTone(exec.status)}`}>
                                {exec.status}
                              </span>
                            </td>
                            <td className="px-4 py-3 text-slate-200">{formatDateTime(exec.start_time)}</td>
                            <td className="px-4 py-3 text-slate-200">
                              {exec.end_time ? (
                                formatExecutionDuration(exec)
                              ) : (
                                <span className="inline-flex items-center gap-1 text-amber-200">
                                  <Clock3 className="h-3 w-3" />
                                  In progress
                                </span>
                              )}
                            </td>
                            <td className="px-4 py-3 text-slate-200">
                              {exec.exit_reason ?? (exec.exit_code !== undefined ? exec.exit_code : '—')}
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
              </div>

              <ExecutionDetailCard
                className="h-full"
                execution={selectedExecution}
                promptText={selectedPromptText}
                outputText={selectedOutputText}
                isLoadingPrompt={isFetchingPrompt}
                isLoadingOutput={isFetchingSelectedOutput}
              />
            </div>
          </TabsContent>

          <TabsContent value="performance" className="flex-1 flex-col space-y-4">
            <div className="flex flex-wrap items-center gap-3">
              <Select value={profileFilter} onValueChange={(value) => setProfileFilter(value)}>
                <SelectTrigger className="w-56">
                  <SelectValue placeholder="Profile" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All profiles</SelectItem>
                  {(profiles ?? []).map((profile) => (
                    <SelectItem key={profile.id} value={profile.id}>
                      {profile.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>

              <Input
                placeholder="Filter by scenario..."
                value={scenarioFilter}
                onChange={(e) => setScenarioFilter(e.target.value)}
                className="w-full md:w-64"
              />

              <div className="text-xs text-slate-500">
                {filteredPerformance.length} {filteredPerformance.length === 1 ? 'run' : 'runs'}
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
              <div className="rounded-lg border border-white/5 bg-slate-900/70 p-4">
                <div className="text-xs uppercase text-slate-400 mb-1">Executions</div>
                <div className="text-xl font-semibold text-white">{filteredPerformance.length}</div>
              </div>
              <div className="rounded-lg border border-white/5 bg-slate-900/70 p-4">
                <div className="text-xs uppercase text-slate-400 mb-1">Avg improvement</div>
                <div className="text-xl font-semibold text-emerald-300">
                  {averageImprovement >= 0 ? '+' : ''}
                  {averageImprovement.toFixed(1)}%
                </div>
              </div>
              <div className="rounded-lg border border-white/5 bg-slate-900/70 p-4">
                <div className="text-xs uppercase text-slate-400 mb-1">Avg rating</div>
                <div className="text-xl font-semibold text-amber-200">
                  {averageRating !== null ? averageRating.toFixed(1) : '—'}
                  <span className="text-sm text-slate-500 ml-1">/5</span>
                </div>
              </div>
            </div>

            <div className="flex-1 grid grid-cols-1 lg:grid-cols-5 gap-4 overflow-hidden">
              <div className="lg:col-span-2 rounded-lg border border-white/5 bg-slate-900/70 overflow-hidden min-h-[260px]">
                <div className="max-h-[55vh] overflow-y-auto divide-y divide-white/5">
                  {isLoadingPerformance && filteredPerformance.length === 0 && (
                    <div className="p-10 text-center text-slate-500">Loading performance history...</div>
                  )}
                  {!isLoadingPerformance && filteredPerformance.length === 0 && (
                    <div className="p-10 text-center text-slate-500">No performance runs yet</div>
                  )}
                  {filteredPerformance.map((perf) => {
                    const improvement = computeImprovement(perf);
                    const durationText = formatDurationMs(perf.total_duration);
                    const rating = perf.user_feedback?.rating;
                    const perfId = String(perf.execution_id ?? (perf as any).id ?? '');
                    return (
                      <button
                        key={perfId || perf.scenario_name || `perf-${perf.executed_at}`}
                        onClick={() => setSelectedPerformanceId(perfId)}
                        className={`w-full text-left p-4 transition-colors border-l-4 ${
                          selectedPerformanceId === perfId
                            ? 'bg-white/10 border-emerald-400/70'
                            : 'hover:bg-white/5 border-transparent'
                        }`}
                      >
                        <div className="flex items-center justify-between gap-2">
                          <div>
                            <div className="text-sm font-semibold text-white">
                              {perf.scenario_name || 'Unknown scenario'}
                            </div>
                            <div className="text-xs text-slate-300">
                              {profileNameMap[perf.profile_id] || 'Unknown profile'}
                            </div>
                          </div>
                          <div className={`text-sm font-semibold ${improvement >= 0 ? 'text-emerald-300' : 'text-rose-300'}`}>
                            {improvement >= 0 ? '+' : ''}
                            {improvement.toFixed(1)}%
                          </div>
                        </div>
                        <div className="mt-2 flex flex-wrap items-center gap-3 text-xs text-slate-200">
                          <span>{formatDateTime(perf.executed_at)}</span>
                          <span>•</span>
                          <span>{perf.total_iterations} iterations</span>
                          <span>•</span>
                          <span>{durationText}</span>
                          {rating ? (
                            <>
                              <span>•</span>
                              <span className="text-amber-200">{'★'.repeat(rating)}{'☆'.repeat(5 - rating)}</span>
                            </>
                          ) : null}
                        </div>
                      </button>
                    );
                  })}
                </div>
              </div>

              <div className="lg:col-span-3 rounded-lg border border-white/5 bg-slate-900/70 p-4 overflow-y-auto min-h-[260px]">
                {!selectedPerformance ? (
                  <div className="h-full flex items-center justify-center text-slate-500 text-sm">
                    {isLoadingPerformance
                      ? 'Loading performance runs...'
                      : performanceHistory.length === 0
                        ? 'No performance runs found'
                        : 'Select a run to see details'}
                  </div>
                ) : (
                  <div className="space-y-4">
                    <div className="flex flex-wrap items-start justify-between gap-3">
                      <div>
                        <div className="text-sm uppercase text-slate-400">Scenario</div>
                        <div className="text-lg font-semibold text-white">{selectedPerformance.scenario_name}</div>
                        <div className="text-sm text-slate-300">
                          Profile: {profileNameMap[selectedPerformance.profile_id] || selectedPerformance.profile_id}
                        </div>
                        <div className="text-sm text-slate-300">
                          Executed {formatDateTime(selectedPerformance.executed_at)}
                        </div>
                      </div>
                      <div className="text-right">
                        <div className="text-xs uppercase text-slate-400">Iterations</div>
                        <div className="text-xl font-semibold text-white">{selectedPerformance.total_iterations}</div>
                        <div className="text-xs uppercase text-slate-400 mt-2">Duration</div>
                        <div className="text-lg font-semibold text-white">
                          {formatDurationMs(selectedPerformance.total_duration)}
                        </div>
                      </div>
                    </div>

                    <div className="rounded-lg border border-white/5 bg-slate-800/60 overflow-hidden">
                      <div className="grid grid-cols-3 text-xs text-slate-300 bg-white/5">
                        <div className="px-3 py-2">Metric</div>
                        <div className="px-3 py-2">Start</div>
                        <div className="px-3 py-2">End</div>
                      </div>
                      <div className="divide-y divide-white/5 text-sm">
                        {['operational_targets_percentage', 'operational_targets_passing', 'build_status'].map((key) => {
                          const start = (selectedPerformance.start_metrics as any)?.[key];
                          const end = (selectedPerformance.end_metrics as any)?.[key];
                          const label =
                            key === 'operational_targets_percentage'
                              ? 'Operational targets %'
                              : key === 'operational_targets_passing'
                                ? 'Targets passing'
                                : 'Build status';
                          return (
                            <div key={key} className="grid grid-cols-3 px-3 py-2">
                              <div className="text-slate-200">{label}</div>
                              <div className="text-slate-300">{start ?? '—'}</div>
                              <div className="text-slate-300">{end ?? '—'}</div>
                            </div>
                          );
                        })}
                      </div>
                    </div>

                    <div className="space-y-2">
                      <div className="text-sm font-semibold text-white">Phase breakdown</div>
                      {selectedPerformance.phase_breakdown.length === 0 ? (
                        <div className="text-sm text-slate-400">No phase data recorded</div>
                      ) : (
                        <div className="space-y-2">
                          {selectedPerformance.phase_breakdown.map((phase, idx) => {
                            const effectiveness = Math.min(100, Math.max(0, Math.round((phase.effectiveness || 0) * 100)));
                            const improvements = Object.entries(phase.metric_deltas || {})
                              .map(([metric, delta]) => `${metric}: ${delta > 0 ? '+' : ''}${delta.toFixed(1)}`)
                              .join(', ');

                            return (
                              <div key={`${phase.mode}-${idx}`} className="border border-white/5 rounded-md p-3 bg-slate-800/60">
                                <div className="flex items-center justify-between gap-3">
                                  <div className="flex items-center gap-3">
                                    <span className="text-xs text-slate-400">Phase {idx + 1}</span>
                                    <span className="px-2 py-1 text-xs rounded border border-white/10 bg-white/5">
                                      {phase.mode}
                                    </span>
                                  </div>
                                  <div className="text-xs text-slate-300">
                                    {phase.iterations} iterations • {formatDurationMs(phase.duration)}
                                  </div>
                                </div>
                                <div className="mt-2 h-2 w-full rounded-full bg-white/5">
                                  <div
                                    className="h-2 rounded-full bg-emerald-400"
                                    style={{ width: `${effectiveness}%` }}
                                  />
                                </div>
                                <div className="mt-2 text-xs text-slate-400">
                                  Effectiveness {effectiveness}% {improvements ? `• ${improvements}` : ''}
                                </div>
                              </div>
                            );
                          })}
                        </div>
                      )}
                    </div>
                  </div>
                )}
              </div>
            </div>
          </TabsContent>
        </Tabs>
      </DialogContent>
    </Dialog>
  );
}
