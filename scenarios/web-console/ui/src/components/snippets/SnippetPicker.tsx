import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Braces, Clock, List, Pencil, Pin, PinOff, Plus, Search, X } from "lucide-react";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { IconButton } from "@vrooli/react-component-library/IconButton";
import { Input } from "@vrooli/react-component-library/Input/1";
import { InputGroup } from "@vrooli/react-component-library/InputGroup";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";
import { SwipeActions, type SwipeAction } from "@vrooli/react-component-library/SwipeActions";
import { Tabs } from "@vrooli/react-component-library/Tabs/1";

import type { SnippetDTO } from "../../api/snippets";
import { useSnippets } from "../../hooks/useSnippets";
import { strings } from "../../consts/strings";
import { cn } from "../../lib/classnames";
import { distinctVariables, renderSnippet } from "../../lib/snippetVars";
import { formatRelativeTime } from "../MessageJumpList.helpers";
import { SnippetSaveSheet } from "./SnippetSaveSheet";
import { SnippetVariableSheet } from "./SnippetVariableSheet";

type Segment = "recent" | "pinned" | "all";

/** How many entries the Recent list shows before it stops being "recent". */
const RECENT_LIMIT = 8;

interface SnippetPickerProps {
  open: boolean;
  onClose: () => void;
  onInsert: (text: string, snippet: SnippetDTO) => void | Promise<void>;
  autoValues?: Record<string, string>;
  onNew?: () => void;
}

