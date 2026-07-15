import { useEffect, useState } from "react";

import { Button } from "../../components/ui/button";
import { type ComponentExample } from "../../api/components";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

interface PropsExperimentPanelProps {
  example?: ComponentExample;
  status?: "idle" | "applying" | "applied" | "error";
  message?: string;
  onApply: (props: Record<string, unknown>) => void;
  onReset: () => void;
}

function indexedProps(example?: ComponentExample): Record<string, unknown> {
  if (!example?.propsJson) return {};
  try {
    const parsed = JSON.parse(example.propsJson) as unknown;
    return parsed && typeof parsed === "object" && !Array.isArray(parsed)
      ? parsed as Record<string, unknown>
      : {};
  } catch {
    return {};
  }
}

/** A deliberately raw JSON editor: the catalog has no prop-type schema. */
export function PropsExperimentPanel({ example, status = "idle", message, onApply, onReset }: PropsExperimentPanelProps) {
  const { t } = useTranslation();
  const [draft, setDraft] = useState("{}");
  const [parseError, setParseError] = useState("");

  useEffect(() => {
    setDraft(JSON.stringify(indexedProps(example), null, 2));
    setParseError("");
  }, [example]);

  const apply = () => {
    try {
      const parsed = JSON.parse(draft) as unknown;
      if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
        setParseError(t(strings.components.editor.propsObjectError));
        return;
      }
      setParseError("");
      onApply(parsed as Record<string, unknown>);
    } catch {
      setParseError(t(strings.components.editor.propsJsonError));
    }
  };

  return (
    <section data-testid={selectors.components.editor.propsPanel} aria-label={t(strings.components.editor.tryProps)} className="mt-3 rounded-md border border-app-border bg-app-surface p-3">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <div>
          <h3 className="text-sm font-semibold text-app-foreground">{t(strings.components.editor.tryProps)}</h3>
          <p className="mt-1 text-xs text-app-muted-foreground">{t(strings.components.editor.tryPropsDescription)}</p>
        </div>
        {status === "applied" && <span className="text-xs text-app-success">{t(strings.components.editor.propsApplied)}</span>}
      </div>
      <label className="mt-3 block text-xs text-app-muted-foreground" htmlFor="rcl-props-experiment">
        {t(strings.components.editor.indexedPropsLabel)}
      </label>
      <textarea
        id="rcl-props-experiment"
        data-testid={selectors.components.editor.propsDraft}
        value={draft}
        onChange={(event) => setDraft(event.target.value)}
        spellCheck={false}
        className="mt-1 min-h-40 w-full resize-y rounded-md border border-app-border bg-app-background p-2 font-mono text-xs text-app-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
      />
      {(parseError || (status === "error" && message)) && <p data-testid={selectors.components.editor.propsError} className="mt-2 text-xs text-app-danger">{parseError || message}</p>}
      <div className="mt-3 flex gap-2">
        <Button data-testid={selectors.components.editor.propsApply} type="button" className="h-8 px-3 text-xs" disabled={status === "applying"} onClick={apply}>
          {status === "applying" ? t(strings.components.editor.applyingProps) : t(strings.components.editor.applyTemporaryProps)}
        </Button>
        <Button data-testid={selectors.components.editor.propsReset} type="button" variant="secondary" className="h-8 px-3 text-xs" onClick={onReset}>
          {t(strings.components.editor.resetIndexedProps)}
        </Button>
      </div>
    </section>
  );
}
