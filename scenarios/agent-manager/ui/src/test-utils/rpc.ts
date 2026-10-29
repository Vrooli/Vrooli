// Helpers for tests that stub `fetch` under a Connect call to Agent Manager.

const SERVICE = "agent_manager.v1.AgentManagerService";

type FetchCall = readonly unknown[];

/** The procedure a recorded Connect call invoked, e.g. "PurgeData"; "" for a non-procedure URL. */
export function rpcMethod(call: FetchCall): string {
  const url = String(call[0]);
  const marker = `/${SERVICE}/`;
  const at = url.indexOf(marker);
  return at < 0 ? "" : url.slice(at + marker.length);
}

/** The recorded calls of one procedure, in order. */
export function rpcCalls(calls: readonly FetchCall[], method: string): FetchCall[] {
  return calls.filter((call) => rpcMethod(call) === method);
}

/** The JSON request message of a recorded Connect call. */
export function rpcBody(call: FetchCall): Record<string, unknown> {
  const body = (call[1] as RequestInit | undefined)?.body;
  const text = ArrayBuffer.isView(body) ? Buffer.from(body.buffer, body.byteOffset, body.byteLength).toString("utf8") : String(body ?? "{}");
  return JSON.parse(text) as Record<string, unknown>;
}
