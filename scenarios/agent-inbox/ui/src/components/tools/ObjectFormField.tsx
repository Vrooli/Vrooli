/**
 * ObjectFormField - Renders an object field in a schema form.
 * Recursively renders child properties.
 */

import type { ParameterSchema } from "../../lib/api";
import { formatLabel, labelClass } from "./schemaFormUtils";
import { SchemaFormField } from "./SchemaFormField";

interface ObjectFormFieldProps {
  name: string;
  schema: ParameterSchema;
  value: unknown;
  onChange: (value: unknown) => void;
  required?: boolean;
  disabled?: boolean;
  depth?: number;
}

export function ObjectFormField({
  name,
  schema,
  value,
  onChange,
  required = false,
  disabled = false,
  depth = 0,
}: ObjectFormFieldProps) {
  const { description } = schema;
  const containerStyle = depth > 0 ? { marginLeft: `${depth * 16}px` } : {};
  const objectValue: Record<string, unknown> =
    value && typeof value === "object" && !Array.isArray(value)
      ? (value as Record<string, unknown>)
      : {};
  const schemaProperties = schema.properties ?? {};

  const updateProperty = (propName: string, newValue: unknown) => {
    onChange({ ...objectValue, [propName]: newValue });
  };

  return (
    <div className="mb-4" style={containerStyle}>
      <label className={labelClass}>
        {formatLabel(name)}
        {required && <span className="text-red-400 ml-1">*</span>}
      </label>
      {description && <p className="text-xs text-slate-500 mb-2">{description}</p>}
      <div className="space-y-2 pl-4 border-l border-white/10">
        {Object.entries(schemaProperties).map(([propName, propSchema]) => (
          <SchemaFormField
            key={propName}
            name={propName}
            schema={propSchema}
            value={objectValue[propName]}
            onChange={(v) => updateProperty(propName, v)}
            disabled={disabled}
            depth={depth + 1}
          />
        ))}
      </div>
    </div>
  );
}
