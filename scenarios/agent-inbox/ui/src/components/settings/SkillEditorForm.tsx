import { useCallback, useState } from "react";
import { Eye, ChevronDown, Tag, Construction } from "lucide-react";
import { IconSelector, SKILL_ICON_OPTIONS } from "@/components/shared/IconSelector";
import { CategoryPathEditor } from "@/components/shared/CategoryPathEditor";
import { getSkillModesAtLevel } from "@/data/skills";

interface SkillEditorFormProps {
  name: string;
  onNameChange: (value: string) => void;
  description: string;
  onDescriptionChange: (value: string) => void;
  icon: string;
  onIconChange: (value: string) => void;
  draft: boolean;
  onDraftChange: (value: boolean) => void;
  modes: string[];
  onModesChange: (value: string[]) => void;
  tagsInput: string;
  onTagsInputChange: (value: string) => void;
  targetToolId: string;
  onTargetToolIdChange: (value: string) => void;
  content: string;
  onContentChange: (value: string) => void;
  errors: Record<string, string>;
  readOnly: boolean;
}

export function SkillEditorForm({
  name,
  onNameChange,
  description,
  onDescriptionChange,
  icon,
  onIconChange,
  draft,
  onDraftChange,
  modes,
  onModesChange,
  tagsInput,
  onTagsInputChange,
  targetToolId,
  onTargetToolIdChange,
  content,
  onContentChange,
  errors,
  readOnly,
}: SkillEditorFormProps) {
  const [showPreview, setShowPreview] = useState(false);

  const getSuggestionsAtLevel = useCallback(
    (level: number, parentPath: string[]): string[] => {
      return getSkillModesAtLevel(level, parentPath);
    },
    []
  );

  const inputClass = (hasError: boolean) =>
    `w-full px-3 py-2 bg-slate-800 border rounded-lg text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 ${
      hasError ? "border-red-500" : "border-white/10"
    } ${readOnly ? "opacity-70 cursor-not-allowed" : ""}`;

  return (
    <div className="h-full md:grid md:grid-cols-[1fr_2fr] md:gap-6 space-y-4 md:space-y-0 overflow-y-auto md:overflow-hidden">
      {/* Left Column - Metadata */}
      <div className="space-y-4 md:h-full md:min-h-0 md:overflow-y-auto md:pr-2">
        {/* Name */}
        <div>
          <label className="block text-sm font-medium text-slate-300 mb-1">
            Name {!readOnly && "*"}
          </label>
          <input
            type="text"
            value={name}
            onChange={(e) => onNameChange(e.target.value)}
            placeholder="e.g., Screaming Architecture"
            disabled={readOnly}
            className={inputClass(!!errors.name)}
          />
          {errors.name && (
            <p className="text-xs text-red-400 mt-1">{errors.name}</p>
          )}
        </div>

        {/* Description */}
        <div>
          <label className="block text-sm font-medium text-slate-300 mb-1">
            Description {!readOnly && "*"}
          </label>
          <input
            type="text"
            value={description}
            onChange={(e) => onDescriptionChange(e.target.value)}
            placeholder="Brief description of what this skill provides"
            disabled={readOnly}
            className={inputClass(!!errors.description)}
          />
          {errors.description && (
            <p className="text-xs text-red-400 mt-1">{errors.description}</p>
          )}
        </div>

        {/* Icon */}
        <div>
          <label className="block text-sm font-medium text-slate-300 mb-1">
            Icon
          </label>
          <IconSelector
            value={icon}
            onChange={onIconChange}
            icons={SKILL_ICON_OPTIONS}
            disabled={readOnly}
          />
        </div>

        {/* Draft Status */}
        {!readOnly && (
          <div className="flex items-center gap-3 p-3 bg-slate-800/50 border border-white/5 rounded-lg">
            <input
              type="checkbox"
              id="draft"
              checked={draft}
              onChange={(e) => onDraftChange(e.target.checked)}
              className="rounded bg-slate-700 border-white/20 text-orange-500 focus:ring-orange-500"
            />
            <div className="flex-1">
              <label htmlFor="draft" className="text-sm text-slate-300 cursor-pointer flex items-center gap-2">
                <Construction className="h-4 w-4 text-orange-400" />
                Mark as draft
              </label>
              <p className="text-xs text-slate-500 mt-0.5">
                Draft skills show a warning that they may not be fully working
              </p>
            </div>
          </div>
        )}

        {/* Category Path */}
        <CategoryPathEditor
          value={modes}
          onChange={onModesChange}
          getSuggestionsAtLevel={getSuggestionsAtLevel}
          label="Category Path"
          placeholder="Select or type category..."
          disabled={readOnly}
        />

        {/* Tags */}
        <div>
          <div className="flex items-center gap-2 mb-1">
            <Tag className="h-4 w-4 text-slate-400" />
            <label className="text-sm font-medium text-slate-300">
              Tags
            </label>
          </div>
          <input
            type="text"
            value={tagsInput}
            onChange={(e) => onTagsInputChange(e.target.value)}
            placeholder="architecture, audit, clean-code, domain-driven"
            disabled={readOnly}
            className={`w-full px-3 py-2 bg-slate-800 border border-white/10 rounded-lg text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 ${
              readOnly ? "opacity-70 cursor-not-allowed" : ""
            }`}
          />
          {!readOnly && (
            <p className="text-xs text-slate-500 mt-1">
              Comma-separated tags for search and filtering
            </p>
          )}
        </div>

        {/* Target Tool ID (optional) */}
        <div>
          <label className="block text-sm font-medium text-slate-300 mb-1">
            Target Tool ID {!readOnly && "(optional)"}
          </label>
          <input
            type="text"
            value={targetToolId}
            onChange={(e) => onTargetToolIdChange(e.target.value)}
            placeholder="e.g., spawn_coding_agent"
            disabled={readOnly}
            className={`w-full px-3 py-2 bg-slate-800 border border-white/10 rounded-lg text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 ${
              readOnly ? "opacity-70 cursor-not-allowed" : ""
            }`}
          />
          {!readOnly && (
            <p className="text-xs text-slate-500 mt-1">
              If set, this skill will only be sent to the specified tool
            </p>
          )}
        </div>
      </div>

      {/* Right Column - Content */}
      <div className="flex flex-col md:h-full md:min-h-0 md:overflow-y-auto space-y-4">
        {/* Content */}
        <div className="flex-1 flex flex-col">
          <label className="block text-sm font-medium text-slate-300 mb-1">
            Skill Content {!readOnly && "*"}
          </label>
          <textarea
            value={content}
            onChange={(e) => onContentChange(e.target.value)}
            placeholder="The methodology, knowledge, or expertise content that will be injected into the agent's context..."
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
              Use Markdown formatting. This content will be injected into the agent's context when the skill is activated.
            </p>
          )}
        </div>

        {/* Preview toggle */}
        <div>
          <button
            type="button"
            onClick={() => setShowPreview(!showPreview)}
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
            <div className="mt-2 p-3 bg-slate-800/50 border border-white/5 rounded-lg max-h-64 overflow-y-auto">
              <pre className="text-sm text-slate-300 whitespace-pre-wrap font-mono">
                {content || "(no content)"}
              </pre>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
