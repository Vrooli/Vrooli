import { useMemo, useState } from 'react'
import { Loader2, Sparkles, Eye, EyeOff, Wand2, Maximize2, Minimize2, CheckCircle, Lightbulb, WrapText, FileCode, AlertTriangle, RefreshCw, Check, X } from 'lucide-react'
import { Button } from '../ui/button'
import { Textarea } from '../ui/textarea'
import { Input } from '../ui/input'
import { DiffViewer } from './DiffViewer'
import { ReferencePRDSelector } from './ReferencePRDSelector'
import type { AIAction, AIInsertMode } from '../../hooks/useAIAssistant'
import { buildPrompt } from '../../hooks/useAIAssistant'
import { extractSectionContent } from '../../utils/prdStructure'
import type { Draft } from '../../types'

const DEFAULT_SECTIONS = [
  { value: '🎯 Full PRD', label: '🎯 Full PRD' },
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
  template: string
}

// Section-specific templates with placeholders
const TEMPLATES_BY_SECTION: Record<string, TemplateDefinition[]> = {
  '🎯 Full PRD': [
    {
      id: 'full-prd-comprehensive',
      label: 'Comprehensive PRD',
      description: 'Generate a complete PRD with all sections',
      section: '🎯 Full PRD',
      template: 'Generate a comprehensive PRD for this scenario, including all standard sections (Overview, Operational Targets, Tech Direction, Dependencies, UX & Branding, etc.).\n\n[INSERT HIGH-LEVEL REQUIREMENTS AND GOALS HERE]',
    },
    {
      id: 'full-prd-from-idea',
      label: 'From Idea',
      description: 'Transform a rough idea into a full PRD',
      section: '🎯 Full PRD',
      template: 'Transform this rough idea into a comprehensive PRD:\n\n[INSERT YOUR IDEA OR CONCEPT HERE]',
    },
  ],
  '🎯 Overview': [
    {
      id: 'overview-problem-statement',
      label: 'Problem Statement',
      description: 'Articulate the core problem this scenario solves',
      section: '🎯 Overview',
      template: 'Define the problem statement for this scenario, focusing on what user pain points or business needs it addresses.\n\n[INSERT PROBLEM DETAILS HERE]',
    },
    {
      id: 'overview-success-criteria',
      label: 'Success Criteria',
      description: 'Define measurable success metrics',
      section: '🎯 Overview',
      template: 'Define 3-5 key success criteria that will indicate this scenario has achieved its goals.\n\n[INSERT SUCCESS METRICS HERE]',
    },
  ],
  '🎯 Operational Targets': [
    {
      id: 'targets-p0',
      label: 'P0 Targets',
      description: 'Generate critical P0 targets',
      section: '🎯 Operational Targets',
      template: 'List the absolutely critical (P0) features that must be delivered for this scenario to be viable.\n\n[INSERT CRITICAL FEATURES HERE]',
    },
    {
      id: 'targets-p1',
      label: 'P1 Targets',
      description: 'Generate important P1 targets',
      section: '🎯 Operational Targets',
      template: 'List the high-priority (P1) features that should be delivered soon after initial launch.\n\n[INSERT HIGH-PRIORITY FEATURES HERE]',
    },
    {
      id: 'targets-p2',
      label: 'P2 Targets',
      description: 'Generate nice-to-have P2 targets',
      section: '🎯 Operational Targets',
      template: 'List the nice-to-have (P2) features that can be deferred but add significant value.\n\n[INSERT NICE-TO-HAVE FEATURES HERE]',
    },
  ],
  '🧱 Tech Direction Snapshot': [
    {
      id: 'tech-stack',
      label: 'Tech Stack',
      description: 'Suggest appropriate technologies',
      section: '🧱 Tech Direction Snapshot',
      template: 'Recommend an appropriate tech stack (frontend, backend, database, etc.) for this scenario.\n\n[INSERT TECH REQUIREMENTS OR PREFERENCES HERE]',
    },
    {
      id: 'tech-architecture',
      label: 'Architecture',
      description: 'Design system architecture',
      section: '🧱 Tech Direction Snapshot',
      template: 'Describe the high-level architecture pattern (monolith, microservices, etc.) and key components.\n\n[INSERT ARCHITECTURE REQUIREMENTS HERE]',
    },
    {
      id: 'tech-integrations',
      label: 'Integrations',
      description: 'List required integrations',
      section: '🧱 Tech Direction Snapshot',
      template: 'List external services, APIs, or scenarios this scenario needs to integrate with.\n\n[INSERT INTEGRATION REQUIREMENTS HERE]',
    },
  ],
  '🤝 Dependencies & Launch Plan': [
    {
      id: 'dependencies-list',
      label: 'Dependencies',
      description: 'Identify scenario dependencies',
      section: '🤝 Dependencies & Launch Plan',
      template: 'List all scenarios, resources, or external dependencies required for this scenario to function.\n\n[INSERT DEPENDENCY DETAILS HERE]',
    },
    {
      id: 'launch-phases',
      label: 'Launch Phases',
      description: 'Create phased rollout plan',
      section: '🤝 Dependencies & Launch Plan',
      template: 'Create a 3-4 phase launch plan with milestones and deliverables for each phase.\n\n[INSERT TIMELINE OR PHASE DETAILS HERE]',
    },
  ],
  '🎨 UX & Branding': [
    {
      id: 'ux-user-flows',
      label: 'User Flows',
      description: 'Design core user flows',
      section: '🎨 UX & Branding',
      template: 'Describe the primary user flows and interactions for this scenario.\n\n[INSERT USER FLOW DETAILS HERE]',
    },
    {
      id: 'ux-accessibility',
      label: 'Accessibility',
      description: 'Define accessibility requirements',
      section: '🎨 UX & Branding',
      template: 'Outline accessibility standards and requirements (WCAG compliance, keyboard navigation, etc.).\n\n[INSERT ACCESSIBILITY REQUIREMENTS HERE]',
    },
    {
      id: 'ux-design-system',
      label: 'Design System',
      description: 'Define design patterns and components',
      section: '🎨 UX & Branding',
      template: 'Describe the design system, UI components, and styling approach for this scenario.\n\n[INSERT DESIGN PREFERENCES HERE]',
    },
  ],
  '📎 Appendix': [
    {
      id: 'appendix-glossary',
      label: 'Glossary',
      description: 'Create terms glossary',
      section: '📎 Appendix',
      template: 'Generate a glossary of key terms and concepts used in this PRD.\n\n[INSERT TERMS TO INCLUDE HERE]',
    },
    {
      id: 'appendix-references',
      label: 'References',
      description: 'Add reference links',
      section: '📎 Appendix',
      template: 'List relevant documentation, research, or reference materials for this scenario.\n\n[INSERT REFERENCE MATERIALS HERE]',
    },
  ],
}

