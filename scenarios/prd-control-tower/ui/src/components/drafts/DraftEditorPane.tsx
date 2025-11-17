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
        <Link to={`/requirements/${draft.entity_type}/${draft.entity_name}`}>
          <Button variant="outline" size="sm" className="gap-2">
            <ListTree size={14} />
            View Requirements
          </Button>
        </Link>
      </div>

      {/* CRITICAL: Unlinked Targets Alert Banner */}
      {hasCriticalOrphans && (
        <div className="rounded-xl border-2 border-red-500 bg-gradient-to-r from-red-50 to-orange-50 p-6 shadow-lg">
          <div className="flex items-start gap-4">
            <div className="rounded-full bg-red-500 p-2 text-white shadow-md">
              <AlertTriangle size={24} className="shrink-0" />
            </div>
            <div className="flex-1 space-y-3">
              <div>
                <h3 className="text-lg font-bold text-red-900 mb-1">⚠️ BLOCKING: Critical Targets Without Requirements</h3>
                <p className="text-sm text-red-800">
                  {orphanedP0Targets.length > 0 && (
                    <><strong className="text-red-900">{orphanedP0Targets.length} P0 target{orphanedP0Targets.length !== 1 ? 's' : ''}</strong> MUST have linked requirements before publishing. </>
                  )}
                  {orphanedP1Targets.length > 0 && (
                    <><strong className="text-red-900">{orphanedP1Targets.length} P1 target{orphanedP1Targets.length !== 1 ? 's' : ''}</strong> SHOULD have linked requirements.</>
                  )}
                </p>
              </div>
              <div className="rounded-lg bg-white border border-red-200 p-4 space-y-2">
                <p className="text-xs font-semibold text-red-900 mb-2">Unlinked Targets:</p>
                <div className="grid gap-2 md:grid-cols-2">
                  {orphanedP0Targets.slice(0, 4).map(target => (
                    <div key={target.id} className="flex items-start gap-2 text-xs text-red-800 bg-red-50 rounded px-2 py-1.5 border border-red-200">
                      <span className="font-semibold text-red-600 shrink-0">P0:</span>
                      <span className="line-clamp-2">{target.title}</span>
                    </div>
                  ))}
                  {orphanedP1Targets.slice(0, 4).map(target => (
                    <div key={target.id} className="flex items-start gap-2 text-xs text-orange-800 bg-orange-50 rounded px-2 py-1.5 border border-orange-200">
                      <span className="font-semibold text-orange-600 shrink-0">P1:</span>
                      <span className="line-clamp-2">{target.title}</span>
                    </div>
                  ))}
                  {(orphanedP0Targets.length + orphanedP1Targets.length > 8) && (
                    <div className="col-span-2 text-xs text-red-700 italic">
                      ...and {orphanedP0Targets.length + orphanedP1Targets.length - 8} more. Scroll down to "Operational Targets Editor" to link them.
                    </div>
                  )}
                </div>
              </div>
              <div className="flex items-center justify-between gap-4 flex-wrap">
                <p className="text-xs text-red-700 flex items-center gap-2">
                  <strong>Action Required:</strong> Link these targets to requirements in the Targets Editor below.
                </p>
                <Button
                  onClick={scrollToTargetsEditor}
                  size="sm"
                  className="bg-red-600 hover:bg-red-700 text-white shadow-md"
                >
                  <AlertTriangle size={14} className="mr-2" />
                  Fix Now - Jump to Editor
                </Button>
              </div>
            </div>
          </div>
        </div>
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
