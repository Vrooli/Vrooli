import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { ArchitectureFinding } from "@vrooli/proto-types/architecture/v1/findings_pb";

import { migrationClient } from "../../../api/migration";

/**
 * Stable React Query cache keys for the migration feature. One key-builder
 * per surface so cache-key drift can't sneak in via inline string assembly.
 */
export const migrationKeys = {
  all: () => ["migration"] as const,
  list: (scenario: string) => [...migrationKeys.all(), "list", scenario] as const,
  status: (id: string) => [...migrationKeys.all(), "status", id] as const,
  next: (id: string) => [...migrationKeys.all(), "next", id] as const,
};

export interface UseListMigrationsArgs {
  scenario: string;
  enabled?: boolean;
}

/** List the migrations for one scenario (newest first). */
export function useListMigrations({ scenario, enabled = true }: UseListMigrationsArgs) {
  return useQuery({
    queryKey: migrationKeys.list(scenario),
    queryFn: () => migrationClient.listMigrations({ scenario }),
    enabled: enabled && scenario.length > 0,
  });
}

export interface UseMigrationStatusArgs {
  id: string;
  enabled?: boolean;
}

/** Full status projection (migration + tracked findings + rollups). */
export function useMigrationStatus({ id, enabled = true }: UseMigrationStatusArgs) {
  return useQuery({
    queryKey: migrationKeys.status(id),
    queryFn: () => migrationClient.getMigrationStatus({ migrationId: id }),
    enabled: enabled && id.length > 0,
  });
}

/** Prioritized worklist of open findings (regressions → cycles → severity). */
export function useNextStep({ id, enabled = true }: UseMigrationStatusArgs) {
  return useQuery({
    queryKey: migrationKeys.next(id),
    queryFn: () => migrationClient.nextMigrationStep({ migrationId: id }),
    enabled: enabled && id.length > 0,
  });
}

export interface CreateMigrationArgs {
  name: string;
  findings: ArchitectureFinding[];
}

/** Open a migration for the scenario, ingesting the parsed findings. */
export function useCreateMigration(scenario: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ name, findings }: CreateMigrationArgs) =>
      migrationClient.createMigration({ scenario, name, findings }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: migrationKeys.list(scenario) });
    },
  });
}

/** Invalidate the per-migration caches after a lifecycle mutation. */
function invalidateMigration(queryClient: ReturnType<typeof useQueryClient>, id: string) {
  void queryClient.invalidateQueries({ queryKey: migrationKeys.status(id) });
  void queryClient.invalidateQueries({ queryKey: migrationKeys.next(id) });
}

export interface ResolveFindingArgs {
  stableId: string;
  note?: string;
}

export function useResolveFinding(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ stableId, note }: ResolveFindingArgs) =>
      migrationClient.resolveFinding({ migrationId: id, stableId, note: note ?? "" }),
    onSuccess: () => invalidateMigration(queryClient, id),
  });
}

export function useApplyFinding(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (stableId: string) => migrationClient.applyFinding({ migrationId: id, stableId }),
    onSuccess: () => invalidateMigration(queryClient, id),
  });
}

/** Reconcile a fresh audit photograph against the tracked findings. */
export function useReauditMigration(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (findings: ArchitectureFinding[]) =>
      migrationClient.reauditMigration({ migrationId: id, findings }),
    onSuccess: () => invalidateMigration(queryClient, id),
  });
}

export function useCloseMigration(id: string, scenario: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => migrationClient.closeMigration({ migrationId: id }),
    onSuccess: () => {
      invalidateMigration(queryClient, id);
      void queryClient.invalidateQueries({ queryKey: migrationKeys.list(scenario) });
    },
  });
}
