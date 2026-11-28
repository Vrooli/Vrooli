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
  const changedPaths = new Set(paths);
  if (changedPaths.size === 0) return;

  // Match all cached diff variants in one cache pass. Staging a large batch
  // should not perform three separate cache scans per file.
  void queryClient.invalidateQueries({
    queryKey: queryKeys.diffFamily(repoId),
    predicate: query => typeof query.queryKey[3] === "string" && changedPaths.has(query.queryKey[3]),
  });
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
    void queryClient.invalidateQueries({ queryKey: queryKeys.diffFamily(repoId) });
  }
}

export function invalidateRepositoryRegistry(queryClient: QueryClient) {
  void queryClient.invalidateQueries({ queryKey: queryKeys.repos });
  void queryClient.invalidateQueries({ queryKey: queryKeys.activeRepo });
}

export function invalidateCredentials(queryClient: QueryClient, repoId?: string | null) {
  void queryClient.invalidateQueries({ queryKey: queryKeys.credentials(repoId) });
}
