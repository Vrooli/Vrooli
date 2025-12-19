import { FC } from 'react';
import { numberInputHandler } from '@hooks/useSyncedField';
import type { NumberFieldProps } from './types';

const inputClasses =
  'w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none';

/**
 * Number input field for node forms.
 *
 * @example
 * ```tsx
 * const timeoutMs = useSyncedNumber(params?.timeoutMs ?? 30000, {
 *   min: 100,
 *   max: 120000,
 *   onCommit: (v) => updateParams({ timeoutMs: v }),
 * });
 *
 * <NodeNumberField field={timeoutMs} label="Timeout (ms)" min={100} max={120000} />
 * ```
 */
export const NodeNumberField: FC<NumberFieldProps> = ({
  field,
  label,
  description,
  min,
  max,
  step,
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
      type="number"
      value={field.value}
      onChange={numberInputHandler(field.setValue)}
      onBlur={field.commit}
      min={min}
      max={max}
      step={step}
      placeholder={placeholder}
      disabled={disabled}
      className={inputClasses}
    />
    {description && <p className="text-[10px] text-gray-500 mt-1">{description}</p>}
  </div>
);

export default NodeNumberField;
