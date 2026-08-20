/** @vrooliComponentSource data-display.description-list */
import { useEffect, useMemo, useState } from "react";

import { Button } from "../../components/Button";
import { type ComponentStory } from "../../api/components";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

const EMPTY_RECORD: Record<string, unknown> = {};

interface PropsExperimentPanelProps {
  storyId?: string;
  storyName?: string;
  storyDescription?: string;
  initialArgs?: Record<string, unknown>;
  initialEnvironment?: Record<string, string>;
  storyContract?: ComponentStory;
  status?: "idle" | "applying" | "applied" | "error";
  message?: string;
  onApply: (props: Record<string, unknown>, environment?: Record<string, string>) => void;
  onReset: () => void;
}

type ControlKind = "text" | "number" | "boolean" | "select" | "json";
interface ControlDefinition {
  key: string;
  label: string;
  kind: ControlKind;
  options?: unknown[];
  minimum?: number;
  maximum?: number;
  minLength?: number;
  maxLength?: number;
  required?: boolean;
  format?: string;
  defaultValue?: unknown;
  visibleWhen?: { path: string; equals: unknown };
}

interface EnvironmentControl {
  key: string;
  label: string;
  adapter: string;
  options: string[];
}

function parseJsonObject(raw: string): Record<string, unknown> | undefined {
  try {
    const parsed: unknown = JSON.parse(raw);
    return parsed && typeof parsed === "object" && !Array.isArray(parsed)
      ? (parsed as Record<string, unknown>)
      : undefined;
  } catch {
    return undefined;
  }
}

function controlsFor(storyContract?: ComponentStory): ControlDefinition[] {
  if (!storyContract?.argsJson) return [];
  const parsed = parseJsonObject(storyContract.argsJson);
  if (!Array.isArray(parsed?.fields)) return [];

  return parsed.fields.flatMap((field) => {
    if (!field || typeof field !== "object" || Array.isArray(field)) return [];
    const value = field as Record<string, unknown>;
    if (typeof value.path !== "string" || !value.path.trim()) return [];
    const kind = value.kind === "enum" ? "select" : value.kind;
    const controlKind = ["object", "array", "structured"].includes(String(kind))
      ? "json"
      : kind;
    if (!["text", "number", "boolean", "select", "json"].includes(String(controlKind)))
      return [];
    const visibleWhen = value.visibleWhen;
    return [
      {
        key: value.path,
        label: typeof value.label === "string" && value.label.trim() ? value.label : value.path,
        kind: controlKind as ControlKind,
        options: Array.isArray(value.options) ? value.options : undefined,
        minimum: typeof value.minimum === "number" ? value.minimum : undefined,
        maximum: typeof value.maximum === "number" ? value.maximum : undefined,
        minLength: typeof value.minLength === "number" ? value.minLength : undefined,
        maxLength: typeof value.maxLength === "number" ? value.maxLength : undefined,
        required: value.required === true,
        format: typeof value.format === "string" ? value.format : undefined,
        defaultValue: value.default,
        visibleWhen:
          visibleWhen &&
          typeof visibleWhen === "object" &&
          !Array.isArray(visibleWhen) &&
          typeof (visibleWhen as Record<string, unknown>).path === "string"
            ? {
                path: (visibleWhen as Record<string, unknown>).path as string,
                equals: (visibleWhen as Record<string, unknown>).equals,
              }
            : undefined,
      },
    ];
  });
}

function storyArgs(
  storyContract?: ComponentStory,
  storyID?: string,
): Record<string, unknown> | undefined {
  if (!storyContract?.storiesJson || !storyID) return undefined;
  try {
    const stories = JSON.parse(storyContract.storiesJson) as Array<{
      id?: unknown;
      args?: unknown;
    }>;
    const story = stories.find((candidate) => candidate?.id === storyID);
    return story?.args && typeof story.args === "object" && !Array.isArray(story.args)
      ? (story.args as Record<string, unknown>)
      : undefined;
  } catch {
    return undefined;
  }
}

function defaultsFor(controls: ControlDefinition[]): Record<string, unknown> {
  return controls.reduce<Record<string, unknown>>(
    (values, control) =>
      control.defaultValue === undefined
        ? values
        : withValueAtPath(values, control.key, control.defaultValue),
    {},
  );
}

