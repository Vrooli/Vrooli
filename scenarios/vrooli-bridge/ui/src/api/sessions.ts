import { buildApiUrl } from "@vrooli/api-base";

import { authedFetch, decodeApiError, REST_API_BASE } from "./client";

/** Owner-gated kill switch for an interactive Bridge session. */
export async function killSession(sessionId: string): Promise<void> {
  const id = sessionId.trim();
  if (!id) throw new Error("session id is required");
  const response = await authedFetch(
    buildApiUrl(`/channel/session/${encodeURIComponent(id)}`, { baseUrl: REST_API_BASE }),
    { method: "DELETE" },
  );
  if (!response.ok) throw await decodeApiError(response);
}
