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
import type { Settings } from "../types";

export interface RuntimeConfig {
  searchDebounceMs: number;
  toastDurationMs: number;
  confirmDestructiveActions: boolean;
}

export function useRuntimeConfig(): RuntimeConfig {
  const { data: settings } = useQuery<Settings, Error>({
    queryKey: ["settings"],
    queryFn: () => settingsService.get(),
    ...defaultQueryOptions,
  });

  return {
    searchDebounceMs: settings?.searchDebounceMs ?? uiBehaviorConfig.searchDebounceMs,
    toastDurationMs: settings?.toastDurationMs ?? uiBehaviorConfig.toastDurationMs,
    confirmDestructiveActions: settings?.confirmDestructiveActions ?? uiBehaviorConfig.confirmDestructiveActions,
  };
}
