import { useId, type FC } from 'react';
import { FORM_INPUT_CLASSES, FORM_LABEL_CLASSES, FORM_DESCRIPTION_CLASSES } from './styles';

export interface FormNumberInputProps {
  /** Current value (undefined displays as empty) */
  value: number | undefined;
  /** Called when value changes (undefined when field is cleared) */
  onChange: (value: number | undefined) => void;
  /** Label text displayed above the input */
  label: string;
  /** Optional description text below the input */
  description?: string;
  /** Placeholder text when empty */
  placeholder?: string;
  /** Minimum allowed value */
  min?: number;
  /** Maximum allowed value */
  max?: number;
  /** Step increment (determines if parseInt or parseFloat is used) */
  step?: number;
  /** Whether the input is disabled */
  disabled?: boolean;
  /** Error message to display (also adds error styling) */
  error?: string;
  /** Additional CSS classes for the wrapper */
  className?: string;
}

/**
 * A controlled number input with label and description support.
 *
 * Handles undefined ↔ empty string conversion automatically.
 * Uses parseFloat for fractional steps, parseInt otherwise.
 *
 * @example
 * ```tsx
 * <FormNumberInput
 *   value={settings.width}
 *   onChange={(v) => updateSettings('width', v)}
 *   label="Width"
 *   placeholder="1920"
 *   min={0}
 * />
 * ```
 */
const ERROR_INPUT_CLASSES =
  'w-full rounded-md border border-red-300 dark:border-red-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-red-500 focus:border-transparent focus:outline-none';

export const FormNumberInput: FC<FormNumberInputProps> = ({
  value,
  onChange,
  label,
  description,
  placeholder,
  min,
  max,
  step,
  disabled,
  error,
  className,
}) => {
  const id = useId();

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const raw = e.target.value;
    if (raw === '') {
      onChange(undefined);
      return;
    }
    // Use parseFloat for fractional steps, parseInt otherwise
    const useFloat = step !== undefined && step % 1 !== 0;
    const parsed = useFloat ? parseFloat(raw) : parseInt(raw, 10);
    onChange(isNaN(parsed) ? undefined : parsed);
  };

  return (
    <div className={className}>
      <label htmlFor={id} className={FORM_LABEL_CLASSES}>
        {label}
      </label>
      <input
        id={id}
        type="number"
        value={value ?? ''}
        onChange={handleChange}
        placeholder={placeholder}
        min={min}
        max={max}
        step={step}
        disabled={disabled}
        className={error ? ERROR_INPUT_CLASSES : FORM_INPUT_CLASSES}
        aria-invalid={!!error}
        aria-describedby={error ? `${id}-error` : undefined}
      />
      {error && (
        <p id={`${id}-error`} className="text-xs text-red-600 dark:text-red-400 mt-1">
          {error}
        </p>
      )}
      {description && !error && <p className={FORM_DESCRIPTION_CLASSES}>{description}</p>}
    </div>
  );
};
