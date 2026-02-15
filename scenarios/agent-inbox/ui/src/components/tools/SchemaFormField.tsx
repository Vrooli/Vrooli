/**
 * SchemaFormField Component
 *
 * A recursive component that renders form fields from JSON Schema (ParameterSchema).
 * Supports various field types: string, number, boolean, array, object, and enums.
 */

import { Plus, Trash2 } from "lucide-react";
import type { ParameterSchema } from "../../lib/api";
import { Button } from "../ui/button";
import { Input } from "../ui/input";
import { Switch } from "../ui/switch";

export interface SchemaFormFieldProps {
  /** Field name (used for label and key) */
  name: string;
  /** JSON Schema for this field */
  schema: ParameterSchema;
  /** Current field value */
  value: unknown;
  /** Callback when value changes */
  onChange: (value: unknown) => void;
  /** Whether this field is required */
  required?: boolean;
  /** Whether the field is disabled */
  disabled?: boolean;
  /** Nesting depth (for styling) */
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

  const labelClass = "block text-sm font-medium text-slate-300 mb-1";
  const inputClass =
    "w-full bg-white/5 border border-white/10 rounded-md px-3 py-2 text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/50 disabled:opacity-50";

  const containerStyle = depth > 0 ? { marginLeft: `${depth * 16}px` } : {};

  const formatLabel = (s: string) =>
    s
      .replace(/_/g, " ")
      .replace(/([a-z])([A-Z])/g, "$1 $2")
      .replace(/^./, (c) => c.toUpperCase());

  if (enumValues && enumValues.length > 0) {
    return (
      <div className="mb-4" style={containerStyle}>
        <label className={labelClass}>
          {formatLabel(name)}
          {required && <span className="text-red-400 ml-1">*</span>}
        </label>
        {description && <p className="text-xs text-slate-500 mb-2">{description}</p>}
        <select
          value={(value as string) ?? ""}
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
          checked={(value as boolean) ?? false}
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
          value={(value as number) ?? ""}
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
    const arrayValue = (value as unknown[]) ?? [];
    const itemSchema = schema.items;

    const addItem = () => {
      const newItem = getDefaultValue(itemSchema);
      onChange([...arrayValue, newItem]);
    };

    const removeItem = (index: number) => {
      const newArray = [...arrayValue];
      newArray.splice(index, 1);
      onChange(newArray);
    };

    const updateItem = (index: number, newValue: unknown) => {
      const newArray = [...arrayValue];
      newArray[index] = newValue;
      onChange(newArray);
    };

    return (
      <div className="mb-4" style={containerStyle}>
        <label className={labelClass}>
          {formatLabel(name)}
          {required && <span className="text-red-400 ml-1">*</span>}
        </label>
        {description && <p className="text-xs text-slate-500 mb-2">{description}</p>}
        <div className="space-y-2 pl-4 border-l border-white/10">
          {arrayValue.map((item, index) => (
            <div key={index} className="flex items-start gap-2">
              <div className="flex-1">
                <SchemaFormField
                  name={`${name}[${index}]`}
                  schema={itemSchema}
                  value={item}
                  onChange={(v) => updateItem(index, v)}
                  disabled={disabled}
                  depth={depth + 1}
                />
              </div>
              <Button
                variant="ghost"
                size="icon"
                onClick={() => removeItem(index)}
                disabled={disabled}
                className="shrink-0 mt-6"
              >
                <Trash2 className="h-4 w-4 text-red-400" />
              </Button>
            </div>
          ))}
          <Button variant="secondary" size="sm" onClick={addItem} disabled={disabled} className="mt-2">
            <Plus className="h-4 w-4 mr-1" />
            Add Item
          </Button>
        </div>
      </div>
    );
  }

  if (type === "object" && schema.properties) {
    const objectValue = (value as Record<string, unknown>) ?? {};
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
          value={(value as string) ?? ""}
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
        value={(value as string) ?? ""}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled}
        placeholder={`Enter ${formatLabel(name).toLowerCase()}...`}
        data-testid={`field-${name}`}
      />
    </div>
  );
}

/**
 * Get a default value for a schema type.
 */
function getDefaultValue(schema: ParameterSchema): unknown {
  if (schema?.default !== undefined) {
    return schema.default;
  }

  switch (schema?.type) {
    case "string":
      return "";
    case "number":
    case "integer":
      return undefined;
    case "boolean":
      return false;
    case "array":
      return [];
    case "object":
      if (schema.properties) {
        const obj: Record<string, unknown> = {};
        const schemaProperties = schema.properties ?? {};
        for (const [key, propSchema] of Object.entries(schemaProperties)) {
          obj[key] = getDefaultValue(propSchema);
        }
        return obj;
      }
      return {};
    default:
      return undefined;
  }
}

export { getDefaultValue };
