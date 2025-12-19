/**
 * useUrlInheritance Hook
 *
 * Provides URL inheritance pattern for nodes that can either specify their own URL
 * or inherit from upstream nodes (typically NavigateNode).
 *
 * Handles:
 * - Local draft state for URL input
 * - Effective URL computation (stored vs inherited)
 * - Commit/reset/keyboard handlers for URL field
 */

import { useState, useEffect, useMemo, useCallback } from 'react';
import type { KeyboardEvent } from 'react';
import { useUpstreamUrl } from './useUpstreamUrl';
import { useNodeData } from './useNodeData';

export interface UseUrlInheritanceResult {
  /** The URL draft value (for controlled input) */
  urlDraft: string;
  /** Set the URL draft value (for controlled input onChange) */
  setUrlDraft: (value: string) => void;
  /** The effective URL (stored URL if set, otherwise inherited from upstream) */
  effectiveUrl: string | null;
  /** Whether this node has a custom URL (not inherited) */
  hasCustomUrl: boolean;
  /** The URL inherited from upstream (if any) */
  upstreamUrl: string | null;
  /** Commit the current draft to node data (for onBlur) */
  commitUrl: () => void;
  /** Reset URL to inherit from upstream */
  resetUrl: () => void;
  /** Keyboard handler for Enter (commit) and Escape (reset draft) */
  handleUrlKeyDown: (event: KeyboardEvent<HTMLInputElement>) => void;
}

/**
 * Hook for managing URL input with inheritance from upstream nodes.
 *
 * @param nodeId - The ID of the node
 * @returns URL state and handlers for the URL input field
 *
 * @example
 * ```tsx
 * const {
 *   urlDraft, setUrlDraft,
 *   effectiveUrl, hasCustomUrl, upstreamUrl,
 *   commitUrl, resetUrl, handleUrlKeyDown
 * } = useUrlInheritance(nodeId);
 *
 * <input
 *   value={urlDraft}
 *   onChange={(e) => setUrlDraft(e.target.value)}
 *   onBlur={commitUrl}
 *   onKeyDown={handleUrlKeyDown}
 *   placeholder={upstreamUrl ?? 'https://example.com'}
 * />
 * {hasCustomUrl && <button onClick={resetUrl}>Reset</button>}
 * ```
 */
export function useUrlInheritance(nodeId: string): UseUrlInheritanceResult {
  const upstreamUrl = useUpstreamUrl(nodeId);
  const { data, updateData } = useNodeData(nodeId);

  // Get stored URL from node data
  const storedUrl = typeof data?.url === 'string' ? data.url : '';

  // Local draft state for URL input
  const [urlDraft, setUrlDraft] = useState<string>(storedUrl);

  // Sync draft with stored value when it changes externally
  useEffect(() => {
    setUrlDraft(storedUrl);
  }, [storedUrl]);

  // Compute effective URL
  const trimmedStoredUrl = useMemo(() => storedUrl.trim(), [storedUrl]);
  const effectiveUrl = useMemo(() => {
    if (trimmedStoredUrl.length > 0) {
      return trimmedStoredUrl;
    }
    return upstreamUrl ?? null;
  }, [trimmedStoredUrl, upstreamUrl]);

  const hasCustomUrl = trimmedStoredUrl.length > 0;

  // Commit draft to node data
  const commitDraft = useCallback(
    (raw: string) => {
      const trimmed = raw.trim();
      updateData({ url: trimmed.length > 0 ? trimmed : null } as Record<string, unknown>);
    },
    [updateData],
  );

  const commitUrl = useCallback(() => {
    commitDraft(urlDraft);
  }, [commitDraft, urlDraft]);

  // Reset to inherit from upstream
  const resetUrl = useCallback(() => {
    setUrlDraft('');
    updateData({ url: null } as Record<string, unknown>);
  }, [updateData]);

  // Keyboard handler for URL input
  const handleUrlKeyDown = useCallback(
    (event: KeyboardEvent<HTMLInputElement>) => {
      if (event.key === 'Enter') {
        event.preventDefault();
        commitDraft(urlDraft);
      } else if (event.key === 'Escape') {
        event.preventDefault();
        setUrlDraft(storedUrl);
      }
    },
    [commitDraft, storedUrl, urlDraft],
  );

  return {
    urlDraft,
    setUrlDraft,
    effectiveUrl,
    hasCustomUrl,
    upstreamUrl,
    commitUrl,
    resetUrl,
    handleUrlKeyDown,
  };
}

export default useUrlInheritance;
