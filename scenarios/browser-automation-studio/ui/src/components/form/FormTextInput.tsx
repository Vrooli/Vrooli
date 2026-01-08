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
export const FormTextInput: FC<FormTextInputProps> = ({
  value,
  onChange,
  label,
  description,
  placeholder,
  type = 'text',
  disabled,
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
        className={FORM_INPUT_CLASSES}
      />
      {description && <p className={FORM_DESCRIPTION_CLASSES}>{description}</p>}
    </div>
  );
};
