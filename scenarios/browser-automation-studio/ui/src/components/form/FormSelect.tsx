import { useId } from 'react';
import { FORM_SELECT_CLASSES, FORM_LABEL_CLASSES, FORM_DESCRIPTION_CLASSES } from './styles';

export interface FormSelectOption<T extends string = string> {
  value: T;
  label: string;
}

export interface FormSelectProps<T extends string = string> {
  /** Current selected value */
  value: T | undefined;
  /** Called when selection changes */
  onChange: (value: T) => void;
  /** Available options */
  options: FormSelectOption<T>[];
  /** Label text displayed above the select */
  label: string;
  /** Optional description text below the select */
  description?: string;
  /** Whether the select is disabled */
  disabled?: boolean;
  /** Additional CSS classes for the wrapper */
  className?: string;
}

/**
 * A controlled select/dropdown with label and description support.
 *
 * Supports generic typing for type-safe option values.
 *
 * @example
 * ```tsx
 * <FormSelect
 *   value={settings.colorScheme}
 *   onChange={(v) => updateSettings('color_scheme', v)}
 *   label="Color Scheme"
 *   options={[
 *     { value: 'light', label: 'Light' },
 *     { value: 'dark', label: 'Dark' },
 *   ]}
 * />
 * ```
 */
export function FormSelect<T extends string = string>({
  value,
  onChange,
  options,
  label,
  description,
  disabled,
  className,
}: FormSelectProps<T>) {
  const id = useId();

  const handleChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    onChange(e.target.value as T);
  };

  return (
    <div className={className}>
      <label htmlFor={id} className={FORM_LABEL_CLASSES}>
        {label}
      </label>
      <select
        id={id}
        value={value ?? ''}
        onChange={handleChange}
        disabled={disabled}
        className={FORM_SELECT_CLASSES}
      >
        {options.map((opt) => (
          <option key={opt.value} value={opt.value}>
            {opt.label}
          </option>
        ))}
      </select>
      {description && <p className={FORM_DESCRIPTION_CLASSES}>{description}</p>}
    </div>
  );
}
