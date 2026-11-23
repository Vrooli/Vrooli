import { useState, type FormEvent } from "react";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { useNamedSelection, type NamedSelectionClient } from "./useNamedSelection";

/** Parts of a picture people name most often; each is a phrase the default model selects well. */
const QUICK_PHRASES = [
  { phrase: "hair", label: strings.workspace.mask.select.hair },
  { phrase: "jacket", label: strings.workspace.mask.select.jacket },
] as const;

export interface MaskSelectByNameProps {
  /** The picture to select in (the current canvas image), or null. */
  image: File | null;
  /** Receives the selection's mask and the phrase that produced it. */
  onSelected: (mask: File, phrase: string) => void;
  /** Injected seam for tests; the live service client otherwise. */
  client?: NamedSelectionClient;
}

/**
 * Name what to change instead of painting it: a typed phrase or a quick choice
 * runs a segmentation job and hands its mask to the edit. A phone can type a word
 * where a finger cannot trace hair.
 */
export function MaskSelectByName({ image, onSelected, client }: MaskSelectByNameProps) {
  const { t } = useTranslation();
  const selection = useNamedSelection(client);
  const [phrase, setPhrase] = useState("");

  const run = async (text: string) => {
    if (!image || selection.busy || !text.trim()) return;
    const mask = await selection.select(image, text);
    if (mask) onSelected(mask, text.trim());
  };

  const submit = (event: FormEvent) => {
    event.preventDefault();
    void run(phrase);
  };

  return (
    <form
      data-testid={selectors.workspace.mask.selectRoot}
      onSubmit={submit}
      className="flex flex-col gap-2"
      aria-label={t(strings.workspace.mask.select.title)}
    >
      <label className="flex flex-col gap-1 text-xs text-app-muted-foreground">
        <span>{t(strings.workspace.mask.select.title)}</span>
        <div className="flex gap-2">
          <input
            data-testid={selectors.workspace.mask.selectInput}
            type="text"
            value={phrase}
            maxLength={120}
            placeholder={t(strings.workspace.mask.select.placeholder)}
            onChange={(event) => setPhrase(event.target.value)}
            className="min-w-0 flex-1 rounded-control border border-app-border bg-app-surface px-2 py-2 text-sm text-app-foreground"
          />
          <button
            type="submit"
            data-testid={selectors.workspace.mask.selectRun}
            disabled={!image || selection.busy || !phrase.trim()}
            className="rounded-control bg-app-primary px-3 py-2 text-sm font-medium text-app-primary-foreground disabled:opacity-50"
          >
            {selection.busy ? t(strings.workspace.mask.select.busy) : t(strings.workspace.mask.select.run)}
          </button>
        </div>
      </label>
      <div className="flex flex-wrap gap-2" role="group" aria-label={t(strings.workspace.mask.select.quick)}>
        {QUICK_PHRASES.map((quick) => (
          <button
            key={quick.phrase}
            type="button"
            data-testid={selectors.workspace.maskQuickSelect({ phrase: quick.phrase })}
            disabled={!image || selection.busy}
            onClick={() => {
              setPhrase(quick.phrase);
              void run(quick.phrase);
            }}
            className="rounded-control border border-app-border px-3 py-2 text-sm text-app-foreground disabled:opacity-50"
          >
            {t(quick.label)}
          </button>
        ))}
      </div>
      {selection.busy && (
        <p data-testid={selectors.workspace.mask.selectBusy} role="status" className="text-xs text-app-muted-foreground">
          {t(strings.workspace.mask.select.working)}
        </p>
      )}
      {selection.error && (
        <p data-testid={selectors.workspace.mask.selectError} role="alert" className="text-xs text-app-danger">
          {selection.error}
        </p>
      )}
    </form>
  );
}
