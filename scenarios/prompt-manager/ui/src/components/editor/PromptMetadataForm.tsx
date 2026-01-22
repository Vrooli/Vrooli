/**
 * PromptMetadataForm - Form fields for prompt metadata.
 *
 * Includes:
 * - Name input
 * - Description textarea
 * - Icon selector
 * - Tags input
 * - Draft toggle
 * - Category path (modes) editor
 */

import { cn } from '@/lib/utils'
import type { PromptFormState, ValidationResult } from '@/types/editor'
import { IconSelector } from '../shared/IconSelector'
import { CategoryPathEditor } from './CategoryPathEditor'

interface PromptMetadataFormProps {
  formState: PromptFormState
  onFieldChange: <K extends keyof PromptFormState>(field: K, value: PromptFormState[K]) => void
  onModesChange: (modes: string[]) => void
  getSuggestionsAtLevel: (level: number, parentPath: string[]) => string[]
  validation: ValidationResult
}

/**
 * Metadata form component for prompt editing.
 */
export function PromptMetadataForm({
  formState,
  onFieldChange,
  onModesChange,
  getSuggestionsAtLevel,
  validation,
}: PromptMetadataFormProps) {
  return (
    <div className="space-y-4">
      {/* Name and Icon row */}
      <div className="flex gap-3">
        <IconSelector
          value={formState.icon}
          onChange={(v) => onFieldChange('icon', v)}
        />
        <div className="flex-1">
          <label htmlFor="name" className="block text-sm font-medium text-slate-300 mb-1">
            Name <span className="text-red-400">*</span>
          </label>
          <input
            id="name"
            type="text"
            value={formState.name}
            onChange={(e) => onFieldChange('name', e.target.value)}
            placeholder="Prompt name..."
            className={cn(
              'w-full px-3 py-2 bg-slate-800 border rounded-lg text-white text-sm',
              'placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500',
              validation.errors.name ? 'border-red-500' : 'border-white/10'
            )}
          />
          {validation.errors.name && (
            <p className="mt-1 text-xs text-red-400">{validation.errors.name}</p>
          )}
        </div>
      </div>

      {/* Description */}
      <div>
        <label htmlFor="description" className="block text-sm font-medium text-slate-300 mb-1">
          Description
        </label>
        <textarea
          id="description"
          value={formState.description}
          onChange={(e) => onFieldChange('description', e.target.value)}
          placeholder="Brief description of what this prompt does..."
          rows={2}
          className={cn(
            'w-full px-3 py-2 bg-slate-800 border rounded-lg text-white text-sm resize-none',
            'placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500',
            validation.errors.description ? 'border-red-500' : 'border-white/10'
          )}
        />
        {validation.errors.description && (
          <p className="mt-1 text-xs text-red-400">{validation.errors.description}</p>
        )}
      </div>

      {/* Modes (Category Path) */}
      <CategoryPathEditor
        value={formState.modes}
        onChange={onModesChange}
        getSuggestionsAtLevel={getSuggestionsAtLevel}
        label="Modes"
        placeholder="Select or type mode..."
      />

      {/* Tags and Draft row */}
      <div className="flex gap-4">
        <div className="flex-1">
          <label htmlFor="tags" className="block text-sm font-medium text-slate-300 mb-1">
            Tags
          </label>
          <input
            id="tags"
            type="text"
            value={formState.tags}
            onChange={(e) => onFieldChange('tags', e.target.value)}
            placeholder="Comma-separated tags..."
            className="w-full px-3 py-2 bg-slate-800 border border-white/10 rounded-lg text-white text-sm placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500"
          />
          <p className="mt-1 text-xs text-slate-500">Separate multiple tags with commas</p>
        </div>

        <div className="flex items-end pb-6">
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={formState.draft}
              onChange={(e) => onFieldChange('draft', e.target.checked)}
              className="w-4 h-4 rounded border-white/20 bg-slate-800 text-indigo-600 focus:ring-indigo-500 focus:ring-offset-0"
            />
            <span className="text-sm text-slate-300">Draft</span>
          </label>
        </div>
      </div>

      {/* Folder selector */}
      <div>
        <label className="block text-sm font-medium text-slate-300 mb-2">
          Storage Location
        </label>
        <div className="space-y-2">
          <label
            className={cn(
              'flex items-start gap-3 p-3 rounded-lg border cursor-pointer transition-colors',
              formState.folder === 'internal'
                ? 'border-indigo-500 bg-indigo-500/10'
                : 'border-white/10 hover:border-white/20 bg-slate-800/50'
            )}
          >
            <input
              type="radio"
              name="folder"
              value="internal"
              checked={formState.folder === 'internal'}
              onChange={() => onFieldChange('folder', 'internal')}
              className="mt-0.5 w-4 h-4 text-indigo-600 bg-slate-800 border-white/20 focus:ring-indigo-500 focus:ring-offset-0"
            />
            <div className="flex-1">
              <div className="text-sm font-medium text-white">Internal</div>
              <div className="text-xs text-slate-400 mt-0.5">
                Personal prompts, gitignored. Only visible on this machine.
              </div>
            </div>
          </label>
          <label
            className={cn(
              'flex items-start gap-3 p-3 rounded-lg border cursor-pointer transition-colors',
              formState.folder === 'core'
                ? 'border-indigo-500 bg-indigo-500/10'
                : 'border-white/10 hover:border-white/20 bg-slate-800/50'
            )}
          >
            <input
              type="radio"
              name="folder"
              value="core"
              checked={formState.folder === 'core'}
              onChange={() => onFieldChange('folder', 'core')}
              className="mt-0.5 w-4 h-4 text-indigo-600 bg-slate-800 border-white/20 focus:ring-indigo-500 focus:ring-offset-0"
            />
            <div className="flex-1">
              <div className="text-sm font-medium text-white">Core</div>
              <div className="text-xs text-slate-400 mt-0.5">
                Shared prompts, git-tracked. Available across all instances.
              </div>
            </div>
          </label>
        </div>
      </div>
    </div>
  )
}
