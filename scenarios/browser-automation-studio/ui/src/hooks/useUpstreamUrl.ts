import { useState, useEffect, useCallback } from 'react';
import { useWorkflowStore } from '../stores/workflowStore';
import { getUpstreamUrl, getUpstreamUrlAsync } from '../utils/nodeUtils';

/**
 * Hook to get the upstream URL for a given node.
 *
 * PERFORMANCE OPTIMIZATION: Uses a Zustand selector that computes the upstream URL
 * directly, so the component only re-renders when the actual URL value changes,
 * not on every node/edge modification in the workflow.
 *
 * For scenario navigation, it resolves to the actual HTTP URL asynchronously.
 */
export function useUpstreamUrl(nodeId: string): string | null {
  // Selector computes upstream URL - only re-renders when result changes
  const syncUrl = useWorkflowStore(
    useCallback(
      (state) => getUpstreamUrl(nodeId, state.nodes, state.edges),
      [nodeId]
    )
  );

  const [resolvedUrl, setResolvedUrl] = useState<string | null>(syncUrl);

  // Handle async resolution for scenario:// URLs
  useEffect(() => {
    if (syncUrl?.startsWith('scenario://')) {
      // For scenario URLs, resolve asynchronously
      const { nodes, edges } = useWorkflowStore.getState();
      void getUpstreamUrlAsync(nodeId, nodes, edges).then((url) => {
        setResolvedUrl(url);
      });
    } else {
      // For regular URLs, use the sync result directly
      setResolvedUrl(syncUrl);
    }
  }, [nodeId, syncUrl]);

  return resolvedUrl;
}