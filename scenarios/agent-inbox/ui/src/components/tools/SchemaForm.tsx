/**
 * SchemaForm Component
 *
 * A container component that renders a form from JSON Schema (ToolParameters).
 * Handles form state, validation, and submission.
 */

import { useState, useCallback, useMemo } from "react";
import { Loader2 } from "lucide-react";
import type { ToolParameters } from "../../lib/api";
import { Button } from "../ui/button";
import { SchemaFormField, getDefaultValue } from "./SchemaFormField";

export interface SchemaFormProps {
  /** Tool parameters schema */
  parameters: ToolParameters;
  /** Initial form values */
  initialValues?: Record<string, unknown>;
  /** Callback when form is submitted with valid values */
  onSubmit: (values: Record<string, unknown>) => void;
  /** Callback when cancel is clicked */
  onCancel?: () => void;
  /** Whether form is currently submitting */
  isSubmitting?: boolean;
  /** Custom submit button text */
  submitLabel?: string;
  /** Custom cancel button text */
  cancelLabel?: string;
}

/**
 * Renders a form from tool parameters schema.
 */
export function SchemaForm({
  parameters,
  initialValues,
  onSubmit,
  onCancel,
  isSubmitting = false,
  submitLabel = "Execute",
  cancelLabel = "Cancel",
}: SchemaFormProps) {
  // Initialize form values with defaults
  // Defensive: parameters.properties might be undefined for tools with no parameters
  const defaultValues = useMemo(() => {
    const values: Record<string, unknown> = {};
    const properties = parameters?.properties ?? {};
    for (const [name, schema] of Object.entries(properties)) {
      if (initialValues?.[name] !== undefined) {
        values[name] = initialValues[name];
      } else {
        values[name] = getDefaultValue(schema);
      }
    }
    return values;
  }, [parameters?.properties, initialValues]);

  const [values, setValues] = useState<Record<string, unknown>>(defaultValues);
  const [errors, setErrors] = useState<string[]>([]);

  // Update a single field value
  const updateField = useCallback((name: string, value: unknown) => {
    setValues((prev) => ({ ...prev, [name]: value }));
    // Clear errors when user makes changes
    setErrors([]);
  }, []);

  // Validate the form
  const validate = useCallback((): boolean => {
    const validationErrors: string[] = [];
    const requiredFields = parameters?.required ?? [];
    const props = parameters?.properties ?? {};

    for (const fieldName of requiredFields) {
      const value = values[fieldName];
      if (value === undefined || value === null || value === "") {
        // Only look up schema if properties exist
        const schema = props[fieldName];
        const label = fieldName
          .replace(/_/g, " ")
          .replace(/([a-z])([A-Z])/g, "$1 $2")
          .replace(/^./, (c) => c.toUpperCase());
        validationErrors.push(`${label} is required`);
      }
    }

    setErrors(validationErrors);
    return validationErrors.length === 0;
  }, [values, parameters]);

  // Handle form submission
  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      if (validate()) {
        // Filter out undefined values
        const cleanedValues: Record<string, unknown> = {};
        for (const [key, value] of Object.entries(values)) {
          if (value !== undefined && value !== "") {
            cleanedValues[key] = value;
          }
        }
        onSubmit(cleanedValues);
      }
    },
    [validate, values, onSubmit]
  );

  const required = parameters?.required ?? [];
  const properties = parameters?.properties ?? {};
  const propertyEntries = Object.entries(properties);

  // No parameters - just show submit button
  if (propertyEntries.length === 0) {
    return (
      <form onSubmit={handleSubmit} className="space-y-4">
        <p className="text-sm text-slate-400">
          This tool has no parameters. Click execute to run it.
        </p>
        <div className="flex justify-end gap-3 pt-4 border-t border-white/10">
          {onCancel && (
            <Button
              type="button"
              variant="secondary"
              onClick={onCancel}
              disabled={isSubmitting}
            >
              {cancelLabel}
            </Button>
          )}
          <Button type="submit" disabled={isSubmitting}>
            {isSubmitting ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                Executing...
              </>
            ) : (
              submitLabel
            )}
          </Button>
        </div>
      </form>
    );
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4" data-testid="schema-form">
      {/* Form fields */}
      <div className="space-y-1">
        {propertyEntries.map(([name, schema]) => (
          <SchemaFormField
            key={name}
            name={name}
            schema={schema}
            value={values[name]}
            onChange={(v) => updateField(name, v)}
            required={required.includes(name)}
            disabled={isSubmitting}
          />
        ))}
      </div>

      {/* Validation errors */}
      {errors.length > 0 && (
        <div
          className="bg-red-500/10 border border-red-500/30 rounded-md p-3"
          data-testid="validation-errors"
        >
          <ul className="text-sm text-red-400 list-disc list-inside">
            {errors.map((error, i) => (
              <li key={i}>{error}</li>
            ))}
          </ul>
        </div>
      )}

      {/* Actions */}
      <div className="flex justify-end gap-3 pt-4 border-t border-white/10">
        {onCancel && (
          <Button
            type="button"
            variant="secondary"
            onClick={onCancel}
            disabled={isSubmitting}
          >
            {cancelLabel}
          </Button>
        )}
        <Button type="submit" disabled={isSubmitting}>
          {isSubmitting ? (
            <>
              <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              Executing...
            </>
          ) : (
            submitLabel
          )}
        </Button>
      </div>
    </form>
  );
}
