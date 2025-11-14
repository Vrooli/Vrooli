import { useMemo, useRef, useState } from 'react'
import ReactMarkdown from 'react-markdown'
import { Link } from 'react-router-dom'
import { Loader2, Save, RotateCcw, RefreshCcw, CheckCircle2, AlertTriangle, Info, FileEdit, Sparkles } from 'lucide-react'
import type { Draft, DraftSaveStatus, ViewMode, OperationalTargetsResponse, RequirementGroup } from '../../types'
import { ViewModes } from '../../types'
import type { DraftMetrics } from '../../utils/formatters'
import { analyzeDraftStructure } from '../../utils/prdStructure'
import { DraftMetaDialog } from './DraftMetaDialog'
import { StructureChecklist } from './StructureChecklist'
import { OperationalTargetsPanel } from './OperationalTargetsPanel'
import { RequirementSummaryPanel } from './RequirementSummaryPanel'
import { AIAssistantDialog } from './AIAssistantDialog'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '../ui/card'
import { Button } from '../ui/button'
import { Badge } from '../ui/badge'
import { Textarea } from '../ui/textarea'
import { Separator } from '../ui/separator'
import { cn } from '../../lib/utils'
import { buildApiUrl } from '../../utils/apiClient'

interface DraftEditorPaneProps {
  draft: Draft
  editorContent: string
  viewMode: ViewMode
  hasUnsavedChanges: boolean
  saving: boolean
  refreshing: boolean
  saveStatus: DraftSaveStatus | null
  draftMetrics: DraftMetrics
  metaDialogOpen: boolean
  targetsData: OperationalTargetsResponse | null
  targetsLoading: boolean
  targetsError: string | null
  requirementsData: RequirementGroup[] | null
  requirementsLoading: boolean
  requirementsError: string | null
  onContentChange: (value: string) => void
  onSave: () => void
  onDiscard: () => void
  onRefresh: () => void
  onViewModeChange: (mode: ViewMode) => void
  onOpenMeta: () => void
  onCloseMeta: () => void
}

