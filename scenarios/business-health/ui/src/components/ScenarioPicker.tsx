import { useState, type FormEvent } from "react";

import { Button } from "./ui/button";
import { Input } from "./ui/input";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

export interface ScenarioPickerProps {
  /** Called with a trimmed, non-empty slug on submit or recent-chip click. */
  readonly onSelect: (scenario: string) => void;
  /** Most-recent-first slugs to offer as one-click chips. */
  readonly recents: readonly string[];
  /** Clear the persisted recents. */
  readonly onClearRecents: () => void;
  /** Seed the input (e.g. the currently loaded scenario). */
  readonly initialValue?: string;
}

/**
 * Reusable scenario chooser: a text input + submit plus a row of recent-slug
 * chips. Owns only its input draft; the parent owns the "active scenario" and
 * the query it drives. Shared by the matrix, wizard, and findings surfaces so
 * the entry affordance is identical everywhere.
 */
export function ScenarioPicker({
  onSelect,
  recents,
  onClearRecents,
  initialValue = "",
}: ScenarioPickerProps) {
  const { t } = useTranslation();
  const [draft, setDraft] = useState(initialValue);

  const submit = (event: FormEvent) => {
    event.preventDefault();
    const slug = draft.trim();
    if (slug) onSelect(slug);
  };

  return (
    <div className="flex flex-col gap-2">
      <form
        data-testid={selectors.scenarioPicker.form}
        onSubmit={submit}
        className="flex flex-wrap items-end gap-2"
      >
        <label className="flex min-w-0 flex-1 flex-col gap-1">
          <span className="text-xs font-medium uppercase tracking-wide text-app-muted-foreground">
            {t(strings.common.scenarioLabel)}
          </span>
          <Input
            data-testid={selectors.scenarioPicker.input}
            value={draft}
            onChange={(event) => setDraft(event.target.value)}
            placeholder={t(strings.common.scenarioPlaceholder)}
            autoComplete="off"
            spellCheck={false}
            className="border-app-border bg-app-surface text-app-foreground placeholder:text-app-muted-foreground focus:ring-app-focus"
          />
        </label>
        <Button data-testid={selectors.scenarioPicker.submit} type="submit">
          {t(strings.common.load)}
        </Button>
      </form>
      {recents.length > 0 && (
        <div
          data-testid={selectors.scenarioPicker.recent}
          className="flex flex-wrap items-center gap-2"
        >
          <span className="text-xs uppercase tracking-wide text-app-muted-foreground">
            {t(strings.common.recentLabel)}
          </span>
          {recents.map((slug) => (
            <button
              key={slug}
              type="button"
              data-testid={selectors.scenarioPicker.recentItem({ scenario: slug })}
              onClick={() => {
                setDraft(slug);
                onSelect(slug);
              }}
              className="rounded-pill border border-app-border bg-app-surface-muted px-3 py-1 font-mono text-xs text-app-foreground transition-colors hover:bg-app-surface"
            >
              {slug}
            </button>
          ))}
          <button
            type="button"
            data-testid={selectors.scenarioPicker.clear}
            onClick={onClearRecents}
            className="rounded-control px-2 py-1 text-xs text-app-muted-foreground underline-offset-2 hover:text-app-foreground hover:underline"
          >
            {t(strings.common.clearRecent)}
          </button>
        </div>
      )}
    </div>
  );
}