function environmentControls(storyContract?: ComponentStory): EnvironmentControl[] {
  if (!storyContract?.environmentJson) return [];
  const parsed = parseJsonObject(storyContract.environmentJson);
  if (!Array.isArray(parsed?.fixtures)) return [];
  return parsed.fixtures.flatMap((fixture) => {
    if (!fixture || typeof fixture !== "object" || Array.isArray(fixture)) return [];
    const value = fixture as Record<string, unknown>;
    return typeof value.key === "string" &&
      Array.isArray(value.options) &&
      value.options.length > 0 &&
      value.options.every((option) => typeof option === "string")
      ? [
          {
            key: value.key,
            label: value.key,
            adapter: typeof value.adapter === "string" ? value.adapter : "fixture",
            options: value.options as string[],
          },
        ]
      : [];
  });
}

function valueAtPath(values: Record<string, unknown>, path: string): unknown {
  return path
    .split(".")
    .reduce<unknown>(
      (current, segment) =>
        current && typeof current === "object" && !Array.isArray(current)
          ? (current as Record<string, unknown>)[segment]
          : undefined,
      values,
    );
}

function withValueAtPath(
  values: Record<string, unknown>,
  path: string,
  value: unknown,
): Record<string, unknown> {
  const [head, ...tail] = path.split(".");
  if (!head) return values;
  if (tail.length === 0) return { ...values, [head]: value };
  const current = values[head];
  return {
    ...values,
    [head]: withValueAtPath(
      current && typeof current === "object" && !Array.isArray(current)
        ? (current as Record<string, unknown>)
        : {},
      tail.join("."),
      value,
    ),
  };
}

function optionKey(value: unknown): string {
  return JSON.stringify(value);
}

function optionLabel(value: unknown): string {
  if (value === null) return "None";
  if (typeof value === "string") return value;
  if (typeof value === "boolean") return value ? "True" : "False";
  return JSON.stringify(value);
}

function controlId(path: string): string {
  return `rcl-preview-control-${path.replace(/[^a-zA-Z0-9_-]+/g, "-")}`;
}

function isVisible(control: ControlDefinition, values: Record<string, unknown>): boolean {
  if (!control.visibleWhen) return true;
  return optionKey(valueAtPath(values, control.visibleWhen.path)) === optionKey(control.visibleWhen.equals);
}

function defaultEnvironment(
  fixtures: EnvironmentControl[],
  initial: Record<string, string>,
): Record<string, string> {
  return fixtures.reduce<Record<string, string>>((result, fixture) => {
    result[fixture.key] = initial[fixture.key] ?? fixture.options[0] ?? "";
    return result;
  }, {});
}

function validateControls(
  values: Record<string, unknown>,
  controls: ControlDefinition[],
): Record<string, string> {
  const errors: Record<string, string> = {};
  for (const control of controls) {
    if (!isVisible(control, values)) continue;
    const value = valueAtPath(values, control.key);
    if (value === undefined) {
      if (control.required && control.defaultValue === undefined) errors[control.key] = "A value is required.";
      continue;
    }
    if (control.kind === "text" && typeof value !== "string") errors[control.key] = "Must be text.";
    if (control.kind === "number" && (typeof value !== "number" || !Number.isFinite(value))) errors[control.key] = "Must be a finite number.";
    if (control.kind === "boolean" && typeof value !== "boolean") errors[control.key] = "Must be true or false.";
    if (control.kind === "select" && control.options && !control.options.some((option) => optionKey(option) === optionKey(value))) errors[control.key] = "Choose one of the declared options.";
    if (control.kind === "json" && (value === null || typeof value !== "object")) errors[control.key] = "Must be structured JSON.";
    if (typeof control.minimum === "number" && typeof value === "number" && value < control.minimum) errors[control.key] = `Must be at least ${control.minimum}.`;
    if (typeof control.maximum === "number" && typeof value === "number" && value > control.maximum) errors[control.key] = `Must be at most ${control.maximum}.`;
    if (typeof control.minLength === "number" && typeof value === "string" && value.length < control.minLength) errors[control.key] = `Must be at least ${control.minLength} characters.`;
    if (typeof control.maxLength === "number" && typeof value === "string" && value.length > control.maxLength) errors[control.key] = `Must be at most ${control.maxLength} characters.`;
  }
  return errors;
}