export function DraftEditorPane({
  draft,
  editorContent,
  viewMode,
  hasUnsavedChanges,
  saving,
  refreshing,
  saveStatus,
  draftMetrics,
  metaDialogOpen,
  targetsData,
  targetsLoading,
  targetsError,
  requirementsData,
  requirementsLoading,
  requirementsError,
  onContentChange,
  onSave,
  onDiscard,
  onRefresh,
  onViewModeChange,
  onOpenMeta,
  onCloseMeta,
}: DraftEditorPaneProps) {
  const viewOptions: Array<{ label: string; mode: ViewMode }> = [
    { label: 'Edit', mode: ViewModes.EDIT },
    { label: 'Split', mode: ViewModes.SPLIT },
    { label: 'Preview', mode: ViewModes.PREVIEW },
  ]

  const showEditor = viewMode === ViewModes.EDIT || viewMode === ViewModes.SPLIT
  const showPreview = viewMode === ViewModes.PREVIEW || viewMode === ViewModes.SPLIT
  const statusVariant = draft.status === 'draft' ? 'warning' : 'success'
  const structureSummary = useMemo(() => analyzeDraftStructure(editorContent), [editorContent])
  const textareaRef = useRef<HTMLTextAreaElement | null>(null)
  const [aiDialogOpen, setAIDialogOpen] = useState(false)
  const [aiSection, setAISection] = useState('Executive Summary')
  const [aiContext, setAIContext] = useState('')
  const [aiResult, setAIResult] = useState<string | null>(null)
  const [aiGenerating, setAIGenerating] = useState(false)
  const [aiError, setAIError] = useState<string | null>(null)

  const targets = targetsData?.targets ?? null
  const unmatchedRequirements = targetsData?.unmatched_requirements ?? null
  const requirementGroups = requirementsData ?? null

  const openAIDialog = () => {
    if (textareaRef.current) {
      const selection = textareaRef.current.value.slice(
        textareaRef.current.selectionStart,
        textareaRef.current.selectionEnd,
      )
      if (selection.trim()) {
        setAIContext(selection.trim())
      }
    }
    setAIResult(null)
    setAIError(null)
    setAIDialogOpen(true)
  }

  const handleAIGenerate = async () => {
    setAIGenerating(true)
    setAIError(null)
    try {
      const response = await fetch(buildApiUrl(`/drafts/${draft.id}/ai/generate-section`), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ section: aiSection, context: aiContext }),
      })
      if (!response.ok) {
        const message = await response.text()
        throw new Error(message || 'AI request failed')
      }
      const data = (await response.json()) as { generated_text?: string; success?: boolean; message?: string }
      if (!data.generated_text) {
        throw new Error(data.message || 'AI response missing content')
      }
      setAIResult(data.generated_text.trim())
    } catch (err) {
      setAIError(err instanceof Error ? err.message : 'Unexpected AI error')
    } finally {
      setAIGenerating(false)
    }
  }

  const applyAIResult = (mode: 'insert' | 'replace') => {
    if (!aiResult || !textareaRef.current) {
      return
    }
    const textarea = textareaRef.current
    const start = textarea.selectionStart
    const end = textarea.selectionEnd
    const snippet = aiResult
    const before = editorContent.slice(0, start)
    const after = mode === 'replace' ? editorContent.slice(end) : editorContent.slice(start)
    const nextContent = `${before}${snippet}${after}`
    const cursorPosition = before.length + snippet.length
    onContentChange(nextContent)
    requestAnimationFrame(() => {
      if (textareaRef.current) {
        textareaRef.current.setSelectionRange(cursorPosition, cursorPosition)
        textareaRef.current.focus()
      }
    })
    setAIDialogOpen(false)
  }

  return (
    <section className="space-y-4" aria-labelledby="draft-editor-heading">
      <div className="text-sm text-muted-foreground">
        <Link to="/drafts" className="text-primary hover:underline">
          Drafts
        </Link>{' '}
        / <span className="capitalize">{draft.entity_type}</span> / <span className="font-medium text-foreground">{draft.entity_name}</span>
      </div>

      <Card>
        <CardHeader className="gap-4">
          <div className="flex items-start justify-between gap-4">
            <div>
              <CardTitle id="draft-editor-heading" className="text-2xl">
                {draft.entity_name}
              </CardTitle>
              <CardDescription className="flex items-center gap-3 pt-2">
                <Badge variant="secondary" className="capitalize">
                  {draft.entity_type}
                </Badge>
                <Badge variant={statusVariant} className="gap-1 capitalize">
                  <FileEdit size={14} /> {draft.status === 'draft' ? 'Draft in progress' : draft.status}
                </Badge>
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={onOpenMeta}
                  aria-haspopup="dialog"
                  aria-expanded={metaDialogOpen}
                >
                  <Info size={16} className="mr-2" /> Details
                </Button>
              </CardDescription>
            </div>
            <div className="flex flex-wrap gap-2">
              <Button onClick={onSave} disabled={saving || !hasUnsavedChanges}>
                {saving ? <Loader2 size={16} className="mr-2 animate-spin" /> : <Save size={16} className="mr-2" />}
                Save
              </Button>
              <Button variant="secondary" onClick={onDiscard} disabled={!hasUnsavedChanges}>
                <RotateCcw size={16} className="mr-2" /> Discard
              </Button>
              <Button variant="outline" onClick={onRefresh} disabled={refreshing}>
                {refreshing ? <Loader2 size={16} className="mr-2 animate-spin" /> : <RefreshCcw size={16} className="mr-2" />}
                Refresh
              </Button>
              <Button variant="outline" onClick={openAIDialog}>
                <Sparkles size={16} className="mr-2" /> AI Assist
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-wrap gap-2" role="group" aria-label="Editor view mode">
            {viewOptions.map((option) => (
              <Button
                key={option.mode}
                type="button"
                variant={viewMode === option.mode ? 'default' : 'outline'}
                onClick={() => onViewModeChange(option.mode)}
              >
                {option.label}
              </Button>
            ))}
          </div>

          {saveStatus && (
            <div
              className={cn(
                'flex items-center gap-2 rounded-xl border px-4 py-2 text-sm',
                saveStatus.type === 'success'
                  ? 'border-emerald-200 bg-emerald-50 text-emerald-800'
                  : 'border-amber-200 bg-amber-50 text-amber-900',
              )}
            >
              {saveStatus.type === 'success' ? <CheckCircle2 size={18} /> : <AlertTriangle size={18} />}
              <span>{saveStatus.message}</span>
            </div>
          )}

          <div
            className={cn(
              'grid gap-6',
              viewMode === ViewModes.SPLIT ? 'lg:grid-cols-2' : 'grid-cols-1',
            )}
          >
            {showEditor && (
              <div className="space-y-3">
                <label htmlFor="draft-content" className="text-sm font-semibold text-slate-700">
                  Markdown source
                </label>
                <Textarea
                  id="draft-content"
                  className="min-h-[480px]"
                  value={editorContent}
                  onChange={(event) => onContentChange(event.target.value)}
                  ref={textareaRef}
                  spellCheck={false}
                />
                <p className="text-xs text-muted-foreground">
                  Tip: Use markdown headings (e.g., # Overview) to match the PRD structure.
                </p>
              </div>
            )}

            {showPreview && (
              <div className="space-y-3">
                <span className="text-sm font-semibold text-slate-700">Live preview</span>
                <div className="min-h-[480px] rounded-2xl border bg-white/80 p-4 text-sm leading-relaxed shadow-inner">
                  {editorContent.trim().length === 0 ? (
                    <p className="text-muted-foreground">Start editing to see a formatted preview.</p>
                  ) : (
                    <div className="space-y-4 text-slate-800">
                      <ReactMarkdown>{editorContent}</ReactMarkdown>
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-4">
          <CardTitle className="text-lg">Structure & Coverage</CardTitle>
          <CardDescription>Ensure the draft follows the standard template and links operational targets to requirements.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          <StructureChecklist summary={structureSummary} />
          <Separator />
          <OperationalTargetsPanel
            targets={targets}
            unmatchedRequirements={unmatchedRequirements ?? null}
            loading={targetsLoading}
            error={targetsError}
          />
          <Separator />
          <RequirementSummaryPanel
            groups={requirementGroups}
            loading={requirementsLoading}
            error={requirementsError}
          />
        </CardContent>
      </Card>

      <DraftMetaDialog draft={draft} metrics={draftMetrics} open={metaDialogOpen} onClose={onCloseMeta} />
      <AIAssistantDialog
        open={aiDialogOpen}
        section={aiSection}
        context={aiContext}
        generating={aiGenerating}
        result={aiResult}
        error={aiError}
        onSectionChange={setAISection}
        onContextChange={setAIContext}
        onGenerate={handleAIGenerate}
        onInsert={applyAIResult}
        onClose={() => setAIDialogOpen(false)}
      />
    </section>
  )
}
