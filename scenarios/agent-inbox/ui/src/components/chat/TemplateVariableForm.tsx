/**
 * TemplateVariableForm - Collapsible form for filling template variables.
 *
 * Renders form fields based on template variable definitions.
 * Appears above the message input when a template is active.
 */
import { ChevronDown, ChevronUp } from "lucide-react";
import { useState } from "react";
import type { Template, ActiveTemplate } from "@/lib/types/templates";

interface TemplateVariableFormProps {
  activeTemplate: ActiveTemplate;
  onUpdateVariables: (values: Record<string, string>) => void;
  missingFields: string[];
}

export function TemplateVariableForm({
  activeTemplate,
  onUpdateVariables,
  missingFields,
}: TemplateVariableFormProps) {
  const [isExpanded, setIsExpanded] = useState(true);
  const { template, variableValues } = activeTemplate;

  const handleChange = (name: string, value: string) => {
    onUpdateVariables({ [name]: value });
  };

  if (template.variables.length === 0) {
    return null;
  }

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
            ({template.variables.length} variable
            {template.variables.length !== 1 ? "s" : ""})
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
          {template.variables.map((variable) => {
            const value = variableValues[variable.name] || "";
            const isMissing = missingFields.includes(variable.label);

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
                    id={`var-${variable.name}`}
                    value={value}
                    onChange={(e) => handleChange(variable.name, e.target.value)}
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
                    id={`var-${variable.name}`}
                    value={value}
                    onChange={(e) => handleChange(variable.name, e.target.value)}
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
                    id={`var-${variable.name}`}
                    type="text"
                    value={value}
                    onChange={(e) => handleChange(variable.name, e.target.value)}
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
