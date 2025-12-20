/**
 * useResiliencePanel Hook
 *
 * Provides a ready-to-use ResiliencePanel component bound to a node's resilience settings.
 * Eliminates the repetitive getValue + updateData + ResiliencePanel pattern in node components.
 *
 * @example
 * ```tsx
 * const ResilienceSection = useResiliencePanel(id);
 *
 * return (
 *   <BaseNode ...>
 *     {/* node fields *\/}
 *     <ResilienceSection />
 *   </BaseNode>
 * );
 * ```
 */

import { FC, memo, useCallback, useMemo } from 'react';
import { useNodeData } from './useNodeData';
import ResiliencePanel from '@/domains/workflows/nodes/ResiliencePanel';
import type { ResilienceSettings } from '@/types/workflow';

/**
 * Hook that returns a memoized ResiliencePanel component bound to a node's settings.
 *
 * @param nodeId - The ReactFlow node ID
 * @returns A React component that renders the ResiliencePanel with proper bindings
 */
export function useResiliencePanel(nodeId: string): FC {
  const { getValue, updateData } = useNodeData(nodeId);

  const resilienceConfig = getValue<ResilienceSettings>('resilience');

  const handleChange = useCallback(
    (next?: ResilienceSettings) => {
      updateData({ resilience: next ?? null });
    },
    [updateData],
  );

  // Create a memoized component that can be rendered directly
  const BoundResiliencePanel = useMemo(() => {
    const Component: FC = () => (
      <ResiliencePanel value={resilienceConfig} onChange={handleChange} />
    );
    Component.displayName = 'BoundResiliencePanel';
    return memo(Component);
  }, [resilienceConfig, handleChange]);

  return BoundResiliencePanel;
}

/**
 * Alternative hook that returns props instead of a component.
 * Use this when you need more control over the ResiliencePanel rendering.
 *
 * @example
 * ```tsx
 * const resilience = useResiliencePanelProps(id);
 *
 * return (
 *   <BaseNode ...>
 *     {showResilience && <ResiliencePanel {...resilience} />}
 *   </BaseNode>
 * );
 * ```
 */
export function useResiliencePanelProps(nodeId: string): {
  value: ResilienceSettings | undefined;
  onChange: (next?: ResilienceSettings) => void;
} {
  const { getValue, updateData } = useNodeData(nodeId);

  const value = getValue<ResilienceSettings>('resilience');

  const onChange = useCallback(
    (next?: ResilienceSettings) => {
      updateData({ resilience: next ?? null });
    },
    [updateData],
  );

  return { value, onChange };
}

export default useResiliencePanel;
