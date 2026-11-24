import { useQuery } from "@tanstack/react-query";
import { fetchTargets } from "./host";
import { onboardingTargetCapability } from "./hostCapabilities";

/** Resolve non-local selection against the owner target registry before any
 * target-scoped profile or host read is allowed to run. */
export function useOnboardingTargetCapability(target: string) {
  const needsRegistry = target.trim() !== "" && target.trim() !== "local";
  const targetsQuery = useQuery({
    queryKey: ["onboarding-targets", target.trim()],
    queryFn: fetchTargets,
    enabled: needsRegistry,
    retry: false,
    staleTime: 0,
  });
  const capability = onboardingTargetCapability(target, targetsQuery.data?.targets);
  return {
    capability,
    registryScoped: needsRegistry,
    resolving: needsRegistry && (targetsQuery.isLoading || targetsQuery.isFetching),
    resolutionError: needsRegistry && targetsQuery.isError ? targetsQuery.error : null,
    refreshTarget: targetsQuery.refetch,
  };
}
