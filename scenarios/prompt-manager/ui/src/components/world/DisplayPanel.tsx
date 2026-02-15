/**
 * DisplayPanel - UI for displaying selected skills.
 * Shows selection count, format options, and preview.
 */

import { useState, useMemo } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Copy, Check, X, FileCode, FileText, Braces, Terminal, ChevronDown, ChevronUp } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { api } from '@/lib/api'
import type { Skill } from '@/types'
import type { DisplayFormat } from '@/types/world'
import { displaySkills, generatePreview, validateForDisplay } from '@/services/skillDisplayService'

interface DisplayPanelProps {
  selectedSkills: Skill[]
  onClear: () => void
  onDisplay?: (combined: string, format: DisplayFormat) => void
}

const FORMAT_OPTIONS: Array<{ value: DisplayFormat; label: string; icon: React.ReactNode }> = [
  { value: 'xml', label: 'XML', icon: <FileCode className="h-4 w-4" /> },
  { value: 'markdown', label: 'Markdown', icon: <FileText className="h-4 w-4" /> },
  { value: 'json', label: 'JSON', icon: <Braces className="h-4 w-4" /> },
  { value: 'cli', label: 'CLI', icon: <Terminal className="h-4 w-4" /> },
]

export function DisplayPanel({ selectedSkills, onClear, onDisplay }: DisplayPanelProps) {
  const [format, setFormat] = useState<DisplayFormat>('xml')
  const [copied, setCopied] = useState(false)
  const [showPreview, setShowPreview] = useState(false)

  // Validation
  const validation = useMemo(
    () => validateForDisplay(selectedSkills),
    [selectedSkills]
  )

  // Display output
  const displayResult = useMemo(
    () => displaySkills(selectedSkills, format),
    [selectedSkills, format]
  )

  // Preview
  const preview = useMemo(
    () => generatePreview(selectedSkills, format, 800),
    [selectedSkills, format]
  )

  // Handle copy - uses API for authoritative display
  const handleCopy = async () => {
    try {
      // Get displayed content from API
      const identifiers = selectedSkills.map((p) => p.id)
      const response = await api.displaySkills(identifiers, format)

      // Copy to clipboard
      await navigator.clipboard.writeText(response.combined)
      setCopied(true)
      onDisplay?.(response.combined, format)
      setTimeout(() => setCopied(false), 2000)
    } catch (error) {
      console.error('Failed to display and copy skills:', error)
      // Fallback to client-side display if API fails
      try {
        await navigator.clipboard.writeText(displayResult.combined)
        setCopied(true)
        onDisplay?.(displayResult.combined, format)
        setTimeout(() => setCopied(false), 2000)
      } catch (fallbackError) {
        console.error('Fallback copy also failed:', fallbackError)
      }
    }
  }

  if (selectedSkills.length === 0) {
    return null
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: 20 }}
      className="absolute bottom-4 left-4 right-4 bg-card/95 backdrop-blur-sm border border-border rounded-lg shadow-xl overflow-hidden"
    >
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-border">
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium text-foreground">
              {selectedSkills.length} skill{selectedSkills.length !== 1 ? 's' : ''} selected
            </span>
            <span className="text-xs text-muted-foreground">
              ~{displayResult.totalTokens.toLocaleString()} tokens
            </span>
          </div>

          {/* Warnings */}
          {validation.warnings.length > 0 && (
            <span className="text-xs text-amber-400" title={validation.warnings.join('\n')}>
              {validation.warnings.length} warning{validation.warnings.length !== 1 ? 's' : ''}
            </span>
          )}
        </div>

        <button
          onClick={onClear}
          className="p-1 text-muted-foreground hover:text-foreground transition-colors"
          title="Clear selection"
        >
          <X className="h-4 w-4" />
        </button>
      </div>

      {/* Selected skills list */}
      <div className="px-4 py-2 border-b border-border">
        <div className="flex flex-wrap gap-2">
          {selectedSkills.map((skill) => (
            <span
              key={skill.id}
              className="inline-flex items-center gap-1 px-2 py-1 bg-muted text-foreground text-xs rounded-md"
            >
              {skill.name}
            </span>
          ))}
        </div>
      </div>

      {/* Format selection and actions */}
      <div className="flex items-center justify-between px-4 py-3">
        <div className="flex items-center gap-2">
          <span className="text-xs text-muted-foreground">Format:</span>
          <div className="flex gap-1">
            {FORMAT_OPTIONS.map((option) => (
              <button
                key={option.value}
                onClick={() => setFormat(option.value)}
                className={`flex items-center gap-1.5 px-2 py-1 text-xs rounded transition-colors ${
                  format === option.value
                    ? 'bg-indigo-600 text-white'
                    : 'bg-muted text-foreground hover:bg-muted/70'
                }`}
              >
                {option.icon}
                {option.label}
              </button>
            ))}
          </div>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={() => setShowPreview(!showPreview)}
            className="flex items-center gap-1 px-2 py-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
          >
            Preview
            {showPreview ? <ChevronDown className="h-3 w-3" /> : <ChevronUp className="h-3 w-3" />}
          </button>

          <Button
            size="sm"
            onClick={() => void handleCopy()}
            disabled={!validation.valid}
            className="gap-1.5"
          >
            {copied ? (
              <>
                <Check className="h-4 w-4" />
                Copied!
              </>
            ) : (
              <>
                <Copy className="h-4 w-4" />
                Copy Display
              </>
            )}
          </Button>
        </div>
      </div>

      {/* Preview section */}
      <AnimatePresence>
        {showPreview && (
          <motion.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: 'auto', opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            className="border-t border-border"
          >
            <pre className="p-4 text-xs font-mono text-foreground bg-muted/50 max-h-48 overflow-auto whitespace-pre-wrap">
              {preview}
            </pre>
          </motion.div>
        )}
      </AnimatePresence>
    </motion.div>
  )
}
