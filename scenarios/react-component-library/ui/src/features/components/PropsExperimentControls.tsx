import type { Dispatch, SetStateAction } from "react";
import { Input } from "@vrooli/react-component-library/Input/1";
import {
  controlId,
  isVisible,
  optionKey,
  optionLabel,
  type ControlDefinition,
  valueAtPath,
} from "./propsExperimentModel";

export function PropsExperimentControls({
  controls,
  values,
  fieldErrors,
  jsonDrafts,
  setJsonDrafts,
  setFieldErrors,
  setValue,
}: {
  controls: ControlDefinition[];
  values: Record<string, unknown>;
  fieldErrors: Record<string, string>;
  jsonDrafts: Record<string, string>;
  setJsonDrafts: Dispatch<SetStateAction<Record<string, string>>>;
  setFieldErrors: Dispatch<SetStateAction<Record<string, string>>>;
  setValue: (key: string, value: unknown) => void;
}) {
  return (
    <fieldset className="mt-space-sm space-y-space-xs border-0 p-0">
      <legend className="text-xs font-semibold uppercase tracking-[0.08em] text-app-muted-foreground">
        Props
      </legend>
      {controls.map((control) => {
        if (!isVisible(control, values)) return null;
        const id = controlId(control.key);
        const value = valueAtPath(values, control.key);
        const error = fieldErrors[control.key];
        const hint = control.format || (control.kind === "json" ? "Structured JSON" : undefined);
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
                    value={
                      control.options?.some((option) => optionKey(option) === optionKey(value))
                        ? optionKey(value)
                        : ""
                    }
                    required={control.required}
                    aria-invalid={error ? "true" : undefined}
                    onChange={(event) =>
                      setValue(
                        control.key,
                        control.options?.find((option) => optionKey(option) === event.target.value),
                      )
                    }
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
                        setFieldErrors((current) => ({
                          ...current,
                          [control.key]: "Invalid JSON.",
                        }));
                      }
                    }}
                  />
                ) : (
                  <Input
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
                    onChange={(event) =>
                      setValue(
                        control.key,
                        control.kind === "number"
                          ? event.target.value === ""
                            ? undefined
                            : Number(event.target.value)
                          : event.target.value,
                      )
                    }
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
  );
}

export function PropsExperimentEmptyState() {
  return (
    <div className="mt-space-sm rounded-control border border-dashed border-app-border bg-app-background/60 p-space-xs">
      <p className="text-sm font-medium text-app-foreground">
        This story declares no configurable scalar arguments.
      </p>
      <p className="mt-space-3xs text-xs leading-relaxed text-app-muted-foreground">
        This story is driven by its named harness and interactions. Add declared fields to make safe
        component props editable here.
      </p>
    </div>
  );
}
