/**
 * TemplateIndicator - Shows when a template is active for the current message.
 *
 * Displays a small pill/badge in the message input footer area.
 * Uses blue color scheme to distinguish from green web search and purple forced tool.
 */
import { FileText, Pencil, X } from "lucide-react";
import type { Template } from "@/lib/types/templates";

interface TemplateIndicatorProps {
  template: Template;
  onClear: () => void;
  onEdit: () => void;
}

export function TemplateIndicator({
  template,
  onClear,
  onEdit,
}: TemplateIndicatorProps) {
  return (
    <div
      className="inline-flex items-center gap-1.5 px-2 py-1 rounded-full bg-blue-500/20 border border-blue-500/30 text-xs text-blue-400"
      data-testid="template-indicator"
    >
      <FileText className="h-3 w-3" />
      <span
        className="max-w-[150px] truncate"
        title={`Template: ${template.name}`}
      >
        {template.name}
      </span>
      <button
        onClick={onEdit}
        className="hover:text-blue-300 transition-colors"
        aria-label="Edit template variables"
        data-testid="template-edit-button"
      >
        <Pencil className="h-3 w-3" />
      </button>
      <button
        onClick={onClear}
        className="hover:text-blue-300 transition-colors"
        aria-label="Clear template"
        data-testid="template-clear-button"
      >
        <X className="h-3 w-3" />
      </button>
    </div>
  );
}
