/** @vrooliComponentSource data-display.description-list */
import { useEffect, useMemo, useState } from "react";

import { Button } from "@vrooli/react-component-library/Button/2";
import { type ComponentStory } from "../../api/components";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import {
  EMPTY_RECORD,
  controlsFor,
  defaultEnvironment,
  defaultsFor,
  environmentControls,
  parseJsonObject,
  storyArgs,
  validateControls,
  withValueAtPath,
} from "./propsExperimentModel";
import { PropsExperimentControls, PropsExperimentEmptyState } from "./PropsExperimentControls";

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
        <PropsExperimentControls
          controls={controls}
          values={values}
          fieldErrors={fieldErrors}
          jsonDrafts={jsonDrafts}
          setJsonDrafts={setJsonDrafts}
          setFieldErrors={setFieldErrors}
          setValue={setValue}
        />
      ) : (
        <PropsExperimentEmptyState />
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
            <label
              key={fixture.key}
              htmlFor={`rcl-preview-fixture-${fixture.key}`}
              className="block text-xs font-medium text-app-foreground"
            >
              <span className="flex items-center justify-between gap-space-xs">
                <span>{fixture.label}</span>
                <span className="font-normal text-app-muted-foreground">{fixture.adapter}</span>
              </span>
              <select
                id={`rcl-preview-fixture-${fixture.key}`}
                aria-label={fixture.label}
                className="mt-space-3xs h-control-sm w-full rounded-control border border-app-border bg-app-background px-space-2xs text-sm"
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
