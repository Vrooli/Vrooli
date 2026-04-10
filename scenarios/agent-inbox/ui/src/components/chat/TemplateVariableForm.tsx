/**
 * TemplateVariableForm - Collapsible form for filling template variables.
 *
 * Renders form fields based on template variable definitions.
 * Appears above the message input when a template is active.
 * Supports keyboard navigation with Tab cycling to message input.
 */
import { ChevronDown, ChevronUp } from "lucide-react";
import { useState, useRef, useEffect, useCallback } from "react";
import type { ActiveTemplate } from "@/lib/types/templates";

interface TemplateVariableFormProps {
  activeTemplate: ActiveTemplate;
  onUpdateVariables: (values: Record<string, string>) => void;
  missingFields: string[];
  /** Called when user tabs past the last form field */
  onTabOut?: () => void;
  /** Whether to auto-focus the first field */
  autoFocus?: boolean;
}

export function TemplateVariableForm({
  activeTemplate,
  onUpdateVariables,
  missingFields,
  onTabOut,
  autoFocus = false,
}: TemplateVariableFormProps) {
  const [isExpanded, setIsExpanded] = useState(true);
  const { template, variableValues } = activeTemplate;
  const firstFieldRef = useRef<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>(null);
  const lastFieldRef = useRef<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>(null);

  const handleChange = (name: string, value: string) => {
    onUpdateVariables({ [name]: value });
  };

  // Auto-focus the first field when form appears
  useEffect(() => {
    if (autoFocus && isExpanded && firstFieldRef.current) {
      // Small delay to ensure the form is rendered
      const timer = setTimeout(() => {
        firstFieldRef.current?.focus();
      }, 50);
      return () => clearTimeout(timer);
    }
  }, [autoFocus, isExpanded]);

  // Handle Tab on the last field to move focus to message input
  const handleLastFieldKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Tab" && !e.shiftKey && onTabOut) {
        e.preventDefault();
        onTabOut();
      }
    },
    [onTabOut]
  );

  // Defensive: guard against undefined/empty variables array
  const variables = template.variables ?? [];
  if (variables.length === 0) {
    return null;
  }

  const lastIndex = variables.length - 1;

  return (
    <div
      className="border-b border-white/10 bg-slate-800/50"
      data-testid="template-variable-form"
    >
      {/* Header */}
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className="w-full flex items-center justify-between px-4 py-2 text-sm text-slate-300 hover:text-white transition-colors"
        data-testid="template-form-toggle"
      >
        <span className="flex items-center gap-2">
          <span className="font-medium text-blue-400">{template.name}</span>
          <span className="text-slate-500">
            ({variables.length} variable
            {variables.length !== 1 ? "s" : ""})
          </span>
          {missingFields.length > 0 && (
            <span className="text-xs text-red-400">
              {missingFields.length} required
            </span>
          )}
        </span>
        {isExpanded ? (
          <ChevronUp className="h-4 w-4" />
        ) : (
          <ChevronDown className="h-4 w-4" />
        )}
      </button>

      {/* Form fields */}
      {isExpanded && (
        <div className="px-4 pb-4 space-y-3" data-testid="template-form-fields">
          {variables.map((variable, index) => {
            const value = variableValues[variable.name] || "";
            const isMissing = missingFields.includes(variable.label);
            const isFirst = index === 0;
            const isLast = index === lastIndex;

            return (
              <div key={variable.name} className="space-y-1">
                <label
                  htmlFor={`var-${variable.name}`}
                  className="flex items-center gap-1 text-xs font-medium text-slate-400"
                >
                  {variable.label}
                  {variable.required && (
                    <span className="text-red-400">*</span>
                  )}
                </label>

                {variable.type === "select" ? (
                  <select
                    ref={isFirst ? firstFieldRef as React.RefObject<HTMLSelectElement> : isLast ? lastFieldRef as React.RefObject<HTMLSelectElement> : undefined}
                    id={`var-${variable.name}`}
                    value={value}
                    onChange={(e) => handleChange(variable.name, e.target.value)}
                    onKeyDown={isLast ? handleLastFieldKeyDown : undefined}
                    className={`
                      w-full px-3 py-2 bg-slate-900 border rounded-lg text-white text-sm
                      focus:outline-none focus:ring-2 focus:ring-blue-500/50
                      ${isMissing ? "border-red-500/50" : "border-white/10"}
                    `}
                    data-testid={`var-input-${variable.name}`}
                  >
                    <option value="">Select {variable.label.toLowerCase()}...</option>
                    {variable.options?.map((option) => (
                      <option key={option} value={option}>
                        {option}
                      </option>
                    ))}
                  </select>
                ) : variable.type === "textarea" ? (
                  <textarea
                    ref={isFirst ? firstFieldRef as React.RefObject<HTMLTextAreaElement> : isLast ? lastFieldRef as React.RefObject<HTMLTextAreaElement> : undefined}
                    id={`var-${variable.name}`}
                    value={value}
                    onChange={(e) => handleChange(variable.name, e.target.value)}
                    onKeyDown={isLast ? handleLastFieldKeyDown : undefined}
                    placeholder={variable.placeholder}
                    rows={2}
                    className={`
                      w-full px-3 py-2 bg-slate-900 border rounded-lg text-white text-sm
                      placeholder:text-slate-500 resize-none
                      focus:outline-none focus:ring-2 focus:ring-blue-500/50
                      ${isMissing ? "border-red-500/50" : "border-white/10"}
                    `}
                    data-testid={`var-input-${variable.name}`}
                  />
                ) : (
                  <input
                    ref={isFirst ? firstFieldRef as React.RefObject<HTMLInputElement> : isLast ? lastFieldRef as React.RefObject<HTMLInputElement> : undefined}
                    id={`var-${variable.name}`}
                    type="text"
                    value={value}
                    onChange={(e) => handleChange(variable.name, e.target.value)}
                    onKeyDown={isLast ? handleLastFieldKeyDown : undefined}
                    placeholder={variable.placeholder}
                    className={`
                      w-full px-3 py-2 bg-slate-900 border rounded-lg text-white text-sm
                      placeholder:text-slate-500
                      focus:outline-none focus:ring-2 focus:ring-blue-500/50
                      ${isMissing ? "border-red-500/50" : "border-white/10"}
                    `}
                    data-testid={`var-input-${variable.name}`}
                  />
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
