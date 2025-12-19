import { selectInputHandler } from '@hooks/useSyncedField';
import type { SelectFieldProps } from './types';

const selectClasses =
  'w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none';

/**
 * Select/dropdown field for node forms.
 * Commits immediately on change (no blur needed).
 *
 * @example
 * ```tsx
 * const MODES = [
 *   { value: 'newest', label: 'Newest tab' },
 *   { value: 'oldest', label: 'Oldest tab' },
 * ];
 *
 * const mode = useSyncedSelect(params?.mode ?? 'newest', {
 *   onCommit: (v) => updateParams({ mode: v }),
 * });
 *
 * <NodeSelectField field={mode} label="Mode" options={MODES} />
 * ```
 */
export function NodeSelectField<T extends string = string>({
  field,
  label,
  description,
  options,
  disabled,
  icon: Icon,
  iconClassName,
  className,
}: SelectFieldProps<T>) {
  return (
    <div className={className}>
      <label className="text-gray-400 block mb-1 text-xs">
        {Icon && <Icon size={12} className={`inline mr-1 ${iconClassName ?? ''}`} />}
        {label}
      </label>
      <select
        value={field.value}
        onChange={selectInputHandler(field.setValue, field.commit)}
        disabled={disabled}
        className={selectClasses}
      >
        {options.map((opt) => (
          <option key={opt.value} value={opt.value}>
            {opt.label}
          </option>
        ))}
      </select>
      {description && <p className="text-[10px] text-gray-500 mt-1">{description}</p>}
    </div>
  );
}

export default NodeSelectField;
