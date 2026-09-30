import { createClient } from "@connectrpc/connect";
import { createScenarioConnectTransport } from "@vrooli/api-base";
import { AgentManagerService } from "@vrooli/proto-types/agent-manager/v1/api/service_pb";
import { useQuery } from "@tanstack/react-query";
import { resolveAgentManagerApiBase } from "../../lib/api";

export const effortBoardClient = createClient(AgentManagerService, createScenarioConnectTransport({ baseUrl: resolveAgentManagerApiBase() }));

// Reads the same owner projection used by the CLI and standing supervisor.
// Refresh does not enroll efforts, grant authority, scan disk or launch agents.
export function useEffortBoard(pageToken: string, effortRef = "") {
  return useQuery({
    queryKey: ["effort-board", pageToken, effortRef],
    queryFn: ({ signal }) => effortBoardClient.getEffortBoard({ pageSize: 25, pageToken, ...(effortRef ? { effortRef } : {}) }, { signal, timeoutMs: 30_000 }),
    staleTime: 30_000,
    retry: false,
    refetchOnWindowFocus: false,
  });
}
