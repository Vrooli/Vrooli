/**
 * ArrayFormField - Renders an array field in a schema form.
 * Supports adding, removing, and editing items.
 */

import { Plus, Trash2 } from "lucide-react";
import type { ParameterSchema } from "../../lib/api";
import { Button } from "../ui/button";
import { getDefaultValue, formatLabel, labelClass } from "./schemaFormUtils";
import { SchemaFormField } from "./SchemaFormField";

interface ArrayFormFieldProps {
  name: string;
  schema: ParameterSchema;
  value: unknown;
  onChange: (value: unknown) => void;
  required?: boolean;
  disabled?: boolean;
  depth?: number;
}

export function ArrayFormField({
  name,
  schema,
  value,
  onChange,
  required = false,
  disabled = false,
  depth = 0,
}: ArrayFormFieldProps) {
  const { description } = schema;
  const containerStyle = depth > 0 ? { marginLeft: `${depth * 16}px` } : {};
  const arrayValue = (value as unknown[]) ?? [];
  const itemSchema = schema.items!;

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
