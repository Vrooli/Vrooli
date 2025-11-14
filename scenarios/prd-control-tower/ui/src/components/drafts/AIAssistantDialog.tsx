import { useMemo } from 'react'
import { Loader2, Sparkles } from 'lucide-react'
import { Button } from '../ui/button'
import { Textarea } from '../ui/textarea'
import { Input } from '../ui/input'

const DEFAULT_SECTIONS = [
  'Executive Summary',
  'Functional Requirements',
  'Performance Criteria',
  'Quality Gates',
  'Technical Architecture',
  'Resource Dependencies',
  'CLI Interface Contract',
]

interface AIAssistantDialogProps {
  open: boolean
  section: string
  context: string
  generating: boolean
  result: string | null
  error: string | null
  onSectionChange: (value: string) => void
  onContextChange: (value: string) => void
  onGenerate: () => void
  onInsert: (mode: 'insert' | 'replace') => void
  onClose: () => void
}

export function AIAssistantDialog({
  open,
  section,
  context,
  generating,
  result,
  error,
  onSectionChange,
  onContextChange,
  onGenerate,
  onInsert,
  onClose,
}: AIAssistantDialogProps) {
  if (!open) {
    return null
  }

  const sectionSuggestions = useMemo(() => DEFAULT_SECTIONS.filter(item => item.toLowerCase().includes(section.toLowerCase())).slice(0, 5), [section])

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

          <div className="flex items-center gap-3">
            <Button onClick={onGenerate} disabled={generating || !section} className="flex items-center gap-2">
              {generating ? <Loader2 size={16} className="animate-spin" /> : <Sparkles size={16} />} Generate with AI
            </Button>
            {error && <span className="text-sm text-amber-700">{error}</span>}
          </div>
        </div>

        {result && (
          <div className="mt-6 space-y-3">
            <p className="text-sm font-medium text-slate-700">Generated suggestion</p>
            <pre className="max-h-64 overflow-auto rounded-2xl border bg-slate-50 p-4 text-sm text-slate-800">
              {result}
            </pre>
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
