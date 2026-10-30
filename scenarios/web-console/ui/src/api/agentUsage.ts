import { capabilitiesClient } from "./capabilities";

/** Target-scoped provider reading. The server controls match confidence. */
export async function getAgentUsage(
  request: { targetId: string; agentId: string },
  options: { signal: AbortSignal },
) {
  const value = await capabilitiesClient.agentUsage(request, options);
  return {
    targetId: value.targetId,
    agentId: value.agentId,
    match: value.match,
    reason: value.reason,
    source: value.source,
    observedAt: value.observedAt,
    windows: value.windows.map((window) => ({
      pool: window.pool,
      windowMinutes: Number(window.windowMinutes),
      usedPercent: window.usedPercent,
      resetsAt: window.resetsAt,
    })),
  };
}
