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
  { value: 'custom', label: '✏️ Custom Section' },
]

const MODEL_OPTIONS = [
  { value: 'default', label: 'Default (scenario setting)' },
  { value: 'openrouter/x-ai/grok-code-fast-1', label: 'x-ai/grok-code-fast-1' },
  { value: 'openrouter/google/gemini-2.5-flash', label: 'google/gemini-2.5-flash' },
  { value: 'openrouter/openai/gpt-5', label: 'openai/gpt-5' },
  { value: 'openrouter/x-ai/grok-4-fast:free', label: 'x-ai/grok-4-fast:free' },
]

interface TemplateDefinition {
  id: string
  label: string
  description: string
  section: string
  contextText: string
}

// Section-specific templates
const TEMPLATES_BY_SECTION: Record<string, TemplateDefinition[]> = {
  '🎯 Overview': [
    {
      id: 'overview-problem-statement',
      label: 'Problem Statement',
      description: 'Articulate the core problem this scenario solves',
      section: '🎯 Overview',
      contextText: 'Define the problem statement for this scenario, focusing on what user pain points or business needs it addresses.',
    },
    {
      id: 'overview-success-criteria',
      label: 'Success Criteria',
      description: 'Define measurable success metrics',
      section: '🎯 Overview',
      contextText: 'Define 3-5 key success criteria that will indicate this scenario has achieved its goals.',
    },
  ],
  '🎯 Operational Targets': [
    {
      id: 'targets-p0',
      label: 'P0 Targets',
      description: 'Generate critical P0 targets',
      section: '🎯 Operational Targets',
      contextText: 'List the absolutely critical (P0) features that must be delivered for this scenario to be viable.',
    },
    {
      id: 'targets-p1',
      label: 'P1 Targets',
      description: 'Generate important P1 targets',
      section: '🎯 Operational Targets',
      contextText: 'List the high-priority (P1) features that should be delivered soon after initial launch.',
    },
    {
      id: 'targets-p2',
      label: 'P2 Targets',
      description: 'Generate nice-to-have P2 targets',
      section: '🎯 Operational Targets',
      contextText: 'List the nice-to-have (P2) features that can be deferred but add significant value.',
    },
  ],
  '🧱 Tech Direction Snapshot': [
    {
      id: 'tech-stack',
      label: 'Tech Stack',
      description: 'Suggest appropriate technologies',
      section: '🧱 Tech Direction Snapshot',
      contextText: 'Recommend an appropriate tech stack (frontend, backend, database, etc.) for this scenario.',
    },
    {
      id: 'tech-architecture',
      label: 'Architecture',
      description: 'Design system architecture',
      section: '🧱 Tech Direction Snapshot',
      contextText: 'Describe the high-level architecture pattern (monolith, microservices, etc.) and key components.',
    },
    {
      id: 'tech-integrations',
      label: 'Integrations',
      description: 'List required integrations',
      section: '🧱 Tech Direction Snapshot',
      contextText: 'List external services, APIs, or scenarios this scenario needs to integrate with.',
    },
  ],
  '🤝 Dependencies & Launch Plan': [
    {
      id: 'dependencies-list',
      label: 'Dependencies',
      description: 'Identify scenario dependencies',
      section: '🤝 Dependencies & Launch Plan',
      contextText: 'List all scenarios, resources, or external dependencies required for this scenario to function.',
    },
    {
      id: 'launch-phases',
      label: 'Launch Phases',
      description: 'Create phased rollout plan',
      section: '🤝 Dependencies & Launch Plan',
      contextText: 'Create a 3-4 phase launch plan with milestones and deliverables for each phase.',
    },
  ],
  '🎨 UX & Branding': [
    {
      id: 'ux-user-flows',
      label: 'User Flows',
      description: 'Design core user flows',
      section: '🎨 UX & Branding',
      contextText: 'Describe the primary user flows and interactions for this scenario.',
    },
    {
      id: 'ux-accessibility',
      label: 'Accessibility',
      description: 'Define accessibility requirements',
      section: '🎨 UX & Branding',
      contextText: 'Outline accessibility standards and requirements (WCAG compliance, keyboard navigation, etc.).',
    },
    {
      id: 'ux-design-system',
      label: 'Design System',
      description: 'Define design patterns and components',
      section: '🎨 UX & Branding',
      contextText: 'Describe the design system, UI components, and styling approach for this scenario.',
    },
  ],
  '📎 Appendix': [
    {
      id: 'appendix-glossary',
      label: 'Glossary',
      description: 'Create terms glossary',
      section: '📎 Appendix',
      contextText: 'Generate a glossary of key terms and concepts used in this PRD.',
    },
    {
      id: 'appendix-references',
      label: 'References',
      description: 'Add reference links',
      section: '📎 Appendix',
      contextText: 'List relevant documentation, research, or reference materials for this scenario.',
    },
  ],
}

