/**
 * CombinePanel - UI for combining selected skills.
 * Shows selection count, format options, and preview.
 */

import { useState, useMemo } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Copy, Check, X, FileCode, FileText, Braces, ChevronDown, ChevronUp } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { api } from '@/lib/api'
import type { Skill } from '@/types'
import type { CombineFormat } from '@/types/world'
import { combineSkills, generatePreview, validateForCombine } from '@/services/skillCombineService'

interface CombinePanelProps {
  selectedSkills: Skill[]
  onClear: () => void
  onCombine?: (combined: string, format: CombineFormat) => void
}

const FORMAT_OPTIONS: Array<{ value: CombineFormat; label: string; icon: React.ReactNode }> = [
  { value: 'xml', label: 'XML', icon: <FileCode className="h-4 w-4" /> },
  { value: 'markdown', label: 'Markdown', icon: <FileText className="h-4 w-4" /> },
  { value: 'json', label: 'JSON', icon: <Braces className="h-4 w-4" /> },
]

export function CombinePanel({ selectedSkills, onClear, onCombine }: CombinePanelProps) {
  const [format, setFormat] = useState<CombineFormat>('xml')
  const [copied, setCopied] = useState(false)
  const [showPreview, setShowPreview] = useState(false)

  // Validation
  const validation = useMemo(
    () => validateForCombine(selectedSkills),
    [selectedSkills]
  )

  // Combined output
  const combineResult = useMemo(
    () => combineSkills(selectedSkills, format),
    [selectedSkills, format]
  )

  // Preview
  const preview = useMemo(
    () => generatePreview(selectedSkills, format, 800),
    [selectedSkills, format]
  )

  // Handle copy - uses API for authoritative combining
  const handleCopy = async () => {
    try {
      // Get combined content from API
      const skillIds = selectedSkills.map((p) => p.id)
      const response = await api.combineSkills(skillIds, format)

      // Copy to clipboard
      await navigator.clipboard.writeText(response.combined)
      setCopied(true)
      onCombine?.(response.combined, format)
      setTimeout(() => setCopied(false), 2000)
    } catch (error) {
      console.error('Failed to combine and copy skills:', error)
      // Fallback to client-side combining if API fails
      try {
        await navigator.clipboard.writeText(combineResult.combined)
        setCopied(true)
        onCombine?.(combineResult.combined, format)
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
      className="absolute bottom-4 left-4 right-4 bg-slate-800/95 backdrop-blur-sm border border-slate-700 rounded-lg shadow-xl overflow-hidden"
    >
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-slate-700">
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium text-slate-200">
              {selectedSkills.length} skill{selectedSkills.length !== 1 ? 's' : ''} selected
            </span>
            <span className="text-xs text-slate-400">
              ~{combineResult.totalTokens.toLocaleString()} tokens
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
          className="p-1 text-slate-400 hover:text-slate-200 transition-colors"
          title="Clear selection"
        >
          <X className="h-4 w-4" />
        </button>
      </div>

      {/* Selected skills list */}
      <div className="px-4 py-2 border-b border-slate-700/50">
        <div className="flex flex-wrap gap-2">
          {selectedSkills.map((skill) => (
            <span
              key={skill.id}
              className="inline-flex items-center gap-1 px-2 py-1 bg-slate-700/50 text-slate-300 text-xs rounded-md"
            >
              {skill.name}
            </span>
          ))}
        </div>
      </div>

      {/* Format selection and actions */}
      <div className="flex items-center justify-between px-4 py-3">
        <div className="flex items-center gap-2">
          <span className="text-xs text-slate-400">Format:</span>
          <div className="flex gap-1">
            {FORMAT_OPTIONS.map((option) => (
              <button
                key={option.value}
                onClick={() => setFormat(option.value)}
                className={`flex items-center gap-1.5 px-2 py-1 text-xs rounded transition-colors ${
                  format === option.value
                    ? 'bg-indigo-600 text-white'
                    : 'bg-slate-700 text-slate-300 hover:bg-slate-600'
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
            className="flex items-center gap-1 px-2 py-1 text-xs text-slate-400 hover:text-slate-200 transition-colors"
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
                Copy Combined
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
            className="border-t border-slate-700"
          >
            <pre className="p-4 text-xs font-mono text-slate-300 bg-slate-900/50 max-h-48 overflow-auto whitespace-pre-wrap">
              {preview}
            </pre>
          </motion.div>
        )}
      </AnimatePresence>
    </motion.div>
  )
}
