import { useMemo, useState } from 'react'
import { Loader2, Sparkles, Eye, EyeOff, Wand2, Maximize2, Minimize2, CheckCircle, Lightbulb, WrapText } from 'lucide-react'
import { Button } from '../ui/button'
import { Textarea } from '../ui/textarea'
import { Input } from '../ui/input'
import { DiffViewer } from './DiffViewer'
import type { AIAction } from '../../hooks/useAIAssistant'

const DEFAULT_SECTIONS = [
  { value: '🎯 Overview', label: '🎯 Overview' },
  { value: '🎯 Operational Targets', label: '🎯 Operational Targets' },
  { value: '🧱 Tech Direction Snapshot', label: '🧱 Tech Direction Snapshot' },
  { value: '🤝 Dependencies & Launch Plan', label: '🤝 Dependencies & Launch Plan' },
  { value: '🎨 UX & Branding', label: '🎨 UX & Branding' },
  { value: '📎 Appendix', label: '📎 Appendix' },
]

const PROMPT_TEMPLATES = [
  {
    id: 'fill-operational-targets',
    label: 'Fill Operational Targets',
    description: 'Generate P0/P1/P2 targets based on scenario requirements',
    section: '🎯 Operational Targets',
    contextPlaceholder: 'Describe what this scenario does and what the key requirements are...',
  },
  {
    id: 'expand-overview',
    label: 'Expand Overview',
    description: 'Generate a detailed overview with problem statement and success criteria',
    section: '🎯 Overview',
    contextPlaceholder: 'Provide a brief description of the scenario...',
  },
  {
    id: 'tech-stack',
    label: 'Tech Stack Details',
    description: 'Suggest appropriate tech stack and architecture',
    section: '🧱 Tech Direction Snapshot',
    contextPlaceholder: 'Describe the type of application and any requirements or constraints...',
  },
  {
    id: 'launch-plan',
    label: 'Launch Plan',
    description: 'Create a phased launch plan with milestones',
    section: '🤝 Dependencies & Launch Plan',
    contextPlaceholder: 'Describe the scope and any known dependencies...',
  },
  {
    id: 'ux-flows',
    label: 'UX Flows',
    description: 'Design user experience and interaction flows',
    section: '🎨 UX & Branding',
    contextPlaceholder: 'Describe the primary users and what they need to accomplish...',
  },
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
  const [selectedTemplate, setSelectedTemplate] = useState<string | null>(null)

  const handleTemplateSelect = (templateId: string) => {
    const template = PROMPT_TEMPLATES.find(t => t.id === templateId)
    if (template) {
      setSelectedTemplate(templateId)
      onSectionChange(template.section)
      // Clear context to show the placeholder
      if (!context.trim()) {
        // Only set if context is empty
      }
    }
  }

  const currentContextPlaceholder = useMemo(() => {
    if (selectedTemplate) {
      const template = PROMPT_TEMPLATES.find(t => t.id === selectedTemplate)
      return template?.contextPlaceholder || 'Paste notes, requirements, or the existing section to guide the model'
    }
    return 'Paste notes, requirements, or the existing section to guide the model'
  }, [selectedTemplate])

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
          {/* Template Selection */}
          <div className="rounded-lg border border-blue-200 bg-blue-50 p-3">
            <label className="flex flex-col gap-2">
              <span className="font-medium text-blue-900">Quick Templates</span>
              <p className="text-xs text-blue-700">Select a template to auto-fill section and get context guidance</p>
              <div className="grid grid-cols-2 gap-2 mt-2">
                {PROMPT_TEMPLATES.map(template => (
                  <button
                    key={template.id}
                    type="button"
                    className={`rounded-lg border p-2 text-left transition ${
                      selectedTemplate === template.id
                        ? 'border-blue-500 bg-blue-100 text-blue-900'
                        : 'border-slate-200 bg-white text-slate-700 hover:border-blue-300'
                    }`}
                    onClick={() => handleTemplateSelect(template.id)}
                  >
                    <div className="font-semibold text-sm">{template.label}</div>
                    <div className="text-xs text-slate-600 mt-0.5">{template.description}</div>
                  </button>
                ))}
              </div>
            </label>
          </div>

          <label className="flex flex-col gap-2">
            <span className="font-medium text-slate-700">Section name</span>
            <select
              value={section}
              onChange={event => onSectionChange(event.target.value)}
              className="rounded-md border border-slate-300 bg-white px-3 py-2 text-sm"
            >
              <option value="">Select a section...</option>
              {DEFAULT_SECTIONS.map(sec => (
                <option key={sec.value} value={sec.value}>{sec.label}</option>
              ))}
            </select>
            <p className="text-xs text-muted-foreground">Or type a custom section name</p>
            <Input value={section} onChange={event => onSectionChange(event.target.value)} placeholder="e.g., Custom Section" />
          </label>

          <label className="flex flex-col gap-2">
            <span className="font-medium text-slate-700">Context (optional)</span>
            <Textarea
              className="min-h-[120px]"
              value={context}
              onChange={event => onContextChange(event.target.value)}
              placeholder={currentContextPlaceholder}
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
