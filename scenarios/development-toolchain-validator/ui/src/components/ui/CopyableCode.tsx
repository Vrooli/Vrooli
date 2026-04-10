import { useState, useCallback } from "react";
import { Copy, Check } from "lucide-react";

// ─────────────────────────────────────────────────────────────────────────────
// CopyableCode Primitive
// [REQ:P0-001] Reference Scenario Registry - CLI command display
// ─────────────────────────────────────────────────────────────────────────────
//
// Displays code/command text with a copy-to-clipboard button.
// Provides visual feedback when the copy action succeeds.
// ─────────────────────────────────────────────────────────────────────────────

interface CopyableCodeProps {
  /** The code/command to display */
  code: string;
  /** Optional label for the copy button */
  label?: string;
  /** Optional test ID */
  testId?: string;
  /** Size variant */
  size?: "sm" | "default";
}

export function CopyableCode({
  code,
  label,
  testId,
  size = "default"
}: CopyableCodeProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      // Reset after 2 seconds
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Fallback for browsers without clipboard API
      const textArea = document.createElement("textarea");
      textArea.value = code;
      document.body.appendChild(textArea);
      textArea.select();
      document.execCommand("copy");
      document.body.removeChild(textArea);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  }, [code]);

  const sizeClasses = {
    sm: "text-xs p-3",
    default: "text-sm p-4"
  };

  const iconSize = size === "sm" ? "h-3 w-3" : "h-4 w-4";

  return (
    <div
      data-testid={testId}
      className={`
        group relative bg-slate-800/50 rounded-lg font-mono
        ${sizeClasses[size]}
      `}
    >
      {label && (
        <div className="flex items-center gap-2 text-slate-500 mb-2">
          <span className="text-xs uppercase tracking-wide">{label}</span>
        </div>
      )}
      <div className="flex items-start justify-between gap-4">
        <code className="text-slate-300 break-all flex-1">{code}</code>
        <button
          type="button"
          onClick={handleCopy}
          className={`
            shrink-0 p-1.5 rounded-md transition-all
            ${copied
              ? "bg-emerald-500/20 text-emerald-400"
              : "bg-white/5 text-slate-400 hover:text-slate-300 hover:bg-white/10"
            }
            focus:outline-none focus:ring-2 focus:ring-indigo-500/50
          `}
          aria-label={copied ? "Copied!" : "Copy to clipboard"}
          title={copied ? "Copied!" : "Copy to clipboard"}
        >
          {copied ? (
            <Check className={iconSize} aria-hidden="true" />
          ) : (
            <Copy className={iconSize} aria-hidden="true" />
          )}
        </button>
      </div>
    </div>
  );
}
