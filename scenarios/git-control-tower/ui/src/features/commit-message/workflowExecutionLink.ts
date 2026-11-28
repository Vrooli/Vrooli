const AGENT_MANAGER_UI_PORT = 21238;

/** Builds a link to the Agent Manager workflow console on this browser host. */
export function workflowExecutionLink(executionId: string, hostname: string, protocol = "http:") {
  const id = executionId.trim();
  if (!id || !hostname.trim()) return "";
  const url = new URL("/workflows", `${protocol}//${hostname}:${AGENT_MANAGER_UI_PORT}`);
  url.searchParams.set("executionId", id);
  return url.toString();
}
