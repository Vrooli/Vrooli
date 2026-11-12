import { useOverview } from "../api/activity";

/** The research home's absolute path, from the overview every screen shares. */
export function useResearchHome(): string | undefined {
  return useOverview().data?.researchHome || undefined;
}
