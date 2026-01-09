/**
 * useElementPicker Hook
 *
 * Provides a reusable element selection handler for workflow nodes that use
 * NodeSelectorField. Handles updating both action params (selector) and
 * node data (elementInfo) when an element is picked.
 *
 * @example
 * ```tsx
 * // Basic usage with action params
 * const elementPicker = useElementPicker(id);
 *
 * <NodeSelectorField
 *   field={selector}
 *   effectiveUrl={effectiveUrl}
 *   onElementSelect={elementPicker.onSelect}
 * />
 * ```
 *
 * @example
 * ```tsx
 * // With custom field names
 * const elementPicker = useElementPicker(id, {
 *   selectorField: 'focusSelector',
 *   elementInfoField: 'focusElement',
 * });
 * ```
 *
 * @example
 * ```tsx
 * // Data-only mode (no action params)
 * const elementPicker = useElementPicker(id, {
 *   mode: 'data-only',
 * });
 * ```
 */

import { useCallback } from 'react';
import { useNodeData } from './useNodeData';
import { useActionParams } from './useActionParams';
import type { ElementInfo } from '@/types/elements';

export interface UseElementPickerOptions {
  /**
   * The field name in action params for the selector.
   * @default 'selector'
   */
  selectorField?: string;

  /**
   * The field name in node.data for element info.
   * @default 'elementInfo'
   */
  elementInfoField?: string;

  /**
   * Whether to store selector in action params or just node.data.
   * - 'action-params': Update action params (V2 native) + node data
   * - 'data-only': Only update node.data (legacy pattern)
   * @default 'action-params'
   */
  mode?: 'action-params' | 'data-only';
}

export interface UseElementPickerResult {
  /**
   * Callback to pass to NodeSelectorField's onElementSelect prop.
   */
  onSelect: (selector: string, elementInfo: ElementInfo) => void;
}

/**
 * Hook for handling element selection in workflow nodes.
 *
 * @param nodeId - The ReactFlow node ID
 * @param options - Configuration options
 * @returns Object with onSelect callback
 */
export function useElementPicker(
  nodeId: string,
  options: UseElementPickerOptions = {},
): UseElementPickerResult {
  const {
    selectorField = 'selector',
    elementInfoField = 'elementInfo',
    mode = 'action-params',
  } = options;

  const { updateData } = useNodeData(nodeId);
  const { updateParams } = useActionParams(nodeId);

  const onSelect = useCallback(
    (selector: string, elementInfo: ElementInfo) => {
      if (mode === 'action-params') {
        // V2 Native: Update selector in action params
        updateParams({ [selectorField]: selector || undefined });
        // Keep elementInfo in node.data (not part of proto)
        updateData({ [elementInfoField]: elementInfo });
      } else {
        // Legacy: Store both in node.data
        updateData({
          [selectorField]: selector || undefined,
          [elementInfoField]: elementInfo,
        });
      }
    },
    [mode, selectorField, elementInfoField, updateParams, updateData],
  );

  return { onSelect };
}

export default useElementPicker;
