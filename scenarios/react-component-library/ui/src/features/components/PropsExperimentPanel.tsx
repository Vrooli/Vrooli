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

type ControlKind = "text" | "number" | "boolean" | "select";
interface ControlDefinition { key: string; label?: string; kind: ControlKind; options?: string[]; group?: string; advanced?: boolean }

function controlsFor(example?: ComponentExample): ControlDefinition[] {
  if (!example?.controlsJson) return [];
  try {
    const parsed = JSON.parse(example.controlsJson) as { fields?: unknown };
    if (!Array.isArray(parsed.fields)) return [];
    return parsed.fields.filter((field): field is ControlDefinition => Boolean(
      field && typeof field === "object" && typeof (field as ControlDefinition).key === "string" &&
      ["text", "number", "boolean", "select"].includes((field as ControlDefinition).kind),
    ));
  } catch { return []; }
}

function indexedProps(example?: Pick<ComponentExample, "propsJson">): Record<string, unknown> {
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

export function PropsExperimentPanel({ example, status = "idle", message, onApply, onReset }: PropsExperimentPanelProps) {
  const { t } = useTranslation();
  const [draft, setDraft] = useState("{}");
  const [parseError, setParseError] = useState("");
  const controls = controlsFor(example);
  const [showAdvanced, setShowAdvanced] = useState(controls.length === 0);
  const values = indexedProps({ ...example, propsJson: draft });

  useEffect(() => {
    setDraft(JSON.stringify(indexedProps(example), null, 2));
    setParseError("");
    setShowAdvanced(controlsFor(example).length === 0);
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

  const setValue = (key: string, value: unknown) => {
    const next = { ...values, [key]: value };
    setDraft(JSON.stringify(next, null, 2));
    setParseError("");
  };

  return (
    <section data-testid={selectors.components.editor.propsPanel} aria-label={t(strings.components.editor.tryProps)} className="mt-3 rounded-md border border-app-border bg-app-surface p-3">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <div>
          <h3 className="text-sm font-semibold text-app-foreground">{t(strings.components.editor.tryProps)}</h3>
          <p className="mt-1 text-xs text-app-muted-foreground">Temporary changes affect only {example?.displayName || "this state"} and disappear when you reset, reload, or leave.</p>
        </div>
        {status === "applied" && <span className="text-xs text-app-success">{t(strings.components.editor.propsApplied)}</span>}
      </div>
      {controls.filter((control) => !control.advanced).map((control) => (
        <label key={control.key} className="mt-3 block text-xs font-medium text-app-foreground">
          {control.label || control.key}
          {control.kind === "boolean" ? (
            <input className="ml-2 h-4 w-4 align-middle" type="checkbox" checked={Boolean(values[control.key])} onChange={(event) => setValue(control.key, event.target.checked)} />
          ) : control.kind === "select" ? (
            <select className="mt-1 h-9 w-full rounded-md border border-app-border bg-app-background px-2 text-sm" value={String(values[control.key] ?? "")} onChange={(event) => setValue(control.key, event.target.value)}>
              {(control.options ?? []).map((option) => <option key={option} value={option}>{option}</option>)}
            </select>
          ) : (
            <input className="mt-1 h-9 w-full rounded-md border border-app-border bg-app-background px-2 text-sm" type={control.kind === "number" ? "number" : "text"} value={String(values[control.key] ?? "")} onChange={(event) => setValue(control.key, control.kind === "number" ? Number(event.target.value) : event.target.value)} />
          )}
        </label>
      ))}
      {controls.length === 0 && <p className="mt-3 text-xs text-app-muted-foreground">This state has no declared quick controls. Advanced JSON is available when you need it.</p>}
      {controls.length > 0 && <Button type="button" variant="ghost" className="mt-3 h-8 px-0 text-xs text-app-primary" aria-expanded={showAdvanced} onClick={() => setShowAdvanced((visible) => !visible)}>Advanced JSON</Button>}
      {showAdvanced && <>
        <label className="mt-2 block text-xs text-app-muted-foreground" htmlFor="rcl-props-experiment">{t(strings.components.editor.indexedPropsLabel)}</label>
        <textarea id="rcl-props-experiment" data-testid={selectors.components.editor.propsDraft} value={draft} onChange={(event) => setDraft(event.target.value)} spellCheck={false} className="mt-1 min-h-40 w-full resize-y rounded-md border border-app-border bg-app-background p-2 font-mono text-xs text-app-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50" />
      </>}
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
