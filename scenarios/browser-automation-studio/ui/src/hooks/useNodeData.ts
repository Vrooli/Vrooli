/**
 * useNodeData Hook
 *
 * Provides a reusable pattern for reading and updating node.data in ReactFlow nodes.
 * Extracted from the common updateNodeData pattern used across most node components.
 */

import { useCallback, useMemo } from 'react';
import { useReactFlow } from 'reactflow';

export interface UseNodeDataResult<T extends Record<string, unknown> = Record<string, unknown>> {
  /** The current node data */
  data: T;
  /** Update node data with partial object - handles cleanup of null/undefined values */
  updateData: (updates: Partial<T>) => void;
  /** Get a typed value from node data */
  getValue: <V>(key: keyof T) => V | undefined;
}

/**
 * Hook for reading and updating node.data in workflow nodes.
 *
 * @param nodeId - The ID of the node to access
 * @returns Object with data, updateData function, and getValue helper
 *
 * @example
 * ```tsx
 * const { data, updateData, getValue } = useNodeData<MyNodeData>(nodeId);
 *
 * // Read data
 * const resilience = getValue<ResilienceSettings>('resilience');
 *
 * // Update data
 * updateData({ resilience: newSettings });
 * ```
 */
export function useNodeData<T extends Record<string, unknown> = Record<string, unknown>>(
  nodeId: string,
): UseNodeDataResult<T> {
  const { getNodes, setNodes } = useReactFlow();

  // Get the current node's data
  const data = useMemo(() => {
    const nodes = getNodes();
    const node = nodes.find((n) => n.id === nodeId);
    return (node?.data ?? {}) as T;
  }, [getNodes, nodeId]);

  // Update node data with partial object
  const updateData = useCallback(
    (updates: Partial<T>) => {
      setNodes((nodes) =>
        nodes.map((node) => {
          if (node.id !== nodeId) {
            return node;
          }

          const nextData = { ...(node.data ?? {}) } as Record<string, unknown>;

          for (const [key, value] of Object.entries(updates)) {
            // Special handling for URL field - trim and remove if empty
            if (key === 'url') {
              const trimmed = typeof value === 'string' ? value.trim() : '';
              if (trimmed) {
                nextData.url = trimmed;
              } else {
                delete nextData.url;
              }
              continue;
            }

            // Remove undefined/null values, otherwise set
            if (value === undefined || value === null) {
              delete nextData[key];
            } else {
              nextData[key] = value;
            }
          }

          return {
            ...node,
            data: nextData,
          };
        }),
      );
    },
    [nodeId, setNodes],
  );

  // Get a typed value from node data
  const getValue = useCallback(
    <V,>(key: keyof T): V | undefined => {
      return data[key] as V | undefined;
    },
    [data],
  );

  return {
    data,
    updateData,
    getValue,
  };
}

export default useNodeData;