interface ReferencePRDData {
  name: string
  content: string
}

interface AIAssistantDialogProps {
  open: boolean
  draft: Draft
  section: string
  context: string
  action: AIAction
  model: string
  generating: boolean
  result: string | null
  error: string | null
  insertMode: AIInsertMode
  hasSelection: boolean
  includeExistingContent: boolean
  referencePRDs: ReferencePRDData[]
  originalContent?: string
  selectionStart?: number
  selectionEnd?: number
  onSectionChange: (value: string) => void
  onContextChange: (value: string) => void
  onActionChange: (action: AIAction) => void
  onModelChange: (model: string) => void
  onInsertModeChange: (mode: AIInsertMode) => void
  onIncludeExistingContentChange: (value: boolean) => void
  onReferencePRDsChange: (prds: ReferencePRDData[]) => void
  onGenerate: () => void
  onQuickAction: (action: AIAction) => void
  onConfirm: () => void
  onRegenerate: () => void
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
  draft,
  section,
  context,
  action,
  model,
  generating,
  result,
  error,
  insertMode,
  hasSelection,
  includeExistingContent,
  referencePRDs,
  originalContent = '',
  selectionStart = 0,
  selectionEnd = 0,
  onSectionChange,
  onContextChange,
  onModelChange,
  onInsertModeChange,
  onIncludeExistingContentChange,
  onReferencePRDsChange,
  onGenerate,
  onQuickAction,
  onConfirm,
  onRegenerate,
  onClose,
}: AIAssistantDialogProps) {
  const [showDiff, setShowDiff] = useState(true) // Default to showing diff
  const [showPromptPreview, setShowPromptPreview] = useState(false)
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
    onContextChange(template.template)
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

    if (insertMode === 'cursor') {
      // Cursor mode: insert at current cursor position (or replace selection)
      const before = originalContent.slice(0, selectionStart)
      const after = originalContent.slice(selectionEnd)
      return `${before}${result}${after}`
    } else {
      // Section mode: replace the entire selected section (or full PRD)
      const isFullPRD = effectiveSection === '🎯 Full PRD' || effectiveSection === 'Full PRD'

      if (isFullPRD) {
        // Replace entire content
        return result
      } else {
        // Replace the specific section
        const sectionContent = extractSectionContent(originalContent, effectiveSection)

        if (sectionContent) {
          // Replace the section content (preserving the section header)
          const lines = originalContent.split('\n')
          const before = lines.slice(0, sectionContent.startLine + 1).join('\n') + '\n'
          const after = lines.slice(sectionContent.endLine).join('\n')
          return `${before}${result}\n${after}`
        } else {
          // Section doesn't exist - just insert at cursor
          const before = originalContent.slice(0, selectionStart)
          const after = originalContent.slice(selectionEnd)
          return `${before}${result}${after}`
        }
      }
    }
  }, [result, insertMode, effectiveSection, originalContent, selectionStart, selectionEnd])

  // Generate descriptive labels for the insertion modes
  const insertModeLabels = useMemo(() => {
    const isFullPRD = effectiveSection === '🎯 Full PRD' || effectiveSection === 'Full PRD'
    const cursorLabel = hasSelection ? 'Replace Selection' : 'Insert at Cursor'
    const sectionLabel = isFullPRD ? 'Replace Full PRD' : `Replace "${effectiveSection}" Section`

    return {
      cursor: cursorLabel,
      section: sectionLabel,
    }
  }, [effectiveSection, hasSelection])

  // Extract current section content for preview
  const currentSectionContent = useMemo(() => {
    if (!effectiveSection) {
      return null
    }

    // For Full PRD, show the entire content
    if (effectiveSection === '🎯 Full PRD' || effectiveSection === 'Full PRD') {
      return {
        content: originalContent,
        startLine: 0,
        endLine: originalContent.split('\n').length,
        isFullDocument: true
      }
    }

    const extracted = extractSectionContent(originalContent, effectiveSection)
    return extracted ? { ...extracted, isFullDocument: false } : null
  }, [originalContent, effectiveSection])

  // Generate full prompt preview
  const promptPreview = useMemo(() => {
    if (!effectiveSection) return ''
    return buildPrompt(draft, effectiveSection, context, action, includeExistingContent, referencePRDs)
  }, [draft, effectiveSection, context, action, includeExistingContent, referencePRDs])

  if (!open) {
    return null
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/80 px-4 backdrop-blur-sm" role="dialog" aria-modal>
      <div className="w-full max-w-5xl rounded-3xl border-2 bg-white shadow-2xl max-h-[90vh] overflow-y-auto">
        <div className="sticky top-0 z-10 flex items-center justify-between gap-3 border-b bg-gradient-to-r from-violet-50 to-white p-6">
          <div className="flex items-center gap-3">
            <div className="rounded-xl bg-gradient-to-br from-violet-500 to-violet-600 p-2.5 text-white shadow-md">
              <Sparkles size={20} strokeWidth={2.5} />
            </div>
            <div>
              <p className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">AI Assistant</p>
              <h2 className="text-xl font-bold text-slate-900">Generate PRD Content</h2>
            </div>
          </div>
          <Button variant="outline" onClick={onClose} size="sm">
            Close
          </Button>
        </div>
        <div className="p-6">

        {/* Two-column layout on desktop */}
        <div className="grid gap-6 lg:grid-cols-2">
          {/* Left Column: Configuration */}
          <div className="space-y-5">
            {/* Step 1: Section Selection */}
            <div className="space-y-3">
              <div className="flex items-center gap-2">
                <div className="flex h-6 w-6 items-center justify-center rounded-full bg-violet-500 text-xs font-bold text-white">1</div>
                <label className="font-semibold text-slate-900">Choose Section</label>
              </div>
              <select
                value={isCustomSection ? 'custom' : section}
                onChange={event => handleSectionDropdownChange(event.target.value)}
                className="w-full rounded-xl border-2 border-slate-200 bg-white px-4 py-3 text-sm font-medium shadow-sm transition focus:border-violet-400 focus:ring-2 focus:ring-violet-100"
              >
                <option value="">Select section to generate...</option>
                {DEFAULT_SECTIONS.map(sec => (
                  <option key={sec.value} value={sec.value}>{sec.label}</option>
                ))}
              </select>

              {/* Custom section input - shown when "Custom" is selected */}
              {isCustomSection && (
                <div className="animate-in fade-in slide-in-from-top-2 duration-200">
                  <Input
                    value={customSection}
                    onChange={event => handleCustomSectionChange(event.target.value)}
                    placeholder="Enter custom section name (e.g., Security Considerations)"
                    className="border-2 py-3 text-sm"
                  />
                </div>
              )}
            </div>

            {/* Current Section Preview - What Will Be Replaced */}
            {currentSectionContent && (
              <div className="rounded-lg border-2 border-amber-300 bg-amber-50 p-4">
                <div className="flex items-center justify-between gap-2 mb-2">
                  <div className="flex items-center gap-2">
                    <AlertTriangle size={16} className="text-amber-700" />
                    <span className="text-sm font-semibold text-amber-900">
                      {currentSectionContent.isFullDocument ? 'Entire Document Will Be Replaced' : 'Section To Be Replaced'}
                    </span>
                  </div>
                  <span className="text-xs text-amber-700 font-mono">
                    Lines {currentSectionContent.startLine + 1}–{currentSectionContent.endLine}
                  </span>
                </div>

                <p className="text-xs text-amber-800 mb-3 font-medium">
                  {currentSectionContent.isFullDocument
                    ? '⚠️ AI will replace your ENTIRE PRD content. Make sure this is what you want!'
                    : `⚠️ AI will replace the "${effectiveSection}" section with new generated content.`
                  }
                </p>

                <div className="rounded-lg border border-amber-400 bg-white">
                  {/* Section header indicator */}
                  <div className="bg-amber-100 px-3 py-2 border-b border-amber-300">
                    <div className="flex items-center justify-between text-xs">
                      <span className="font-semibold text-amber-900">
                        {currentSectionContent.isFullDocument ? 'Current Full Document' : `Current: ${effectiveSection}`}
                      </span>
                      <span className="text-amber-700">
                        {currentSectionContent.content.length} chars
                      </span>
                    </div>
                  </div>

                  {/* Content preview */}
                  <div className="max-h-40 overflow-auto p-3">
                    <pre className="text-xs text-slate-700 whitespace-pre-wrap font-mono leading-relaxed">
                      {currentSectionContent.content.length > 500
                        ? currentSectionContent.content.substring(0, 500) + '\n\n... [truncated] ...'
                        : currentSectionContent.content || '(Empty - will be created)'}
                    </pre>
                  </div>
                </div>

                {!currentSectionContent.content && (
                  <p className="text-xs text-amber-700 mt-2 italic">
                    ℹ️ This section doesn't exist yet. AI will create it in the correct position.
                  </p>
                )}
              </div>
            )}

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

            {/* Advanced Options - Collapsible */}
            <details className="group rounded-xl border-2 border-slate-200 bg-slate-50/50">
              <summary className="cursor-pointer p-4 font-semibold text-slate-700 hover:bg-slate-100/50 transition flex items-center gap-2 select-none">
                <span className="text-sm">Advanced Options</span>
                <span className="ml-auto text-xs text-slate-500 group-open:hidden">(Click to expand)</span>
              </summary>
              <div className="space-y-4 border-t border-slate-200 bg-white p-4">
                {/* AI Model Selector */}
                <div>
                  <label className="flex flex-col gap-2">
                    <span className="text-sm font-medium text-slate-700">AI Model</span>
                    <select
                      value={model}
                      onChange={event => onModelChange(event.target.value)}
                      className="rounded-lg border-2 border-slate-200 bg-white px-3 py-2 text-sm transition focus:border-violet-400 focus:ring-2 focus:ring-violet-100"
                    >
                      {MODEL_OPTIONS.map(option => (
                        <option key={option.value} value={option.value}>
                          {option.label}
                        </option>
                      ))}
                    </select>
                  </label>
                </div>

                {/* Include Existing Content Toggle */}
                <div className="flex items-center gap-3 rounded-lg border border-slate-200 bg-slate-50 p-3">
                  <input
                    type="checkbox"
                    id="include-existing"
                    checked={includeExistingContent}
                    onChange={event => onIncludeExistingContentChange(event.target.checked)}
                    className="h-4 w-4 rounded border-slate-300"
                  />
                  <label htmlFor="include-existing" className="flex-1 cursor-pointer">
                    <div className="font-medium text-sm text-slate-700">Include existing PRD content</div>
                    <div className="text-xs text-slate-600 mt-0.5">
                      AI will see your current PRD for better context
                    </div>
                  </label>
                </div>

                {/* Reference PRDs Section */}
                <ReferencePRDSelector
                  selectedPRDs={referencePRDs}
                  onSelectionChange={onReferencePRDsChange}
                />
              </div>
            </details>

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

            {/* Generate Button with Action Summary */}
            <div className="space-y-4">
              {effectiveSection && !generating && (
                <div className="rounded-xl border-2 border-blue-200 bg-gradient-to-br from-blue-50 to-white p-4">
                  <div className="flex items-start gap-2">
                    <AlertTriangle size={16} className="mt-0.5 shrink-0 text-blue-600" />
                    <div className="space-y-1 text-sm">
                      <p className="font-medium text-blue-900">
                        {currentSectionContent?.isFullDocument
                          ? 'Will replace your ENTIRE PRD'
                          : currentSectionContent?.content
                            ? `Will replace the "${effectiveSection}" section`
                            : `Will create new "${effectiveSection}" section`
                        }
                      </p>
                      <p className="text-xs text-blue-700">
                        Model: <span className="font-medium">{model === 'default' ? 'Default (scenario setting)' : model.replace('openrouter/', '')}</span>
                      </p>
                      {context.trim() && (
                        <p className="text-xs text-blue-600 mt-1">
                          ✓ Custom context provided ({context.length} characters)
                        </p>
                      )}
                    </div>
                  </div>
                </div>
              )}

              <Button
                onClick={onGenerate}
                disabled={generating || !effectiveSection}
                size="lg"
                className="w-full h-14 text-base font-semibold shadow-lg hover:shadow-xl hover:scale-105 active:scale-100 transition-all duration-200"
              >
                {generating ? (
                  <>
                    <Loader2 size={20} className="animate-spin" />
                    Generating...
                  </>
                ) : (
                  <>
                    <Sparkles size={20} />
                    Generate with AI
                  </>
                )}
              </Button>
              {!effectiveSection && (
                <p className="text-sm text-center text-slate-500 italic">
                  Select a section above to enable generation
                </p>
              )}
              {error && (
                <div className="rounded-lg border-2 border-amber-200 bg-amber-50 p-3 text-sm text-amber-900">
                  <div className="flex items-start gap-2">
                    <AlertTriangle size={14} className="mt-0.5 shrink-0" />
                    <span>{error}</span>
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* Right Column: Context Input (larger) */}
          <div className="space-y-5">
            <div className="space-y-3">
              <div className="flex items-center gap-2">
                <div className="flex h-6 w-6 items-center justify-center rounded-full bg-violet-500 text-xs font-bold text-white">2</div>
                <label className="font-semibold text-slate-900">Provide Context <span className="text-sm font-normal text-slate-500">(optional)</span></label>
              </div>
              <Textarea
                className="min-h-[280px] resize-none border-2 p-4 text-sm leading-relaxed transition focus:border-violet-400 focus:ring-2 focus:ring-violet-100"
                value={context}
                onChange={event => onContextChange(event.target.value)}
                placeholder="Describe what you want this section to cover, paste requirements, or add any notes to guide the AI...&#10;&#10;Example:&#10;- Feature must support real-time collaboration&#10;- Target 95th percentile latency < 200ms&#10;- Integration with existing auth system required"
              />
              <p className="flex items-start gap-2 text-xs text-slate-600">
                <Lightbulb size={14} className="mt-0.5 shrink-0 text-amber-500" />
                <span>Add specific requirements, constraints, or examples to get better AI-generated content</span>
              </p>
            </div>

            {/* Prompt Preview */}
            <details className="rounded-lg border border-slate-200 bg-slate-50" open={showPromptPreview} onToggle={(e) => setShowPromptPreview((e.target as HTMLDetailsElement).open)}>
              <summary className="cursor-pointer p-3 text-sm font-medium text-slate-700 hover:bg-slate-100 flex items-center gap-2">
                <FileCode size={14} />
                Preview Full AI Prompt
                <span className="ml-auto text-xs text-slate-500">
                  {promptPreview.length.toLocaleString()} characters
                </span>
              </summary>
              <div className="p-3 pt-0">
                <p className="text-xs text-slate-600 mb-2">
                  This is the exact prompt that will be sent to the AI model:
                </p>
                <div className="max-h-80 overflow-auto rounded border border-slate-300 bg-white p-3">
                  <pre className="text-xs text-slate-800 whitespace-pre-wrap font-mono">
                    {promptPreview || '(No prompt generated yet - select a section)'}
                  </pre>
                </div>
              </div>
            </details>
          </div>
        </div>

        {/* Result Preview with Mode Selector and Action Buttons */}
        {result && (
          <div className="mt-6 space-y-4">
            {/* Header with Mode Selector */}
            <div className="flex items-center justify-between gap-4">
              <p className="text-sm font-medium text-slate-700">Preview & Apply</p>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setShowDiff(!showDiff)}
                className="flex items-center gap-2"
              >
                {showDiff ? <EyeOff size={14} /> : <Eye size={14} />}
                {showDiff ? 'Hide' : 'Show'} Diff
              </Button>
            </div>

            {/* Insertion Mode Selector */}
            <div className="flex gap-2 p-1 bg-slate-100 rounded-lg">
              <button
                type="button"
                onClick={() => onInsertModeChange('cursor')}
                className={`flex-1 px-4 py-2 text-sm font-medium rounded-md transition ${
                  insertMode === 'cursor'
                    ? 'bg-white text-slate-900 shadow-sm'
                    : 'text-slate-600 hover:text-slate-900'
                }`}
              >
                {insertModeLabels.cursor}
              </button>
              <button
                type="button"
                onClick={() => onInsertModeChange('section')}
                className={`flex-1 px-4 py-2 text-sm font-medium rounded-md transition ${
                  insertMode === 'section'
                    ? 'bg-white text-slate-900 shadow-sm'
                    : 'text-slate-600 hover:text-slate-900'
                }`}
              >
                {insertModeLabels.section}
              </button>
            </div>

            {/* Mode Description */}
            <div className="rounded-lg border border-blue-200 bg-blue-50 p-3">
              <p className="text-xs text-blue-900">
                <strong>Selected mode:</strong>{' '}
                {insertMode === 'cursor'
                  ? hasSelection
                    ? `Will replace the currently selected text (${selectionEnd - selectionStart} characters) with the AI-generated content.`
                    : `Will insert the AI-generated content at the current cursor position (character ${selectionStart}).`
                  : effectiveSection === '🎯 Full PRD' || effectiveSection === 'Full PRD'
                    ? 'Will replace the ENTIRE PRD content with the AI-generated content.'
                    : `Will replace the entire "${effectiveSection}" section with the AI-generated content.`
                }
              </p>
            </div>

            {/* Preview */}
            {showDiff && previewContent ? (
              <div className="rounded-2xl border bg-white overflow-hidden">
                <DiffViewer
                  original={originalContent}
                  modified={previewContent}
                  title={`Preview: ${insertModeLabels[insertMode]}`}
                />
                <div className="border-t bg-green-50 p-3">
                  <p className="text-xs text-green-800 flex items-center gap-2">
                    <Sparkles size={12} />
                    Green highlights show the content that will be added. Red highlights show content that will be removed.
                  </p>
                </div>
              </div>
            ) : (
              <div className="space-y-2">
                <p className="text-xs text-slate-600">AI-generated content:</p>
                <pre className="max-h-64 overflow-auto rounded-2xl border bg-slate-50 p-4 text-sm text-slate-800 whitespace-pre-wrap">
                  {result}
                </pre>
              </div>
            )}

            {/* Action Buttons */}
            <div className="flex flex-wrap gap-3 pt-2">
              <Button
                onClick={onConfirm}
                size="lg"
                className="flex items-center gap-2 shadow-md hover:shadow-lg hover:scale-105 active:scale-100 transition-all duration-200"
                variant="default"
              >
                <Check size={18} />
                Confirm & Apply
              </Button>
              <Button
                onClick={onRegenerate}
                disabled={generating}
                size="lg"
                className="flex items-center gap-2 hover:scale-105 active:scale-100 transition-all duration-200"
                variant="secondary"
              >
                <RefreshCw size={18} className={generating ? 'animate-spin' : ''} />
                Regenerate
              </Button>
              <Button
                onClick={onClose}
                size="lg"
                className="flex items-center gap-2 hover:scale-105 active:scale-100 transition-all duration-200"
                variant="outline"
              >
                <X size={18} />
                Cancel
              </Button>
            </div>
          </div>
        )}
        </div>
      </div>
    </div>
  )
}
