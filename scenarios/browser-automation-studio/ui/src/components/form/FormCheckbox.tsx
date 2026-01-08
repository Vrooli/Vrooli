import type { FC } from 'react';
import { FORM_CHECKBOX_CLASSES, FORM_DESCRIPTION_CLASSES } from './styles';

export interface FormCheckboxProps {
  /** Current checked state */
  checked: boolean;
  /** Called when checked state changes */
  onChange: (checked: boolean) => void;
  /** Label text displayed next to the checkbox */
  label: string;
  /** Optional description text below the checkbox */
  description?: string;
  /** Whether the checkbox is disabled */
  disabled?: boolean;
  /** Additional CSS classes for the wrapper */
  className?: string;
}

/**
 * A controlled checkbox with inline label and optional description.
 *
 * @example
 * ```tsx
 * <FormCheckbox
 *   checked={settings.enabled}
 *   onChange={(v) => updateSettings('enabled', v)}
 *   label="Enable feature"
 *   description="When enabled, this feature will be active"
 * />
 * ```
 */
export const FormCheckbox: FC<FormCheckboxProps> = ({
  checked,
  onChange,
  label,
  description,
  disabled,
  className,
}) => {
  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    onChange(e.target.checked);
  };

  return (
    <div className={className}>
      <label className="flex items-center gap-2">
        <input
          type="checkbox"
          checked={checked}
          onChange={handleChange}
          disabled={disabled}
          className={FORM_CHECKBOX_CLASSES}
        />
        <span className="text-sm text-gray-700 dark:text-gray-300">{label}</span>
      </label>
      {description && <p className={`${FORM_DESCRIPTION_CLASSES} ml-6`}>{description}</p>}
    </div>
  );
};
