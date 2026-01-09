import { useState, useEffect, useCallback } from "react";
import { fetchLinkPreview, type LinkPreviewData } from "../../../lib/api";

export type LinkPreview = LinkPreviewData;

interface UseLinkPreviewReturn {
  preview: LinkPreview | null;
  isLoading: boolean;
  error: string | null;
  fetch: () => void;
}

// In-memory cache for link previews
const previewCache = new Map<string, LinkPreview | null>();
const pendingFetches = new Map<string, Promise<LinkPreview | null>>();

/**
 * Hook for fetching and caching link preview metadata.
 * Only fetches when `fetch` is called (lazy loading for hover).
 */
export function useLinkPreview(url: string): UseLinkPreviewReturn {
  const [preview, setPreview] = useState<LinkPreview | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Check cache on mount
  useEffect(() => {
    if (previewCache.has(url)) {
      setPreview(previewCache.get(url) || null);
    }
  }, [url]);

  const fetchPreviewCallback = useCallback(async () => {
    // Already cached
    if (previewCache.has(url)) {
      setPreview(previewCache.get(url) || null);
      return;
    }

    // Already fetching
    if (pendingFetches.has(url)) {
      try {
        const result = await pendingFetches.get(url);
        setPreview(result ?? null);
      } catch {
        setError("Failed to fetch preview");
      }
      return;
    }

    setIsLoading(true);
    setError(null);

    const fetchPromise = fetchLinkPreview(url);
    pendingFetches.set(url, fetchPromise);

    try {
      const result = await fetchPromise;
      previewCache.set(url, result);
      setPreview(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to fetch preview");
      previewCache.set(url, null); // Cache failures too
    } finally {
      setIsLoading(false);
      pendingFetches.delete(url);
    }
  }, [url]);

  return {
    preview,
    isLoading,
    error,
    fetch: fetchPreviewCallback,
  };
}
