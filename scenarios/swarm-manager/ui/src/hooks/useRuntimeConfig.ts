/**
 * Runtime Config Hook - Reads UI behavior settings from the unified settings API
 *
 * Falls back to compile-time defaults from config/index.ts when settings
 * haven't loaded yet.
 */

import { useQuery } from "@tanstack/react-query";
import { useCallback, useMemo } from "react";
import { uiBehaviorConfig } from "../config";
import { defaultQueryOptions } from "../lib";
import { settingsService } from "../services";
import { DEFAULT_SETTINGS } from "../services/settings-service";
import type { Settings } from "../types";
import type { DeleteConfirmLevel } from "../types/settings";
import { defaultLevelFor, type DeletableEntityType } from "../lib/deletable-entities";

export type { DeletableEntityType };

export interface RuntimeConfig {
  searchDebounceMs: number;
  toastDurationMs: number;
  getDeleteConfirmLevel: (entityType: DeletableEntityType) => DeleteConfirmLevel;
}

export function useRuntimeConfig(): RuntimeConfig {
  const { data: settings } = useQuery<Settings>({
    queryKey: ["settings"],
    queryFn: () => settingsService.get(),
    ...defaultQueryOptions,
  });

  const dc = settings?.deleteConfirmation ?? DEFAULT_SETTINGS.deleteConfirmation;

  // Stabilize the returned callback and object across renders. Without this,
  // every render produces a fresh getDeleteConfirmLevel identity, which
  // propagates through consumers that list it (and objects derived from it) as
  // effect/callback dependencies — turning a single state change into an
  // unbounded re-render loop (e.g. the file-browser header-slot effect).
  const getDeleteConfirmLevel = useCallback(
    (entityType: DeletableEntityType) => dc[entityType] ?? defaultLevelFor(entityType),
    [dc],
  );

  return useMemo(
    () => ({
      searchDebounceMs: settings?.searchDebounceMs ?? uiBehaviorConfig.searchDebounceMs,
      toastDurationMs: settings?.toastDurationMs ?? uiBehaviorConfig.toastDurationMs,
      getDeleteConfirmLevel,
    }),
    [settings?.searchDebounceMs, settings?.toastDurationMs, getDeleteConfirmLevel],
  );
}
