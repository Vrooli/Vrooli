import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  getLiveState,
  getFiles,
  getFileContent,
  getDrift,
  killProcess,
  restartProcess,
  getHistory,
  getLogs,
  checkDNS,
  controlCaddy,
  getTLSInfo,
  renewTLS,
  type LiveStateResult,
  type FileEntry,
  type DriftReport,
  type KillProcessRequest,
  type RestartRequest,
  type HistoryEvent,
  type LogEntry,
  type CaddyAction,
  type DNSCheckResponse,
  type TLSInfoResponse,
} from "../lib/api";

/**
 * Hook to fetch live state from VPS.
 * Returns processes, ports, system info, and caddy status.
 */
export function useLiveState(deploymentId: string | null) {
  return useQuery({
    queryKey: ["liveState", deploymentId],
    queryFn: async () => {
      if (!deploymentId) return null;
      const res = await getLiveState(deploymentId);
      return res.result;
    },
    enabled: !!deploymentId,
    staleTime: 30000, // Consider data stale after 30s
    refetchOnWindowFocus: false, // Don't auto-refetch on focus
  });
}

/**
 * Hook to list files in a directory on VPS.
 */
export function useFiles(deploymentId: string | null, path?: string) {
  return useQuery({
    queryKey: ["files", deploymentId, path],
    queryFn: async () => {
      if (!deploymentId) return null;
      const res = await getFiles(deploymentId, path);
      return { path: res.path, entries: res.entries };
    },
    enabled: !!deploymentId,
    staleTime: 60000, // Files don't change as often
  });
}

/**
 * Hook to read file content from VPS.
 */
export function useFileContent(deploymentId: string | null, path: string | null) {
  return useQuery({
    queryKey: ["fileContent", deploymentId, path],
    queryFn: async () => {
      if (!deploymentId || !path) return null;
      const res = await getFileContent(deploymentId, path);
      return {
        path: res.path,
        content: res.content,
        sizeBytes: res.size_bytes,
        truncated: res.truncated,
      };
    },
    enabled: !!deploymentId && !!path,
    staleTime: 60000,
  });
}

/**
 * Hook to get drift report comparing manifest vs actual state.
 */
export function useDrift(deploymentId: string | null) {
  return useQuery({
    queryKey: ["drift", deploymentId],
    queryFn: async () => {
      if (!deploymentId) return null;
      const res = await getDrift(deploymentId);
      return res.result;
    },
    enabled: !!deploymentId,
    staleTime: 30000,
  });
}

/**
 * Hook to kill a process on VPS.
 */
export function useKillProcess(deploymentId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (request: KillProcessRequest) => {
      return killProcess(deploymentId, request);
    },
    onSuccess: () => {
      // Invalidate live state to refresh process list
      queryClient.invalidateQueries({ queryKey: ["liveState", deploymentId] });
      queryClient.invalidateQueries({ queryKey: ["drift", deploymentId] });
    },
  });
}

/**
 * Hook to restart a scenario or resource on VPS.
 */
export function useRestartProcess(deploymentId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (request: RestartRequest) => {
      return restartProcess(deploymentId, request);
    },
    onSuccess: () => {
      // Invalidate live state and drift to refresh
      queryClient.invalidateQueries({ queryKey: ["liveState", deploymentId] });
      queryClient.invalidateQueries({ queryKey: ["drift", deploymentId] });
    },
  });
}

/**
 * Helper to format uptime from seconds.
 */
export function formatUptime(seconds: number): string {
  if (seconds < 60) {
    return `${seconds}s`;
  }
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) {
    return `${minutes}m`;
  }
  const hours = Math.floor(minutes / 60);
  if (hours < 24) {
    const remainingMinutes = minutes % 60;
    return `${hours}h ${remainingMinutes}m`;
  }
  const days = Math.floor(hours / 24);
  const remainingHours = hours % 24;
  return `${days}d ${remainingHours}h`;
}

/**
 * Helper to format bytes to human readable.
 */
export function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
}

/**
 * Helper to format MB to human readable.
 */
export function formatMB(mb: number): string {
  if (mb < 1024) return `${mb} MB`;
  return `${(mb / 1024).toFixed(1)} GB`;
}

/**
 * Helper to get time since timestamp.
 */
