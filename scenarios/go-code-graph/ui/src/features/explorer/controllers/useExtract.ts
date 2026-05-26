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
  result: (scenarioPath: string, includeVendor: boolean) =>
    [...extractKeys.all(), scenarioPath, includeVendor] as const,
};

export interface ExtractParams {
  /** Resolved Go module root (scenario-relative or absolute). */
  readonly scenarioPath: string;
  /** When true, the loader descends into vendor/ and the module cache. */
  readonly includeVendor: boolean;
}

/**
 * Run GoCodeGraphService.Extract for the given params. The query stays
 * disabled until a non-empty `scenarioPath` is supplied (i.e. after the user
 * submits the extract bar), so the workbench renders its idle state first.
 */
export function useExtract(params: ExtractParams | null): UseQueryResult<ExtractResponse> {
  const scenarioPath = params?.scenarioPath ?? "";
  const includeVendor = params?.includeVendor ?? false;
  return useQuery({
    queryKey: extractKeys.result(scenarioPath, includeVendor),
    queryFn: () =>
      goCodeGraphClient.extract({ scenarioPath, includeVendor }),
    enabled: scenarioPath.length > 0,
  });
}