export function PropsExperimentPanel({
  storyId,
  storyName,
  storyDescription,
  initialArgs = EMPTY_RECORD,
  initialEnvironment = EMPTY_RECORD as Record<string, string>,
  storyContract,
  status = "idle",
  message,
  onApply,
  onReset,
}: PropsExperimentPanelProps) {
  const { t } = useTranslation();
  const controls = useMemo(() => controlsFor(storyContract), [storyContract]);
  const fixtures = useMemo(() => environmentControls(storyContract), [storyContract]);
  const baselineArgs = useMemo(
    () => ({ ...defaultsFor(controls), ...(storyArgs(storyContract, storyId) ?? initialArgs) }),
    [controls, initialArgs, storyContract, storyId],
  );
  const baselineDraft = useMemo(() => JSON.stringify(baselineArgs, null, 2), [baselineArgs]);
  const baselineEnvironment = useMemo(
    () => defaultEnvironment(fixtures, initialEnvironment),
    [fixtures, initialEnvironment],
  );
  const [draft, setDraft] = useState(baselineDraft);
  const [parseError, setParseError] = useState("");
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [jsonDrafts, setJsonDrafts] = useState<Record<string, string>>({});
  const [environment, setEnvironment] = useState<Record<string, string>>(baselineEnvironment);
  const [lastAppliedDraft, setLastAppliedDraft] = useState(baselineDraft);
  const [lastAppliedEnvironment, setLastAppliedEnvironment] =
    useState<Record<string, string>>(baselineEnvironment);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const initialArgsKey = JSON.stringify(initialArgs);
  const initialEnvironmentKey = JSON.stringify(initialEnvironment);
  const values = parseJsonObject(draft) ?? {};
  const baselineEnvironmentKey = JSON.stringify(baselineEnvironment);
  const isDirty =
    draft !== lastAppliedDraft ||
    JSON.stringify(environment) !== JSON.stringify(lastAppliedEnvironment);
  const hasStoryOverride =
    draft !== baselineDraft || JSON.stringify(environment) !== baselineEnvironmentKey;

  useEffect(() => {
    setDraft(baselineDraft);
    setParseError("");
    setFieldErrors({});
    setJsonDrafts({});
    setEnvironment(baselineEnvironment);
    setLastAppliedDraft(baselineDraft);
    setLastAppliedEnvironment(baselineEnvironment);
    setShowAdvanced(false);
    // The serialized keys intentionally make session edits reset when the selected
    // story baseline changes, without requiring callers to replace the contract object.
  }, [baselineDraft, baselineEnvironment, initialArgsKey, initialEnvironmentKey, storyId]);

  const setValue = (key: string, value: unknown) => {
    const next = withValueAtPath(values, key, value);
    setDraft(JSON.stringify(next, null, 2));
    setParseError("");
    setFieldErrors((current) => {
      const nextErrors = { ...current };
      delete nextErrors[key];
      return nextErrors;
    });
    setJsonDrafts((current) => {
      const nextDrafts = { ...current };
      delete nextDrafts[key];
      return nextDrafts;
    });
  };

  const apply = () => {
    const parsed = parseJsonObject(draft);
    if (!parsed) {
      setParseError(t(strings.components.editor.propsJsonError));
      return;
    }
    const errors = validateControls(parsed, controls);
    if (Object.keys(errors).length > 0) {
      setFieldErrors(errors);
      setParseError("Fix the highlighted controls before applying.");
      return;
    }
    setParseError("");
    setFieldErrors({});
    setLastAppliedDraft(draft);
    setLastAppliedEnvironment(environment);
    if (fixtures.length > 0) onApply(parsed, environment);
    else onApply(parsed);
  };

  const reset = () => {
    setDraft(baselineDraft);
    setParseError("");
    setFieldErrors({});
    setJsonDrafts({});
    setEnvironment(baselineEnvironment);
    setLastAppliedDraft(baselineDraft);
    setLastAppliedEnvironment(baselineEnvironment);
    setShowAdvanced(false);
    onReset();
  };

  return (
    <section
      data-testid={selectors.components.editor.propsPanel}
      aria-label={t(strings.components.editor.tryProps)}
      className="rounded-panel border border-app-border bg-app-surface p-space-sm"
    >
      <div className="flex flex-wrap items-start justify-between gap-space-xs">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-space-2xs">
            <h3 className="text-sm font-semibold text-app-foreground">
              {t(strings.components.editor.tryProps)}
            </h3>
            {isDirty && (
              <span className="rounded-full bg-app-warning/15 px-space-2xs py-space-3xs text-[0.6875rem] font-semibold text-app-warning">
                Unsaved changes
              </span>
            )}
            {!isDirty && status === "applied" && (
              <span className="rounded-full bg-app-success/15 px-space-2xs py-space-3xs text-[0.6875rem] font-semibold text-app-success">
                Applied to preview
              </span>
            )}
            {!isDirty && status !== "applied" && hasStoryOverride && (
              <span className="rounded-full bg-app-primary/15 px-space-2xs py-space-3xs text-[0.6875rem] font-semibold text-app-primary">
                Temporary override
              </span>
            )}
          </div>
          <p className="mt-space-3xs text-xs text-app-muted-foreground">
            {storyName ? `Editing ${storyName}. ` : "Editing this story. "}
            Changes are temporary and stay inside the selected preview.
          </p>
          {storyDescription && (
            <p className="mt-space-3xs max-w-prose text-xs leading-relaxed text-app-muted-foreground">
              {storyDescription}
            </p>
          )}
        </div>
      </div>

      {controls.length > 0 ? (
        <fieldset className="mt-space-sm space-y-space-xs border-0 p-0">
          <legend className="text-xs font-semibold uppercase tracking-[0.08em] text-app-muted-foreground">
            Props
          </legend>
          {controls.map((control) => {
            if (!isVisible(control, values)) return null;
            const id = controlId(control.key);
            const value = valueAtPath(values, control.key);
            const error = fieldErrors[control.key];
            const hint = control.format ||
              (control.kind === "json" ? "Structured JSON" : undefined);
            return (
              <div key={control.key} className="min-w-0">
                {control.kind === "boolean" ? (
                  <label
                    htmlFor={id}
                    className="flex min-h-control-sm items-center gap-space-2xs text-sm font-medium text-app-foreground"
                  >
                    <input
                      id={id}
                      aria-label={control.label}
                      className="h-icon-sm w-icon-sm accent-app-primary"
                      type="checkbox"
                      checked={Boolean(value)}
                      required={control.required}
                      aria-invalid={error ? "true" : undefined}
                      onChange={(event) => setValue(control.key, event.target.checked)}
                    />
                    <span>{control.label}</span>
                    {control.required && <span className="text-app-danger">*</span>}
                  </label>
                ) : (
                  <label htmlFor={id} className="block text-xs font-medium text-app-foreground">
                    <span className="flex items-center justify-between gap-space-xs">
                      <span>
                        {control.label}
                        {control.required && <span className="ml-space-3xs text-app-danger">*</span>}
                      </span>
                      {hint && <span className="font-normal text-app-muted-foreground">{hint}</span>}
                    </span>
                    {control.kind === "select" ? (
                      <select
                        id={id}
                        aria-label={control.label}
                        className="mt-space-3xs h-control-sm w-full rounded-control border border-app-border bg-app-background px-space-2xs text-sm"
                        value={control.options?.some((option) => optionKey(option) === optionKey(value)) ? optionKey(value) : ""}
                        required={control.required}
                        aria-invalid={error ? "true" : undefined}
                        onChange={(event) => {
                          const next = control.options?.find((option) => optionKey(option) === event.target.value);
                          setValue(control.key, next);
                        }}
                      >
                        <option value="">Choose {control.label.toLowerCase()}</option>
                        {(control.options ?? []).map((option) => (
                          <option key={optionKey(option)} value={optionKey(option)}>
                            {optionLabel(option)}
                          </option>
                        ))}
                      </select>
                    ) : control.kind === "json" ? (
                      <textarea
                        id={id}
                        aria-label={control.label}
                        className="mt-space-3xs min-h-surface-tiny w-full resize-y rounded-control border border-app-border bg-app-background p-space-2xs font-mono text-xs"
                        value={jsonDrafts[control.key] ?? JSON.stringify(value ?? {}, null, 2)}
                        spellCheck={false}
                        aria-invalid={error ? "true" : undefined}
                        onChange={(event) => {
                          const raw = event.target.value;
                          setJsonDrafts((current) => ({ ...current, [control.key]: raw }));
                          try {
                            setValue(control.key, JSON.parse(raw));
                          } catch {
                            setFieldErrors((current) => ({ ...current, [control.key]: "Invalid JSON." }));
                          }
                        }}
                      />
                    ) : (
                      <input
                        id={id}
                        aria-label={control.label}
                        className="mt-space-3xs h-control-sm w-full rounded-control border border-app-border bg-app-background px-space-2xs text-sm"
                        type={control.kind === "number" ? "number" : "text"}
                        min={control.minimum}
                        max={control.maximum}
                        minLength={control.minLength}
                        maxLength={control.maxLength}
                        value={String(value ?? "")}
                        required={control.required}
                        aria-invalid={error ? "true" : undefined}
                        onChange={(event) => {
                          const raw = event.target.value;
                          setValue(control.key, control.kind === "number" ? (raw === "" ? undefined : Number(raw)) : raw);
                        }}
                      />
                    )}
                  </label>
                )}
                {error && (
                  <p className="mt-space-3xs text-xs text-app-danger" id={`${id}-error`}>
                    {error}
                  </p>
                )}
              </div>
            );
          })}
        </fieldset>
      ) : (
        <div className="mt-space-sm rounded-control border border-dashed border-app-border bg-app-background/60 p-space-xs">
          <p className="text-sm font-medium text-app-foreground">
            This story declares no configurable scalar arguments.
          </p>
          <p className="mt-space-3xs text-xs leading-relaxed text-app-muted-foreground">
            This story is driven by its named harness and interactions. Add declared fields to make
            safe component props editable here.
          </p>
        </div>
      )}

      {fixtures.length > 0 && (
        <fieldset className="mt-space-sm space-y-space-xs border-0 border-t border-app-border p-0 pt-space-sm">
          <legend className="text-xs font-semibold uppercase tracking-[0.08em] text-app-muted-foreground">
            Fixtures
          </legend>
          <p className="text-xs leading-relaxed text-app-muted-foreground">
            External adapter states declared by this story contract.
          </p>
          {fixtures.map((fixture) => (
            <label key={fixture.key} htmlFor={`rcl-preview-fixture-${fixture.key}`} className="block text-xs font-medium text-app-foreground">
              <span className="flex items-center justify-between gap-space-xs">
                <span>{fixture.label}</span>
                <span className="font-normal text-app-muted-foreground">{fixture.adapter}</span>
              </span>
              <select
                id={`rcl-preview-fixture-${fixture.key}`}
                aria-label={fixture.label}
                className="mt-space-3xs h-control-sm w-full rounded-control border border-app-border bg-app-background px-space-2xs text-sm"
                value={environment[fixture.key] ?? fixture.options[0] ?? ""}
                onChange={(event) => setEnvironment((current) => ({ ...current, [fixture.key]: event.target.value }))}
              >
                {fixture.options.map((option) => (
                  <option key={option} value={option}>
                    {option}
                  </option>
                ))}
              </select>
            </label>
          ))}
        </fieldset>
      )}

      {controls.length > 0 && (
        <div className="mt-space-sm border-t border-app-border pt-space-xs">
          <Button
            type="button"
            variant="ghost"
            className="h-control-tight px-0 text-xs text-app-primary"
            aria-expanded={showAdvanced}
            aria-controls="rcl-props-experiment-advanced"
            onClick={() => setShowAdvanced((visible) => !visible)}
          >
            {showAdvanced ? "Hide advanced JSON" : "Advanced JSON"}
          </Button>
          {showAdvanced && (
            <textarea
              id="rcl-props-experiment-advanced"
              data-testid={selectors.components.editor.propsDraft}
              value={draft}
              onChange={(event) => {
                setDraft(event.target.value);
                setParseError("");
                setFieldErrors({});
              }}
              spellCheck={false}
              className="mt-space-2xs min-h-surface-tall w-full resize-y rounded-control border border-app-border bg-app-background p-space-2xs font-mono text-xs text-app-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
            />
          )}
        </div>
      )}

      {(parseError || (status === "error" && message)) && (
        <p
          data-testid={selectors.components.editor.propsError}
          className="mt-space-xs rounded-control bg-app-danger/10 px-space-xs py-space-2xs text-xs text-app-danger"
          role="alert"
        >
          {parseError || message}
        </p>
      )}
      <div className="mt-space-sm flex flex-wrap gap-space-2xs border-t border-app-border pt-space-sm">
        <Button
          data-testid={selectors.components.editor.propsApply}
          type="button"
          className="h-control-tight px-space-xs text-xs"
          disabled={status === "applying" || Object.keys(fieldErrors).length > 0}
          onClick={apply}
        >
          {status === "applying"
            ? t(strings.components.editor.applyingProps)
            : t(strings.components.editor.applyTemporaryProps)}
        </Button>
        <Button
          data-testid={selectors.components.editor.propsReset}
          type="button"
          variant="secondary"
          className="h-control-tight px-space-xs text-xs"
          onClick={reset}
        >
          {t(strings.components.editor.resetIndexedProps)}
        </Button>
      </div>
    </section>
  );
}
