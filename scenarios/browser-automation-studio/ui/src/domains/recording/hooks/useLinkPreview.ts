/**
 * useLinkPreview Hook
 *
 * Fetches and caches OpenGraph metadata for URLs.
 * Uses both in-memory and localStorage caching for persistence.
 */

import { useState, useEffect, useCallback, useMemo } from 'react';
import { getApiBase } from '@/config';

/** Link preview data from the API */
export interface LinkPreviewData {
  title?: string;
  description?: string;
  image?: string;
  favicon?: string;
  site_name?: string;
}

/** Cached preview entry with timestamp */
interface CachedPreview {
  data: LinkPreviewData | null;
  fetchedAt: number;
}

const STORAGE_KEY = 'browser-automation-studio:link-previews';
const CACHE_TTL_MS = 7 * 24 * 60 * 60 * 1000; // 7 days

// In-memory cache for faster access
const memoryCache = new Map<string, CachedPreview>();
const pendingFetches = new Map<string, Promise<LinkPreviewData | null>>();

/**
 * Load cached previews from localStorage
 */
function loadCacheFromStorage(): Map<string, CachedPreview> {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored) {
      const parsed = JSON.parse(stored) as Record<string, CachedPreview>;
      const now = Date.now();
      const cache = new Map<string, CachedPreview>();

      // Filter out expired entries
      for (const [url, entry] of Object.entries(parsed)) {
        if (now - entry.fetchedAt < CACHE_TTL_MS) {
          cache.set(url, entry);
        }
      }

      return cache;
    }
  } catch {
    // Invalid or unavailable
  }
  return new Map();
}

/**
 * Save cache to localStorage
 */
function saveCacheToStorage(cache: Map<string, CachedPreview>): void {
  try {
    const obj: Record<string, CachedPreview> = {};
    for (const [url, entry] of cache.entries()) {
      obj[url] = entry;
    }
    localStorage.setItem(STORAGE_KEY, JSON.stringify(obj));
  } catch {
    // localStorage unavailable
  }
}

// Initialize memory cache from localStorage
if (typeof window !== 'undefined') {
  const stored = loadCacheFromStorage();
  for (const [url, entry] of stored.entries()) {
    memoryCache.set(url, entry);
  }
}

/**
 * Fetch a single link preview from the API
 */
async function fetchLinkPreview(url: string): Promise<LinkPreviewData | null> {
  const apiBase = getApiBase();
  const apiUrl = `${apiBase}/link-preview?url=${encodeURIComponent(url)}`;

  const res = await fetch(apiUrl, {
    headers: { 'Content-Type': 'application/json' },
  });

  if (res.status === 204) {
    // No content - preview unavailable
    return null;
  }

  if (!res.ok) {
    throw new Error(`Failed to fetch link preview: ${res.status}`);
  }

  return res.json();
}

/**
 * Get a cached preview (from memory or localStorage)
 */
function getCachedPreview(url: string): LinkPreviewData | null | undefined {
  const cached = memoryCache.get(url);
  if (!cached) {
    return undefined; // Not cached
  }

  const now = Date.now();
  if (now - cached.fetchedAt >= CACHE_TTL_MS) {
    memoryCache.delete(url);
    return undefined; // Expired
  }

  return cached.data;
}

/**
 * Set a cached preview
 */
function setCachedPreview(url: string, data: LinkPreviewData | null): void {
  const entry: CachedPreview = {
    data,
    fetchedAt: Date.now(),
  };
  memoryCache.set(url, entry);
  saveCacheToStorage(memoryCache);
}

interface UseLinkPreviewReturn {
  preview: LinkPreviewData | null;
  isLoading: boolean;
  error: string | null;
  fetch: () => void;
}

/**
 * Hook for fetching and caching a single link preview.
 * Only fetches when `fetch` is called (lazy loading).
 */
export function useLinkPreview(url: string): UseLinkPreviewReturn {
  const [preview, setPreview] = useState<LinkPreviewData | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Check cache on mount
  useEffect(() => {
    const cached = getCachedPreview(url);
    if (cached !== undefined) {
      setPreview(cached);
    }
  }, [url]);

  const fetchPreviewCallback = useCallback(async () => {
    // Already cached
    const cached = getCachedPreview(url);
    if (cached !== undefined) {
      setPreview(cached);
      return;
    }

    // Already fetching
    if (pendingFetches.has(url)) {
      try {
        const result = await pendingFetches.get(url);
        setPreview(result ?? null);
      } catch {
        setError('Failed to fetch preview');
      }
      return;
    }

    setIsLoading(true);
    setError(null);

    const fetchPromise = fetchLinkPreview(url);
    pendingFetches.set(url, fetchPromise);

    try {
      const result = await fetchPromise;
      setCachedPreview(url, result);
      setPreview(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch preview');
      setCachedPreview(url, null); // Cache failures too
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

interface UseLinkPreviewsBatchReturn {
  previews: Map<string, LinkPreviewData | null>;
  isLoading: boolean;
  fetchAll: () => void;
}

/**
 * Hook for fetching multiple link previews in batch.
 * Automatically fetches on mount for URLs not already cached.
 */
export function useLinkPreviewsBatch(urls: string[]): UseLinkPreviewsBatchReturn {
  const [previews, setPreviews] = useState<Map<string, LinkPreviewData | null>>(new Map());
  const [isLoading, setIsLoading] = useState(false);

  // Stable reference for URLs to avoid infinite loops
  const urlsKey = useMemo(() => urls.join('|'), [urls]);

  // Initialize with cached previews
  useEffect(() => {
    const initial = new Map<string, LinkPreviewData | null>();
    for (const url of urls) {
      const cached = getCachedPreview(url);
      if (cached !== undefined) {
        initial.set(url, cached);
      }
    }
    setPreviews(initial);
  }, [urlsKey, urls]);

  const fetchAll = useCallback(async () => {
    // Find URLs that need fetching
    const toFetch = urls.filter((url) => getCachedPreview(url) === undefined);

    if (toFetch.length === 0) {
      return;
    }

    setIsLoading(true);

    const results = new Map<string, LinkPreviewData | null>(previews);

    // Fetch in parallel with concurrency limit
    const CONCURRENCY = 3;
    for (let i = 0; i < toFetch.length; i += CONCURRENCY) {
      const batch = toFetch.slice(i, i + CONCURRENCY);
      const fetchPromises = batch.map(async (url) => {
        try {
          // Check if already pending
          if (pendingFetches.has(url)) {
            const result = await pendingFetches.get(url);
            return { url, data: result };
          }

          const fetchPromise = fetchLinkPreview(url);
          pendingFetches.set(url, fetchPromise);

          try {
            const result = await fetchPromise;
            setCachedPreview(url, result);
            return { url, data: result };
          } finally {
            pendingFetches.delete(url);
          }
        } catch {
          setCachedPreview(url, null);
          return { url, data: null };
        }
      });

      const batchResults = await Promise.all(fetchPromises);
      for (const { url, data } of batchResults) {
        results.set(url, data ?? null);
      }
      setPreviews(new Map(results));
    }

    setIsLoading(false);
  }, [urlsKey, urls, previews]);

  // Auto-fetch on mount
  useEffect(() => {
    void fetchAll();
    // Only run on mount/URL change
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [urlsKey]);

  return {
    previews,
    isLoading,
    fetchAll,
  };
}

/**
 * Get domain from URL for display
 */
export function extractDomainFromUrl(url: string): string {
  try {
    const parsed = new URL(url);
    return parsed.hostname.replace(/^www\./, '');
  } catch {
    return url;
  }
}
