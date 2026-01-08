import type { FC } from 'react';

export interface FormToggleCardProps {
  /** Current checked state */
  checked: boolean;
  /** Called when checked state changes */
  onChange: (checked: boolean) => void;
  /** Label text displayed next to the checkbox */
  label: string;
  /** Optional description text below the label */
  description?: string;
  /** Whether the toggle is disabled */
  disabled?: boolean;
  /** Additional CSS classes for the wrapper */
  className?: string;
}

/**
 * A toggle checkbox displayed as a card with label and description.
 *
 * Similar to FormCheckbox but with a card-style container for settings lists.
 *
 * @example
 * ```tsx
 * <FormToggleCard
 *   checked={settings.enabled}
 *   onChange={(v) => updateSettings('enabled', v)}
 *   label="Enable feature"
 *   description="When enabled, this feature will be active"
 * />
 * ```
 */
export const FormToggleCard: FC<FormToggleCardProps> = ({
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
    <label
      className={`flex items-start gap-3 p-3 rounded-lg border border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-800/50 cursor-pointer ${className ?? ''}`}
    >
      <input
        type="checkbox"
        checked={checked}
        onChange={handleChange}
        disabled={disabled}
        className="mt-0.5 rounded border-gray-300 dark:border-gray-600"
      />
      <div>
        <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
          {label}
        </span>
        {description && (
          <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
            {description}
          </p>
        )}
      </div>
    </label>
  );
};
