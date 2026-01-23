import { X } from 'lucide-react';
import { inputClassName } from './FormField';

export interface ClearableInputProps {
  /** Current value */
  value: string;
  /** Change handler */
  onChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
  /** Clear handler - called when clear button is clicked */
  onClear: () => void;
  /** Input type (text, url, email, etc.) */
  type?: string;
  /** Placeholder text */
  placeholder?: string;
  /** Input name attribute */
  name?: string;
  /** Input id attribute */
  id?: string;
  /** Test ID for the input */
  testId?: string;
  /** Additional className for the container */
  className?: string;
  /** Whether the input is disabled */
  disabled?: boolean;
  /** Title for the clear button */
  clearTitle?: string;
}

/**
 * ClearableInput - An input with a conditional clear button.
 *
 * The clear button only appears when the input has a value.
 * Extracted from the repeated pattern in BrandingSettings.tsx.
 *
 * @example
 * ```tsx
 * <ClearableInput
 *   value={form.tagline}
 *   onChange={handleInput('tagline')}
 *   onClear={() => handleClearField('tagline')}
 *   placeholder="Your catchy tagline"
 *   clearTitle="Clear tagline"
 * />
 * ```
 */
export function ClearableInput({
  value,
  onChange,
  onClear,
  type = 'text',
  placeholder,
  name,
  id,
  testId,
  className,
  disabled,
  clearTitle = 'Clear',
}: ClearableInputProps) {
  return (
    <div className={`flex gap-2 ${className ?? ''}`}>
      <input
        type={type}
        value={value}
        onChange={onChange}
        placeholder={placeholder}
        name={name}
        id={id}
        disabled={disabled}
        className={`${inputClassName} flex-1`}
        data-testid={testId}
      />
      {value && (
        <button
          type="button"
          onClick={onClear}
          className="mt-1 p-2 text-slate-400 hover:text-rose-400"
          title={clearTitle}
          disabled={disabled}
        >
          <X className="h-4 w-4" />
        </button>
      )}
    </div>
  );
}
