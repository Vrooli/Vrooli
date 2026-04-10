/**
 * Utility functions for SchemaFormField.
 */

import type { ParameterSchema } from "../../lib/api";

/**
 * Get a default value for a schema type.
 */
export function getDefaultValue(schema: ParameterSchema): unknown {
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

/**
 * Format a field name into a human-readable label.
 */
export const formatLabel = (s: string) =>
  s
    .replace(/_/g, " ")
    .replace(/([a-z])([A-Z])/g, "$1 $2")
    .replace(/^./, (c) => c.toUpperCase());

/** Common CSS class for field labels */
export const labelClass = "block text-sm font-medium text-slate-300 mb-1";

/** Common CSS class for inputs */
export const inputClass =
  "w-full bg-white/5 border border-white/10 rounded-md px-3 py-2 text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/50 disabled:opacity-50";
