/**
 * @libraryId react-component-library:ValidationAdapter
 * @displayName ValidationAdapter
 * @description
 * @version 1.0.2
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
export type OperatorInputKind =
  | "secret"
  | "choice"
  | "confirm"
  | "path"
  | "enum"
  | "boolean"
  | "duration"
  | "confirmation";
export interface OperatorInput {
  id: string;
  kind: OperatorInputKind;
  label: string;
  description?: string;
  required?: boolean;
  defaultValue?: string | boolean;
  options?: Array<{ label: string; value: string }>;
  candidates?: Array<{
    label: string;
    value: string;
    status?: string;
    risk?: string;
    remediation?: string;
  }>;
  validation?: string;
}
export interface ValidationAdapterProps {
  inputs?: OperatorInput[];
  values?: Record<string, string | boolean>;
  onChange?: (id: string, value: string | boolean) => void;
  className?: string;
}

export function toGeneratedFields(inputs: OperatorInput[] = []) {
  return inputs.map((input) => ({
    name: input.id,
    label: input.label,
    description: input.description,
    required: input.required,
    defaultValue: input.defaultValue,
    options: input.options ?? input.candidates?.map((c) => ({ label: c.label, value: c.value })),
    type:
      input.kind === "boolean" || input.kind === "confirm" || input.kind === "confirmation"
        ? "checkbox"
        : input.kind === "secret"
          ? "text"
          : input.kind === "choice" || input.kind === "enum"
            ? "select"
            : "text",
  }));
}

export default function ValidationAdapter({
  inputs = [],
  values = {},
  onChange,
  className = "",
}: ValidationAdapterProps) {
  return (
    <section
      className={`rcl-component validation-adapter ${className}`.trim()}
      aria-label="Outstanding questions"
    >
      {inputs.map((input) => (
        <label key={input.id} style={{ display: "grid", gap: 4, marginBlock: 12 }}>
          <span>
            {input.label}
            {input.required ? " *" : ""}
          </span>
          {input.description && <small>{input.description}</small>}
          {input.kind === "secret" ? (
            <input
              type="password"
              autoComplete="new-password"
              value={String(values[input.id] ?? "")}
              onChange={(e) => onChange?.(input.id, e.target.value)}
            />
          ) : input.kind === "boolean" ||
            input.kind === "confirm" ||
            input.kind === "confirmation" ? (
            <input
              type="checkbox"
              checked={Boolean(values[input.id] ?? input.defaultValue)}
              onChange={(e) => onChange?.(input.id, e.target.checked)}
            />
          ) : input.options || input.candidates ? (
            <select
              value={String(values[input.id] ?? input.defaultValue ?? "")}
              onChange={(e) => onChange?.(input.id, e.target.value)}
            >
              <option value="">Select…</option>
              {(input.options ?? input.candidates ?? []).map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          ) : (
            <input
              type={input.kind === "duration" ? "text" : "text"}
              value={String(values[input.id] ?? input.defaultValue ?? "")}
              onChange={(e) => onChange?.(input.id, e.target.value)}
            />
          )}
        </label>
      ))}
    </section>
  );
}
