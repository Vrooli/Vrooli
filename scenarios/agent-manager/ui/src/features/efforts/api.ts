import { useQuery } from "@tanstack/react-query";
import { agentManager, CALL_DEADLINE } from "../../lib/api";

// Reads the same owner projection used by the CLI and standing supervisor.
// Refresh does not enroll efforts, grant authority, scan disk or launch agents.
export function useEffortBoard(pageToken: string, effortRef = "") {
  return useQuery({
    queryKey: ["effort-board", pageToken, effortRef],
    queryFn: ({ signal }) => agentManager.getEffortBoard({ pageSize: 25, pageToken, ...(effortRef ? { effortRef } : {}) }, { signal, ...CALL_DEADLINE }),
    staleTime: 30_000,
    retry: false,
    refetchOnWindowFocus: false,
  });
}
