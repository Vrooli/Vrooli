import { useMemo } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { FileEdit, AlertTriangle, Search } from 'lucide-react'

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
      <div className="app-container space-y-6" data-layout="dual">
        <TopNav />
        <DraftWorkspaceSkeleton />
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
    <Card className="border-2 border-dashed bg-gradient-to-br from-white via-violet-50/20 to-slate-50">
      <CardContent className="flex flex-col items-center gap-6 py-16 text-center">
        <div className="rounded-full bg-gradient-to-br from-violet-100 to-violet-200 p-6 text-violet-600 shadow-inner">
          <FileEdit size={56} />
        </div>
        <div className="space-y-3 max-w-lg">
          <h3 className="text-xl font-bold text-slate-900">
            {drafts.length === 0 ? 'No drafts yet' : 'No matches found'}
          </h3>
          <p className="text-sm sm:text-base text-slate-600 leading-relaxed">
            {drafts.length === 0
              ? 'Draft PRDs live here while you build and validate them.'
              : `No drafts match "${filter}". Try adjusting your search or clear the filter to see all drafts.`}
          </p>
          {drafts.length === 0 && (
            <div className="rounded-xl border border-violet-100 bg-white p-4 text-left space-y-3 mt-4">
              <p className="text-xs font-semibold uppercase tracking-wider text-violet-600 text-center">Quick Start Guide</p>
              <div className="flex items-start gap-3">
                <span className="flex h-6 w-6 items-center justify-center rounded-full bg-violet-100 text-xs font-bold text-violet-700 shrink-0">1</span>
                <p className="text-sm text-slate-700">Go to the <Link to="/catalog" className="font-semibold text-violet-600 underline">Catalog</Link> page</p>
              </div>
              <div className="flex items-start gap-3">
                <span className="flex h-6 w-6 items-center justify-center rounded-full bg-violet-100 text-xs font-bold text-violet-700 shrink-0">2</span>
                <p className="text-sm text-slate-700">Find a scenario or resource you want to document</p>
              </div>
              <div className="flex items-start gap-3">
                <span className="flex h-6 w-6 items-center justify-center rounded-full bg-violet-100 text-xs font-bold text-violet-700 shrink-0">3</span>
                <p className="text-sm text-slate-700">Click <strong>"Edit PRD"</strong> to create a draft</p>
              </div>
              <div className="flex items-start gap-3">
                <span className="flex h-6 w-6 items-center justify-center rounded-full bg-violet-100 text-xs font-bold text-violet-700 shrink-0">4</span>
                <p className="text-sm text-slate-700">Return here to edit, validate, and publish</p>
              </div>
            </div>
          )}
        </div>
        <div className="flex flex-wrap gap-3 justify-center">
          {drafts.length === 0 ? (
            <Button asChild size="lg" className="h-12 gap-2 font-semibold shadow-md hover:shadow-lg">
              <Link to="/catalog">
                <Search size={18} />
                <span>Browse catalog</span>
              </Link>
            </Button>
          ) : (
            <Button variant="outline" size="lg" onClick={() => setFilter('')} className="h-12 gap-2">
              <Search size={18} />
              <span>Clear filter</span>
            </Button>
          )}
        </div>
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
          navigate(`/scenario/${result.scenario_type}/${encoded}?tab=prd`)
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
            className="pl-9 pr-20"
            placeholder="Search drafts by name, type, or owner..."
            value={filter}
            onChange={(event) => setFilter(event.target.value)}
            aria-label="Search drafts"
          />
          {!filter && (
            <div className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2">
              <span className="hidden sm:inline-block text-xs text-slate-400">
                Tip: Use filters to narrow results
              </span>
            </div>
          )}
          {filter && (
            <button
              type="button"
              className="absolute right-3 top-1/2 -translate-y-1/2 rounded-lg bg-slate-100 px-3 py-1.5 text-xs font-medium text-slate-600 hover:bg-slate-200 active:scale-95 transition-all"
              onClick={() => setFilter('')}
              aria-label="Clear search"
            >
              Clear
            </button>
          )}
        </div>
      </div>

      <section className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <Card className="border-2 border-violet-100 bg-gradient-to-br from-white to-violet-50/30">
          <CardHeader className="pb-3">
            <CardDescription className="text-xs font-semibold uppercase tracking-wider">Total drafts</CardDescription>
            <CardTitle className="text-4xl font-bold text-violet-700">{drafts.length}</CardTitle>
          </CardHeader>
        </Card>
        <Card className="border border-slate-200 bg-white/80 hover:border-slate-300 transition-colors">
          <CardHeader className="pb-3">
            <CardDescription className="text-xs font-semibold uppercase tracking-wider">Scenarios</CardDescription>
            <CardTitle className="text-4xl font-bold text-slate-700">{drafts.filter((draft) => draft.entity_type === 'scenario').length}</CardTitle>
          </CardHeader>
        </Card>
        <Card className="border border-slate-200 bg-white/80 hover:border-slate-300 transition-colors">
          <CardHeader className="pb-3">
            <CardDescription className="text-xs font-semibold uppercase tracking-wider">Resources</CardDescription>
            <CardTitle className="text-4xl font-bold text-slate-700">{drafts.filter((draft) => draft.entity_type === 'resource').length}</CardTitle>
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

function DraftWorkspaceSkeleton() {
  return (
    <div className="space-y-6">
      <section className="rounded-3xl border bg-white/90 p-6 shadow-soft-lg">
        <div className="space-y-4 animate-pulse">
          <div className="h-4 w-32 rounded-full bg-slate-200" />
          <div className="h-8 w-64 rounded-full bg-slate-200" />
          <div className="h-4 w-full max-w-2xl rounded-full bg-slate-100" />
        </div>
      </section>

      <div className="grid gap-6 lg:grid-cols-[1fr,420px]">
        <Card className="border bg-white/90 shadow-soft-lg">
          <CardContent className="space-y-4">
            <div className="h-10 w-full rounded-xl bg-slate-100" />
            <div className="grid gap-3 sm:grid-cols-2">
              {Array.from({ length: 4 }).map((_, idx) => (
                <div key={`draft-card-${idx}`} className="h-32 rounded-2xl border border-slate-100 bg-slate-50" />
              ))}
            </div>
          </CardContent>
        </Card>

        <Card className="border bg-white/90 shadow-soft-lg">
          <CardContent className="space-y-4">
            <div className="h-6 w-32 rounded-full bg-slate-100" />
            <div className="space-y-3">
              {Array.from({ length: 5 }).map((_, idx) => (
                <div key={`meta-${idx}`} className="h-4 w-full rounded-full bg-slate-100" />
              ))}
            </div>
            <div className="h-48 rounded-2xl border border-dashed border-slate-200 bg-slate-50" />
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
