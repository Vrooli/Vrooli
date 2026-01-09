/**
 * useActionParams Hook
 *
 * Provides typed access to V2 ActionDefinition params for workflow nodes.
 * This hook enables components to read and write action params directly,
 * making the action field the single source of truth.
 */

import { useCallback, useMemo } from 'react';
import { useReactFlow } from 'reactflow';
import type { Node } from 'reactflow';
import {
  extractParams,
  extractMetadata,
  updateActionParams as updateParams,
  updateActionMetadata,
  type ActionDefinition,
  type ActionMetadata,
} from '@utils/actionParams';

export interface UseActionParamsResult<T> {
  /** The typed params for this action (e.g., ClickParams, NavigateParams) */
  params: T | undefined;
  /** The action metadata (label, etc.) */
  metadata: ActionMetadata | undefined;
  /** The full action definition */
  action: ActionDefinition | undefined;
  /** Update params with a partial object - merges with existing params */
  updateParams: (updates: Partial<T>) => void;
  /** Update metadata with a partial object - merges with existing metadata */
  updateMetadata: (updates: Partial<ActionMetadata>) => void;
  /** Replace the entire action definition */
  setAction: (action: ActionDefinition) => void;
}

/**
 * Hook for accessing and updating V2 ActionDefinition params in workflow nodes.
 *
 * @param nodeId - The ID of the node to access
 * @returns Object with params, metadata, and update functions
 *
 * @example
 * ```tsx
 * const { params, updateParams, metadata, updateMetadata } = useActionParams<ClickParams>(nodeId);
 *
 * // Read params
 * const selector = params?.selector ?? '';
 *
 * // Update params
 * updateParams({ selector: newSelector });
 *
 * // Update metadata (label)
 * updateMetadata({ label: 'Click Submit' });
 * ```
 */
export function useActionParams<T>(
  nodeId: string,
): UseActionParamsResult<T> {
  const { getNodes, setNodes } = useReactFlow();

  // Get the current node's action
  const nodeAction = useMemo(() => {
    const nodes = getNodes();
    const node = nodes.find((n) => n.id === nodeId);
    return (node as Node & { action?: ActionDefinition })?.action;
  }, [getNodes, nodeId]);

  // Extract typed params from action
  const params = useMemo(() => extractParams<T>(nodeAction), [nodeAction]);

  // Extract metadata from action
  const metadata = useMemo(() => extractMetadata(nodeAction), [nodeAction]);

  // Update the node's action in ReactFlow state
  const updateNodeAction = useCallback(
    (newAction: ActionDefinition) => {
      setNodes((nodes) =>
        nodes.map((node) => {
          if (node.id !== nodeId) {
            return node;
          }
          return {
            ...node,
            action: newAction,
          } as Node & { action: ActionDefinition };
        }),
      );
    },
    [nodeId, setNodes],
  );

  // Update params with partial object
  const handleUpdateParams = useCallback(
    (updates: Partial<T>) => {
      if (!nodeAction) {
        console.warn(`useActionParams: Cannot update params - node ${nodeId} has no action`);
        return;
      }
      const newAction = updateParams<T>(nodeAction, updates);
      updateNodeAction(newAction);
    },
    [nodeAction, nodeId, updateNodeAction],
  );

  // Update metadata with partial object
  const handleUpdateMetadata = useCallback(
    (updates: Partial<ActionMetadata>) => {
      if (!nodeAction) {
        console.warn(`useActionParams: Cannot update metadata - node ${nodeId} has no action`);
        return;
      }
      const newAction = updateActionMetadata(nodeAction, updates);
      updateNodeAction(newAction);
    },
    [nodeAction, nodeId, updateNodeAction],
  );

  // Set the entire action
  const setAction = useCallback(
    (action: ActionDefinition) => {
      updateNodeAction(action);
    },
    [updateNodeAction],
  );

  return {
    params,
    metadata,
    action: nodeAction,
    updateParams: handleUpdateParams,
    updateMetadata: handleUpdateMetadata,
    setAction,
  };
}

export default useActionParams;
