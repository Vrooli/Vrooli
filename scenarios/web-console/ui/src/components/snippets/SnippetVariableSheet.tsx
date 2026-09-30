import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";

import type { SnippetDTO } from "../../api/snippets";
import { distinctVariables, renderSnippet } from "../../lib/snippetVars";
import { strings } from "../../consts/strings";

interface SnippetVariableSheetProps {
  open: boolean;
  snippet: SnippetDTO;
  autoValues: Record<string, string>;
  onClose: () => void;
  onInsert: (text: string) => void | Promise<void>;
}

export function SnippetVariableSheet({ open, snippet, autoValues, onClose, onInsert }: SnippetVariableSheetProps) {
  const { t } = useTranslation();
  const names = useMemo(() => distinctVariables(snippet.body), [snippet.body]);
  const [values, setValues] = useState<Record<string, string>>({});
  const [inserting, setInserting] = useState(false);
  useEffect(() => { if (open) setValues({}); }, [open, snippet.id]);
  const supplied = { ...autoValues, ...values };
  const preview = renderSnippet(snippet.body, supplied);
  const insertPreview = async () => {
    setInserting(true);
    try {
      await onInsert(preview);
    } finally {
      setInserting(false);
    }
  };

  if (!open) return null;
  return (
    <ResponsiveDialog open onClose={onClose} size="md" title={t(strings.snippets.variables.title, { name: snippet.name })} closeLabel={t(strings.snippets.close)} testId="snippet-variable-sheet" avoidKeyboard>
      <div className="space-y-4 p-4">
        {names.map((name) => Object.prototype.hasOwnProperty.call(autoValues, name) ? (
          <div key={name} data-testid={`snippet-variable-readonly-${name}`} className="flex min-h-11 items-center justify-between rounded-lg border border-wc-default px-3 text-sm">
            <span className="text-wc-text-muted">{name}</span><span className="text-wc-text-primary">{autoValues[name]}</span>
          </div>
        ) : (
          <label key={name} className="block space-y-1 text-xs text-wc-text-secondary">
            <span>{name}</span>
            <input data-testid={`snippet-variable-input-${name}`} value={values[name] ?? ""} onChange={(event) => { setValues((current) => ({ ...current, [name]: event.target.value })); }} className="min-h-11 w-full rounded-lg border border-wc-default bg-wc-surface-input px-3 text-sm text-wc-text-primary" />
          </label>
        ))}
        <div className="space-y-1">
          <div className="text-xs text-wc-text-secondary">{t(strings.snippets.variables.preview)}</div>
          <div data-testid="snippet-variable-preview" className="whitespace-pre-wrap rounded-lg border border-wc-default bg-wc-surface-base p-3 text-sm text-wc-text-primary"><mark className="bg-wc-accent/15 text-inherit">{preview}</mark></div>
        </div>
        <div className="flex justify-end gap-2">
          <button type="button" onClick={onClose} className="min-h-11 rounded-lg px-4 text-sm text-wc-text-secondary">{t(strings.snippets.cancel)}</button>
          <button type="button" data-testid="snippet-variable-insert" disabled={inserting} onClick={() => { void insertPreview(); }} className="min-h-11 rounded-lg bg-wc-accent/20 px-4 text-sm font-medium text-wc-text-primary">{t(strings.snippets.variables.insert)}</button>
        </div>
      </div>
    </ResponsiveDialog>
  );
}
