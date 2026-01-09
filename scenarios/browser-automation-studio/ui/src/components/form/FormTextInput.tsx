import { useId, type FC } from 'react';
import { FORM_INPUT_CLASSES, FORM_LABEL_CLASSES, FORM_DESCRIPTION_CLASSES } from './styles';

export interface FormTextInputProps {
  /** Current value (undefined displays as empty) */
  value: string | undefined;
  /** Called when value changes (undefined when field is cleared) */
  onChange: (value: string | undefined) => void;
  /** Label text displayed above the input */
  label: string;
  /** Optional description text below the input */
  description?: string;
  /** Placeholder text when empty */
  placeholder?: string;
  /** Input type (text, password, email, etc.) */
  type?: 'text' | 'password' | 'email' | 'url';
  /** Whether the input is disabled */
  disabled?: boolean;
  /** Error message to display (also adds error styling) */
  error?: string;
  /** Additional CSS classes for the wrapper */
  className?: string;
}

/**
 * A controlled text input with label and description support.
 *
 * Handles undefined ↔ empty string conversion automatically.
 *
 * @example
 * ```tsx
 * <FormTextInput
 *   value={settings.locale}
 *   onChange={(v) => updateSettings('locale', v)}
 *   label="Locale"
 *   placeholder="en-US"
 * />
 * ```
 */
const ERROR_INPUT_CLASSES =
  'w-full rounded-md border border-red-300 dark:border-red-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-red-500 focus:border-transparent focus:outline-none';

export const FormTextInput: FC<FormTextInputProps> = ({
  value,
  onChange,
  label,
  description,
  placeholder,
  type = 'text',
  disabled,
  error,
  className,
}) => {
  const id = useId();

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const raw = e.target.value;
    onChange(raw === '' ? undefined : raw);
  };

  return (
    <div className={className}>
      <label htmlFor={id} className={FORM_LABEL_CLASSES}>
        {label}
      </label>
      <input
        id={id}
        type={type}
        value={value ?? ''}
        onChange={handleChange}
        placeholder={placeholder}
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
