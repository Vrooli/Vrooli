import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";

import { HEADER_COLORS } from "../../consts/config";
import { strings } from "../../consts/strings";
import { cn } from "../../lib/classnames";
import { useSnippets } from "../../hooks/useSnippets";
import type { SnippetDTO } from "../../api/snippets";
import { SnippetBodyEditor } from "./SnippetBodyEditor";

export function defaultSnippetName(body: string): string {
  const first = (body.split(/\r?\n/, 1)[0] ?? "")
    .replace(/^\s*(?:[-*+]\s+|#{1,6}\s+)/, "")
    .trim();
  if (first.length <= 40) return first;
  const candidate = first.slice(0, 40);
  const boundary = candidate.lastIndexOf(" ");
  return (boundary >= 20 ? candidate.slice(0, boundary) : candidate).trimEnd();
}

interface SnippetSaveSheetProps {
  open: boolean;
  onClose: () => void;
  mode: "create" | "edit";
  initialBody: string;
  initialName?: string;
  initialColor?: string;
  sourceLabel?: string;
  snippet?: SnippetDTO;
  onSaved?: (snippet: SnippetDTO) => void;
}

export function SnippetSaveSheet({
  open,
  onClose,
  mode,
  initialBody,
  initialName,
  initialColor,
  sourceLabel,
  snippet,
  onSaved,
}: SnippetSaveSheetProps) {
  const { t } = useTranslation();
  const { save } = useSnippets();
  const [name, setName] = useState("");
  const [body, setBody] = useState("");
  const [color, setColor] = useState<string>(HEADER_COLORS[0]);
  const [saveError, setSaveError] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!open) return;
    setName(initialName ?? snippet?.name ?? defaultSnippetName(initialBody));
    setBody(initialBody);
    const preferred = initialColor ?? snippet?.color;
    setColor(
      preferred && HEADER_COLORS.includes(preferred as typeof HEADER_COLORS[number])
        ? preferred
        : HEADER_COLORS[0],
    );
    setSaveError("");
  }, [open, initialBody, initialName, initialColor, snippet]);

  const handleSave = async () => {
    if (saving) return;
    setSaving(true);
    setSaveError("");
    try {
      const saved = await save({
        id: mode === "edit" ? snippet?.id : undefined,
        name,
        body,
        color,
        pinned: mode === "edit" ? snippet?.pinned : undefined,
        sort_order: snippet?.sort_order,
      });
      onSaved?.(saved);
      onClose();
    } catch {
      setSaveError(t(strings.snippets.save.error));
    } finally {
      setSaving(false);
    }
  };

  if (!open) return null;
  return (
    <ResponsiveDialog open onClose={onClose} size="md" title={t(mode === "edit" ? strings.snippets.save.titleEdit : strings.snippets.save.titleCreate)} closeLabel={t(strings.snippets.close)} testId="snippet-save-sheet" avoidKeyboard>
      <div className="space-y-4 p-4">
        {sourceLabel && <p className="text-xs text-wc-text-muted">{sourceLabel}</p>}
        <label className="block space-y-1 text-xs text-wc-text-secondary">
          <span>{t(strings.snippets.name)}</span>
          <input data-testid="snippet-save-name" value={name} onChange={(event) => { setName(event.target.value); }} className="min-h-11 w-full rounded-lg border border-wc-default bg-wc-surface-input px-3 text-sm text-wc-text-primary" />
        </label>
        <div>
          <div className="mb-1 text-xs text-wc-text-secondary">{t(strings.snippets.color)}</div>
          <div className="flex flex-wrap gap-1.5">
            {HEADER_COLORS.map((swatch) => (
              <button key={swatch} type="button" data-testid={`snippet-color-${swatch}`} aria-label={t(strings.snippets.chooseColor, { color: swatch })} aria-pressed={color === swatch} onClick={() => { setColor(swatch); }} className={cn("h-11 w-11 rounded-full border border-wc-default", color === swatch && "ring-2 ring-wc-accent")} style={{ backgroundColor: swatch }} />
            ))}
          </div>
        </div>
        <label className="block space-y-1 text-xs text-wc-text-secondary">
          <span>{t(strings.snippets.body)}</span>
          <SnippetBodyEditor body={body} onChange={setBody} />
        </label>
        {saveError && <p data-testid="snippet-save-error" className="text-xs text-red-400">{saveError}</p>}
        <div className="flex justify-end gap-2">
          <button type="button" onClick={onClose} className="min-h-11 rounded-lg px-4 text-sm text-wc-text-secondary">{t(strings.snippets.cancel)}</button>
          <button type="button" data-testid="snippet-save-submit" disabled={saving || !name.trim()} onClick={() => void handleSave()} className="min-h-11 rounded-lg bg-wc-accent/20 px-4 text-sm font-medium text-wc-text-primary disabled:opacity-50">{t(saving ? strings.snippets.save.saving : strings.snippets.save.save)}</button>
        </div>
      </div>
    </ResponsiveDialog>
  );
}
