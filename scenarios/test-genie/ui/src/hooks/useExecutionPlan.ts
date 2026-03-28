import { useQuery } from "@tanstack/react-query";
import { previewSuiteExecution, type ExecuteSuiteInput, type ExecutionPlanPreview } from "../lib/api";

function normalizeList(values?: string[]): string {
  if (!values || values.length === 0) return "";
  return values.join(",");
}

export function useExecutionPlan(input: ExecuteSuiteInput, enabled = true) {
  const scenarioName = input.scenarioName.trim();

  return useQuery<ExecutionPlanPreview>({
    queryKey: [
      "execution-plan",
      scenarioName,
      input.preset ?? "",
      Boolean(input.failFast),
      normalizeList(input.phases),
      normalizeList(input.skip)
    ],
    queryFn: () =>
      previewSuiteExecution({
        ...input,
        scenarioName
      }),
    enabled: enabled && scenarioName.length > 0,
    staleTime: 30_000
  });
}
