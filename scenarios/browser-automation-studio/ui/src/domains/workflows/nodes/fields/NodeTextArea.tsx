import { FC } from 'react';
import { textInputHandler } from '@hooks/useSyncedField';
import type { TextAreaProps } from './types';

const textareaClasses =
  'w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none resize-none';

/**
 * Textarea field for node forms.
 *
 * @example
 * ```tsx
 * const value = useSyncedString(params?.value ?? '', {
 *   onCommit: (v) => updateParams({ value: v }),
 * });
 *
 * <NodeTextArea field={value} label="Value" rows={3} placeholder="Enter content..." />
 * ```
 */
export const NodeTextArea: FC<TextAreaProps> = ({
  field,
  label,
  description,
  placeholder,
  disabled,
  rows = 2,
  icon: Icon,
  iconClassName,
  className,
}) => (
  <div className={className}>
    <label className="text-gray-400 block mb-1 text-xs">
      {Icon && <Icon size={12} className={`inline mr-1 ${iconClassName ?? ''}`} />}
      {label}
    </label>
    <textarea
      rows={rows}
      value={field.value}
      onChange={textInputHandler(field.setValue)}
      onBlur={field.commit}
      placeholder={placeholder}
      disabled={disabled}
      className={textareaClasses}
    />
    {description && <p className="text-[10px] text-gray-500 mt-1">{description}</p>}
  </div>
);

export default NodeTextArea;
