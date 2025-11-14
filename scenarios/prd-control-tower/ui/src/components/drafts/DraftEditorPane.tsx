import { useMemo, useState } from 'react'
import ReactMarkdown from 'react-markdown'
import { Link } from 'react-router-dom'
import type { Draft, DraftSaveStatus, ViewMode, OperationalTargetsResponse, RequirementGroup } from '../../types'
import { ViewModes } from '../../types'
import type { DraftMetrics } from '../../utils/formatters'
import { analyzeDraftStructure } from '../../utils/prdStructure'
import { useAIAssistant } from '../../hooks/useAIAssistant'
import { DraftMetaDialog } from './DraftMetaDialog'
import { StructureChecklist } from './StructureChecklist'
import { OperationalTargetsPanel } from './OperationalTargetsPanel'
import { RequirementSummaryPanel } from './RequirementSummaryPanel'
import { AIAssistantDialog } from './AIAssistantDialog'
import { PublishPreviewDialog } from './PublishPreviewDialog'
import { EditorToolbar } from './EditorToolbar'
import { EditorStatusBadges } from './EditorStatusBadges'
import { ViewModeSwitcher } from './ViewModeSwitcher'
import { SaveStatusNotification } from './SaveStatusNotification'
import { Card, CardHeader, CardTitle, CardContent } from '../ui/card'
import { Textarea } from '../ui/textarea'
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
  onContentChange: (value: string) => void
  onSave: () => void
  onDiscard: () => void
  onRefresh: () => void
  onViewModeChange: (mode: ViewMode) => void
  onOpenMeta: () => void
  onCloseMeta: () => void
  onPublishSuccess?: () => void
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

  const showEditor = viewMode === ViewModes.EDIT || viewMode === ViewModes.SPLIT
  const showPreview = viewMode === ViewModes.PREVIEW || viewMode === ViewModes.SPLIT
  const structureSummary = useMemo(() => analyzeDraftStructure(editorContent), [editorContent])

  const targets = targetsData?.targets ?? null
  const unmatchedRequirements = targetsData?.unmatched_requirements ?? null
  const requirementGroups = requirementsData ?? null

  // AI Assistant hook handles all AI-related state and operations
  const {
    textareaRef,
    aiDialogOpen,
    aiSection,
    aiContext,
    aiResult,
    aiGenerating,
    aiError,
    openAIDialog,
    closeAIDialog,
    setAISection,
    setAIContext,
    handleAIGenerate,
    applyAIResult,
  } = useAIAssistant({ draftId: draft.id, editorContent, onContentChange })

  return (
    <section className="space-y-4" aria-labelledby="draft-editor-heading">
      {/* Breadcrumb Navigation */}
      <div className="text-sm text-muted-foreground">
        <Link to="/drafts" className="text-primary hover:underline">
          Drafts
        </Link>{' '}
        / <span className="capitalize">{draft.entity_type}</span> / <span className="font-medium text-foreground">{draft.entity_name}</span>
      </div>

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
                <Textarea
                  id="draft-content"
                  className="min-h-[480px]"
                  value={editorContent}
                  onChange={(event) => onContentChange(event.target.value)}
                  ref={textareaRef}
                  spellCheck={false}
                />
                <p className="text-xs text-muted-foreground">Tip: Use markdown headings (e.g., # Overview) to match the PRD structure.</p>
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

      {/* Dialogs */}
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
        onClose={closeAIDialog}
      />
      <PublishPreviewDialog
        draft={draft}
        open={publishDialogOpen}
        onClose={() => setPublishDialogOpen(false)}
        onPublishSuccess={() => {
          setPublishDialogOpen(false)
          if (onPublishSuccess) {
            onPublishSuccess()
          }
        }}
      />
    </section>
  )
}