export function getTimeSince(timestamp: string): string {
  const date = new Date(timestamp);
  const now = new Date();
  const seconds = Math.floor((now.getTime() - date.getTime()) / 1000);

  if (seconds < 5) return "just now";
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

/**
 * Hook to fetch deployment history timeline.
 */
export function useHistory(deploymentId: string | null) {
  return useQuery({
    queryKey: ["history", deploymentId],
    queryFn: async () => {
      if (!deploymentId) return null;
      const res = await getHistory(deploymentId);
      return res.history;
    },
    enabled: !!deploymentId,
    staleTime: 30000,
  });
}

/**
 * Hook to fetch aggregated logs from VPS.
 */
export function useLogs(
  deploymentId: string | null,
  options: { source?: string; level?: string; tail?: number; search?: string } = {}
) {
  return useQuery({
    queryKey: ["logs", deploymentId, options.source, options.level, options.tail, options.search],
    queryFn: async () => {
      if (!deploymentId) return null;
      const res = await getLogs(deploymentId, options);
      return {
        logs: res.logs,
        total: res.total,
        filtered: res.filtered,
        sources: res.sources,
      };
    },
    enabled: !!deploymentId,
    staleTime: 15000, // Logs change frequently
    refetchOnWindowFocus: false,
  });
}

/**
 * Helper to format duration in milliseconds.
 */
export function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  const seconds = Math.floor(ms / 1000);
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ${seconds % 60}s`;
  const hours = Math.floor(minutes / 60);
  return `${hours}h ${minutes % 60}m`;
}

/**
 * Helper to get event type display info.
 */
export function getEventTypeInfo(type: string): { label: string; color: string; icon: string } {
  const typeMap: Record<string, { label: string; color: string; icon: string }> = {
    deployment_created: { label: "Created", color: "blue", icon: "plus" },
    bundle_built: { label: "Bundle Built", color: "purple", icon: "package" },
    preflight_started: { label: "Preflight Started", color: "slate", icon: "check-circle" },
    preflight_completed: { label: "Preflight Done", color: "emerald", icon: "check-circle" },
    setup_started: { label: "Setup Started", color: "amber", icon: "settings" },
    setup_completed: { label: "Setup Done", color: "emerald", icon: "check-circle" },
    deploy_started: { label: "Deploy Started", color: "blue", icon: "rocket" },
    deploy_completed: { label: "Deployed", color: "emerald", icon: "check-circle" },
    deploy_failed: { label: "Deploy Failed", color: "red", icon: "x-circle" },
    inspection: { label: "Inspected", color: "slate", icon: "search" },
    stopped: { label: "Stopped", color: "amber", icon: "pause" },
    restarted: { label: "Restarted", color: "blue", icon: "refresh" },
    autoheal_triggered: { label: "Autoheal", color: "purple", icon: "heart" },
  };

  return typeMap[type] || { label: type, color: "slate", icon: "info" };
}

/**
 * Helper to get log level display info.
 */
export function getLogLevelInfo(level: string): { color: string; bg: string } {
  const levelMap: Record<string, { color: string; bg: string }> = {
    ERROR: { color: "text-red-400", bg: "bg-red-500/20" },
    WARN: { color: "text-amber-400", bg: "bg-amber-500/20" },
    INFO: { color: "text-blue-400", bg: "bg-blue-500/20" },
    DEBUG: { color: "text-slate-400", bg: "bg-slate-500/20" },
  };

  return levelMap[level.toUpperCase()] || { color: "text-slate-400", bg: "bg-slate-500/20" };
}

// ============================================================================
// Edge/TLS Hooks (Ground Truth Redesign - Enhancement)
// ============================================================================

/**
 * Hook to check DNS resolution (domain points to VPS).
 */
export function useDNSCheck(deploymentId: string | null) {
  return useQuery({
    queryKey: ["dnsCheck", deploymentId],
    queryFn: async () => {
      if (!deploymentId) return null;
      return checkDNS(deploymentId);
    },
    enabled: !!deploymentId,
    staleTime: 60000, // DNS doesn't change often
  });
}

/**
 * Hook to control Caddy service (start, stop, restart, reload).
 */
export function useCaddyControl(deploymentId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (action: CaddyAction) => {
      return controlCaddy(deploymentId, action);
    },
    onSuccess: () => {
      // Invalidate live state to refresh caddy status
      queryClient.invalidateQueries({ queryKey: ["liveState", deploymentId] });
      queryClient.invalidateQueries({ queryKey: ["drift", deploymentId] });
    },
  });
}

/**
 * Hook to fetch detailed TLS certificate information.
 */
export function useTLSInfo(deploymentId: string | null) {
  return useQuery({
    queryKey: ["tlsInfo", deploymentId],
    queryFn: async () => {
      if (!deploymentId) return null;
      return getTLSInfo(deploymentId);
    },
    enabled: !!deploymentId,
    staleTime: 300000, // TLS info rarely changes (5 min)
  });
}

/**
 * Hook to force TLS certificate renewal.
 */
export function useTLSRenew(deploymentId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      return renewTLS(deploymentId);
    },
    onSuccess: () => {
      // Invalidate TLS info to refresh certificate status
      queryClient.invalidateQueries({ queryKey: ["tlsInfo", deploymentId] });
      queryClient.invalidateQueries({ queryKey: ["liveState", deploymentId] });
    },
  });
}
