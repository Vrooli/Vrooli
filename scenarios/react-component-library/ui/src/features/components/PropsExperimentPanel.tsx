/** @vrooliComponentSource data-display.description-list */
import { useEffect, useState } from "react";

import { Button } from "../../components/Button";
import { type ComponentStory } from "../../api/components";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

interface PropsExperimentPanelProps {
  storyId?: string;
  storyName?: string;
  initialArgs?: Record<string, unknown>;
  initialEnvironment?: Record<string, string>;
  storyContract?: ComponentStory;
  status?: "idle" | "applying" | "applied" | "error";
  message?: string;
  onApply: (props: Record<string, unknown>, environment?: Record<string, string>) => void;
  onReset: () => void;
}

type ControlKind = "text" | "number" | "boolean" | "select";
interface ControlDefinition {
  key: string;
  label?: string;
  kind: ControlKind;
  options?: string[];
  minimum?: number;
  maximum?: number;
  minLength?: number;
  maxLength?: number;
  defaultValue?: unknown;
}

function controlsFor(storyContract?: ComponentStory): ControlDefinition[] {
  if (storyContract?.argsJson) {
    try {
      const parsed = JSON.parse(storyContract.argsJson) as { fields?: unknown };
      if (Array.isArray(parsed.fields)) {
        return parsed.fields.flatMap((field) => {
          if (!field || typeof field !== "object") return [];
          const value = field as {
            path?: unknown;
            label?: unknown;
            kind?: unknown;
            options?: unknown;
            default?: unknown;
          };
          if (typeof value.path !== "string") return [];
          const kind = value.kind === "enum" ? "select" : value.kind;
          if (!["text", "number", "boolean", "select"].includes(String(kind))) return [];
          return [
            {
              key: value.path,
              label: typeof value.label === "string" ? value.label : value.path,
              kind: kind as ControlKind,
              options: Array.isArray(value.options)
                ? value.options.filter((option): option is string => typeof option === "string")
                : undefined,
              minimum:
                typeof (value as { minimum?: unknown }).minimum === "number"
                  ? (value as { minimum: number }).minimum
                  : undefined,
              maximum:
                typeof (value as { maximum?: unknown }).maximum === "number"
                  ? (value as { maximum: number }).maximum
                  : undefined,
              minLength:
                typeof (value as { minLength?: unknown }).minLength === "number"
                  ? (value as { minLength: number }).minLength
                  : undefined,
              maxLength:
                typeof (value as { maxLength?: unknown }).maxLength === "number"
                  ? (value as { maxLength: number }).maxLength
                  : undefined,
              defaultValue: value.default,
            },
          ];
        });
      }
    } catch {
      /* validation errors are surfaced by the server/runtime */
    }
  }
  return [];
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
    if (!story?.args || typeof story.args !== "object" || Array.isArray(story.args))
      return undefined;
    return story.args as Record<string, unknown>;
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

function environmentControls(
  storyContract?: ComponentStory,
): Array<{ key: string; label: string; options: string[] }> {
  if (!storyContract?.environmentJson) return [];
  try {
    const parsed = JSON.parse(storyContract.environmentJson) as {
      fixtures?: Array<{ key?: unknown; options?: unknown }>;
    };
    return Array.isArray(parsed.fixtures)
      ? parsed.fixtures.flatMap((fixture) =>
          typeof fixture?.key === "string" &&
          Array.isArray(fixture.options) &&
          fixture.options.every((option) => typeof option === "string")
            ? [{ key: fixture.key, label: fixture.key, options: fixture.options }]
            : [],
        )
      : [];
  } catch {
    return [];
  }
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

export function PropsExperimentPanel({
  storyId,
  storyName,
  initialArgs = {},
  initialEnvironment = {},
  storyContract,
  status = "idle",
  message,
  onApply,
  onReset,
}: PropsExperimentPanelProps) {
  const { t } = useTranslation();
  const [draft, setDraft] = useState("{}");
  const [parseError, setParseError] = useState("");
  const controls = controlsFor(storyContract);
  const fixtures = environmentControls(storyContract);
  const [environment, setEnvironment] = useState<Record<string, string>>({});
  const [showAdvanced, setShowAdvanced] = useState(false);
  const initialArgsKey = JSON.stringify(initialArgs);
  const initialEnvironmentKey = JSON.stringify(initialEnvironment);
  const values = (() => {
    try {
      const parsed = JSON.parse(draft);
      return parsed && typeof parsed === "object" && !Array.isArray(parsed)
        ? (parsed as Record<string, unknown>)
        : {};
    } catch {
      return {};
    }
  })();

  useEffect(() => {
    // A named story is a baseline, not a second schema. Start with the
    // component schema's defaults and let that story override only its
    // deliberate differences, so every generated field is visible and useful.
    setDraft(
      JSON.stringify(
        {
          ...defaultsFor(controlsFor(storyContract)),
          ...(storyArgs(storyContract, storyId) ?? initialArgs),
        },
        null,
        2,
      ),
    );
    setParseError("");
    setEnvironment(initialEnvironment);
    setShowAdvanced(false);
  }, [initialArgsKey, initialEnvironmentKey, storyContract, storyId]);

  const apply = () => {
    try {
      const parsed = JSON.parse(draft) as unknown;
      if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
        setParseError(t(strings.components.editor.propsObjectError));
        return;
      }
      setParseError("");
      if (fixtures.length > 0) onApply(parsed as Record<string, unknown>, environment);
      else onApply(parsed as Record<string, unknown>);
    } catch {
      setParseError(t(strings.components.editor.propsJsonError));
    }
  };

  const setValue = (key: string, value: unknown) => {
    const next = withValueAtPath(values, key, value);
    setDraft(JSON.stringify(next, null, 2));
    setParseError("");
  };

  return (
    <section
      data-testid={selectors.components.editor.propsPanel}
      aria-label={t(strings.components.editor.tryProps)}
      className="mt-space-xs rounded-md border border-app-border bg-app-surface p-space-xs"
    >
      <div className="flex flex-wrap items-baseline justify-between gap-space-2xs">
        <div>
          <h3 className="text-sm font-semibold text-app-foreground">
            {t(strings.components.editor.tryProps)}
          </h3>
          <p className="mt-space-3xs text-xs text-app-muted-foreground">
            Temporary changes affect only {storyName || "this story"} and disappear when you reset,
            reload, or leave.
          </p>
        </div>
        {status === "applied" && (
          <span className="text-xs text-app-success">
            {t(strings.components.editor.propsApplied)}
          </span>
        )}
      </div>
      {controls.map((control) => (
        <label
          key={control.key}
          className="mt-space-xs block text-xs font-medium text-app-foreground"
        >
          {control.label || control.key}
          {control.kind === "boolean" ? (
            <input
              className="ml-2 h-4 w-4 align-middle"
              type="checkbox"
              checked={Boolean(valueAtPath(values, control.key))}
              onChange={(event) => setValue(control.key, event.target.checked)}
            />
          ) : control.kind === "select" ? (
            <select
              className="mt-space-3xs h-9 w-full rounded-md border border-app-border bg-app-background px-space-2xs text-sm"
              value={String(valueAtPath(values, control.key) ?? "")}
              onChange={(event) => setValue(control.key, event.target.value)}
            >
              {(control.options ?? []).map((option) => (
                <option key={option} value={option}>
                  {option}
                </option>
              ))}
            </select>
          ) : (
            <input
              className="mt-space-3xs h-9 w-full rounded-md border border-app-border bg-app-background px-space-2xs text-sm"
              type={control.kind === "number" ? "number" : "text"}
              min={control.minimum}
              max={control.maximum}
              minLength={control.minLength}
              maxLength={control.maxLength}
              value={String(valueAtPath(values, control.key) ?? "")}
              onChange={(event) =>
                setValue(
                  control.key,
                  control.kind === "number" ? Number(event.target.value) : event.target.value,
                )
              }
            />
          )}
        </label>
      ))}
      {controls.length === 0 && (
        <p className="mt-space-xs text-xs text-app-muted-foreground">
          This story declares no configurable scalar arguments.
        </p>
      )}
      {fixtures.length > 0 && (
        <div className="mt-space-sm border-t border-app-border pt-space-xs">
          <h4 className="text-xs font-semibold text-app-foreground">Environment</h4>
          <p className="mt-space-3xs text-xs text-app-muted-foreground">
            Only this story’s declared fixture states are available.
          </p>
          {fixtures.map((fixture) => (
            <label
              key={fixture.key}
              className="mt-space-xs block text-xs font-medium text-app-foreground"
            >
              {fixture.label}
              <select
                className="mt-space-3xs h-9 w-full rounded-md border border-app-border bg-app-background px-space-2xs text-sm"
                value={environment[fixture.key] ?? fixture.options[0] ?? ""}
                onChange={(event) =>
                  setEnvironment((current) => ({ ...current, [fixture.key]: event.target.value }))
                }
              >
                {fixture.options.map((option) => (
                  <option key={option} value={option}>
                    {option}
                  </option>
                ))}
              </select>
            </label>
          ))}
        </div>
      )}
      {controls.length > 0 && (
        <Button
          type="button"
          variant="ghost"
          className="mt-space-xs h-8 px-0 text-xs text-app-primary"
          aria-expanded={showAdvanced}
          onClick={() => setShowAdvanced((visible) => !visible)}
        >
          Advanced JSON
        </Button>
      )}
      {showAdvanced && (
        <>
          <label
            className="mt-space-2xs block text-xs text-app-muted-foreground"
            htmlFor="rcl-props-experiment"
          >
            {t(strings.components.editor.indexedPropsLabel)}
          </label>
          <textarea
            id="rcl-props-experiment"
            data-testid={selectors.components.editor.propsDraft}
            value={draft}
            onChange={(event) => setDraft(event.target.value)}
            spellCheck={false}
            className="mt-space-3xs min-h-40 w-full resize-y rounded-md border border-app-border bg-app-background p-space-2xs font-mono text-xs text-app-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
          />
        </>
      )}
      {(parseError || (status === "error" && message)) && (
        <p
          data-testid={selectors.components.editor.propsError}
          className="mt-space-2xs text-xs text-app-danger"
        >
          {parseError || message}
        </p>
      )}
      <div className="mt-space-xs flex gap-space-2xs">
        <Button
          data-testid={selectors.components.editor.propsApply}
          type="button"
          className="h-8 px-space-xs text-xs"
          disabled={status === "applying"}
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
          className="h-8 px-space-xs text-xs"
          onClick={onReset}
        >
          {t(strings.components.editor.resetIndexedProps)}
        </Button>
      </div>
    </section>
  );
}
