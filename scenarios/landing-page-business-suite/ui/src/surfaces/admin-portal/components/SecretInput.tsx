import { useState } from 'react';
import { Eye, EyeOff, Loader2, ShieldCheck } from 'lucide-react';
import { inputClassName } from './formFieldClasses';
import { cn } from '../../../shared/lib/utils';

export interface SecretInputProps {
  /** Whether a secret value is already stored on the server */
  isSet: boolean;
  /** Current input value (for new entries) */
  value: string;
  /** Change handler for the input */
  onChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
  /** Placeholder text when no value is set */
  placeholder?: string;
  /** Callback to reveal the secret - should return the revealed value */
  onReveal: () => Promise<string>;
  /** Input name attribute */
  name?: string;
  /** Input id attribute */
  id?: string;
  /** Test ID for the input */
  testId?: string;
  /** Additional className for the wrapper */
  className?: string;
  /** Whether the input is disabled */
  disabled?: boolean;
}

/**
 * SecretInput - An input for secret values that shows a masked state when saved.
 *
 * When a secret is already set (isSet=true) and no new value is being typed:
 * - Shows masked circles (●●●●●●●●) indicating a value exists
 * - Shows an eye icon to reveal the actual value
 * - Clicking the eye fetches and displays the real secret
 *
 * When the user starts typing, it switches to a standard input.
 *
 * @example
 * ```tsx
 * <SecretInput
 *   isSet={stripeSettings?.secret_key_set ?? false}
 *   value={stripeForm.secretKey}
 *   onChange={handleStripeInput('secretKey')}
 *   placeholder="rk_live_..."
 *   onReveal={() => revealStripeSecret('secret_key').then(r => r.value)}
 * />
 * ```
 */
export function SecretInput({
  isSet,
  value,
  onChange,
  placeholder,
  onReveal,
  name,
  id,
  testId,
  className,
  disabled,
}: SecretInputProps) {
  const [revealing, setRevealing] = useState(false);
  const [revealedValue, setRevealedValue] = useState<string | null>(null);
  const [showRevealed, setShowRevealed] = useState(false);
  const [revealError, setRevealError] = useState<string | null>(null);
  const [isReplacing, setIsReplacing] = useState(false);

  // Show masked state when: secret is set, user hasn't typed a new value, and not showing revealed
  const showMaskedState = isSet && value === '' && !showRevealed && !isReplacing;

  const handleRevealClick = async () => {
    if (revealing) return;

    if (revealedValue !== null) {
      // Toggle visibility if already fetched
      setShowRevealed(!showRevealed);
      return;
    }

    // Fetch the secret
    setRevealing(true);
    setRevealError(null);
    try {
      const secret = await onReveal();
      setRevealedValue(secret);
      setShowRevealed(true);
    } catch (error) {
      setRevealError(error instanceof Error ? error.message : 'Failed to reveal secret');
    } finally {
      setRevealing(false);
    }
  };

  const handleHideClick = () => {
    setShowRevealed(false);
  };

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    // When user starts typing, clear any revealed state
    if (revealedValue !== null) {
      setRevealedValue(null);
      setShowRevealed(false);
    }
    onChange(e);
  };

  // When showing revealed value
  if (showRevealed && revealedValue !== null) {
    return (
      <div className={cn('relative', className)}>
        <input
          type="text"
          value={revealedValue}
          readOnly
          className={cn(inputClassName, 'pr-10 font-mono text-sm bg-slate-800/50')}
          data-testid={testId}
        />
        <button
          type="button"
          onClick={handleHideClick}
          className="absolute right-2 top-1/2 -translate-y-1/2 text-slate-500 hover:text-slate-300"
          aria-label="Hide secret"
        >
          <EyeOff className="h-4 w-4" />
        </button>
      </div>
    );
  }

  // When showing masked state (secret is set but not revealed)
  if (showMaskedState) {
    return (
      <div className={cn('relative', className)}>
        <div
          className={cn(
            inputClassName,
            'flex items-center justify-between cursor-default pr-10'
          )}
        >
          <span className="flex items-center gap-2 text-slate-400">
            <ShieldCheck className="h-4 w-4 text-emerald-400" />
            <span className="tracking-wider">{'●'.repeat(12)}</span>
          </span>
        </div>
        <button
          type="button"
          onClick={() => { void handleRevealClick(); }}
          disabled={disabled || revealing}
          className="absolute right-2 top-1/2 -translate-y-1/2 text-slate-500 hover:text-slate-300 disabled:opacity-50"
          aria-label={revealing ? 'Loading...' : 'Reveal secret'}
        >
          {revealing ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <Eye className="h-4 w-4" />
          )}
        </button>
        <button
          type="button"
          onClick={() => { setIsReplacing(true); }}
          disabled={disabled}
          className="mt-2 text-xs font-medium text-blue-300 hover:text-blue-200 disabled:opacity-50"
        >
          Replace secret
        </button>
        {revealError && (
          <p className="mt-1 text-xs text-rose-400">{revealError}</p>
        )}
      </div>
    );
  }

  // Standard input for entering new values
  return (
    <div className={cn('relative', className)}>
      <input
        type="text"
        value={value}
        onChange={handleInputChange}
        placeholder={placeholder}
        name={name}
        id={id}
        disabled={disabled}
        className={cn(inputClassName, isSet && value === '' ? 'pr-10' : '')}
        data-testid={testId}
      />
      {isSet && value === '' && (
        <button
          type="button"
          onClick={() => { void handleRevealClick(); }}
          disabled={disabled || revealing}
          className="absolute right-2 top-1/2 -translate-y-1/2 text-slate-500 hover:text-slate-300 disabled:opacity-50"
          aria-label={revealing ? 'Loading...' : 'Reveal secret'}
        >
          {revealing ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <Eye className="h-4 w-4" />
          )}
        </button>
      )}
    </div>
  );
}
