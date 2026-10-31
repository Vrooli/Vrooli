import { useState } from 'react';
import { Eye, EyeOff } from 'lucide-react';
import { inputClassName } from './formFieldClasses';

export interface PasswordInputProps {
  /** Current value */
  value: string;
  /** Change handler */
  onChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
  /** Placeholder text */
  placeholder?: string;
  /** Input name attribute */
  name?: string;
  /** Input id attribute */
  id?: string;
  /** Test ID for the input */
  testId?: string;
  /** Additional className for the input */
  className?: string;
  /** Autocomplete attribute */
  autoComplete?: string;
  /** Whether the input is disabled */
  disabled?: boolean;
}

/**
 * PasswordInput - A password input with show/hide toggle.
 *
 * Extracted from BrandingSettings.tsx for reuse across admin forms.
 *
 * @example
 * ```tsx
 * <PasswordInput
 *   value={form.smtp_password}
 *   onChange={handleInput('smtp_password')}
 *   placeholder="Your app password"
 * />
 * ```
 */
export function PasswordInput({
  value,
  onChange,
  placeholder,
  name,
  id,
  testId,
  className,
  autoComplete = 'current-password',
  disabled,
}: PasswordInputProps) {
  const [show, setShow] = useState(false);

  return (
    <div className="relative">
      <input
        type={show ? 'text' : 'password'}
        value={value}
        onChange={onChange}
        placeholder={placeholder}
        name={name}
        id={id}
        autoComplete={autoComplete}
        disabled={disabled}
        className={`${inputClassName} pr-10 ${className ?? ''}`}
        data-testid={testId}
      />
      <button
        type="button"
        onClick={() => { setShow(!show); }}
        className="absolute right-2 top-1/2 -translate-y-1/2 text-slate-500 hover:text-slate-300"
        aria-label={show ? 'Hide password' : 'Show password'}
        disabled={disabled}
      >
        {show ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
      </button>
    </div>
  );
}
