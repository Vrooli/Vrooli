import { useMemo, useState } from 'react'
import { Loader2, Sparkles, Eye, EyeOff, Wand2, Maximize2, Minimize2, CheckCircle, Lightbulb, WrapText } from 'lucide-react'
import { Button } from '../ui/button'
import { Textarea } from '../ui/textarea'
import { Input } from '../ui/input'
import { DiffViewer } from './DiffViewer'
import type { AIAction } from '../../hooks/useAIAssistant'

const DEFAULT_SECTIONS = [
  '🎯 Overview',
  '🎯 Operational Targets',
  '🧱 Tech Direction Snapshot',
  '🤝 Dependencies & Launch Plan',
  '🎨 UX & Branding',
  '📎 Appendix',
]

interface AIAssistantDialogProps {
  open: boolean
  section: string
  context: string
  action: AIAction
  generating: boolean
  result: string | null
  error: string | null
  hasSelection: boolean
  originalContent?: string  // Full editor content for diff preview
  selectionStart?: number   // For computing diff context
  selectionEnd?: number     // For computing diff context
  onSectionChange: (value: string) => void
  onContextChange: (value: string) => void
  onActionChange: (action: AIAction) => void
  onGenerate: () => void
  onQuickAction: (action: AIAction) => void
  onInsert: (mode: 'insert' | 'replace') => void
  onClose: () => void
}

const QUICK_ACTIONS: Array<{ action: AIAction; label: string; icon: typeof Wand2; description: string }> = [
  { action: 'improve', label: 'Improve', icon: Wand2, description: 'Make more professional and clear' },
  { action: 'expand', label: 'Expand', icon: Maximize2, description: 'Add details and examples' },
  { action: 'simplify', label: 'Simplify', icon: Minimize2, description: 'Make more concise' },
  { action: 'grammar', label: 'Fix Grammar', icon: CheckCircle, description: 'Correct errors' },
  { action: 'technical', label: 'Technical', icon: Lightbulb, description: 'Add technical precision' },
  { action: 'clarify', label: 'Clarify', icon: WrapText, description: 'Improve clarity' },
]

export function AIAssistantDialog({
  open,
  section,
  context,
  action,
  generating,
  result,
  error,
  hasSelection,
  originalContent = '',
  selectionStart = 0,
  selectionEnd = 0,
  onSectionChange,
  onContextChange,
  // onActionChange - not currently used, reserved for future use
  onGenerate,
  onQuickAction,
  onInsert,
  onClose,
}: AIAssistantDialogProps) {
  const [showDiff, setShowDiff] = useState(false)

  const sectionSuggestions = useMemo(
    () => DEFAULT_SECTIONS.filter(item => item.toLowerCase().includes(section.toLowerCase())).slice(0, 5),
    [section],
  )

  // Compute preview of what content will look like after AI insertion
  const previewContent = useMemo(() => {
    if (!result) return null

    const before = originalContent.slice(0, selectionStart)
    const after = originalContent.slice(selectionEnd)
    return `${before}${result}${after}`
  }, [result, originalContent, selectionStart, selectionEnd])

  const selectedText = useMemo(() => {
    return originalContent.slice(selectionStart, selectionEnd)
  }, [originalContent, selectionStart, selectionEnd])

  if (!open) {
    return null
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/80 px-4" role="dialog" aria-modal>
      <div className="w-full max-w-2xl rounded-3xl border bg-white p-6 shadow-2xl">
        <div className="flex items-center justify-between gap-3">
          <div>
            <p className="text-sm uppercase tracking-[0.3em] text-slate-500">AI assistant</p>
            <h2 className="text-2xl font-semibold text-slate-900">Generate section content</h2>
          </div>
          <Button variant="outline" onClick={onClose}>
            Close
          </Button>
        </div>

        <div className="mt-6 space-y-4 text-sm">
          <label className="flex flex-col gap-2">
            <span className="font-medium text-slate-700">Section name</span>
            <Input value={section} onChange={event => onSectionChange(event.target.value)} placeholder="e.g., Executive Summary" />
            {section && sectionSuggestions.length > 0 && (
              <div className="flex flex-wrap gap-2 text-xs text-muted-foreground">
                {sectionSuggestions.map(suggestion => (
                  <button
                    key={suggestion}
                    type="button"
                    className="rounded-full border bg-slate-50 px-3 py-1"
                    onClick={() => onSectionChange(suggestion)}
                  >
                    {suggestion}
                  </button>
                ))}
              </div>
            )}
          </label>

          <label className="flex flex-col gap-2">
            <span className="font-medium text-slate-700">Context (optional)</span>
            <Textarea
              className="min-h-[120px]"
              value={context}
              onChange={event => onContextChange(event.target.value)}
              placeholder="Paste notes, requirements, or the existing section to guide the model"
            />
          </label>

          {hasSelection && context.trim() && (
            <div className="space-y-2">
              <span className="text-sm font-medium text-slate-700">Quick Actions</span>
              <div className="grid grid-cols-2 gap-2">
                {QUICK_ACTIONS.map(({ action: quickAction, label, icon: Icon, description }) => (
                  <Button
                    key={quickAction}
                    variant={action === quickAction ? 'default' : 'outline'}
                    size="sm"
                    onClick={() => onQuickAction(quickAction)}
                    disabled={generating}
                    className="flex items-center gap-2 justify-start"
                    title={description}
                  >
                    <Icon size={14} />
                    <span className="flex-1 text-left">{label}</span>
                  </Button>
                ))}
              </div>
              <p className="text-xs text-muted-foreground">
                Select an action to transform the selected text using AI
              </p>
            </div>
          )}

          <div className="flex items-center gap-3">
            <Button onClick={onGenerate} disabled={generating || !section} className="flex items-center gap-2">
              {generating ? <Loader2 size={16} className="animate-spin" /> : <Sparkles size={16} />} Generate with AI
            </Button>
            {error && <span className="text-sm text-amber-700">{error}</span>}
          </div>
        </div>

        {result && (
          <div className="mt-6 space-y-3">
            <div className="flex items-center justify-between">
              <p className="text-sm font-medium text-slate-700">Generated suggestion</p>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setShowDiff(!showDiff)}
                className="flex items-center gap-2"
              >
                {showDiff ? <EyeOff size={14} /> : <Eye size={14} />}
                {showDiff ? 'Hide' : 'Show'} Diff Preview
              </Button>
            </div>

            {showDiff && previewContent ? (
              <div className="rounded-2xl border bg-white overflow-hidden">
                <DiffViewer
                  original={originalContent}
                  modified={previewContent}
                  title="AI Content Preview"
                />
                <div className="border-t bg-amber-50 p-3">
                  <p className="text-xs text-amber-800 flex items-center gap-2">
                    <Sparkles size={12} />
                    Preview shows how your draft will look after {selectedText ? 'replacing the selection' : 'inserting'} with AI-generated content
                  </p>
                </div>
              </div>
            ) : (
              <pre className="max-h-64 overflow-auto rounded-2xl border bg-slate-50 p-4 text-sm text-slate-800">
                {result}
              </pre>
            )}

            <div className="flex flex-wrap gap-3">
              <Button variant="secondary" onClick={() => onInsert('insert')}>
                Insert at cursor
              </Button>
              <Button variant="outline" onClick={() => onInsert('replace')}>
                Replace selection
              </Button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
