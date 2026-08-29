import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";

import type { SnippetDTO } from "../../api/snippets";
import { useSnippets } from "../../hooks/useSnippets";
import { strings } from "../../consts/strings";
import { cn } from "../../lib/classnames";
import { distinctVariables, renderSnippet } from "../../lib/snippetVars";
import { SnippetVariableSheet } from "./SnippetVariableSheet";

type Segment = "recent" | "pinned" | "all";

interface SnippetPickerProps {
  open: boolean;
  onClose: () => void;
  onInsert: (text: string, snippet: SnippetDTO) => void | Promise<void>;
  autoValues?: Record<string, string>;
  onNew?: () => void;
}

export function SnippetPicker({ open, onClose, onInsert, autoValues = {}, onNew }: SnippetPickerProps) {
  const { t } = useTranslation();
  const { snippets, status, touch } = useSnippets();
  const [filter, setFilter] = useState("");
  const [segment, setSegment] = useState<Segment>("recent");
  const [pending, setPending] = useState<SnippetDTO | null>(null);
  const visible = useMemo(() => {
    const query = filter.trim().toLocaleLowerCase();
    const matches = snippets.filter((snippet) => !query || snippet.name.toLocaleLowerCase().includes(query) || snippet.body.toLocaleLowerCase().includes(query));
    if (segment === "pinned") return matches.filter((snippet) => snippet.pinned);
    if (segment === "recent") return matches.slice(0, 8);
    return matches;
  }, [filter, segment, snippets]);

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

  if (!open) return null;
  return (
    <>
      <ResponsiveDialog open onClose={closePicker} size="md" title={t(strings.snippets.picker.title)} closeLabel={t(strings.snippets.close)} dismissAffordance="close" testId="snippet-picker" avoidKeyboard>
        <div className="flex h-full flex-col gap-3 p-4">
          <div className="flex items-center gap-2">
            <input aria-label={t(strings.snippets.picker.filterAria)} data-testid="snippet-filter" value={filter} onChange={(event) => { setFilter(event.target.value); }} placeholder={t(strings.snippets.picker.filterPlaceholder)} className="min-h-11 min-w-0 flex-1 rounded-lg border border-wc-default bg-wc-surface-input px-3 text-sm text-wc-text-primary" />
            {onNew && <button type="button" data-testid="snippet-new" onClick={onNew} className="min-h-11 rounded-lg px-3 text-sm font-medium text-wc-accent hover:bg-wc-surface-input">{t(strings.snippets.picker.new)}</button>}
          </div>
          <div className="grid grid-cols-3 gap-1 rounded-lg bg-wc-surface-base p-1">
            {(["recent", "pinned", "all"] as const).map((value) => <button key={value} type="button" data-testid={`snippet-segment-${value}`} aria-pressed={segment === value} onClick={() => { setSegment(value); }} className={cn("min-h-11 rounded-md px-2 text-xs text-wc-text-secondary", segment === value && "bg-wc-surface-input text-wc-text-primary")}>{value === "all" ? t(strings.snippets.picker.all, { count: snippets.length }) : t(strings.snippets.picker[value])}</button>)}
          </div>
          <div className="min-h-0 flex-1 space-y-1 overflow-y-auto" data-testid="snippet-list">
            {status === "loading" && <p className="p-3 text-sm text-wc-text-muted">{t(strings.snippets.picker.loading)}</p>}
            {status === "request-error" && <p className="p-3 text-sm text-red-400">{t(strings.snippets.picker.loadError)}</p>}
            {visible.map((snippet) => <button key={snippet.id} type="button" data-testid={`snippet-row-${snippet.id}`} onClick={() => void select(snippet)} className="flex min-h-11 w-full items-center gap-3 rounded-lg border border-wc-default px-3 py-2 text-left hover:border-wc-accent/60">
              <span className="h-3 w-3 shrink-0 rounded-full" style={{ backgroundColor: snippet.color || "currentColor" }} />
              <span className="min-w-0 flex-1"><span className="block truncate text-sm font-medium text-wc-text-primary">{snippet.name}</span><span className="block truncate text-xs text-wc-text-muted">{snippet.body.replace(/\s+/g, " ")}</span></span>
              <span className="text-[11px] text-wc-text-faint">{snippet.use_count}</span>
            </button>)}
          </div>
        </div>
      </ResponsiveDialog>
      {pending && <SnippetVariableSheet open snippet={pending} autoValues={autoValues} onClose={() => { setPending(null); }} onInsert={(text) => insert(text, pending)} />}
    </>
  );
}
