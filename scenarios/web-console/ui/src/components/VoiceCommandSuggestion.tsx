import { useEffect, useRef, useCallback } from "react";
import { Check, X, Mic } from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../consts/strings";
import type { CommandSuggestion } from "../hooks/voice/types";

/** Auto-dismiss timeout for unacted command suggestions. */
const AUTO_DISMISS_MS = 5000;

interface VoiceCommandSuggestionProps {
  suggestion: CommandSuggestion;
  /** Called when the user confirms the command. */
  onConfirm: (suggestion: CommandSuggestion) => void;
  /** Called when the user dismisses (or auto-dismiss fires). */
  onDismiss: (suggestion: CommandSuggestion) => void;
}

export default function VoiceCommandSuggestion({
  suggestion,
  onConfirm,
  onDismiss,
}: VoiceCommandSuggestionProps) {
  const { t } = useTranslation();
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Auto-dismiss after timeout
  useEffect(() => {
    timerRef.current = setTimeout(() => {
      onDismiss(suggestion);
    }, AUTO_DISMISS_MS);
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    };
  }, [suggestion, onDismiss]);

  const handleConfirm = useCallback(() => {
    if (timerRef.current) clearTimeout(timerRef.current);
    onConfirm(suggestion);
  }, [suggestion, onConfirm]);

  const handleDismiss = useCallback(() => {
    if (timerRef.current) clearTimeout(timerRef.current);
    onDismiss(suggestion);
  }, [suggestion, onDismiss]);

  return (
    <div
      data-testid="voice-command-suggestion"
      className="flex items-center gap-2 border-t border-wc-default bg-wc-surface-raised px-2 py-1.5 animate-in slide-in-from-bottom-2 duration-200 md:hidden touch-manipulation select-none"
      onMouseDown={(e) => e.preventDefault()}
    >
      <Mic className="h-3.5 w-3.5 shrink-0 text-cyan-400" />
      <span className="flex-1 text-xs text-wc-text-primary truncate">
        {suggestion.description}
      </span>
      <button
        data-testid="voice-command-confirm"
        tabIndex={-1}
        onPointerDown={(e) => e.preventDefault()}
        onClick={handleConfirm}
        className="shrink-0 rounded border border-green-500/40 bg-green-500/10 p-1.5 text-green-400 transition active:bg-green-500/25 touch-manipulation"
        title={t(strings.voiceCommandSuggestion.executeTitle)}
      >
        <Check className="h-3.5 w-3.5" />
      </button>
      <button
        data-testid="voice-command-dismiss"
        tabIndex={-1}
        onPointerDown={(e) => e.preventDefault()}
        onClick={handleDismiss}
        className="shrink-0 rounded border border-wc-default bg-wc-surface-input p-1.5 text-wc-text-secondary transition active:bg-wc-accent-active touch-manipulation"
        title={t(strings.voiceCommandSuggestion.dismissTitle)}
      >
        <X className="h-3.5 w-3.5" />
      </button>
    </div>
  );
}
