/**
 * useCaptureContent — fetches text content from the review capture endpoint.
 *
 * Used by CLI output and config diff evidence renderers to load capture file
 * content inline. Backed by React Query with infinite stale time since
 * captures are immutable once created.
 */

import { useQuery } from "@tanstack/react-query";
import { buildApiUrl } from "@vrooli/api-base";
import { API_ENDPOINTS } from "../lib/api-endpoints";

const MAX_DISPLAY_SIZE = 50_000; // 50 KB — truncate beyond this

export interface UseCaptureContentResult {
  content: string | null;
  isLoading: boolean;
  error: string | null;
  isTruncated: boolean;
  /** Direct URL to the full capture file (for "open full output" links). */
  captureUrl: string;
}

export function useCaptureContent(
  backlogKind: string,
  backlogName: string,
  capturePath: string,
): UseCaptureContentResult {
  const captureUrl = buildApiUrl(
    API_ENDPOINTS.reviewCapture(backlogKind, backlogName, capturePath),
    { appendSuffix: true },
  );

  const { data, isLoading, error } = useQuery({
    queryKey: ["capture-content", backlogKind, backlogName, capturePath],
    queryFn: async () => {
      const res = await fetch(captureUrl);
      if (!res.ok) {
        throw new Error(`Failed to load capture: ${res.status} ${res.statusText}`);
      }
      const contentType = res.headers.get("Content-Type") ?? "";
      if (contentType && !contentType.startsWith("text/") && !contentType.includes("json") && !contentType.includes("xml")) {
        return { text: null, binary: true };
      }
      const text = await res.text();
      return { text, binary: false };
    },
    enabled: Boolean(capturePath),
    staleTime: Infinity,
  });

  if (data?.binary) {
    return {
      content: null,
      isLoading: false,
      error: "Cannot display binary content. Open the file directly.",
      isTruncated: false,
      captureUrl,
    };
  }

  const rawText = data?.text ?? null;
  const isTruncated = rawText !== null && rawText.length > MAX_DISPLAY_SIZE;
  const content = isTruncated ? rawText.slice(0, MAX_DISPLAY_SIZE) : rawText;

  return {
    content,
    isLoading,
    error: error ? (error instanceof Error ? error.message : "Failed to load capture") : null,
    isTruncated,
    captureUrl,
  };
}
