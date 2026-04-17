/**
 * SchemaFormField Component
 *
 * A recursive component that renders form fields from JSON Schema (ParameterSchema).
 * Supports various field types: string, number, boolean, array, object, and enums.
 */

import type { ParameterSchema } from "../../lib/api";
import { Input } from "../ui/input";
import { Switch } from "../ui/switch";
import { formatLabel, labelClass, inputClass } from "./schemaFormUtils";
import { ArrayFormField } from "./ArrayFormField";
import { ObjectFormField } from "./ObjectFormField";

// Re-export for consumers
export { getDefaultValue } from "./schemaFormUtils";

export interface SchemaFormFieldProps {
  name: string;
  schema: ParameterSchema;
  value: unknown;
  onChange: (value: unknown) => void;
  required?: boolean;
  disabled?: boolean;
  depth?: number;
}

/**
 * Renders a form field based on its JSON Schema type.
 */
export function SchemaFormField({
  name,
  schema,
  value,
  onChange,
  required = false,
  disabled = false,
  depth = 0,
}: SchemaFormFieldProps) {
  const { type, description, enum: enumValues, format } = schema;

  const containerStyle = depth > 0 ? { marginLeft: `${depth * 16}px` } : {};

  if (enumValues && enumValues.length > 0) {
    return (
      <div className="mb-4" style={containerStyle}>
        <label className={labelClass}>
          {formatLabel(name)}
          {required && <span className="text-red-400 ml-1">*</span>}
        </label>
        {description && <p className="text-xs text-slate-500 mb-2">{description}</p>}
        <select
          value={typeof value === "string" ? value : ""}
          onChange={(e) => onChange(e.target.value)}
          disabled={disabled}
          className={inputClass}
          data-testid={`field-${name}`}
        >
          <option value="">Select...</option>
          {enumValues.map((opt) => (
            <option key={opt} value={opt}>
              {opt}
            </option>
          ))}
        </select>
      </div>
    );
  }

  if (type === "boolean") {
    return (
      <div className="mb-4 flex items-start gap-3" style={containerStyle}>
        <Switch
          checked={typeof value === "boolean" ? value : false}
          onCheckedChange={(checked) => onChange(checked)}
          disabled={disabled}
          className="mt-1"
          data-testid={`field-${name}`}
          aria-label={formatLabel(name)}
        />
        <div>
          <span className="text-sm font-medium text-slate-300">
            {formatLabel(name)}
            {required && <span className="text-red-400 ml-1">*</span>}
          </span>
          {description && <p className="text-xs text-slate-500 mt-0.5">{description}</p>}
        </div>
      </div>
    );
  }

  if (type === "number" || type === "integer") {
    return (
      <div className="mb-4" style={containerStyle}>
        <label className={labelClass}>
          {formatLabel(name)}
          {required && <span className="text-red-400 ml-1">*</span>}
        </label>
        {description && <p className="text-xs text-slate-500 mb-2">{description}</p>}
        <Input
          type="number"
          value={typeof value === "number" ? value : ""}
          onChange={(e) => {
            const num = type === "integer" ? parseInt(e.target.value, 10) : parseFloat(e.target.value);
            onChange(isNaN(num) ? undefined : num);
          }}
          disabled={disabled}
          step={type === "integer" ? 1 : "any"}
          data-testid={`field-${name}`}
        />
      </div>
    );
  }

  if (type === "array" && schema.items) {
    return (
      <ArrayFormField
        name={name}
        schema={schema}
        value={value}
        onChange={onChange}
        required={required}
        disabled={disabled}
        depth={depth}
      />
    );
  }

  if (type === "object" && schema.properties) {
    return (
      <ObjectFormField
        name={name}
        schema={schema}
        value={value}
        onChange={onChange}
        required={required}
        disabled={disabled}
        depth={depth}
      />
    );
  }

  const isUrl = format === "uri" || format === "url";
  const isEmail = format === "email";
  const isMultiline =
    description?.toLowerCase().includes("multiline") ||
    name.toLowerCase().includes("content") ||
    name.toLowerCase().includes("description") ||
    name.toLowerCase().includes("body");

  if (isMultiline) {
    return (
      <div className="mb-4" style={containerStyle}>
        <label className={labelClass}>
          {formatLabel(name)}
          {required && <span className="text-red-400 ml-1">*</span>}
        </label>
        {description && <p className="text-xs text-slate-500 mb-2">{description}</p>}
        <textarea
          value={typeof value === "string" ? value : ""}
          onChange={(e) => onChange(e.target.value)}
          disabled={disabled}
          rows={4}
          className={inputClass}
          placeholder={`Enter ${formatLabel(name).toLowerCase()}...`}
          data-testid={`field-${name}`}
        />
      </div>
    );
  }

  return (
    <div className="mb-4" style={containerStyle}>
      <label className={labelClass}>
        {formatLabel(name)}
        {required && <span className="text-red-400 ml-1">*</span>}
      </label>
      {description && <p className="text-xs text-slate-500 mb-2">{description}</p>}
      <Input
        type={isUrl ? "url" : isEmail ? "email" : "text"}
        value={typeof value === "string" ? value : ""}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled}
        placeholder={`Enter ${formatLabel(name).toLowerCase()}...`}
        data-testid={`field-${name}`}
      />
    </div>
  );
}
