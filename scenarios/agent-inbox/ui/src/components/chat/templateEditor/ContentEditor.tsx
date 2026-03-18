/**
 * Content editor component for the template editor.
 * Includes the main textarea, undefined variable warnings, and preview toggle.
 */

import { AlertTriangle, ChevronDown, Eye } from "lucide-react";

interface ContentEditorProps {
  content: string;
  onChange: (content: string) => void;
  readOnly: boolean;
  errors: Record<string, string>;
  undefinedVariables: string[];
  showPreview: boolean;
  onTogglePreview: () => void;
  previewContent: () => string;
}

export function ContentEditor({
  content,
  onChange,
  readOnly,
  errors,
  undefinedVariables,
  showPreview,
  onTogglePreview,
  previewContent,
}: ContentEditorProps) {
  return (
    <div className="flex flex-col md:h-full md:min-h-0 md:overflow-y-auto space-y-4">
      {/* Content */}
      <div className="flex-1 flex flex-col">
        <label className="block text-sm font-medium text-slate-300 mb-1">
          Template Content {!readOnly && "*"}
        </label>
        <textarea
          value={content}
          onChange={(e) => onChange(e.target.value)}
          placeholder="Template text with {{variable_name}} placeholders..."
          rows={14}
          disabled={readOnly}
          className={`flex-1 w-full px-3 py-2 bg-slate-800 border rounded-lg text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 font-mono text-sm min-h-[300px] ${
            errors.content ? "border-red-500" : "border-white/10"
          } ${readOnly ? "opacity-70 cursor-not-allowed" : ""}`}
        />
        {errors.content && (
          <p className="text-xs text-red-400 mt-1">{errors.content}</p>
        )}
        {!readOnly && (
          <p className="text-xs text-slate-500 mt-1">
            Use {"{{variable_name}}"} syntax for placeholders
          </p>
        )}

        {/* Undefined variable warning */}
        {undefinedVariables.length > 0 && (
          <div className="mt-2 p-2 bg-amber-900/20 border border-amber-500/30 rounded-lg flex items-start gap-2">
            <AlertTriangle className="h-4 w-4 text-amber-400 flex-shrink-0 mt-0.5" />
            <p className="text-xs text-amber-300">
              Undefined variables: {undefinedVariables.map(v => `{{${v}}}`).join(', ')}
            </p>
          </div>
        )}
      </div>

      {/* Preview toggle */}
      <div>
        <button
          onClick={onTogglePreview}
          className="flex items-center gap-2 text-sm text-slate-300 hover:text-white transition-colors"
        >
          <Eye className="h-4 w-4" />
          {showPreview ? "Hide Preview" : "Show Preview"}
          <ChevronDown
            className={`h-4 w-4 transition-transform ${
              showPreview ? "rotate-180" : ""
            }`}
          />
        </button>
        {showPreview && (
          <div className="mt-2 p-3 bg-slate-800/50 border border-white/5 rounded-lg">
            <pre className="text-sm text-slate-300 whitespace-pre-wrap font-mono">
              {previewContent()}
            </pre>
          </div>
        )}
      </div>
    </div>
  );
}
