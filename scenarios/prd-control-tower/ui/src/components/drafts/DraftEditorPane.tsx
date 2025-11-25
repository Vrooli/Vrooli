import { useMemo, useState } from 'react'
import ReactMarkdown from 'react-markdown'
import { Link } from 'react-router-dom'
import { AlertTriangle, ListTree, FileEdit } from 'lucide-react'
import type {
  Draft,
  DraftSaveStatus,
  ViewMode,
  OperationalTargetsResponse,
  RequirementGroup,
  DraftValidationResult,
  PublishResponse,
} from '../../types'
import { ViewModes } from '../../types'
import type { DraftMetrics } from '../../utils/formatters'
import { analyzeDraftStructure } from '../../utils/prdStructure'
import { useAIAssistant } from '../../hooks/useAIAssistant'
import { DraftMetaDialog } from './DraftMetaDialog'
import { StructureChecklist } from './StructureChecklist'
import { OperationalTargetsPanel } from './OperationalTargetsPanel'
import { OperationalTargetsEditor } from './OperationalTargetsEditor'
import { RequirementSummaryPanel } from './RequirementSummaryPanel'
import { AIAssistantDialog } from './AIAssistantDialog'
import { PublishPreviewDialog } from './PublishPreviewDialog'
import { EditorToolbar } from './EditorToolbar'
import { EditorStatusBadges } from './EditorStatusBadges'
import { ViewModeSwitcher } from './ViewModeSwitcher'
import { SaveStatusNotification } from './SaveStatusNotification'
import { MonacoMarkdownEditor } from './MonacoMarkdownEditor'
import { PRDValidationPanel } from './PRDValidationPanel'
import { Card, CardHeader, CardTitle, CardContent } from '../ui/card'
import { Button } from '../ui/button'
import { Separator } from '../ui/separator'
import { cn } from '../../lib/utils'
import { IssueCategoryCard, type IssueCategorySection } from '../issues'

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
  // Auto-validation props
  validationResult?: DraftValidationResult | null
  validating?: boolean
  validationError?: string | null
  lastValidatedAt?: Date | null
  onManualValidate?: () => Promise<void>
  // Event handlers
  onContentChange: (value: string) => void
  onSave: () => void
  onDiscard: () => void
  onRefresh: () => void
  onViewModeChange: (mode: ViewMode) => void
  onOpenMeta: () => void
  onCloseMeta: () => void
  onPublishSuccess?: (result: PublishResponse) => void
}

