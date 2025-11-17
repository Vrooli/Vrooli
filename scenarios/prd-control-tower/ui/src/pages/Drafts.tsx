import { useMemo } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { FileEdit, AlertTriangle, Search, Loader2 } from 'lucide-react'

import { useDraftWorkspace } from '../hooks/useDraftWorkspace'
import { DraftCardGrid, DraftEditorPane } from '../components/drafts'
import { Input } from '../components/ui/input'
import { Button } from '../components/ui/button'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '../components/ui/card'
import { TopNav } from '../components/ui/top-nav'
import { Breadcrumbs } from '../components/ui/breadcrumbs'
import { useConfirm } from '../utils/confirmDialog'
import { decodeRouteSegment } from '../utils/formatters'

export default function Drafts() {
  const { entityType: routeEntityType, entityName: routeEntityName } = useParams<{
    entityType?: string
    entityName?: string
  }>()
  const navigate = useNavigate()
  const confirm = useConfirm()

  const decodedRouteName = useMemo(() => decodeRouteSegment(routeEntityName), [routeEntityName])

  const {
    loading,
    refreshing,
    error,
    drafts,
    filter,
    setFilter,
    filteredDrafts,
    selectedDraft,
    viewMode,
    setViewMode,
    editorContent,
    handleEditorChange,
    hasUnsavedChanges,
    saving,
    saveStatus,
    handleSaveDraft,
    handleDiscardChanges,
    handleRefreshDraft,
    handleSelectDraft,
    handleDelete,
    metaDialogOpen,
    openMetaDialog,
    closeMetaDialog,
    targetsData,
    targetsLoading,
    targetsError,
    requirementsData,
    requirementsLoading,
    requirementsError,
    draftMetrics,
    validationResult,
    validating,
    validationError,
    lastValidatedAt,
    triggerValidation,
  } = useDraftWorkspace({
    routeEntityType,
    routeEntityName,
    decodedRouteName,
    navigate,
    confirm,
  })

  if (loading && !refreshing) {
    return (
      <div className="app-container" data-layout="dual">
        <Card className="border-dashed bg-white/80">
          <CardContent className="flex items-center gap-3 text-muted-foreground">
            <Loader2 size={20} className="animate-spin" /> Loading drafts...
          </CardContent>
        </Card>
      </div>
    )
  }

  if (error) {
    return (
      <div className="app-container" data-layout="dual">
        <Card className="border-amber-200 bg-amber-50">
          <CardContent className="flex items-center gap-3 text-amber-900">
            <AlertTriangle size={20} />
            <p>Error loading drafts: {error}</p>
          </CardContent>
        </Card>
      </div>
    )
  }

  const shouldShowDraftEditor = Boolean(selectedDraft)
  const shouldShowDraftNotFound =
    Boolean(routeEntityName && decodedRouteName) && !selectedDraft && !loading && !refreshing
  const isDetailRoute = routeEntityName !== undefined

  const listContent = filteredDrafts.length === 0 ? (
    <Card className="border-dashed bg-slate-50/80">
      <CardContent className="flex flex-col items-center gap-3 py-12 text-center text-muted-foreground">
        <FileEdit size={48} />
        {drafts.length === 0 ? (
          <p>No drafts found. Create a new draft to get started.</p>
        ) : (
          <p>No drafts found matching your search.</p>
        )}
      </CardContent>
    </Card>
  ) : (
    <DraftCardGrid
      drafts={filteredDrafts}
      selectedDraftId={selectedDraft?.id ?? null}
      onSelectDraft={handleSelectDraft}
      onDeleteDraft={handleDelete}
    />
  )

  const detailContent = shouldShowDraftNotFound ? (
    <Card className="border-amber-200 bg-amber-50/70">
      <CardHeader>
        <CardTitle>Draft not found</CardTitle>
        <CardDescription>
          We could not locate a draft for <strong>{decodedRouteName}</strong>. Double-check the draft URL or return to the draft list.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Button asChild variant="secondary">
          <Link to="/drafts">Back to drafts</Link>
        </Button>
      </CardContent>
    </Card>
  ) : shouldShowDraftEditor && selectedDraft ? (
      <DraftEditorPane
        draft={selectedDraft}
      editorContent={editorContent}
      viewMode={viewMode}
      hasUnsavedChanges={hasUnsavedChanges}
      saving={saving}
      refreshing={refreshing}
      saveStatus={saveStatus}
      draftMetrics={draftMetrics}
      metaDialogOpen={metaDialogOpen}
      targetsData={targetsData}
      targetsLoading={targetsLoading}
      targetsError={targetsError}
      requirementsData={requirementsData}
      requirementsLoading={requirementsLoading}
      requirementsError={requirementsError}
      validationResult={validationResult}
      validating={validating}
      validationError={validationError}
      lastValidatedAt={lastValidatedAt}
      onManualValidate={triggerValidation}
      onContentChange={handleEditorChange}
      onSave={handleSaveDraft}
      onDiscard={handleDiscardChanges}
      onRefresh={handleRefreshDraft}
      onViewModeChange={setViewMode}
      onOpenMeta={openMetaDialog}
      onCloseMeta={closeMetaDialog}
      onPublishSuccess={(result) => {
        if (result?.scenario_type && result?.scenario_id) {
          const encoded = encodeURIComponent(result.scenario_id)
          navigate(`/requirements/${result.scenario_type}/${encoded}?tab=prd`)
          return
        }
        navigate('/catalog')
      }}
    />
  ) : null

  if (isDetailRoute) {
    const breadcrumbItems = [
      { label: 'Catalog', to: '/catalog' },
      { label: 'Drafts', to: '/drafts' },
      { label: selectedDraft?.entity_name || 'Draft' },
    ]

    return (
      <div className="app-container space-y-6" data-layout="dual">
        <TopNav />
        <Breadcrumbs items={breadcrumbItems} />
        {detailContent}
      </div>
    )
  }

  return (
    <div className="app-container space-y-8" data-layout="dual">
      <TopNav />
      <header className="rounded-3xl border bg-white/90 p-6 shadow-soft-lg">
        <div className="space-y-2">
          <span className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-500">Draft workspace</span>
          <div className="flex items-center gap-3 text-3xl font-semibold text-slate-900">
            <span className="rounded-2xl bg-slate-100 p-3 text-slate-600">
              <FileEdit size={28} strokeWidth={2.5} />
            </span>
            Draft PRDs
          </div>
          <p className="max-w-3xl text-base text-muted-foreground">Manage and edit PRD drafts before publishing.</p>
        </div>
      </header>

      <div className="flex flex-col gap-3 md:flex-row">
        <div className="relative w-full max-w-xl">
          <Search className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" size={18} />
          <Input
            id="drafts-search"
            type="text"
            className="pl-9"
            placeholder="Search drafts by name, type, or owner..."
            value={filter}
            onChange={(event) => setFilter(event.target.value)}
          />
          {filter && (
            <button
              type="button"
              className="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-slate-500"
              onClick={() => setFilter('')}
              aria-label="Clear search"
            >
              Clear
            </button>
          )}
        </div>
      </div>

      <section className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <Card>
          <CardHeader className="pb-2">
            <CardDescription>Total drafts</CardDescription>
            <CardTitle className="text-3xl">{drafts.length}</CardTitle>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardDescription>Scenarios</CardDescription>
            <CardTitle className="text-3xl">{drafts.filter((draft) => draft.entity_type === 'scenario').length}</CardTitle>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardDescription>Resources</CardDescription>
            <CardTitle className="text-3xl">{drafts.filter((draft) => draft.entity_type === 'resource').length}</CardTitle>
          </CardHeader>
        </Card>
      </section>

      {listContent}

      <div className="flex flex-wrap gap-4 text-sm">
        <Link to="/backlog" className="text-primary hover:underline">
          ← Go to Backlog
        </Link>
        <Link to="/catalog" className="text-primary hover:underline">
          ← Back to Catalog
        </Link>
      </div>
    </div>
  )
}
