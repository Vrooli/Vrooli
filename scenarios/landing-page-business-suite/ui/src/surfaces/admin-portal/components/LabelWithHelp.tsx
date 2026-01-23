import { useState } from 'react';
import { HelpCircle, X } from 'lucide-react';

export interface LabelWithHelpProps {
  /** Label text */
  label: string;
  /** Help text shown in the tooltip */
  help: string;
  /** HTML for attribute to link label to input */
  htmlFor?: string;
  /** Additional className for the label */
  className?: string;
}

/**
 * LabelWithHelp - A label with an expandable help tooltip.
 *
 * Extracted from BrandingSettings.tsx for reuse across admin forms.
 *
 * @example
 * ```tsx
 * <LabelWithHelp
 *   label="SMTP Host"
 *   help="Your email provider's SMTP server address. Common examples: smtp.gmail.com..."
 * />
 * <input type="text" ... />
 * ```
 */
export function LabelWithHelp({ label, help, htmlFor, className }: LabelWithHelpProps) {
  const [showHelp, setShowHelp] = useState(false);

  return (
    <div className={`relative ${className ?? ''}`}>
      <label
        htmlFor={htmlFor}
        className="flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wide text-slate-400"
      >
        {label}
        <button
          type="button"
          onClick={() => setShowHelp(!showHelp)}
          className="text-slate-500 hover:text-slate-300"
          aria-label={`Help for ${label}`}
        >
          <HelpCircle className="h-3.5 w-3.5" />
        </button>
      </label>
      {showHelp && (
        <div className="absolute left-0 top-full z-10 mt-1 w-72 rounded-lg border border-white/10 bg-slate-800 p-3 text-xs text-slate-300 shadow-xl">
          {help}
          <button
            type="button"
            onClick={() => setShowHelp(false)}
            className="absolute right-2 top-2 text-slate-500 hover:text-slate-300"
            aria-label="Close help"
          >
            <X className="h-3 w-3" />
          </button>
        </div>
      )}
    </div>
  );
}
