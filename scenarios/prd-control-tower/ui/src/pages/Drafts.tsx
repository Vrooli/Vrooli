import { useMemo, useEffect, useRef } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { FileEdit, AlertTriangle, Search } from 'lucide-react'

import { useDraftWorkspace } from '../hooks/useDraftWorkspace'
import { DraftCardGrid, DraftEditorPane } from '../components/drafts'
import { Input } from '../components/ui/input'
import { Button } from '../components/ui/button'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '../components/ui/card'
import { Skeleton, CardSkeleton } from '../components/ui/skeleton'
import { TopNav } from '../components/ui/top-nav'
import { Breadcrumbs } from '../components/ui/breadcrumbs'
import { useConfirm } from '../utils/confirmDialog'
import { decodeRouteSegment } from '../utils/formatters'
import { selectors } from '../consts/selectors'

export default function Drafts() {
  const { entityType: routeEntityType, entityName: routeEntityName } = useParams<{
    entityType?: string
    entityName?: string
  }>()
  const navigate = useNavigate()
  const confirm = useConfirm()
  const searchInputRef = useRef<HTMLInputElement>(null)

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

  // Keyboard shortcuts for drafts page
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      const isMac = navigator.platform.toUpperCase().includes('MAC')
      const modKey = isMac ? event.metaKey : event.ctrlKey

      // ⌘K / Ctrl+K: Focus search and auto-select
      if (modKey && event.key === 'k') {
        event.preventDefault()
        searchInputRef.current?.focus()
        searchInputRef.current?.select()
      }

      // ⌘⇧C / Ctrl+Shift+C: Clear filter
      if (modKey && event.shiftKey && event.key === 'C') {
        event.preventDefault()
        setFilter('')
        searchInputRef.current?.focus()
      }

      // Escape: Clear filter if search is focused and has content
      if (event.key === 'Escape' && document.activeElement === searchInputRef.current && filter) {
        event.preventDefault()
        setFilter('')
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [filter, setFilter])

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
      <CardContent className="flex flex-col items-center gap-6 py-12 sm:py-16 text-center">
        <div className="rounded-full bg-gradient-to-br from-violet-100 to-violet-200 p-5 sm:p-6 text-violet-600 shadow-inner">
          <FileEdit size={48} className="sm:w-14 sm:h-14" />
        </div>
        <div className="space-y-3 max-w-lg px-4">
          <h3 className="text-xl sm:text-2xl font-bold text-slate-900">
            {drafts.length === 0 ? 'No drafts yet' : 'No matches found'}
          </h3>
          <p className="text-sm sm:text-base text-slate-600 leading-relaxed">
            {drafts.length === 0
              ? 'Draft PRDs live here while you build and validate them. Start your first PRD to begin.'
              : `No drafts match "${filter}". Try adjusting your search or clear the filter to see all drafts.`}
          </p>
          {drafts.length === 0 && (
            <div className="rounded-xl border border-violet-100 bg-white p-4 sm:p-5 text-left space-y-3 mt-6">
              <p className="text-xs font-semibold uppercase tracking-wider text-violet-600 text-center">Quick Start Guide</p>
              <div className="flex items-start gap-3">
                <span className="flex h-7 w-7 items-center justify-center rounded-full bg-violet-100 text-xs font-bold text-violet-700 shrink-0">1</span>
                <p className="text-sm text-slate-700 leading-relaxed">Go to the <Link to="/catalog" className="font-semibold text-violet-600 underline hover:text-violet-700">Catalog</Link> page</p>
              </div>
              <div className="flex items-start gap-3">
                <span className="flex h-7 w-7 items-center justify-center rounded-full bg-violet-100 text-xs font-bold text-violet-700 shrink-0">2</span>
                <p className="text-sm text-slate-700 leading-relaxed">Find a scenario or resource you want to document</p>
              </div>
              <div className="flex items-start gap-3">
                <span className="flex h-7 w-7 items-center justify-center rounded-full bg-violet-100 text-xs font-bold text-violet-700 shrink-0">3</span>
                <p className="text-sm text-slate-700 leading-relaxed">Click <strong>"Edit PRD"</strong> to create a draft</p>
              </div>
              <div className="flex items-start gap-3">
                <span className="flex h-7 w-7 items-center justify-center rounded-full bg-violet-100 text-xs font-bold text-violet-700 shrink-0">4</span>
                <p className="text-sm text-slate-700 leading-relaxed">Return here to edit, validate, and publish</p>
              </div>
            </div>
          )}
        </div>
        <div className="flex flex-wrap gap-3 justify-center mt-2">
          {drafts.length === 0 ? (
            <Button asChild size="lg" className="min-h-[44px] gap-2 font-semibold shadow-md hover:shadow-lg hover:scale-105 active:scale-95 transition-all">
              <Link to="/catalog">
                <Search size={20} />
                <span>Browse catalog</span>
              </Link>
            </Button>
          ) : (
            <Button variant="outline" size="lg" onClick={() => setFilter('')} className="min-h-[44px] gap-2 hover:bg-slate-50 active:scale-95 transition-all">
              <Search size={20} />
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

      <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div className="relative w-full max-w-xl">
          <Search className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" size={18} />
          <Input
            ref={searchInputRef}
            id="drafts-search"
            data-testid={selectors.drafts.searchInput}
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
        <div className="flex flex-wrap gap-2 text-xs text-slate-500">
          <kbd className="rounded border border-slate-200 bg-slate-50 px-2 py-1 font-mono shadow-sm">⌘K</kbd>
          <span>to search</span>
          <span className="text-slate-300">•</span>
          <kbd className="rounded border border-slate-200 bg-slate-50 px-2 py-1 font-mono shadow-sm">Esc</kbd>
          <span>to clear</span>
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
    <div className="space-y-8">
      {/* Header skeleton */}
      <header className="rounded-3xl border bg-white/90 p-6 shadow-soft-lg">
        <div className="space-y-4">
          <Skeleton className="h-3 w-32" />
          <div className="flex items-center gap-3">
            <Skeleton className="h-12 w-12 rounded-2xl" />
            <Skeleton className="h-8 w-48" />
          </div>
          <Skeleton className="h-4 w-full max-w-2xl" />
        </div>
      </header>

      {/* Search skeleton */}
      <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <Skeleton className="h-10 w-full max-w-xl rounded-xl" />
        <Skeleton className="h-6 w-32 rounded" />
      </div>

      {/* Stats skeleton */}
      <section className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {Array.from({ length: 3 }).map((_, idx) => (
          <Card key={`stat-${idx}`} className="border border-slate-200 bg-white/80" style={{ animationDelay: `${idx * 50}ms` }}>
            <CardHeader className="pb-3">
              <Skeleton className="h-3 w-24 mb-2" />
              <Skeleton className="h-10 w-16" />
            </CardHeader>
          </Card>
        ))}
      </section>

      {/* Drafts grid skeleton */}
      <section className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {Array.from({ length: 6 }).map((_, idx) => (
          <CardSkeleton key={`draft-${idx}`} className="h-44" style={{ animationDelay: `${idx * 40}ms` }} />
        ))}
      </section>
    </div>
  )
}
