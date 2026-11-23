import type { QueryClient } from "@tanstack/react-query";
import { queryKeys } from "./hooks-query-keys";

/** Invalidate the live Git views that can change after a branch/worktree edit. */
export function invalidateBranchWorkspace(queryClient: QueryClient, repoId?: string | null) {
  void queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus(repoId) });
  void queryClient.invalidateQueries({ queryKey: queryKeys.branches(repoId) });
  void queryClient.invalidateQueries({ queryKey: queryKeys.syncStatus(repoId) });
}

/** Reconcile commit-backed views after Git creates a new commit. */
export function invalidateCommitEffects(queryClient: QueryClient, repoId?: string | null) {
  void queryClient.invalidateQueries({ queryKey: queryKeys.repoHistoryFamily(repoId) });
  void queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus(repoId) });
  void queryClient.invalidateQueries({ queryKey: queryKeys.syncStatus(repoId) });
  void queryClient.invalidateQueries({ queryKey: queryKeys.approvedChanges(repoId) });
  void queryClient.invalidateQueries({ queryKey: queryKeys.provenance(repoId) });
}

/** Diffs have separate cache entries for each stage/untracked representation. */
export function invalidateFileDiffVariants(queryClient: QueryClient, repoId: string | null | undefined, paths: string[]) {
  for (const path of paths) {
    for (const [staged, untracked] of [[false, false], [false, true], [true, false]] as const) {
      void queryClient.invalidateQueries({
        queryKey: queryKeys.diff(path, staged, untracked, undefined, "diff", false, repoId),
      });
    }
  }
}

/** Refresh the user-visible repository workspace without changing its filters. */
export function invalidateManualRefresh(queryClient: QueryClient, repoId?: string | null, selectedFile?: string | null) {
  void queryClient.invalidateQueries({ queryKey: queryKeys.health });
  void queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus(repoId) });
  void queryClient.invalidateQueries({ queryKey: queryKeys.repoHistoryFamily(repoId) });
  void queryClient.invalidateQueries({ queryKey: queryKeys.syncStatus(repoId) });
  void queryClient.invalidateQueries({ queryKey: queryKeys.approvedChanges(repoId) });
  void queryClient.invalidateQueries({ queryKey: queryKeys.branches(repoId) });
  if (selectedFile) {
    void queryClient.invalidateQueries({ queryKey: ["repo", "diff", repoId ?? "default"] });
  }
}

export function invalidateRepositoryRegistry(queryClient: QueryClient) {
  void queryClient.invalidateQueries({ queryKey: queryKeys.repos });
  void queryClient.invalidateQueries({ queryKey: queryKeys.activeRepo });
}

export function invalidateCredentials(queryClient: QueryClient, repoId?: string | null) {
  void queryClient.invalidateQueries({ queryKey: queryKeys.credentials(repoId) });
}
