import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Sparkles, Lightbulb, ArrowRight, X } from 'lucide-react'
import toast from 'react-hot-toast'
import { Button } from '../ui/button'
import { Textarea } from '../ui/textarea'
import { parseBacklogInput, createBacklogEntriesRequest } from '../../utils/backlog'

interface QuickAddIdeaDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

/**
 * QuickAddIdeaDialog
 *
 * Quick capture modal for adding ideas to backlog from anywhere in the app.
 * Allows users to paste freeform text and immediately save to backlog.
 *
 * Features:
 * - Textarea for pasting notes
 * - Auto-parse bullet lists
 * - Shows preview of detected ideas
 * - Quick save to backlog
 * - Option to navigate to backlog page for batch operations
 */
export function QuickAddIdeaDialog({ open, onOpenChange }: QuickAddIdeaDialogProps) {
  const navigate = useNavigate()
  const [rawInput, setRawInput] = useState('')
  const [saving, setSaving] = useState(false)

  const parsedIdeas = parseBacklogInput(rawInput, 'scenario')

  const handleQuickSave = async () => {
    if (parsedIdeas.length === 0) {
      toast.error('Please enter at least one idea')
      return
    }

    setSaving(true)
    try {
      await createBacklogEntriesRequest(parsedIdeas)
      toast.success(`Added ${parsedIdeas.length} idea${parsedIdeas.length === 1 ? '' : 's'} to backlog`)
      setRawInput('')
      onOpenChange(false)
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to save ideas'
      toast.error(message)
    } finally {
      setSaving(false)
    }
  }

  const handleOpenBacklog = () => {
    onOpenChange(false)
    navigate('/backlog')
  }

  const handleClear = () => {
    setRawInput('')
  }

  if (!open) {
    return null
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/70 px-4"
      onClick={() => onOpenChange(false)}
      role="dialog"
      aria-modal="true"
      aria-labelledby="quick-add-dialog-title"
    >
      <div
        className="w-full max-w-2xl rounded-2xl border bg-white shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-start justify-between gap-4 border-b p-6">
          <div className="flex items-center gap-3 flex-1">
            <span className="rounded-xl bg-amber-100 p-2 text-amber-600">
              <Lightbulb size={24} />
            </span>
            <div>
              <h2 id="quick-add-dialog-title" className="text-xl font-semibold text-slate-900">
                Quick Add Ideas
              </h2>
              <p className="mt-1 text-sm text-muted-foreground">
                Paste your notes below. We'll auto-detect ideas from bullet lists or line breaks.
              </p>
            </div>
          </div>
          <button
            onClick={() => onOpenChange(false)}
            className="rounded-lg p-2 text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600"
            aria-label="Close dialog"
          >
            <X size={20} />
          </button>
        </div>

        {/* Content */}
        <div className="space-y-4 p-6">
          <Textarea
            value={rawInput}
            onChange={(e) => setRawInput(e.target.value)}
            placeholder={`Paste your ideas here...\n\nExamples:\n- Build a task automation tool\n- Create API wrapper for OpenAI\n- Design referral program generator`}
            rows={8}
            className="resize-none font-mono text-sm"
          />

          {parsedIdeas.length > 0 && (
            <div className="rounded-lg border bg-slate-50 p-3">
              <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-600">
                Detected {parsedIdeas.length} idea{parsedIdeas.length === 1 ? '' : 's'}
              </p>
              <ul className="space-y-1 text-sm">
                {parsedIdeas.slice(0, 5).map((idea) => (
                  <li key={idea.id} className="flex items-start gap-2">
                    <span className="mt-1 text-amber-600">•</span>
                    <span className="flex-1 text-slate-700">
                      {idea.ideaText}
                      <span className="ml-2 text-xs text-muted-foreground">({idea.entityType})</span>
                    </span>
                  </li>
                ))}
                {parsedIdeas.length > 5 && (
                  <li className="text-xs text-muted-foreground">
                    ... and {parsedIdeas.length - 5} more
                  </li>
                )}
              </ul>
            </div>
          )}

          {rawInput && parsedIdeas.length === 0 && (
            <div className="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900">
              <p className="font-medium">No ideas detected</p>
              <p className="mt-1 text-xs text-amber-700">
                Try separating ideas with line breaks or bullet points (-, *, •)
              </p>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-2 border-t p-6">
          <Button variant="ghost" onClick={handleClear} disabled={!rawInput || saving}>
            Clear
          </Button>
          <Button variant="outline" onClick={handleOpenBacklog} disabled={saving}>
            <Sparkles size={16} className="mr-2" />
            Open Full Backlog
          </Button>
          <Button onClick={handleQuickSave} disabled={parsedIdeas.length === 0 || saving}>
            {saving ? (
              'Saving...'
            ) : (
              <>
                Save to Backlog
                <ArrowRight size={16} className="ml-2" />
              </>
            )}
          </Button>
        </div>
      </div>
    </div>
  )
}
