import { FC } from 'react';
import { checkboxInputHandler } from '@hooks/useSyncedField';
import type { CheckboxProps } from './types';

/**
 * Checkbox field for node forms.
 * Commits immediately on change.
 *
 * @example
 * ```tsx
 * const secure = useSyncedBoolean(params?.secure ?? false, {
 *   onCommit: (v) => updateParams({ secure: v || undefined }),
 * });
 *
 * <NodeCheckbox field={secure} label="Secure" />
 * ```
 */
export const NodeCheckbox: FC<CheckboxProps> = ({ field, label, className, disabled }) => (
  <label
    className={`flex items-center gap-2 text-xs text-gray-400 ${disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'} ${className ?? ''}`}
  >
    <input
      type="checkbox"
      checked={field.value}
      onChange={checkboxInputHandler(field.setValue, field.commit)}
      disabled={disabled}
      className="accent-flow-accent"
    />
    {label}
  </label>
);

export default NodeCheckbox;
