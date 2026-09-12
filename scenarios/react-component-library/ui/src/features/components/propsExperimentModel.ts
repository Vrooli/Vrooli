import type { ComponentStory } from "../../api/components";

export const EMPTY_RECORD: Record<string, unknown> = {};

export type ControlKind = "text" | "number" | "boolean" | "select" | "json";

export interface ControlDefinition {
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

export interface EnvironmentControl {
  key: string;
  label: string;
  adapter: string;
  options: string[];
}

export function parseJsonObject(raw: string): Record<string, unknown> | undefined {
  try {
    const parsed: unknown = JSON.parse(raw);
    return parsed && typeof parsed === "object" && !Array.isArray(parsed)
      ? (parsed as Record<string, unknown>)
      : undefined;
  } catch {
    return undefined;
  }
}

export function controlsFor(storyContract?: ComponentStory): ControlDefinition[] {
  const parsed = storyContract?.argsJson ? parseJsonObject(storyContract.argsJson) : undefined;
  if (!Array.isArray(parsed?.fields)) return [];
  return parsed.fields.flatMap((field) => {
    if (!field || typeof field !== "object" || Array.isArray(field)) return [];
    const value = field as Record<string, unknown>;
    if (typeof value.path !== "string" || !value.path.trim()) return [];
    const kind = value.kind === "enum" ? "select" : value.kind;
    const controlKind = ["object", "array", "structured"].includes(String(kind)) ? "json" : kind;
    if (!["text", "number", "boolean", "select", "json"].includes(String(controlKind))) return [];
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

export function storyArgs(storyContract?: ComponentStory, storyID?: string) {
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

export function defaultsFor(controls: ControlDefinition[]) {
  return controls.reduce<Record<string, unknown>>(
    (values, control) =>
      control.defaultValue === undefined
        ? values
        : withValueAtPath(values, control.key, control.defaultValue),
    {},
  );
}

export function environmentControls(storyContract?: ComponentStory): EnvironmentControl[] {
  const parsed = storyContract?.environmentJson
    ? parseJsonObject(storyContract.environmentJson)
    : undefined;
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
            options: value.options,
          },
        ]
      : [];
  });
}

export function valueAtPath(values: Record<string, unknown>, path: string): unknown {
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

export function withValueAtPath(
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

export function optionKey(value: unknown): string {
  return JSON.stringify(value);
}

export function optionLabel(value: unknown): string {
  if (value === null) return "None";
  if (typeof value === "string") return value;
  if (typeof value === "boolean") return value ? "True" : "False";
  return JSON.stringify(value);
}

export function controlId(path: string): string {
  return `rcl-preview-control-${path.replace(/[^a-zA-Z0-9_-]+/g, "-")}`;
}

export function isVisible(control: ControlDefinition, values: Record<string, unknown>): boolean {
  return (
    !control.visibleWhen ||
    optionKey(valueAtPath(values, control.visibleWhen.path)) ===
      optionKey(control.visibleWhen.equals)
  );
}

export function defaultEnvironment(
  fixtures: EnvironmentControl[],
  initial: Record<string, string>,
) {
  return fixtures.reduce<Record<string, string>>((result, fixture) => {
    result[fixture.key] = initial[fixture.key] ?? fixture.options[0] ?? "";
    return result;
  }, {});
}

export function validateControls(
  values: Record<string, unknown>,
  controls: ControlDefinition[],
): Record<string, string> {
  const errors: Record<string, string> = {};
  for (const control of controls) {
    if (!isVisible(control, values)) continue;
    const value = valueAtPath(values, control.key);
    if (value === undefined) {
      if (control.required && control.defaultValue === undefined)
        errors[control.key] = "A value is required.";
      continue;
    }
    if (control.kind === "text" && typeof value !== "string") errors[control.key] = "Must be text.";
    if (control.kind === "number" && (typeof value !== "number" || !Number.isFinite(value)))
      errors[control.key] = "Must be a finite number.";
    if (control.kind === "boolean" && typeof value !== "boolean")
      errors[control.key] = "Must be true or false.";
    if (
      control.kind === "select" &&
      control.options &&
      !control.options.some((option) => optionKey(option) === optionKey(value))
    )
      errors[control.key] = "Choose one of the declared options.";
    if (control.kind === "json" && (value === null || typeof value !== "object"))
      errors[control.key] = "Must be structured JSON.";
    if (typeof control.minimum === "number" && typeof value === "number" && value < control.minimum)
      errors[control.key] = `Must be at least ${control.minimum}.`;
    if (typeof control.maximum === "number" && typeof value === "number" && value > control.maximum)
      errors[control.key] = `Must be at most ${control.maximum}.`;
    if (
      typeof control.minLength === "number" &&
      typeof value === "string" &&
      value.length < control.minLength
    )
      errors[control.key] = `Must be at least ${control.minLength} characters.`;
    if (
      typeof control.maxLength === "number" &&
      typeof value === "string" &&
      value.length > control.maxLength
    )
      errors[control.key] = `Must be at most ${control.maxLength} characters.`;
  }
  return errors;
}
