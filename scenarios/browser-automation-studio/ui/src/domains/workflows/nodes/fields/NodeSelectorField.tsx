import { FC, useState, useCallback } from 'react';
import { Target, X as XIcon } from 'lucide-react';
import { textInputHandler, type UseSyncedFieldResult } from '@hooks/useSyncedField';
import { ElementPickerModal } from '../../components';
import type { ElementInfo } from '@/types/elements';

export interface NodeSelectorFieldProps {
  /** Field state from useSyncedString for the selector */
  field: UseSyncedFieldResult<string>;
  /** The effective URL for the page (enables element picker when set) */
  effectiveUrl: string | null;
  /** Callback when an element is selected from the picker */
  onElementSelect?: (selector: string, elementInfo: ElementInfo) => void;
  /** Callback when the selector is cleared (only called if showClear is true and field has value) */
  onClear?: () => void;
  /** Label text (default: none - just shows the input) */
  label?: string;
  /** Placeholder text (default: "CSS Selector...") */
  placeholder?: string;
  /** Additional className for the wrapper div */
  className?: string;
  /** Whether to show the element picker button (default: true) */
  showPicker?: boolean;
  /** Whether to show a clear button when the field has a value (default: false) */
  showClear?: boolean;
}

/**
 * Selector input field with optional element picker integration.
 *
 * Features:
 * - Text input for CSS selector
 * - Element picker button (enabled when effectiveUrl is set)
 * - ElementPickerModal integration
 *
 * @example
 * ```tsx
 * const selector = useSyncedString(params?.selector ?? '', {
 *   onCommit: (v) => updateParams({ selector: v || undefined }),
 * });
 *
 * <NodeSelectorField
 *   field={selector}
 *   effectiveUrl={effectiveUrl}
 *   onElementSelect={(sel, info) => {
 *     selector.setValue(sel);
 *     updateParams({ selector: sel });
 *     updateData({ elementInfo: info });
 *   }}
 * />
 * ```
 */
export const NodeSelectorField: FC<NodeSelectorFieldProps> = ({
  field,
  effectiveUrl,
  onElementSelect,
  onClear,
  label,
  placeholder = 'CSS Selector...',
  className,
  showPicker = true,
  showClear = false,
}) => {
  const [showElementPicker, setShowElementPicker] = useState(false);

  const handleElementSelect = useCallback(
    (selector: string, elementInfo: ElementInfo) => {
      field.setValue(selector);
      field.commit();
      onElementSelect?.(selector, elementInfo);
      setShowElementPicker(false);
    },
    [field, onElementSelect],
  );

  const handleClear = useCallback(() => {
    field.setValue('');
    field.commit();
    onClear?.();
  }, [field, onClear]);

  const canUsePicker = Boolean(effectiveUrl);
  const hasValue = field.value.length > 0;

  return (
    <>
      <div className={className ?? 'mb-2'}>
        {label && (
          <label className="text-gray-400 block mb-1 text-xs">{label}</label>
        )}
        <div className="flex items-center gap-1">
          <input
            type="text"
            placeholder={placeholder}
            className="flex-1 px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
            value={field.value}
            onChange={textInputHandler(field.setValue)}
            onBlur={field.commit}
          />
          {showPicker && (
            <div
              className="inline-block"
              title={!canUsePicker ? 'Set a page URL before picking elements' : undefined}
            >
              <button
                type="button"
                onClick={() => canUsePicker && setShowElementPicker(true)}
                className={`p-1.5 bg-flow-bg rounded border border-gray-700 transition-colors ${
                  canUsePicker ? 'hover:bg-gray-700 cursor-pointer' : 'opacity-50 cursor-not-allowed'
                }`}
                title={canUsePicker ? `Pick element from ${effectiveUrl}` : undefined}
                disabled={!canUsePicker}
              >
                <Target size={14} className={canUsePicker ? 'text-gray-400' : 'text-gray-600'} />
              </button>
            </div>
          )}
          {showClear && hasValue && (
            <button
              type="button"
              className="p-1.5 rounded border border-gray-700 bg-flow-bg text-gray-300 hover:bg-gray-700 transition-colors"
              onClick={handleClear}
              title="Clear selector"
            >
              <XIcon size={12} />
            </button>
          )}
        </div>
      </div>

      {showElementPicker && effectiveUrl && (
        <ElementPickerModal
          isOpen={showElementPicker}
          onClose={() => setShowElementPicker(false)}
          url={effectiveUrl}
          onSelectElement={handleElementSelect}
          selectedSelector={field.value}
        />
      )}
    </>
  );
};

export default NodeSelectorField;