/**
 * Main draft editor component.
 * Orchestrates the draft editing experience with toolbar, editor, preview, and metadata panels.
 */
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
  validationResult,
  validating,
  validationError,
  lastValidatedAt,
  onManualValidate,
  onContentChange,
  onSave,
  onDiscard,
  onRefresh,
  onViewModeChange,
  onOpenMeta,
  onCloseMeta,
  onPublishSuccess,
}: DraftEditorPaneProps) {
  const [publishDialogOpen, setPublishDialogOpen] = useState(false)

  // Scroll to targets editor for quick fixing
  const scrollToTargetsEditor = () => {
    const targetsCard = document.getElementById('operational-targets-editor')
    if (targetsCard) {
      targetsCard.scrollIntoView({ behavior: 'smooth', block: 'start' })
      // Flash highlight effect
      targetsCard.classList.add('ring-2', 'ring-amber-500', 'ring-offset-2')
      setTimeout(() => {
        targetsCard.classList.remove('ring-2', 'ring-amber-500', 'ring-offset-2')
      }, 2000)
    }
  }

  const showEditor = viewMode === ViewModes.EDIT || viewMode === ViewModes.SPLIT
  const showPreview = viewMode === ViewModes.PREVIEW || viewMode === ViewModes.SPLIT
  const structureSummary = useMemo(() => analyzeDraftStructure(editorContent), [editorContent])

  const targets = targetsData?.targets ?? null
  const unmatchedRequirements = targetsData?.unmatched_requirements ?? null
  const requirementGroups = requirementsData ?? null

  // Calculate critical orphaned targets (P0/P1 without requirements)
  const orphanedP0Targets = targets?.filter(t => t.criticality === 'P0' && (!t.linked_requirement_ids || t.linked_requirement_ids.length === 0)) ?? []
  const orphanedP1Targets = targets?.filter(t => t.criticality === 'P1' && (!t.linked_requirement_ids || t.linked_requirement_ids.length === 0)) ?? []
  const hasCriticalOrphans = orphanedP0Targets.length > 0 || orphanedP1Targets.length > 0
  const criticalTargetSections: IssueCategorySection[] = [
    orphanedP0Targets.length > 0
      ? {
          id: 'p0-targets',
          title: 'P0 targets',
          items: orphanedP0Targets.map((target) => target.title),
          maxVisible: 4,
        }
      : null,
    orphanedP1Targets.length > 0
      ? {
          id: 'p1-targets',
          title: 'P1 targets',
          items: orphanedP1Targets.map((target) => target.title),
          maxVisible: 4,
        }
      : null,
  ].filter(Boolean) as IssueCategorySection[]

  // AI Assistant hook handles all AI-related state and operations
  const {
    textareaRef,
    aiDialogOpen,
    aiSection,
    aiContext,
    aiAction,
    aiModel,
    aiResult,
    aiGenerating,
    aiError,
    aiInsertMode,
    selectionStart,
    selectionEnd,
    hasSelection,
    aiIncludeExistingContent,
    aiReferencePRDs,
    openAIDialog,
    closeAIDialog,
    setAISection,
    setAIContext,
    setAIAction,
    setAIModel,
    setAIIncludeExistingContent,
    setAIReferencePRDs,
    setAIInsertMode,
    handleAIGenerate,
    handleQuickAction,
    applyAIResult,
    regenerateAIResult,
  } = useAIAssistant({ draftId: draft.id, editorContent, onContentChange })

  return (
    <section className="space-y-4" aria-labelledby="draft-editor-heading">
      {/* Breadcrumb Navigation */}
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
        <div className="text-sm text-slate-600">
          <Link to="/drafts" className="text-violet-600 hover:text-violet-700 hover:underline font-medium transition-colors">
            Drafts
          </Link>{' '}
          <span className="text-slate-400">/</span> <span className="capitalize">{draft.entity_type}</span> <span className="text-slate-400">/</span> <span className="font-semibold text-slate-900">{draft.entity_name}</span>
        </div>
        <Link to={`/scenario/${draft.entity_type}/${draft.entity_name}?tab=requirements`}>
          <Button variant="outline" size="sm" className="gap-2 h-10 hover:border-violet-300 hover:bg-violet-50 transition-colors">
            <ListTree size={16} />
            <span className="hidden sm:inline">View Requirements</span>
            <span className="sm:hidden">Requirements</span>
          </Button>
        </Link>
      </div>

      {/* CRITICAL: Unlinked Targets Alert Banner */}
      {hasCriticalOrphans && (
        <IssueCategoryCard
          title="⚠️ Blocking: Critical targets without requirements"
          icon={<AlertTriangle size={18} />}
          tone="critical"
          description={
            <span className="text-sm text-rose-900">
              {orphanedP0Targets.length > 0 && (
                <>
                  <strong>{orphanedP0Targets.length} P0 target{orphanedP0Targets.length !== 1 ? 's' : ''}</strong> must have linked requirements before publishing.
                </>
              )}
              {orphanedP0Targets.length > 0 && orphanedP1Targets.length > 0 && ' '}
              {orphanedP1Targets.length > 0 && (
                <>
                  <strong>{orphanedP1Targets.length} P1 target{orphanedP1Targets.length !== 1 ? 's' : ''}</strong> should have linked requirements.
                </>
              )}
            </span>
          }
          sections={criticalTargetSections}
          footer={
            <div className="flex flex-wrap items-center justify-between gap-3">
              <span className="text-xs text-rose-900">Action required: link these targets in the editor below.</span>
              <Button onClick={scrollToTargetsEditor} size="sm" className="gap-2 bg-red-600 text-white hover:bg-red-700">
                <AlertTriangle size={14} /> Fix now
              </Button>
            </div>
          }
          className="border-2 border-rose-200"
        />
      )}

      {/* Main Editor Card */}
      <Card className="shadow-md">
        <CardHeader className="gap-5 bg-gradient-to-br from-white to-slate-50/30 border-b">
          <div className="flex flex-col lg:flex-row items-start lg:items-start justify-between gap-4">
            <div className="space-y-2 min-w-0 flex-1">
              <CardTitle id="draft-editor-heading" className="text-2xl font-bold text-slate-900">
                {draft.entity_name}
              </CardTitle>
              <EditorStatusBadges
                draft={draft}
                metaDialogOpen={metaDialogOpen}
                onOpenMeta={onOpenMeta}
                targetsData={targetsData}
                completenessPercent={structureSummary.completenessPercent}
              />
            </div>
            <div className="w-full lg:w-auto">
              <EditorToolbar
                saving={saving}
                refreshing={refreshing}
                hasUnsavedChanges={hasUnsavedChanges}
                onSave={onSave}
                onDiscard={onDiscard}
                onRefresh={onRefresh}
                onOpenAI={openAIDialog}
                onPublish={() => setPublishDialogOpen(true)}
              />
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <ViewModeSwitcher viewMode={viewMode} onViewModeChange={onViewModeChange} />

          <SaveStatusNotification saveStatus={saveStatus} />

          {/* Editor and Preview Layout */}
          <div className={cn('grid gap-6', viewMode === ViewModes.SPLIT ? 'lg:grid-cols-2' : 'grid-cols-1')}>
            {showEditor && (
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <label htmlFor="draft-content" className="text-sm font-bold text-slate-700 uppercase tracking-wide">
                    Markdown source
                  </label>
                  <span className="text-xs text-slate-500">{editorContent.length.toLocaleString()} characters</span>
                </div>
                <MonacoMarkdownEditor
                  value={editorContent}
                  onChange={onContentChange}
                  textareaRef={textareaRef}
                  onOpenAI={openAIDialog}
                  height={480}
                />
                <div className="rounded-xl border-2 border-violet-200/60 bg-gradient-to-br from-violet-50/70 via-purple-50/50 to-violet-50/70 p-4 shadow-sm">
                  <p className="text-xs sm:text-sm leading-relaxed text-slate-700">
                    <strong className="text-violet-700 font-semibold">💡 Quick tips:</strong> Use markdown headings (e.g., <code className="rounded bg-white px-2 py-0.5 text-[11px] font-mono text-violet-600 border border-violet-200"># Overview</code>) to match the PRD structure.
                    Press <kbd className="mx-1 rounded border-2 border-violet-300 bg-white px-2 py-1 text-xs font-bold text-violet-700 shadow-sm">⌘K</kbd> for AI assist,
                    <kbd className="mx-1 rounded border-2 border-violet-300 bg-white px-2 py-1 text-xs font-bold text-violet-700 shadow-sm">⌘S</kbd> to save.
                  </p>
                </div>
              </div>
            )}

            {showPreview && (
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-bold text-slate-700 uppercase tracking-wide">Live preview</span>
                  <span className="text-xs text-slate-500">{structureSummary.completenessPercent}% complete</span>
                </div>
                <div className="h-[480px] overflow-auto rounded-2xl border-2 bg-white p-6 text-sm leading-relaxed shadow-lg">
                  {editorContent.trim().length === 0 ? (
                    <div className="flex flex-col items-center justify-center h-full text-center space-y-3">
                      <div className="rounded-full bg-slate-100 p-4 text-slate-300">
                        <FileEdit size={40} />
                      </div>
                      <p className="text-slate-500 font-medium">Start editing to see a formatted preview</p>
                      <p className="text-xs text-slate-400">Your markdown will render here in real-time</p>
                    </div>
                  ) : (
                    <div className="prd-content">
                      <ReactMarkdown>{editorContent}</ReactMarkdown>
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Operational Targets Editor */}
      <Card id="operational-targets-editor" className="scroll-mt-6 transition-all duration-300">
        <CardHeader className="pb-4">
          <CardTitle className="text-lg">✏️ Edit Operational Targets & Requirements Linkages</CardTitle>
        </CardHeader>
        <CardContent>
          <OperationalTargetsEditor
            draftId={draft.id}
            requirements={requirementGroups}
          />
        </CardContent>
      </Card>

      {/* PRD Validation & Structure */}
      <div className="grid gap-4 lg:grid-cols-2">
        {/* Validation Panel */}
        <PRDValidationPanel
          draftId={draft.id}
          orphanedTargetsCount={
            targets?.filter(t => (t.criticality === 'P0' || t.criticality === 'P1') && (!t.linked_requirement_ids || t.linked_requirement_ids.length === 0)).length ?? 0
          }
          unmatchedRequirementsCount={unmatchedRequirements?.length ?? 0}
          validationResult={validationResult}
          validating={validating}
          error={validationError}
          lastValidatedAt={lastValidatedAt}
          onManualValidate={onManualValidate}
        />

        {/* Structure & Coverage Card */}
        <Card>
          <CardHeader className="pb-4">
            <CardTitle className="text-lg">Structure & Coverage</CardTitle>
          </CardHeader>
          <CardContent className="space-y-6">
            <StructureChecklist summary={structureSummary} />
            <Separator />
            <OperationalTargetsPanel targets={targets} unmatchedRequirements={unmatchedRequirements ?? null} loading={targetsLoading} error={targetsError} />
            <Separator />
            <RequirementSummaryPanel groups={requirementGroups} loading={requirementsLoading} error={requirementsError} />
          </CardContent>
        </Card>
      </div>

      {/* Dialogs */}
      <DraftMetaDialog draft={draft} metrics={draftMetrics} open={metaDialogOpen} onClose={onCloseMeta} />
      <AIAssistantDialog
        open={aiDialogOpen}
        draft={draft}
        section={aiSection}
        context={aiContext}
        action={aiAction}
        model={aiModel}
        generating={aiGenerating}
        result={aiResult}
        error={aiError}
        insertMode={aiInsertMode}
        hasSelection={hasSelection}
        includeExistingContent={aiIncludeExistingContent}
        referencePRDs={aiReferencePRDs}
        originalContent={editorContent}
        selectionStart={selectionStart}
        selectionEnd={selectionEnd}
        onSectionChange={setAISection}
        onContextChange={setAIContext}
        onActionChange={setAIAction}
        onModelChange={setAIModel}
        onInsertModeChange={setAIInsertMode}
        onIncludeExistingContentChange={setAIIncludeExistingContent}
        onReferencePRDsChange={setAIReferencePRDs}
        onGenerate={handleAIGenerate}
        onQuickAction={handleQuickAction}
        onConfirm={applyAIResult}
        onRegenerate={regenerateAIResult}
        onClose={closeAIDialog}
      />
      <PublishPreviewDialog
        draft={draft}
        open={publishDialogOpen}
        onClose={() => setPublishDialogOpen(false)}
        onPublishSuccess={(result) => {
          setPublishDialogOpen(false)
          if (onPublishSuccess) {
            onPublishSuccess(result)
          }
        }}
        orphanedP0Count={orphanedP0Targets.length}
        orphanedP1Count={orphanedP1Targets.length}
      />
    </section>
  )
}
