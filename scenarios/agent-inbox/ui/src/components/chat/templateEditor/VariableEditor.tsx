/**
 * Variable editor component for the template editor.
 * Allows adding, editing, and removing template variables.
 */

import { Plus, Trash2 } from "lucide-react";
import type { TemplateVariable } from "@/lib/types/templates";
import { VARIABLE_TYPES } from "./types";

interface VariableEditorProps {
  variables: TemplateVariable[];
  errors: Record<string, string>;
  readOnly: boolean;
  onAdd: () => void;
  onUpdate: (index: number, updates: Partial<TemplateVariable>) => void;
  onRemove: (index: number) => void;
}

export function VariableEditor({
  variables,
  errors,
  readOnly,
  onAdd,
  onUpdate,
  onRemove,
}: VariableEditorProps) {
  return (
    <div>
      <div className="flex items-center justify-between mb-2">
        <label className="text-sm font-medium text-slate-300">
          Variables
        </label>
        {!readOnly && (
          <button
            onClick={onAdd}
            className="flex items-center gap-1 px-2 py-1 text-xs rounded bg-indigo-600/20 text-indigo-300 hover:bg-indigo-600/30 transition-colors"
          >
            <Plus className="h-3 w-3" />
            Add Variable
          </button>
        )}
      </div>

      {variables.length === 0 ? (
        <p className="text-sm text-slate-500 italic">
          No variables defined yet.
        </p>
      ) : (
        <div className="space-y-3">
          {variables.map((variable, index) => (
            <VariableRow
              key={index}
              variable={variable}
              index={index}
              errors={errors}
              onUpdate={onUpdate}
              onRemove={onRemove}
            />
          ))}
        </div>
      )}
    </div>
  );
}

interface VariableRowProps {
  variable: TemplateVariable;
  index: number;
  errors: Record<string, string>;
  onUpdate: (index: number, updates: Partial<TemplateVariable>) => void;
  onRemove: (index: number) => void;
}

function VariableRow({ variable, index, errors, onUpdate, onRemove }: VariableRowProps) {
  return (
    <div className="p-3 bg-slate-800/50 border border-white/5 rounded-lg">
      <div className="flex items-start gap-2">
        <div className="flex-1 grid grid-cols-2 gap-2">
          <div>
            <input
              type="text"
              value={variable.name}
              onChange={(e) =>
                onUpdate(index, {
                  name: e.target.value.replace(/\s+/g, "_"),
                })
              }
              placeholder="variable_name"
              className={`w-full px-2 py-1 text-sm bg-slate-700 border rounded text-white placeholder:text-slate-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 ${
                errors[`variable_${index}_name`]
                  ? "border-red-500"
                  : "border-white/10"
              }`}
            />
            <p className="text-xs text-slate-500 mt-0.5">
              Name (no spaces)
            </p>
          </div>
          <div>
            <input
              type="text"
              value={variable.label}
              onChange={(e) =>
                onUpdate(index, { label: e.target.value })
              }
              placeholder="Display Label"
              className={`w-full px-2 py-1 text-sm bg-slate-700 border rounded text-white placeholder:text-slate-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 ${
                errors[`variable_${index}_label`]
                  ? "border-red-500"
                  : "border-white/10"
              }`}
            />
            <p className="text-xs text-slate-500 mt-0.5">
              Label
            </p>
          </div>
          <div>
            <select
              value={variable.type}
              onChange={(e) =>
                onUpdate(index, {
                  type: e.target.value as TemplateVariable["type"],
                })
              }
              className="w-full px-2 py-1 text-sm bg-slate-700 border border-white/10 rounded text-white focus:outline-none focus:ring-1 focus:ring-indigo-500"
            >
              {VARIABLE_TYPES.map((t) => (
                <option key={t} value={t}>
                  {t}
                </option>
              ))}
            </select>
            <p className="text-xs text-slate-500 mt-0.5">
              Type
            </p>
          </div>
          <div>
            <input
              type="text"
              value={variable.placeholder || ""}
              onChange={(e) =>
                onUpdate(index, {
                  placeholder: e.target.value,
                })
              }
              placeholder="Placeholder text..."
              className="w-full px-2 py-1 text-sm bg-slate-700 border border-white/10 rounded text-white placeholder:text-slate-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
            />
            <p className="text-xs text-slate-500 mt-0.5">
              Placeholder
            </p>
          </div>
        </div>
        <button
          onClick={() => onRemove(index)}
          className="p-1 rounded hover:bg-white/10 text-slate-400 hover:text-red-400 transition-colors"
        >
          <Trash2 className="h-4 w-4" />
        </button>
      </div>

      {/* Options for select type */}
      {variable.type === "select" && (
        <div className="mt-2">
          <input
            type="text"
            value={variable.options?.join(", ") || ""}
            onChange={(e) =>
              onUpdate(index, {
                options: e.target.value
                  .split(",")
                  .map((o) => o.trim())
                  .filter(Boolean),
              })
            }
            placeholder="Option 1, Option 2, Option 3"
            className={`w-full px-2 py-1 text-sm bg-slate-700 border rounded text-white placeholder:text-slate-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 ${
              errors[`variable_${index}_options`]
                ? "border-red-500"
                : "border-white/10"
            }`}
          />
          <p className="text-xs text-slate-500 mt-0.5">
            Options (comma-separated)
          </p>
        </div>
      )}

      {/* Required checkbox */}
      <div className="mt-2 flex items-center gap-2">
        <input
          type="checkbox"
          id={`required_${index}`}
          checked={variable.required || false}
          onChange={(e) =>
            onUpdate(index, { required: e.target.checked })
          }
          className="rounded bg-slate-700 border-white/20 text-indigo-500 focus:ring-indigo-500"
        />
        <label
          htmlFor={`required_${index}`}
          className="text-xs text-slate-400"
        >
          Required
        </label>
      </div>
    </div>
  );
}
