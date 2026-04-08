/**
 * Runtime Config Hook - Reads UI behavior settings from the unified settings API
 *
 * Falls back to compile-time defaults from config/index.ts when settings
 * haven't loaded yet.
 */

import { useQuery } from "@tanstack/react-query";
import { uiBehaviorConfig } from "../config";
import { defaultQueryOptions } from "../lib";
import { settingsService } from "../services";
import { DEFAULT_SETTINGS } from "../services/settings-service";
import type { Settings } from "../types";
import type { DeleteConfirmLevel } from "../types/settings";

export type DeletableEntityType = "backlog" | "initiative" | "capture";

export interface RuntimeConfig {
  searchDebounceMs: number;
  toastDurationMs: number;
  getDeleteConfirmLevel: (entityType: DeletableEntityType) => DeleteConfirmLevel;
}

export function useRuntimeConfig(): RuntimeConfig {
  const { data: settings } = useQuery<Settings, Error>({
    queryKey: ["settings"],
    queryFn: () => settingsService.get(),
    ...defaultQueryOptions,
  });

  const dc = settings?.deleteConfirmation ?? DEFAULT_SETTINGS.deleteConfirmation;

  return {
    searchDebounceMs: settings?.searchDebounceMs ?? uiBehaviorConfig.searchDebounceMs,
    toastDurationMs: settings?.toastDurationMs ?? uiBehaviorConfig.toastDurationMs,
    getDeleteConfirmLevel: (entityType: DeletableEntityType) => dc[entityType],
  };
}
