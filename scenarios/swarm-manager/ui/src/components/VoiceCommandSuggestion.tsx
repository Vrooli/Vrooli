import { useCallback, useEffect, useRef } from "react";
import { Check, Mic, X } from "lucide-react";
import type { CommandSuggestion } from "../audio-integration";

const AUTO_DISMISS_MS = 5_000;

export interface VoiceCommandSuggestionProps {
  suggestion: CommandSuggestion;
  onConfirm: (suggestion: CommandSuggestion) => void;
  onDismiss: (suggestion: CommandSuggestion) => void;
}

/** Confirmation surface for commands recognized from a final voice segment. */
export function VoiceCommandSuggestion({ suggestion, onConfirm, onDismiss }: VoiceCommandSuggestionProps) {
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  useEffect(() => {
    timerRef.current = setTimeout(() => onDismiss(suggestion), AUTO_DISMISS_MS);
    return () => { if (timerRef.current) clearTimeout(timerRef.current); };
  }, [onDismiss, suggestion]);
  const confirm = useCallback(() => { if (timerRef.current) clearTimeout(timerRef.current); onConfirm(suggestion); }, [onConfirm, suggestion]);
  const dismiss = useCallback(() => { if (timerRef.current) clearTimeout(timerRef.current); onDismiss(suggestion); }, [onDismiss, suggestion]);
  return (
    <div data-testid="voice-command-suggestion" data-audio-state="command-suggestion" className="flex items-center gap-2 border-t border-slate-700 bg-slate-800 px-2 py-1.5 text-xs">
      <Mic className="h-3.5 w-3.5 shrink-0 text-cyan-400" aria-hidden="true" />
      <span className="min-w-0 flex-1 truncate text-slate-200">{suggestion.description}</span>
      <button type="button" onClick={confirm} className="rounded border border-green-500/40 bg-green-500/10 p-1.5 text-green-400" aria-label="Execute voice command"><Check className="h-3.5 w-3.5" aria-hidden="true" /></button>
      <button type="button" onClick={dismiss} className="rounded border border-slate-600 bg-slate-800 p-1.5 text-slate-300" aria-label="Dismiss voice command"><X className="h-3.5 w-3.5" aria-hidden="true" /></button>
    </div>
  );
}

export default VoiceCommandSuggestion;
