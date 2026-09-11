import { useState } from "react";
import { ChevronDown, Send, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { IconButton } from "@vrooli/react-component-library/IconButton/3";

import { strings } from "../../consts/strings";
import { cn } from "../../lib/classnames";
import type { HandoffSuggestion } from "../../lib/captureRules";

// [REQ:P0-014h] Handoff Capture Rules

interface HandoffSuggestionChipProps {
  /** One rule's matches in one message; several fold into one chip. */
  suggestions: readonly HandoffSuggestion[];
  /** Opens the composer carrying this payload; it never sends. */
  onOpen: (payload: string) => void;
  onDismiss: (suggestions: readonly HandoffSuggestion[]) => void;
}

/** The last path segment, which is what tells two matched paths apart. */
function lastSegment(payload: string): string {
  const segments = payload.split(/[\\/]/).filter(Boolean);
  return segments[segments.length - 1] ?? payload;
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
 *
 * Several matches of one rule are one chip: it counts them, hands them off
 * together, and lists them on request so one can be handed off alone.
 */
export default function HandoffSuggestionChip({
  suggestions,
  onOpen,
  onDismiss,
}: HandoffSuggestionChipProps) {
  const { t } = useTranslation();
  const [expanded, setExpanded] = useState(false);
  const first = suggestions[0];
  if (!first) return null;
  const count = suggestions.length;
  const many = count > 1;
  const payloads = suggestions.map((suggestion) => suggestion.payload);

  return (
    <div
      data-testid="handoff-suggestion"
      data-rule-id={first.ruleId}
      data-count={count}
      className="mt-1.5 rounded-lg border border-dashed border-wc-default bg-wc-surface-base/40 text-xs"
    >
      <div className="flex items-center gap-2 ps-2.5 pe-1 py-1">
        <Send className="h-3.5 w-3.5 shrink-0 text-wc-text-faint" aria-hidden />
        <span className="min-w-0 flex-1">
          <span className="block truncate text-wc-text-secondary">
            {many
              ? t(strings.handoff.suggestionTitleMany, { rule: first.ruleName, count })
              : t(strings.handoff.suggestionTitle, { rule: first.ruleName })}
          </span>
          <span className="block truncate text-[11px] text-wc-text-faint" title={payloads.join("\n")}>
            {many ? payloads.map(lastSegment).join(", ") : first.payload}
          </span>
        </span>
        <button
          type="button"
          onClick={() => { onOpen(payloads.join("\n")); }}
          className="shrink-0 rounded-lg px-2 py-1 text-xs font-medium text-wc-accent transition hover:bg-wc-surface-input"
        >
          {t(strings.handoff.suggestionOpen)}
        </button>
        {many && (
          <IconButton
            data-testid="handoff-suggestion-expand"
            size="xs"
            aria-expanded={expanded}
            aria-label={expanded ? t(strings.handoff.suggestionHideEach) : t(strings.handoff.suggestionShowEach)}
            onClick={() => { setExpanded((value) => !value); }}
            className="shrink-0"
          >
            <ChevronDown className={cn("transition-transform", expanded && "rotate-180")} />
          </IconButton>
        )}
        <IconButton
          data-testid="handoff-suggestion-dismiss"
          size="xs"
          aria-label={many ? t(strings.handoff.suggestionDismissAll, { count }) : t(strings.handoff.suggestionDismiss)}
          onClick={() => { onDismiss(suggestions); }}
          className="shrink-0"
        >
          <X />
        </IconButton>
      </div>
      {many && expanded && (
        <ul className="border-t border-dashed border-wc-default py-0.5">
          {suggestions.map((suggestion) => (
            <li key={suggestion.payload} data-testid="handoff-suggestion-item" className="flex items-center gap-2 ps-8 pe-1">
              <span className="min-w-0 flex-1 truncate text-[11px] text-wc-text-faint" title={suggestion.payload}>
                {suggestion.payload}
              </span>
              <button
                type="button"
                data-testid="handoff-suggestion-item-open"
                onClick={() => { onOpen(suggestion.payload); }}
                className="shrink-0 rounded-lg px-2 py-1 text-xs font-medium text-wc-accent transition hover:bg-wc-surface-input"
              >
                {t(strings.handoff.suggestionOpen)}
              </button>
              <IconButton
                data-testid="handoff-suggestion-item-dismiss"
                size="xs"
                aria-label={t(strings.handoff.suggestionDismiss)}
                onClick={() => { onDismiss([suggestion]); }}
                className="shrink-0"
              >
                <X />
              </IconButton>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
