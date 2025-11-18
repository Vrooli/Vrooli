import { useMemo, useState } from 'react'
import ReactMarkdown from 'react-markdown'
import { Link } from 'react-router-dom'
import { AlertTriangle, ListTree } from 'lucide-react'
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
    aiResult,
    aiGenerating,
    aiError,
    selectionStart,
    selectionEnd,
    hasSelection,
    openAIDialog,
    closeAIDialog,
    setAISection,
    setAIContext,
    setAIAction,
    handleAIGenerate,
    handleQuickAction,
    applyAIResult,
  } = useAIAssistant({ draftId: draft.id, editorContent, onContentChange })

  return (
    <section className="space-y-4" aria-labelledby="draft-editor-heading">
      {/* Breadcrumb Navigation */}
      <div className="flex items-center justify-between gap-4">
        <div className="text-sm text-muted-foreground">
          <Link to="/drafts" className="text-primary hover:underlink">
            Drafts
          </Link>{' '}
          / <span className="capitalize">{draft.entity_type}</span> / <span className="font-medium text-foreground">{draft.entity_name}</span>
        </div>
        <Link to={`/scenario/${draft.entity_type}/${draft.entity_name}?tab=requirements`}>
          <Button variant="outline" size="sm" className="gap-2">
            <ListTree size={14} />
            View Requirements
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
      <Card>
        <CardHeader className="gap-4">
          <div className="flex items-start justify-between gap-4">
            <div>
              <CardTitle id="draft-editor-heading" className="text-2xl">
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
        </CardHeader>
        <CardContent className="space-y-4">
          <ViewModeSwitcher viewMode={viewMode} onViewModeChange={onViewModeChange} />

          <SaveStatusNotification saveStatus={saveStatus} />

          {/* Editor and Preview Layout */}
          <div className={cn('grid gap-6', viewMode === ViewModes.SPLIT ? 'lg:grid-cols-2' : 'grid-cols-1')}>
            {showEditor && (
              <div className="space-y-3">
                <label htmlFor="draft-content" className="text-sm font-semibold text-slate-700">
                  Markdown source
                </label>
                <MonacoMarkdownEditor
                  value={editorContent}
                  onChange={onContentChange}
                  textareaRef={textareaRef}
                  onOpenAI={openAIDialog}
                  height={480}
                />
                <p className="text-xs text-muted-foreground">
                  <strong>Tip:</strong> Use markdown headings (e.g., # Overview) to match the PRD structure.
                  Press <kbd className="px-1.5 py-0.5 text-xs font-semibold bg-slate-100 border rounded">Cmd+K</kbd> for AI assist,{' '}
                  <kbd className="px-1.5 py-0.5 text-xs font-semibold bg-slate-100 border rounded">Cmd+F</kbd> to find/replace.
                </p>
              </div>
            )}

            {showPreview && (
              <div className="space-y-3">
                <span className="text-sm font-semibold text-slate-700">Live preview</span>
                <div className="h-[480px] overflow-auto rounded-2xl border bg-white/80 p-4 text-sm leading-relaxed shadow-inner">
                  {editorContent.trim().length === 0 ? (
                    <p className="text-muted-foreground">Start editing to see a formatted preview.</p>
                  ) : (
                    <div className="space-y-4 text-slate-800 prose prose-sm max-w-none">
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
        section={aiSection}
        context={aiContext}
        action={aiAction}
        generating={aiGenerating}
        result={aiResult}
        error={aiError}
        hasSelection={hasSelection}
        originalContent={editorContent}
        selectionStart={selectionStart}
        selectionEnd={selectionEnd}
        onSectionChange={setAISection}
        onContextChange={setAIContext}
        onActionChange={setAIAction}
        onGenerate={handleAIGenerate}
        onQuickAction={handleQuickAction}
        onInsert={applyAIResult}
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
