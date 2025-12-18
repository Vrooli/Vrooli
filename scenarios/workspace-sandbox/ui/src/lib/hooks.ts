import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  fetchHealth,
  fetchDriverInfo,
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
  type ListFilter,
  type CreateRequest,
  type ApprovalRequest,
} from "./api";

// Query keys for cache management
export const queryKeys = {
  health: ["health"] as const,
  driver: ["driver"] as const,
  sandboxes: (filter?: ListFilter) => ["sandboxes", filter] as const,
  sandbox: (id: string) => ["sandbox", id] as const,
  diff: (id: string) => ["diff", id] as const,
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
