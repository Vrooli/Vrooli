import { Send, X } from "lucide-react";
import { useTranslation } from "react-i18next";

import { strings } from "../../consts/strings";
import type { HandoffSuggestion } from "../../lib/captureRules";

// [REQ:P0-014h] Handoff Capture Rules

interface HandoffSuggestionChipProps {
  suggestion: HandoffSuggestion;
  onOpen: (suggestion: HandoffSuggestion) => void;
  onDismiss: (suggestion: HandoffSuggestion) => void;
}

/**
 * An offer, not an action.
 *
 * Inline in the message block, never a modal: a suggestion that interrupts is
 * a suggestion the operator learns to resent, and one that moves the
 * transcript under them while they are reading is worse. It sits inside the
 * message it belongs to, so the scroll position of everything above it is
 * unchanged.
 *
 * The chip NAMES THE RULE that fired. A wrong suggestion is then a rule the
 * operator can go and edit, rather than an unexplained thing the console did.
 */
export default function HandoffSuggestionChip({
  suggestion,
  onOpen,
  onDismiss,
}: HandoffSuggestionChipProps) {
  const { t } = useTranslation();

  return (
    <div
      data-testid="handoff-suggestion"
      data-rule-id={suggestion.ruleId}
      className="mt-1.5 flex items-center gap-2 rounded-lg border border-dashed border-wc-default bg-wc-surface-base/40 px-2.5 py-1.5 text-xs"
    >
      <Send className="h-3.5 w-3.5 shrink-0 text-wc-text-faint" aria-hidden />
      <span className="min-w-0 flex-1">
        <span className="block truncate text-wc-text-secondary">
          {t(strings.handoff.suggestionTitle, { rule: suggestion.ruleName })}
        </span>
        <span className="block truncate text-[11px] text-wc-text-faint">{suggestion.payload}</span>
      </span>
      <button
        type="button"
        onClick={() => { onOpen(suggestion); }}
        className="shrink-0 rounded-lg px-2 py-1 text-xs font-medium text-wc-accent transition hover:bg-wc-surface-input"
      >
        {t(strings.handoff.suggestionOpen)}
      </button>
      <button
        type="button"
        data-testid="handoff-suggestion-dismiss"
        aria-label={t(strings.handoff.suggestionDismiss)}
        title={t(strings.handoff.suggestionDismiss)}
        onClick={() => { onDismiss(suggestion); }}
        className="shrink-0 rounded-full p-1 text-wc-text-faint transition hover:bg-wc-surface-input hover:text-wc-text-primary"
      >
        <X className="h-3.5 w-3.5" />
      </button>
    </div>
  );
}
