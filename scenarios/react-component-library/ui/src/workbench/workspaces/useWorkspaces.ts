import { useQuery, type UseQueryResult } from "@tanstack/react-query";

import {
  getWorkspace,
  listWorkspaces,
  type Workspace,
  type WorkspaceSummary,
} from "../../api/workspaces";

/** Workspace summaries for the Design index and Start's continue section. */
export function useWorkspaces(): UseQueryResult<WorkspaceSummary[]> {
  return useQuery({
    queryKey: ["wb", "workspaces"],
    queryFn: () => listWorkspaces({ limit: 200 }),
    staleTime: 15_000,
    retry: 0,
  });
}

/** One full aggregate for the workspace studio. */
export function useWorkspace(id: string | null): UseQueryResult<Workspace | null> {
  return useQuery({
    queryKey: ["wb", "workspace", id],
    queryFn: () => (id ? getWorkspace(id) : Promise.resolve(null)),
    enabled: Boolean(id),
    staleTime: 0,
    retry: 0,
  });
}
