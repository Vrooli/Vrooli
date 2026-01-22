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
          <label htmlFor="name" className="block text-sm font-medium text-muted-foreground mb-1">
            Name <span className="text-red-400">*</span>
          </label>
          <input
            id="name"
            type="text"
            value={formState.name}
            onChange={(e) => onFieldChange('name', e.target.value)}
            placeholder="Prompt name..."
            className={cn(
              'w-full px-3 py-2 bg-muted border rounded-lg text-foreground text-sm',
              'placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary',
              validation.errors.name ? 'border-red-500' : 'border-border'
            )}
          />
          {validation.errors.name && (
            <p className="mt-1 text-xs text-red-400">{validation.errors.name}</p>
          )}
        </div>
      </div>

      {/* Description */}
      <div>
        <label htmlFor="description" className="block text-sm font-medium text-muted-foreground mb-1">
          Description
        </label>
        <textarea
          id="description"
          value={formState.description}
          onChange={(e) => onFieldChange('description', e.target.value)}
          placeholder="Brief description of what this prompt does..."
          rows={2}
          className={cn(
            'w-full px-3 py-2 bg-muted border rounded-lg text-foreground text-sm resize-none',
            'placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary',
            validation.errors.description ? 'border-red-500' : 'border-border'
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
          <label htmlFor="tags" className="block text-sm font-medium text-muted-foreground mb-1">
            Tags
          </label>
          <input
            id="tags"
            type="text"
            value={formState.tags}
            onChange={(e) => onFieldChange('tags', e.target.value)}
            placeholder="Comma-separated tags..."
            className="w-full px-3 py-2 bg-muted border border-border rounded-lg text-foreground text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
          />
          <p className="mt-1 text-xs text-muted-foreground">Separate multiple tags with commas</p>
        </div>

        <div className="flex items-end pb-6">
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={formState.draft}
              onChange={(e) => onFieldChange('draft', e.target.checked)}
              className="w-4 h-4 rounded border-border bg-muted text-primary focus:ring-primary focus:ring-offset-0"
            />
            <span className="text-sm text-muted-foreground">Draft</span>
          </label>
        </div>
      </div>

      {/* Folder selector */}
      <div>
        <label className="block text-sm font-medium text-muted-foreground mb-2">
          Storage Location
        </label>
        <div className="space-y-2">
          <label
            className={cn(
              'flex items-start gap-3 p-3 rounded-lg border cursor-pointer transition-colors',
              formState.folder === 'internal'
                ? 'border-primary bg-primary/10'
                : 'border-border hover:border-border/80 bg-muted/50'
            )}
          >
            <input
              type="radio"
              name="folder"
              value="internal"
              checked={formState.folder === 'internal'}
              onChange={() => onFieldChange('folder', 'internal')}
              className="mt-0.5 w-4 h-4 text-primary bg-muted border-border focus:ring-primary focus:ring-offset-0"
            />
            <div className="flex-1">
              <div className="text-sm font-medium text-foreground">Internal</div>
              <div className="text-xs text-muted-foreground mt-0.5">
                Personal prompts, gitignored. Only visible on this machine.
              </div>
            </div>
          </label>
          <label
            className={cn(
              'flex items-start gap-3 p-3 rounded-lg border cursor-pointer transition-colors',
              formState.folder === 'core'
                ? 'border-primary bg-primary/10'
                : 'border-border hover:border-border/80 bg-muted/50'
            )}
          >
            <input
              type="radio"
              name="folder"
              value="core"
              checked={formState.folder === 'core'}
              onChange={() => onFieldChange('folder', 'core')}
              className="mt-0.5 w-4 h-4 text-primary bg-muted border-border focus:ring-primary focus:ring-offset-0"
            />
            <div className="flex-1">
              <div className="text-sm font-medium text-foreground">Core</div>
              <div className="text-xs text-muted-foreground mt-0.5">
                Shared prompts, git-tracked. Available across all instances.
              </div>
            </div>
          </label>
        </div>
      </div>
    </div>
  )
}
