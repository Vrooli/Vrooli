/**
 * CombineActionBar - Footer component for combine mode.
 *
 * Shows selection count, format selector, and action buttons
 * for combining and copying selected skills.
 */

import { FileCode, FileText, Braces, X, Copy, Check } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { CombineFormat } from '@/stores/combineStore'

interface CombineActionBarProps {
  selectedCount: number
  format: CombineFormat
  onFormatChange: (format: CombineFormat) => void
  onCopy: () => void
  onCancel: () => void
  isCopying: boolean
  copySuccess: boolean
}

const FORMAT_OPTIONS: Array<{ value: CombineFormat; label: string; icon: React.ReactNode }> = [
  { value: 'xml', label: 'XML', icon: <FileCode className="h-3.5 w-3.5" /> },
  { value: 'markdown', label: 'MD', icon: <FileText className="h-3.5 w-3.5" /> },
  { value: 'json', label: 'JSON', icon: <Braces className="h-3.5 w-3.5" /> },
]

export function CombineActionBar({
  selectedCount,
  format,
  onFormatChange,
  onCopy,
  onCancel,
  isCopying,
  copySuccess,
}: CombineActionBarProps) {
  return (
    <div className="flex flex-col gap-2">
      {/* Selection count and format selector */}
      <div className="flex items-center justify-between">
        <span className="text-xs text-muted-foreground">
          {selectedCount} skill{selectedCount !== 1 ? 's' : ''} selected
        </span>
        <div className="flex items-center gap-1">
          {FORMAT_OPTIONS.map((option) => (
            <button
              key={option.value}
              type="button"
              onClick={() => onFormatChange(option.value)}
              className={cn(
                'flex items-center gap-1 px-2 py-1 text-[10px] rounded transition-colors',
                format === option.value
                  ? 'bg-primary text-primary-foreground'
                  : 'bg-muted text-muted-foreground hover:bg-muted/80 hover:text-foreground'
              )}
              title={option.label}
            >
              {option.icon}
              <span className="hidden sm:inline">{option.label}</span>
            </button>
          ))}
        </div>
      </div>

      {/* Action buttons */}
      <div className="flex items-center gap-2">
        <button
          type="button"
          onClick={onCancel}
          className={cn(
            'flex-1 flex items-center justify-center gap-2 px-3 py-2 text-sm',
            'bg-muted hover:bg-muted/80 text-foreground rounded-lg transition-colors'
          )}
        >
          <X className="h-4 w-4" />
          Cancel
        </button>
        <button
          type="button"
          onClick={onCopy}
          disabled={selectedCount === 0 || isCopying}
          className={cn(
            'flex-1 flex items-center justify-center gap-2 px-3 py-2 text-sm',
            'rounded-lg transition-colors',
            copySuccess
              ? 'bg-green-600 hover:bg-green-600 text-white'
              : 'bg-primary hover:bg-primary/90 text-primary-foreground',
            (selectedCount === 0 || isCopying) && 'opacity-50 cursor-not-allowed'
          )}
        >
          {copySuccess ? (
            <>
              <Check className="h-4 w-4" />
              Copied!
            </>
          ) : isCopying ? (
            <>
              <div className="h-4 w-4 border-2 border-current border-t-transparent rounded-full animate-spin" />
              Copying...
            </>
          ) : (
            <>
              <Copy className="h-4 w-4" />
              Copy Combined
            </>
          )}
        </button>
      </div>
    </div>
  )
}
