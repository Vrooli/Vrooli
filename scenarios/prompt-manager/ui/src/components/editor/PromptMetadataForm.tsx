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
  disabled?: boolean
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
  disabled = false,
}: PromptMetadataFormProps) {
  return (
    <div className="space-y-4">
      {/* Name and Icon row */}
      <div className="flex gap-3">
        <IconSelector
          value={formState.icon}
          onChange={(v) => onFieldChange('icon', v)}
          disabled={disabled}
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
            disabled={disabled}
            className={cn(
              'w-full px-3 py-2 bg-slate-800 border rounded-lg text-white text-sm',
              'placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500',
              validation.errors.name ? 'border-red-500' : 'border-white/10',
              disabled && 'opacity-50 cursor-not-allowed'
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
          disabled={disabled}
          rows={2}
          className={cn(
            'w-full px-3 py-2 bg-slate-800 border rounded-lg text-white text-sm resize-none',
            'placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500',
            validation.errors.description ? 'border-red-500' : 'border-white/10',
            disabled && 'opacity-50 cursor-not-allowed'
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
        disabled={disabled}
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
            disabled={disabled}
            className={cn(
              'w-full px-3 py-2 bg-slate-800 border border-white/10 rounded-lg text-white text-sm',
              'placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500',
              disabled && 'opacity-50 cursor-not-allowed'
            )}
          />
          <p className="mt-1 text-xs text-slate-500">Separate multiple tags with commas</p>
        </div>

        <div className="flex items-end pb-6">
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={formState.draft}
              onChange={(e) => onFieldChange('draft', e.target.checked)}
              disabled={disabled}
              className={cn(
                'w-4 h-4 rounded border-white/20 bg-slate-800',
                'text-indigo-600 focus:ring-indigo-500 focus:ring-offset-0',
                disabled && 'opacity-50 cursor-not-allowed'
              )}
            />
            <span className="text-sm text-slate-300">Draft</span>
          </label>
        </div>
      </div>

      {/* Target Tool ID (optional, advanced) */}
      <details className="group">
        <summary className="text-xs text-slate-500 cursor-pointer hover:text-slate-300 transition-colors">
          Advanced options
        </summary>
        <div className="mt-2 pt-2 border-t border-white/5">
          <label htmlFor="targetToolId" className="block text-sm font-medium text-slate-300 mb-1">
            Target Tool ID
          </label>
          <input
            id="targetToolId"
            type="text"
            value={formState.targetToolId}
            onChange={(e) => onFieldChange('targetToolId', e.target.value)}
            placeholder="Optional tool ID this prompt targets..."
            disabled={disabled}
            className={cn(
              'w-full px-3 py-2 bg-slate-800 border border-white/10 rounded-lg text-white text-sm',
              'placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500',
              disabled && 'opacity-50 cursor-not-allowed'
            )}
          />
        </div>
      </details>
    </div>
  )
}
