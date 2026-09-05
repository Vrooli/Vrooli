import { useEffect, useMemo, useState, type CSSProperties, type ReactNode } from "react";
import { Loader2, Plus, Search, Send, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";
import { Checkbox } from "@vrooli/react-component-library/Checkbox/1";
import { IconButton } from "@vrooli/react-component-library/IconButton";
import { Input } from "@vrooli/react-component-library/Input/1";
import { InputGroup } from "@vrooli/react-component-library/InputGroup";
import { Textarea } from "@vrooli/react-component-library/Textarea/1";

import { strings } from "../../consts/strings";
import { cn } from "../../lib/classnames";
import { paneAccentStyle } from "../../lib/paneColor";
import type { HandoffResult } from "../../lib/handoff";
import { targetKey, textForTarget, type HandoffTarget } from "../../hooks/useHandoff";
import { Button } from "../ui/button";
import { SnippetPicker } from "../snippets/SnippetPicker";
import type { SnippetDTO } from "../../api/snippets";
import type { HandoffTargetSection } from "../../hooks/useHandoff";

// [REQ:P0-014d] Handoff Between Sessions In A Group
//
// The composer imports the send path and nothing else. It has no knowledge of
// capture rules: a suggestion chip opens this same surface, already populated,
// through the ordinary props below.

/**
 * Above this many targets the list stops being scannable in one glance and
 * earns a filter. Below it the field is chrome over a list already fully
 * visible — which is why it appears rather than always being there.
 */
const FILTER_THRESHOLD = 6;

export interface HandoffComposerProps {
  open: boolean;
  onClose: () => void;
  /** Where the handoff is coming from, for the title. */
  sourceLabel: string;
  /** The text being carried: a path, a passage, or empty. Never classified. */
  payload: string;
  targets: HandoffTargetSection[];
  sourceSessionId?: string;
  cwd?: string;
  selection?: string;
  /** Pre-select these target keys — a suggestion chip names one. */
  initialSelection?: string[];
  onSend: (targets: HandoffTarget[], textFor: (target: HandoffTarget) => string) => Promise<HandoffResult[]>;
}

/**
 * Compose and send a handoff.
 *
 * Two rules the surface exists to enforce:
 *
 *   1. Nothing is dispatched without the operator seeing the exact text. Every
 *      selected target's message is rendered and editable before Send.
 *   2. The result is reported per target, and `queued` is shown as its own
 *      state. Collapsing it into success would tell the operator their
 *      message arrived when it is still sitting in a pending queue.
 */
export default function HandoffComposer({
  open,
  onClose,
  sourceLabel,
  payload,
  targets,
  sourceSessionId,
  cwd = "",
  selection = "",
  initialSelection,
  onSend,
}: HandoffComposerProps) {
  const { t } = useTranslation();
  const [selected, setSelected] = useState<string[]>(initialSelection ?? []);
  // Per-target overrides. Absent means "use the rendered default", so a target
  // the operator never touched follows its role's prompt even if the payload
  // changes underneath.
  const [edited, setEdited] = useState<Record<string, string>>({});
  const [sending, setSending] = useState(false);
  const [results, setResults] = useState<HandoffResult[] | null>(null);
  const [snippetPickerOpen, setSnippetPickerOpen] = useState(false);
  const [messageSource, setMessageSource] = useState<{ snippet: SnippetDTO; rendered: string } | null>(null);
  const [filter, setFilter] = useState("");
  const flatTargets = useMemo(() => targets.flatMap((section) => section.targets), [targets]);

  useEffect(() => {
    if (!open) return;
    // Default to the only target when there is exactly one: the common case
    // should not cost a selection click.
    const only = flatTargets.length === 1 ? flatTargets[0] : undefined;
    const preset = initialSelection ?? (only ? [targetKey(only)] : []);
    setSelected(preset);
    setEdited({});
    setResults(null);
    setSending(false);
    setMessageSource(null);
    setSnippetPickerOpen(false);
    setFilter("");
  }, [open, initialSelection, flatTargets]);

  const query = filter.trim().toLocaleLowerCase();
  const visibleSections = useMemo(() => targets
    .map((section) => ({
      ...section,
      // "Somewhere new" is never filtered away. It is the escape hatch FROM a
      // list too long to read, so hiding it behind a query nothing matched
      // would remove the one row that always applies.
      targets: section.kind === "new" || !query
        ? section.targets
        : section.targets.filter((target) => target.label.toLocaleLowerCase().includes(query)),
    }))
    .filter((section) => section.targets.length > 0), [targets, query]);

  const noMatches = query.length > 0
    && visibleSections.every((section) => section.kind === "new");

  const selectedTargets = useMemo(
    () => flatTargets.filter((target) => selected.includes(targetKey(target))),
    [selected, flatTargets],
  );

  const textFor = (target: HandoffTarget): string => {
    const key = targetKey(target);
    return edited[key] ?? messageSource?.rendered ?? textForTarget(target, payload);
  };

  const toggle = (key: string) => {
    setSelected((prev) => (prev.includes(key) ? prev.filter((k) => k !== key) : [...prev, key]));
  };

  const handleSend = async () => {
    if (selectedTargets.length === 0 || sending) return;
    setSending(true);
    try {
      const outcome = await onSend(selectedTargets, textFor);
      setResults(outcome);
      // The text is kept on any non-delivery so the operator can retry
      // without retyping. Closing here would throw it away.
      if (outcome.every((r) => r.status === "sent")) onClose();
    } finally {
      setSending(false);
    }
  };

  if (!open) return null;

  const subheader = (
    <div
      className="flex flex-col gap-2"
      style={{ paddingInline: "var(--space-md)", paddingBlock: "var(--space-sm)" }}
    >
      {/* The source used to be interpolated into the dialog title, where a
          real session name wrapped it onto two lines and pushed the list
          down. It is context, not the heading. */}
      <p data-testid="handoff-source" className="truncate text-xs text-wc-text-muted" title={sourceLabel}>
        {t(strings.handoff.fromSource, { name: sourceLabel })}
      </p>
      {flatTargets.length > FILTER_THRESHOLD && (
        <InputGroup className="min-w-0" size="md" shape="rounded" testId="handoff-filter-group">
          <InputGroup.Adornment side="leading"><Search aria-hidden /></InputGroup.Adornment>
          <InputGroup.Field>
            <Input
              type="search"
              data-testid="handoff-filter"
              aria-label={t(strings.handoff.filterAria)}
              placeholder={t(strings.handoff.filterPlaceholder)}
              value={filter}
              onChange={(event) => { setFilter(event.target.value); }}
            />
          </InputGroup.Field>
          {query.length > 0 && (
            <InputGroup.Action>
              <IconButton
                type="button"
                data-testid="handoff-filter-clear"
                aria-label={t(strings.handoff.clearFilter)}
                onClick={() => { setFilter(""); }}
                shape="rounded"
                surface="ghost"
                size="sm"
              >
                <X className="h-4 w-4" aria-hidden />
              </IconButton>
            </InputGroup.Action>
          )}
        </InputGroup>
      )}
    </div>
  );

  const footer = (
    <div className="flex w-full items-center justify-end gap-2">
      <Button variant="outline" size="sm" onClick={onClose}>
        <X className="me-1.5 h-4 w-4" aria-hidden />
        {t(strings.handoff.close)}
      </Button>
      <Button
        data-testid="handoff-send"
        size="sm"
        onClick={() => { void handleSend(); }}
        disabled={selectedTargets.length === 0 || sending || snippetPickerOpen}
      >
        {sending
          ? <Loader2 className="me-1.5 h-4 w-4 animate-spin" aria-hidden />
          : <Send className="me-1.5 h-4 w-4" aria-hidden />}
        {sending
          ? t(strings.handoff.sending)
          // The count is on the button because fanning out to three sessions
          // is not something to discover from the results panel afterwards.
          : selectedTargets.length > 1
            ? t(strings.handoff.sendTo, { total: selectedTargets.length })
            : t(strings.handoff.send)}
      </Button>
    </div>
  );

  return (
    <ResponsiveDialog
      avoidKeyboard
      open
      onClose={onClose}
      size="md"
      closeLabel={t(strings.handoff.close)}
      title={t(strings.handoff.handOff)}
      testId="handoff-composer"
      subheader={subheader}
      footer={footer}
      contentPadding="comfortable"
    >
      <div className="space-y-4">
        <div className="flex items-baseline gap-2 rounded-lg border border-wc-default bg-wc-surface-base/50 px-3 py-2">
          <span className="shrink-0 text-[11px] font-semibold uppercase tracking-wider text-wc-text-faint">
            {t(strings.handoff.payloadLabel)}
          </span>
          <span className="min-w-0 flex-1 truncate text-sm text-wc-text-secondary" title={payload}>
            {payload || t(strings.handoff.noPayload)}
          </span>
        </div>

        {flatTargets.length === 0 ? (
          <p data-testid="handoff-no-targets" className="py-6 text-center text-sm text-wc-text-muted">
            {t(strings.handoff.noTargets)}
          </p>
        ) : (
          <>
            <div data-testid="handoff-targets" className="space-y-4">
              {noMatches && (
                <div data-testid="handoff-no-matches" className="flex items-center justify-between gap-2 px-1">
                  <span className="text-sm text-wc-text-muted">{t(strings.handoff.noMatches)}</span>
                  <button
                    type="button"
                    onClick={() => { setFilter(""); }}
                    className="shrink-0 text-xs text-wc-accent underline-offset-2 hover:underline"
                  >
                    {t(strings.handoff.clearFilter)}
                  </button>
                </div>
              )}
              {visibleSections.map((section) => (
                <section key={section.kind} data-testid={`handoff-section-${section.kind}`} className="space-y-2">
                  <div className="text-xs font-semibold uppercase tracking-wider text-wc-text-faint">
                    {t(section.labelKey as never)}
                  </div>
                  <div className="space-y-1.5">
                    {section.targets.map((target) => (
                      <TargetRow
                        key={targetKey(target)}
                        target={target}
                        checked={selected.includes(targetKey(target))}
                        onToggle={() => { toggle(targetKey(target)); }}
                        // Inside "this group" every row shares one group, so
                        // naming it on each row is noise. Elsewhere it is the
                        // most useful thing on the line.
                        showGroup={section.kind !== "group"}
                        waitingLabel={t(strings.handoff.waitingChip)}
                        startsFirstLabel={t(strings.handoff.startsFirst)}
                        unreadLabel={(total) => t(strings.handoff.unreadAria, { total })}
                      />
                    ))}
                  </div>
                </section>
              ))}
            </div>

            {/* One field for the common single-target case; one per target
                only when the selected targets carry different prompts, so
                the ordinary path does not pay for the fan-out path. */}
            {selectedTargets.length > 0 && (
              <section className="space-y-2">
                <div className="flex items-center justify-between gap-2">
                  <span className="text-xs font-semibold uppercase tracking-wider text-wc-text-faint">
                    {t(strings.handoff.message)}
                  </span>
                  {/* The snippet control sits ON the message it rewrites. It
                      used to live three sections above, next to nothing it
                      affected, which is what made it read as unrelated. */}
                  <div className="flex shrink-0 items-center gap-1">
                    <button
                      type="button"
                      data-testid="handoff-message-source"
                      onClick={() => { setSnippetPickerOpen(true); }}
                      className={cn(
                        "flex min-h-8 max-w-[14rem] items-center gap-1.5 rounded-full border px-2.5 text-xs transition-colors",
                        messageSource
                          ? "border-wc-accent/60 bg-wc-accent/10 text-wc-text-primary"
                          : "border-wc-default bg-wc-surface-input text-wc-text-secondary hover:border-wc-accent/50 hover:text-wc-text-primary",
                      )}
                    >
                      {messageSource
                        ? <span className="h-2 w-2 shrink-0 rounded-full" style={{ backgroundColor: messageSource.snippet.color || "currentColor" }} aria-hidden />
                        : <Plus className="h-3.5 w-3.5 shrink-0" aria-hidden />}
                      <span className="truncate">{messageSource?.snippet.name ?? t(strings.handoff.useSnippet)}</span>
                    </button>
                    {messageSource && (
                      <IconButton
                        type="button"
                        data-testid="handoff-message-source-clear"
                        aria-label={t(strings.handoff.clearSnippet)}
                        onClick={() => {
                          // Dropping the snippet restores what each target
                          // would have received, so edits made on top of it
                          // go too — leaving the snippet's words on screen
                          // under a control that says it is off is worse.
                          setMessageSource(null);
                          setEdited({});
                        }}
                        shape="rounded"
                        surface="ghost"
                        size="sm"
                      >
                        <X className="h-3.5 w-3.5" aria-hidden />
                      </IconButton>
                    )}
                  </div>
                </div>
                {selectedTargets.map((target) => {
                  const key = targetKey(target);
                  return (
                    <div key={key} className="space-y-1">
                      {selectedTargets.length > 1 && (
                        <div className="text-[11px] text-wc-text-faint">
                          {t(strings.handoff.messageFor, { label: target.label })}
                        </div>
                      )}
                      <Textarea
                        data-testid={selectedTargets.length === 1 ? "handoff-message" : `handoff-message-${key}`}
                        value={textFor(target)}
                        onChange={(event) => {
                          setEdited((prev) => ({ ...prev, [key]: event.target.value }));
                        }}
                        rows={selectedTargets.length > 1 ? 2 : 4}
                        className="text-sm"
                      />
                    </div>
                  );
                })}
              </section>
            )}
          </>
        )}

        {results && (
          <section data-testid="handoff-results" className="space-y-1.5 rounded-lg border border-wc-default bg-wc-surface-base/50 p-3">
            {results.map((result) => (
              <div
                key={result.targetId}
                data-testid={`handoff-result-${result.targetId}`}
                data-status={result.status}
                className={cn(
                  "text-xs",
                  result.status === "sent" && "text-emerald-300",
                  result.status === "queued" && "text-amber-300",
                  result.status === "failed" && "text-rose-300",
                )}
              >
                {result.status === "sent" && t(strings.handoff.resultSent, { label: result.label })}
                {result.status === "queued" && t(strings.handoff.resultQueued, { label: result.label })}
                {result.status === "failed" && t(strings.handoff.resultFailed, { label: result.label, reason: result.reason ?? "" })}
              </div>
            ))}
            {results.some((r) => r.status !== "sent") && (
              <p className="pt-1 text-[11px] text-wc-text-faint">{t(strings.handoff.keptText)}</p>
            )}
          </section>
        )}
      </div>
      <SnippetPicker
        open={snippetPickerOpen}
        onClose={() => { setSnippetPickerOpen(false); }}
        autoValues={{ payload, cwd, session: sourceSessionId ?? sourceLabel, selection }}
        onInsert={(rendered, snippet) => {
          // Existing edits have to go: `textFor` prefers `edited` over the
          // snippet, so choosing one after typing used to change nothing at
          // all and read as a dead control.
          setEdited({});
          setMessageSource({ rendered, snippet });
        }}
      />
    </ResponsiveDialog>
  );
}

interface TargetRowProps {
  target: HandoffTarget;
  checked: boolean;
  onToggle: () => void;
  showGroup: boolean;
  waitingLabel: string;
  startsFirstLabel: string;
  unreadLabel: (total: number) => string;
}

/**
 * One selectable target.
 *
 * The row IS the shared selection control, rather than a hand-built row with
 * a checkbox dropped inside it. That earlier shape nested a `<label>` in a
 * `<label>` — invalid, and expensive: the shared control is itself a whole
 * 44px row with its own padding, so an empty label reserved 44px of nothing
 * and every target stood 84px tall carrying one line of text.
 */
function TargetRow({
  target,
  checked,
  onToggle,
  showGroup,
  waitingLabel,
  startsFirstLabel,
  unreadLabel,
}: TargetRowProps) {
  const isWaiting = target.kind !== "session";
  const meta = target.meta;
  const accent = paneAccentStyle(meta?.color, meta?.groupColor, "bar");
  const unread = meta?.unreadCount ?? 0;
  const detail: ReactNode = isWaiting
    ? startsFirstLabel
    // Both halves are frequently the only thing separating two rows that both
    // read "/bin/bash". The workspace holds no richer description of a bare
    // terminal, and inventing one would be fiction.
    : [meta?.activityLabel, showGroup ? meta?.groupName : ""].filter(Boolean).join(" · ");

  return (
    <div
      data-testid={`handoff-target-${targetKey(target)}`}
      className={cn(
        "relative rounded-lg border transition-colors",
        checked
          ? "border-wc-accent bg-wc-accent/10"
          : "border-wc-default bg-wc-surface-input hover:border-wc-accent/50",
      )}
      // The shared row pads itself with --space-xs (12px) around a 44px tap
      // target, which is right for a settings list and generous for a
      // pick-list of twenty sessions. Narrowing the token HERE, where the
      // component reads it, keeps the touch floor intact and takes 8px off
      // every row without overriding one library selector.
      style={{ "--space-xs": "8px" } as CSSProperties}
    >
      {accent && (
        <span
          className="pointer-events-none absolute inset-y-2 start-1 w-1 rounded-full"
          style={accent}
          aria-hidden
        />
      )}
      <Checkbox
        className="w-full"
        checked={checked}
        onChange={onToggle}
        label={(
          <span className="flex min-w-0 items-center gap-2">
            <span className="min-w-0 flex-1 truncate">{target.label}</span>
            {unread > 0 && (
              <span
                className="shrink-0 rounded-full bg-wc-accent px-1.5 py-0.5 text-[10px] font-semibold text-wc-accent-fg"
                aria-label={unreadLabel(unread)}
              >
                {unread}
              </span>
            )}
            {isWaiting && (
              <span className="shrink-0 rounded border border-dashed border-wc-default px-1.5 py-0.5 text-[10px] font-normal uppercase tracking-wide text-wc-text-faint">
                {waitingLabel}
              </span>
            )}
          </span>
        )}
        description={detail || undefined}
      />
    </div>
  );
}
