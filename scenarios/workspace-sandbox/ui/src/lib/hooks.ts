import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  fetchHealth,
  fetchDriverInfo,
  fetchDriverOptions,
  selectDriver,
  listSandboxes,
  getSandbox,
  createSandbox,
  deleteSandbox,
  stopSandbox,
  startSandbox,
  getDiff,
  approveSandbox,
  rejectSandbox,
  discardFiles,
  execCommand,
  startProcess,
  listProcesses,
  killProcess,
  killAllProcesses,
  getProcessLogs,
  listLogs,
  type ListFilter,
  type CreateRequest,
  type ApprovalRequest,
  type ExecRequest,
  type StartProcessRequest,
} from "./api";

// Query keys for cache management
export const queryKeys = {
  health: ["health"] as const,
  driver: ["driver"] as const,
  driverOptions: ["driverOptions"] as const,
  sandboxes: (filter?: ListFilter) => ["sandboxes", filter] as const,
  sandbox: (id: string) => ["sandbox", id] as const,
  diff: (id: string) => ["diff", id] as const,
  processes: (sandboxId: string) => ["processes", sandboxId] as const,
  logs: (sandboxId: string) => ["logs", sandboxId] as const,
  processLog: (sandboxId: string, pid: number) => ["processLog", sandboxId, pid] as const,
};

// Health check
export function useHealth() {
  return useQuery({
    queryKey: queryKeys.health,
    queryFn: fetchHealth,
    refetchInterval: 30000, // Refetch every 30 seconds
  });
}

// Driver info
export function useDriverInfo() {
  return useQuery({
    queryKey: queryKeys.driver,
    queryFn: fetchDriverInfo,
  });
}

// Driver options (for Settings dialog)
export function useDriverOptions() {
  return useQuery({
    queryKey: queryKeys.driverOptions,
    queryFn: fetchDriverOptions,
  });
}

// Select driver mutation
export function useSelectDriver() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (driverId: string) => selectDriver(driverId),
    onSuccess: () => {
      // Refresh driver options to show updated state
      queryClient.invalidateQueries({ queryKey: queryKeys.driverOptions });
    },
  });
}

// List sandboxes
export function useSandboxes(filter?: ListFilter) {
  return useQuery({
    queryKey: queryKeys.sandboxes(filter),
    queryFn: () => listSandboxes(filter),
    refetchInterval: 10000, // Refetch every 10 seconds for live updates
  });
}

// Get single sandbox
export function useSandbox(id: string | undefined) {
  return useQuery({
    queryKey: queryKeys.sandbox(id || ""),
    queryFn: () => getSandbox(id!),
    enabled: !!id,
  });
}

// Get diff for sandbox
export function useDiff(id: string | undefined) {
  return useQuery({
    queryKey: queryKeys.diff(id || ""),
    queryFn: () => getDiff(id!),
    enabled: !!id,
  });
}

// Create sandbox mutation
export function useCreateSandbox() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (req: CreateRequest) => createSandbox(req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["sandboxes"] });
    },
  });
}

// Delete sandbox mutation
export function useDeleteSandbox() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => deleteSandbox(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["sandboxes"] });
    },
  });
}

// Stop sandbox mutation
export function useStopSandbox() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => stopSandbox(id),
    onSuccess: (sandbox) => {
      queryClient.setQueryData(queryKeys.sandbox(sandbox.id), sandbox);
      queryClient.invalidateQueries({ queryKey: ["sandboxes"] });
    },
  });
}

// Start sandbox mutation (remount a stopped sandbox)
export function useStartSandbox() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => startSandbox(id),
    onSuccess: (sandbox) => {
      queryClient.setQueryData(queryKeys.sandbox(sandbox.id), sandbox);
      queryClient.invalidateQueries({ queryKey: ["sandboxes"] });
    },
  });
}

// Approve sandbox mutation
export function useApproveSandbox() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      id,
      options,
    }: {
      id: string;
      options?: Partial<ApprovalRequest>;
    }) => approveSandbox(id, options),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.sandbox(id) });
      queryClient.invalidateQueries({ queryKey: ["sandboxes"] });
    },
  });
}

// Reject sandbox mutation
export function useRejectSandbox() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, actor }: { id: string; actor?: string }) =>
      rejectSandbox(id, actor),
    onSuccess: (sandbox) => {
      queryClient.setQueryData(queryKeys.sandbox(sandbox.id), sandbox);
      queryClient.invalidateQueries({ queryKey: ["sandboxes"] });
    },
  });
}

// Discard specific files mutation
export function useDiscardFiles() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      sandboxId,
      fileIds,
      filePaths,
      actor,
    }: {
      sandboxId: string;
      fileIds?: string[];
      filePaths?: string[];
      actor?: string;
    }) => discardFiles(sandboxId, { fileIds, filePaths, actor }),
    onSuccess: (_, { sandboxId }) => {
      // Refresh the diff to show remaining files
      queryClient.invalidateQueries({ queryKey: queryKeys.diff(sandboxId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.sandbox(sandboxId) });
    },
  });
}

// --- Process Execution Hooks ---

// Execute a command synchronously
export function useExecCommand() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      sandboxId,
      request,
    }: {
      sandboxId: string;
      request: ExecRequest;
    }) => execCommand(sandboxId, request),
    onSuccess: (_, { sandboxId }) => {
      // Refresh diff after exec in case files were modified
      queryClient.invalidateQueries({ queryKey: queryKeys.diff(sandboxId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.sandbox(sandboxId) });
    },
  });
}

// Start a background process
export function useStartProcess() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      sandboxId,
      request,
    }: {
      sandboxId: string;
      request: StartProcessRequest;
    }) => startProcess(sandboxId, request),
    onSuccess: (_, { sandboxId }) => {
      // Refresh process list
      queryClient.invalidateQueries({ queryKey: queryKeys.processes(sandboxId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.logs(sandboxId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.sandbox(sandboxId) });
    },
  });
}

// List processes in a sandbox
export function useProcesses(sandboxId: string | undefined, runningOnly?: boolean) {
  return useQuery({
    queryKey: [...queryKeys.processes(sandboxId || ""), runningOnly] as const,
    queryFn: () => listProcesses(sandboxId!, runningOnly),
    enabled: !!sandboxId,
    refetchInterval: 5000, // Refresh every 5 seconds
  });
}

// Kill a process
export function useKillProcess() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ sandboxId, pid }: { sandboxId: string; pid: number }) =>
      killProcess(sandboxId, pid),
    onSuccess: (_, { sandboxId }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.processes(sandboxId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.sandbox(sandboxId) });
    },
  });
}

// Kill all processes in a sandbox
export function useKillAllProcesses() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (sandboxId: string) => killAllProcesses(sandboxId),
    onSuccess: (_, sandboxId) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.processes(sandboxId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.sandbox(sandboxId) });
    },
  });
}

// Get process logs
export function useProcessLogs(
  sandboxId: string | undefined,
  pid: number | undefined,
  options?: { tail?: number; offset?: number }
) {
  return useQuery({
    queryKey: [...queryKeys.processLog(sandboxId || "", pid || 0), options] as const,
    queryFn: () => getProcessLogs(sandboxId!, pid!, options),
    enabled: !!sandboxId && !!pid,
    refetchInterval: 2000, // Refresh every 2 seconds for live log updates
  });
}

// List all logs for a sandbox
export function useLogs(sandboxId: string | undefined) {
  return useQuery({
    queryKey: queryKeys.logs(sandboxId || ""),
    queryFn: () => listLogs(sandboxId!),
    enabled: !!sandboxId,
    refetchInterval: 5000,
  });
}
