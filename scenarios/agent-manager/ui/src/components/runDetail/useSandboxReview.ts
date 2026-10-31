import { useCallback } from "react";
import { fetchWithDeadline } from "../../lib/api";
import { useApiQuery } from "../../hooks/useApiQuery";

/** Resolves the workspace-sandbox URL via the embedded proxy and opens a sandbox's review page. */
export function useSandboxReview(sandboxId?: string) {
  // workspace-sandbox may not be installed; an unavailable proxy simply leaves the link off.
  const { data } = useApiQuery<string | null>(["workspace-sandbox-url"], async (signal) => {
    const res = await fetchWithDeadline(`/embedded/${encodeURIComponent("workspace-sandbox")}/external-url`, { signal });
    return res.ok ? ((await res.json()) as { url?: string }).url ?? null : null;
  });
  const sandboxUrl = data;

  const openSandboxReview = useCallback(() => {
    if (!sandboxId || !sandboxUrl) return;
    window.open(`${sandboxUrl}?sandbox=${sandboxId}&review=true`, "_blank", "noopener,noreferrer");
  }, [sandboxId, sandboxUrl]);

  return { sandboxUrl, openSandboxReview };
}
