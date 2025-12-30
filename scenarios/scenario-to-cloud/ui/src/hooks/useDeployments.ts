import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  listDeployments,
  getDeployment,
  createDeployment,
  executeDeployment,
  inspectDeployment,
  stopDeployment,
  deleteDeployment,
  type DeploymentSummary,
  type Deployment,
} from "../lib/api";

/**
 * Hook to fetch the list of all deployments.
 * Polls every 10 seconds to catch status updates.
 */
export function useDeployments() {
  return useQuery({
    queryKey: ["deployments"],
    queryFn: async () => {
      const res = await listDeployments();
      return res.deployments;
    },
    refetchInterval: 10000, // Poll every 10s for status updates
  });
}

/**
 * Hook to fetch a single deployment by ID.
 * Polls more frequently (3s) when the deployment is in progress.
 */
export function useDeployment(id: string | null) {
  return useQuery({
    queryKey: ["deployment", id],
    queryFn: async () => {
      if (!id) return null;
      const res = await getDeployment(id);
      return res.deployment;
    },
    enabled: !!id,
    refetchInterval: (query) => {
      const data = query.state.data as Deployment | null | undefined;
      // Poll frequently during active phases
      if (
        data?.status === "setup_running" ||
        data?.status === "deploying" ||
        data?.status === "pending"
      ) {
        return 3000;
      }
      return false; // No polling when stable
    },
  });
}

/**
 * Hook to create a new deployment from a manifest.
 */
export function useCreateDeployment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      manifest,
      name,
    }: {
      manifest: unknown;
      name?: string;
    }) => {
      const res = await createDeployment(manifest, name);
      return res.deployment;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["deployments"] });
    },
  });
}

/**
 * Hook to execute (setup + deploy) a deployment.
 */
export function useExecuteDeployment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const res = await executeDeployment(id);
      return res;
    },
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: ["deployments"] });
      queryClient.invalidateQueries({ queryKey: ["deployment", id] });
    },
  });
}

/**
 * Hook to inspect a deployment (fetch status + logs from VPS).
 */
export function useInspectDeployment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const res = await inspectDeployment(id);
      return res.result;
    },
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: ["deployment", id] });
    },
  });
}

/**
 * Hook to stop a deployment on the VPS.
 */
export function useStopDeployment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      return stopDeployment(id);
    },
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: ["deployments"] });
      queryClient.invalidateQueries({ queryKey: ["deployment", id] });
    },
  });
}

/**
 * Hook to delete a deployment record.
 */
export function useDeleteDeployment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      id,
      stopOnVPS,
      cleanupBundles,
    }: {
      id: string;
      stopOnVPS?: boolean;
      cleanupBundles?: boolean;
    }) => {
      return deleteDeployment(id, { stopOnVPS, cleanupBundles });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["deployments"] });
    },
  });
}

/**
 * Helper to get status display info.
 */
export function getStatusInfo(status: DeploymentSummary["status"]) {
  switch (status) {
    case "pending":
      return { label: "Pending", color: "slate", icon: "clock" };
    case "setup_running":
      return { label: "Setting up...", color: "blue", icon: "loader" };
    case "setup_complete":
      return { label: "Setup complete", color: "blue", icon: "check" };
    case "deploying":
      return { label: "Deploying...", color: "blue", icon: "loader" };
    case "deployed":
      return { label: "Deployed", color: "emerald", icon: "check-circle" };
    case "failed":
      return { label: "Failed", color: "red", icon: "x-circle" };
    case "stopped":
      return { label: "Stopped", color: "amber", icon: "pause" };
    default:
      return { label: status, color: "slate", icon: "help" };
  }
}
