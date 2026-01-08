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
        className={FORM_INPUT_CLASSES}
      />
      {description && <p className={FORM_DESCRIPTION_CLASSES}>{description}</p>}
    </div>
  );
};
