import { useQuery } from "@tanstack/react-query";
import { fetchRunArtifacts } from "../lib/api";

export function useRunArtifacts(scenario: string, runId?: string) {
  return useQuery({
    queryKey: ["run-artifacts", scenario, runId],
    queryFn: () => runId ? fetchRunArtifacts(scenario, runId) : Promise.reject(new Error("run id is required")),
    enabled: Boolean(scenario && runId),
    staleTime: 30_000
  });
}
