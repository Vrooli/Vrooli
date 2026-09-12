import { useQuery, type UseQueryResult } from "@tanstack/react-query";

import { goCodeGraphClient } from "../../../api/graph";
import type { ExtractResponse } from "../../../api/graph";

/**
 * Stable React Query cache keys for the Extract surface. One key-builder so
 * cache-key drift can't sneak in via inline string assembly. The key folds in
 * both inputs (path + vendor flag) so toggling include-vendor refetches a
 * distinct cache entry rather than clobbering the default-extraction result.
 */
export const extractKeys = {
  all: () => ["extract"] as const,
  result: (modulePath: string, includeVendor: boolean) =>
    [...extractKeys.all(), modulePath, includeVendor] as const,
};

export interface ExtractParams {
  /** Resolved Go module root (scenario-relative or absolute). */
  readonly modulePath: string;
  /** When true, the loader descends into vendor/ and the module cache. */
  readonly includeVendor: boolean;
}

/**
 * Run GoCodeGraphService.Extract for the given params. The query stays
 * disabled until a non-empty `modulePath` is supplied (i.e. after the user
 * submits the extract bar), so the workbench renders its idle state first.
 */
export function useExtract(params: ExtractParams | null): UseQueryResult<ExtractResponse> {
  const modulePath = params?.modulePath ?? "";
  const includeVendor = params?.includeVendor ?? false;
  return useQuery({
    queryKey: extractKeys.result(modulePath, includeVendor),
    queryFn: () =>
      goCodeGraphClient.extract({ modulePath, includeVendor }),
    enabled: modulePath.length > 0,
  });
}