interface AIAssistantDialogProps {
  open: boolean
  section: string
  context: string
  action: AIAction
  model: string
  generating: boolean
  result: string | null
  error: string | null
  hasSelection: boolean
  originalContent?: string
  selectionStart?: number
  selectionEnd?: number
  onSectionChange: (value: string) => void
  onContextChange: (value: string) => void
  onActionChange: (action: AIAction) => void
  onModelChange: (model: string) => void
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
  model,
  generating,
  result,
  error,
  hasSelection,
  originalContent = '',
  selectionStart = 0,
  selectionEnd = 0,
  onSectionChange,
  onContextChange,
  onModelChange,
  onGenerate,
  onQuickAction,
  onInsert,
  onClose,
}: AIAssistantDialogProps) {
  const [showDiff, setShowDiff] = useState(false)
  const [customSection, setCustomSection] = useState('')

  // Determine if current section is "custom"
  const isCustomSection = section === 'custom' || !DEFAULT_SECTIONS.find(s => s.value === section)
  const effectiveSection = isCustomSection ? customSection : section

  // Get templates for current section
  const availableTemplates = useMemo(() => {
    if (isCustomSection || !section) {
      return []
    }
    return TEMPLATES_BY_SECTION[section] || []
  }, [section, isCustomSection])

  const handleTemplateSelect = (template: TemplateDefinition) => {
    onSectionChange(template.section)
    onContextChange(template.contextText)
  }

  const handleSectionDropdownChange = (value: string) => {
    if (value === 'custom') {
      onSectionChange(value)
      setCustomSection('')
    } else {
      onSectionChange(value)
      setCustomSection('')
    }
  }

  const handleCustomSectionChange = (value: string) => {
    setCustomSection(value)
    onSectionChange(value)
  }

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
      <div className="w-full max-w-5xl rounded-3xl border bg-white p-6 shadow-2xl max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between gap-3 mb-6">
          <div>
            <p className="text-sm uppercase tracking-[0.3em] text-slate-500">AI assistant</p>
            <h2 className="text-2xl font-semibold text-slate-900">Generate section content</h2>
          </div>
          <Button variant="outline" onClick={onClose}>
            Close
          </Button>
        </div>

        {/* Two-column layout on desktop */}
        <div className="grid gap-6 lg:grid-cols-2">
          {/* Left Column: Configuration */}
          <div className="space-y-4">
            {/* Section Selection */}
            <div>
              <label className="flex flex-col gap-2">
                <span className="font-medium text-slate-700">Section name</span>
                <select
                  value={isCustomSection ? 'custom' : section}
                  onChange={event => handleSectionDropdownChange(event.target.value)}
                  className="rounded-md border border-slate-300 bg-white px-3 py-2 text-sm"
                >
                  <option value="">Select a section...</option>
                  {DEFAULT_SECTIONS.map(sec => (
                    <option key={sec.value} value={sec.value}>{sec.label}</option>
                  ))}
                </select>
              </label>

              {/* Custom section input - shown when "Custom" is selected */}
              {isCustomSection && (
                <div className="mt-2">
                  <Input
                    value={customSection}
                    onChange={event => handleCustomSectionChange(event.target.value)}
                    placeholder="Enter custom section name (e.g., Security Considerations)"
                    className="text-sm"
                  />
                </div>
              )}
            </div>

            {/* Quick Templates - shown below section selection */}
            {availableTemplates.length > 0 && (
              <div className="rounded-lg border border-blue-200 bg-blue-50 p-3">
                <span className="font-medium text-blue-900 text-sm">Quick Templates</span>
                <p className="text-xs text-blue-700 mt-1">Click a template to auto-fill context</p>
                <div className="grid grid-cols-1 gap-2 mt-2">
                  {availableTemplates.map(template => (
                    <button
                      key={template.id}
                      type="button"
                      className="rounded-lg border border-slate-200 bg-white p-2 text-left transition hover:border-blue-300 hover:bg-blue-50"
                      onClick={() => handleTemplateSelect(template)}
                    >
                      <div className="font-semibold text-sm">{template.label}</div>
                      <div className="text-xs text-slate-600 mt-0.5">{template.description}</div>
                    </button>
                  ))}
                </div>
              </div>
            )}

            {/* AI Model Selector */}
            <div>
              <label className="flex flex-col gap-2">
                <span className="font-medium text-slate-700">AI Model</span>
                <select
                  value={model}
                  onChange={event => onModelChange(event.target.value)}
                  className="rounded-md border border-slate-300 bg-white px-3 py-2 text-sm"
                >
                  {MODEL_OPTIONS.map(option => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
                <p className="text-xs text-muted-foreground">Select which AI model to use for generation</p>
              </label>
            </div>

            {/* Quick Actions */}
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

            {/* Generate Button */}
            <div className="flex items-center gap-3">
              <Button onClick={onGenerate} disabled={generating || !effectiveSection} className="flex items-center gap-2">
                {generating ? <Loader2 size={16} className="animate-spin" /> : <Sparkles size={16} />} Generate with AI
              </Button>
              {error && <span className="text-sm text-amber-700">{error}</span>}
            </div>
          </div>

          {/* Right Column: Context Input (larger) */}
          <div className="space-y-4">
            <label className="flex flex-col gap-2">
              <span className="font-medium text-slate-700">Context (optional)</span>
              <Textarea
                className="min-h-[320px]"
                value={context}
                onChange={event => onContextChange(event.target.value)}
                placeholder="Paste notes, requirements, or the existing section to guide the model"
              />
              <p className="text-xs text-muted-foreground">
                Provide context, requirements, or existing content to guide the AI generation
              </p>
            </label>
          </div>
        </div>

        {/* Result Preview */}
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
              <pre className="max-h-64 overflow-auto rounded-2xl border bg-slate-50 p-4 text-sm text-slate-800 whitespace-pre-wrap">
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
