import { useId } from 'react';

export interface FormRadioOption<T extends string = string> {
  value: T;
  label: string;
  description?: string;
}

export interface FormRadioGroupProps<T extends string = string> {
  /** Unique name for the radio group */
  name: string;
  /** Current selected value */
  value: T | undefined;
  /** Called when selection changes */
  onChange: (value: T) => void;
  /** Available options */
  options: FormRadioOption<T>[];
  /** Whether the radio group is disabled */
  disabled?: boolean;
  /** Additional CSS classes for the wrapper */
  className?: string;
}

/**
 * A controlled radio group with card-style options.
 *
 * Each option is rendered as a clickable card with label and optional description.
 *
 * @example
 * ```tsx
 * <FormRadioGroup
 *   name="blocking_mode"
 *   value={settings.ad_blocking_mode}
 *   onChange={(v) => onChange('ad_blocking_mode', v)}
 *   options={[
 *     { value: 'none', label: 'Disabled', description: 'No blocking' },
 *     { value: 'ads_only', label: 'Ads Only', description: 'Block ads' },
 *   ]}
 * />
 * ```
 */
export function FormRadioGroup<T extends string = string>({
  name,
  value,
  onChange,
  options,
  disabled,
  className,
}: FormRadioGroupProps<T>): JSX.Element {
  const groupId = useId();

  return (
    <div className={`space-y-2 ${className ?? ''}`}>
      {options.map((option) => (
        <label
          key={option.value}
          className="flex items-start gap-3 p-3 rounded-lg border border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-800/50 cursor-pointer"
        >
          <input
            type="radio"
            name={`${groupId}-${name}`}
            value={option.value}
            checked={value === option.value}
            onChange={() => onChange(option.value)}
            disabled={disabled}
            className="mt-0.5"
          />
          <div>
            <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
              {option.label}
            </span>
            {option.description && (
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
                {option.description}
              </p>
            )}
          </div>
        </label>
      ))}
    </div>
  );
}
