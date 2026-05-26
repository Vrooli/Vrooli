import { useQuery, type UseQueryResult } from "@tanstack/react-query";

import { tsCodeGraphClient } from "../../../api/graph";
import type { ExtractResponse } from "../../../api/graph";

/**
 * Stable React Query cache keys for the Extract surface. One key-builder so
 * cache-key drift can't sneak in via inline string assembly. The key folds in
 * the TS project path; there is no vendor concept here (unlike go-code-graph),
 * so the path alone identifies an extraction.
 */
export const extractKeys = {
  all: () => ["extract"] as const,
  result: (scenarioPath: string) => [...extractKeys.all(), scenarioPath] as const,
};

export interface ExtractParams {
  /** Resolved TS project root (scenario-relative or absolute). */
  readonly scenarioPath: string;
}

/**
 * Run TypeScriptCodeGraphService.Extract for the given params. The query stays
 * disabled until a non-empty `scenarioPath` is supplied (i.e. after the user
 * submits the extract bar), so the workbench renders its idle state first.
 */
export function useExtract(params: ExtractParams | null): UseQueryResult<ExtractResponse> {
  const scenarioPath = params?.scenarioPath ?? "";
  return useQuery({
    queryKey: extractKeys.result(scenarioPath),
    queryFn: () => tsCodeGraphClient.extract({ scenarioPath }),
    enabled: scenarioPath.length > 0,
  });
}
