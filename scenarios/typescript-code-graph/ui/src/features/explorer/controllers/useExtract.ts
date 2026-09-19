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
  result: (projectPath: string) => [...extractKeys.all(), projectPath] as const,
};

export interface ExtractParams {
  /** Resolved TS project root or tsconfig.json path. */
  readonly projectPath: string;
}

/**
 * Run TypeScriptCodeGraphService.Extract for the given params. The query stays
 * disabled until a non-empty `projectPath` is supplied (i.e. after the user
 * submits the extract bar), so the workbench renders its idle state first.
 */
export function useExtract(params: ExtractParams | null): UseQueryResult<ExtractResponse> {
  const projectPath = params?.projectPath ?? "";
  return useQuery({
    queryKey: extractKeys.result(projectPath),
    queryFn: () => tsCodeGraphClient.extract({ projectPath }),
    enabled: projectPath.length > 0,
  });
}
