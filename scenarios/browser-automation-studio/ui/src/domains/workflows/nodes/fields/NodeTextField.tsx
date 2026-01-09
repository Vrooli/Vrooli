import { FC } from 'react';
import { textInputHandler } from '@hooks/useSyncedField';
import type { TextFieldProps } from './types';

const inputClasses =
  'w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none';

/**
 * Text input field for node forms.
 *
 * @example
 * ```tsx
 * const name = useSyncedString(params?.name ?? '', {
 *   onCommit: (v) => updateParams({ name: v || undefined }),
 * });
 *
 * <NodeTextField field={name} label="Name" placeholder="Enter name..." />
 * ```
 */
export const NodeTextField: FC<TextFieldProps> = ({
  field,
  label,
  description,
  placeholder,
  disabled,
  icon: Icon,
  iconClassName,
  className,
}) => (
  <div className={className}>
    <label className="text-gray-400 block mb-1 text-xs">
      {Icon && <Icon size={12} className={`inline mr-1 ${iconClassName ?? ''}`} />}
      {label}
    </label>
    <input
      type="text"
      value={field.value}
      onChange={textInputHandler(field.setValue)}
      onBlur={field.commit}
      placeholder={placeholder}
      disabled={disabled}
      className={inputClasses}
    />
    {description && <p className="text-[10px] text-gray-500 mt-1">{description}</p>}
  </div>
);

export default NodeTextField;