export function SnippetPicker({ open, onClose, onInsert, autoValues = {}, onNew }: SnippetPickerProps) {
  const { t } = useTranslation();
  const { snippets, status, touch, save } = useSnippets();
  const [filter, setFilter] = useState("");
  const [segment, setSegment] = useState<Segment>("recent");
  const [pending, setPending] = useState<SnippetDTO | null>(null);
  const [editing, setEditing] = useState<SnippetDTO | null>(null);
  // Only one row may hold its action track open, so a second swipe closes the
  // first rather than leaving two rows displaced with no way back.
  const [swipeOpenId, setSwipeOpenId] = useState<string | null>(null);
  const [actionError, setActionError] = useState("");

  const matches = useMemo(() => {
    const query = filter.trim().toLocaleLowerCase();
    if (!query) return snippets;
    return snippets.filter((snippet) =>
      snippet.name.toLocaleLowerCase().includes(query) || snippet.body.toLocaleLowerCase().includes(query));
  }, [filter, snippets]);

  const pinnedMatches = useMemo(() => matches.filter((snippet) => snippet.pinned), [matches]);
  const visible = useMemo(() => {
    if (segment === "pinned") return pinnedMatches;
    if (segment === "recent") return matches.slice(0, RECENT_LIMIT);
    return matches;
  }, [matches, pinnedMatches, segment]);

  const closePicker = () => {
    setPending(null);
    onClose();
  };

  const insert = async (text: string, snippet: SnippetDTO) => {
    await onInsert(text, snippet);
    await touch(snippet.id);
    setPending(null);
    onClose();
  };
  const select = async (snippet: SnippetDTO) => {
    const unresolved = distinctVariables(snippet.body).filter((name) => !Object.prototype.hasOwnProperty.call(autoValues, name));
    if (unresolved.length > 0) {
      setPending(snippet);
      return;
    }
    await insert(renderSnippet(snippet.body, autoValues), snippet);
  };

  const togglePin = async (snippet: SnippetDTO) => {
    setActionError("");
    try {
      await save({
        id: snippet.id,
        name: snippet.name,
        body: snippet.body,
        color: snippet.color,
        pinned: !snippet.pinned,
        sort_order: snippet.sort_order,
      });
    } catch {
      setActionError(t(strings.snippets.settings.pinError));
    }
  };

  // The swipe is an accelerator, never the only route: every action here is
  // also reachable from the Snippets settings panel, which is what keyboard
  // and screen-reader users get.
  const swipeActionsFor = (snippet: SnippetDTO): SwipeAction[] => [
    {
      id: "pin",
      label: t(snippet.pinned ? strings.snippets.settings.unpin : strings.snippets.settings.pin),
      icon: snippet.pinned ? <PinOff className="h-4 w-4" aria-hidden /> : <Pin className="h-4 w-4" aria-hidden />,
      tone: "primary",
      onSelect: () => {
        setSwipeOpenId(null);
        void togglePin(snippet);
      },
    },
    {
      id: "edit",
      label: t(strings.snippets.picker.edit),
      icon: <Pencil className="h-4 w-4" aria-hidden />,
      onSelect: () => {
        setSwipeOpenId(null);
        setEditing(snippet);
      },
    },
  ];

  // The tab badge is `aria-hidden` in the shared strip, so the count also goes
  // into the label as screen-reader-only text. Without it, moving the count out
  // of "All (3)" and into a badge would have silently dropped it for assistive
  // technology. A zero badge is omitted rather than shown as an empty pill.
  const tabLabel = (label: string, count: number) => (
    <>
      {label}
      <span className="sr-only"> ({count})</span>
    </>
  );
  const tabItems = [
    { id: "recent", label: tabLabel(t(strings.snippets.picker.recent), Math.min(matches.length, RECENT_LIMIT)), icon: <Clock className="h-4 w-4" aria-hidden />, badge: Math.min(matches.length, RECENT_LIMIT) || undefined },
    { id: "pinned", label: tabLabel(t(strings.snippets.picker.pinned), pinnedMatches.length), icon: <Pin className="h-4 w-4" aria-hidden />, badge: pinnedMatches.length || undefined },
    { id: "all", label: tabLabel(t(strings.snippets.picker.all), matches.length), icon: <List className="h-4 w-4" aria-hidden />, badge: matches.length || undefined },
  ];

  const filtering = filter.trim().length > 0;

  if (!open) return null;
  // Search and the list selector are one band in `subheader`: full-bleed,
  // pinned above the scroll region, and never scrolled away from the list
  // they filter.
  return (
    <>
      <ResponsiveDialog
        open
        onClose={closePicker}
        size="md"
        title={t(strings.snippets.picker.title)}
        closeLabel={t(strings.snippets.close)}
        testId="snippet-picker"
        avoidKeyboard
        contentPadding="none"
        headerActions={onNew && (
          <IconButton
            type="button"
            data-testid="snippet-new"
            aria-label={t(strings.snippets.picker.newAria)}
            onClick={onNew}
            shape="rounded"
            surface="ghost"
            size="md"
          >
            <Plus className="h-4 w-4" aria-hidden />
          </IconButton>
        )}
        subheader={(
          <div
            className="flex flex-col gap-2"
            style={{ paddingInline: "var(--space-md)", paddingBlock: "var(--space-sm)" }}
          >
            <InputGroup size="md" shape="rounded" testId="snippet-filter-group">
              <InputGroup.Adornment side="leading">
                <Search aria-hidden />
              </InputGroup.Adornment>
              <InputGroup.Field>
                <Input
                  type="search"
                  aria-label={t(strings.snippets.picker.filterAria)}
                  data-testid="snippet-filter"
                  value={filter}
                  onChange={(event) => { setFilter(event.target.value); }}
                  placeholder={t(strings.snippets.picker.filterPlaceholder)}
                />
              </InputGroup.Field>
              {filtering && (
                <InputGroup.Action>
                  <IconButton
                    type="button"
                    data-testid="snippet-filter-clear"
                    aria-label={t(strings.snippets.picker.clearFilter)}
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
            <Tabs
              mode="controlled"
              items={tabItems}
              active={segment}
              onChange={(value) => { setSegment(value as Segment); }}
              ariaLabel={t(strings.snippets.picker.tabsAria)}
              itemTestId={(value) => `snippet-segment-${value}`}
            />
          </div>
        )}
      >
        {/* One scroller, not two: the dialog's own content region scrolls, and
            the band above it does not, so the list must not nest a scroller of
            its own inside it. */}
        <div
          className="space-y-1.5"
          style={{ paddingInline: "var(--space-md)", paddingBlock: "var(--space-sm)" }}
        >
          {actionError && <p data-testid="snippet-action-error" className="px-1 text-xs text-red-400">{actionError}</p>}
          <div className="space-y-1.5" data-testid="snippet-list">
            {status === "loading" && <p className="p-3 text-sm text-wc-text-muted">{t(strings.snippets.picker.loading)}</p>}
            {status === "request-error" && <p className="p-3 text-sm text-red-400">{t(strings.snippets.picker.loadError)}</p>}
            {status === "ready" && visible.length === 0 && (
              <EmptyState
                className="py-8"
                title={t(filtering ? strings.snippets.picker.noMatches : strings.snippets.picker.empty)}
                description={t(filtering ? strings.snippets.picker.noMatchesDescription : strings.snippets.picker.emptyDescription)}
                icon={<Search aria-hidden />}
                actionLabel={filtering ? t(strings.snippets.picker.clearFilter) : (onNew ? t(strings.snippets.picker.new) : undefined)}
                onAction={filtering ? () => { setFilter(""); } : onNew}
              />
            )}
            {visible.map((snippet) => (
              <SnippetRow
                key={snippet.id}
                snippet={snippet}
                actions={swipeActionsFor(snippet)}
                swipeLabel={t(strings.snippets.picker.swipeActionsAria, { name: snippet.name })}
                lastUsedLabel={(when: string) => t(strings.snippets.picker.lastUsedAria, { when })}
                variablesLabel={(count: number) => t(strings.snippets.variableCount, { count })}
                open={swipeOpenId === snippet.id}
                onOpenChange={(next) => { setSwipeOpenId(next ? snippet.id : null); }}
                onSelect={() => void select(snippet)}
              />
            ))}
          </div>
        </div>
      </ResponsiveDialog>
      {pending && <SnippetVariableSheet open snippet={pending} autoValues={autoValues} onClose={() => { setPending(null); }} onInsert={(text) => insert(text, pending)} />}
      {editing && (
        <SnippetSaveSheet
          open
          mode="edit"
          snippet={editing}
          initialBody={editing.body}
          initialName={editing.name}
          initialColor={editing.color}
          onClose={() => { setEditing(null); }}
          onSaved={() => { setEditing(null); }}
        />
      )}
    </>
  );
}

interface SnippetRowProps {
  snippet: SnippetDTO;
  actions: SwipeAction[];
  swipeLabel: string;
  lastUsedLabel: (when: string) => string;
  variablesLabel: (count: number) => string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSelect: () => void;
}

function SnippetRow({ snippet, actions, swipeLabel, lastUsedLabel, variablesLabel, open, onOpenChange, onSelect }: SnippetRowProps) {
  const preview = snippet.body.replace(/\s+/g, " ").trim();
  const variableCount = distinctVariables(snippet.body).length;
  const lastUsed = formatRelativeTime(snippet.last_used_at);

  return (
    <SwipeActions
      actions={actions}
      label={swipeLabel}
      // `rest` because the two actions are heterogeneous: a past-threshold
      // release should show the choice, not guess which one was meant.
      releaseMode="rest"
      open={open}
      onOpenChange={onOpenChange}
      // The track is clipped to the wrapper, so its corners must match the
      // row's or the action surface shows through them.
      className="rounded-lg"
      testId={`snippet-swipe-${snippet.id}`}
    >
      <button
        type="button"
        data-testid={`snippet-row-${snippet.id}`}
        onClick={onSelect}
        className={cn(
          "flex min-h-11 w-full items-center gap-3 rounded-lg border border-wc-default bg-wc-surface-raised px-3 py-2.5 text-left",
          "transition-colors hover:border-wc-accent/60 hover:bg-wc-surface-input",
        )}
      >
        <span
          className="h-2.5 w-2.5 shrink-0 rounded-full"
          style={{ backgroundColor: snippet.color || "currentColor" }}
          aria-hidden
        />
        <span className="min-w-0 flex-1">
          <span className="flex items-center gap-1.5">
            <span className="truncate text-sm font-medium text-wc-text-primary">{snippet.name}</span>
            {snippet.pinned && <Pin className="h-3 w-3 shrink-0 text-wc-accent" aria-hidden />}
          </span>
          {preview && <span className="mt-0.5 block truncate text-xs text-wc-text-muted">{preview}</span>}
        </span>
        <span className="flex shrink-0 flex-col items-end gap-0.5 text-[11px] text-wc-text-faint">
          {variableCount > 0 && (
            <span
              className="flex items-center gap-0.5 tabular-nums"
              title={variablesLabel(variableCount)}
              aria-label={variablesLabel(variableCount)}
            >
              <Braces className="h-3 w-3" aria-hidden />
              {variableCount}
            </span>
          )}
          {lastUsed && (
            <span className="tabular-nums" title={lastUsedLabel(lastUsed)} aria-label={lastUsedLabel(lastUsed)}>
              {lastUsed}
            </span>
          )}
        </span>
      </button>
    </SwipeActions>
  );
}
