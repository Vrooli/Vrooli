import { useEffect, useId, useRef } from 'react';

interface CodeInputProps {
  value: string;
  onChange: (value: string) => void;
  onComplete?: (value: string) => void;
  length?: number;
  disabled?: boolean;
  invalid?: boolean;
  autoFocus?: boolean;
  label: string;
  describedBy?: string;
  testId?: string;
}

/**
 * One real input drawn as separate digit cells. A single field keeps paste,
 * iOS/Android one-time-code autofill, password managers and screen readers
 * working, which per-digit inputs routinely break.
 */
export function CodeInput({ value, onChange, onComplete, length = 6, disabled, invalid, autoFocus, label, describedBy, testId }: CodeInputProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const id = useId();
  const lastCompleted = useRef('');

  useEffect(() => {
    if (autoFocus) inputRef.current?.focus();
  }, [autoFocus]);

  useEffect(() => {
    if (value.length < length) lastCompleted.current = '';
  }, [value, length]);

  const handleChange = (raw: string) => {
    const digits = raw.replace(/\D/g, '').slice(0, length);
    onChange(digits);
    if (digits.length === length && lastCompleted.current !== digits) {
      lastCompleted.current = digits;
      onComplete?.(digits);
    }
  };

  const active = Math.min(value.length, length - 1);
  return (
    <div className="auth-code" data-invalid={invalid ? 'true' : undefined} data-disabled={disabled ? 'true' : undefined}>
      <label htmlFor={id} className="sr-only">{label}</label>
      <input
        ref={inputRef}
        id={id}
        className="auth-code-input"
        type="text"
        inputMode="numeric"
        autoComplete="one-time-code"
        pattern="[0-9]*"
        maxLength={length}
        value={value}
        disabled={disabled}
        aria-invalid={invalid || undefined}
        aria-describedby={describedBy}
        data-testid={testId}
        onChange={(event) => { handleChange(event.target.value); }}
        onPaste={(event) => {
          event.preventDefault();
          handleChange(event.clipboardData.getData('text'));
        }}
      />
      <div className="auth-code-cells" aria-hidden="true">
        {Array.from({ length }, (_, index) => (
          <span
            key={index}
            className="auth-code-cell"
            data-filled={index < value.length ? 'true' : undefined}
            data-active={index === active && value.length < length ? 'true' : undefined}
          >
            {value[index] ?? ''}
          </span>
        ))}
      </div>
    </div>
  );
}
