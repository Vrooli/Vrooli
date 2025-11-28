import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import {
  AlertTriangle,
  ClipboardList,
  Layers,
  RefreshCw,
  Search,
  Sparkles,
  StickyNote,
  Tag,
  Target,
} from 'lucide-react'

import { Badge } from '../components/ui/badge'
import { Button } from '../components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../components/ui/card'
import { Input } from '../components/ui/input'
import { Separator } from '../components/ui/separator'
import { Skeleton, CardSkeleton } from '../components/ui/skeleton'
import { Tabs, TabsList, TabsTrigger } from '../components/ui/tabs'
import { HelpTooltip } from '../components/ui/help-tooltip'
import { TopNav } from '../components/ui/top-nav'
import { buildApiUrl } from '../utils/apiClient'
import { usePrepareDraft } from '../utils/useDraft'
import type { CatalogEntry, CatalogResponse } from '../types'
import { CatalogCard, QuickAddIdeaDialog } from '../components/catalog'
import { cn } from '../lib/utils'
import { selectors } from '../consts/selectors'

type CatalogFilter = 'all' | 'scenario' | 'resource'
type CatalogSort = 'status' | 'name_asc' | 'name_desc' | 'coverage' | 'recently_viewed'

export default function Catalog() {
  const [entries, setEntries] = useState<CatalogEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [search, setSearch] = useState('')
  const [typeFilter, setTypeFilter] = useState<CatalogFilter>('all')
  const [sortMode, setSortMode] = useState<CatalogSort>('status')
  const [quickAddOpen, setQuickAddOpen] = useState(false)
  const [selectedLabels, setSelectedLabels] = useState<string[]>([])
  const [availableLabels, setAvailableLabels] = useState<string[]>([])
  const navigate = useNavigate()
  const searchInputRef = useRef<HTMLInputElement>(null)

  const { prepareDraft, preparingId } = usePrepareDraft({
    onSuccess: (entityType, entityName) => {
      setEntries(prev =>
        prev.map(item =>
          item.type === entityType && item.name === entityName
            ? { ...item, has_draft: true }
            : item,
        ),
      )
    },
  })

  const fetchCatalog = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const response = await fetch(buildApiUrl('/catalog?include_requirements=1&include_visits=1'))
      if (!response.ok) {
        throw new Error(`Failed to fetch catalog: ${response.statusText}`)
      }
      const data: CatalogResponse = await response.json()
      setEntries(data.entries)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [])

  const fetchLabels = useCallback(async () => {
    try {
      const response = await fetch(buildApiUrl('/catalog/labels'))
      if (!response.ok) {
        console.warn('Failed to fetch labels')
        return
      }
      const data: { labels: string[] } = await response.json()
      setAvailableLabels(data.labels || [])
    } catch (err) {
      console.warn('Failed to fetch labels:', err)
    }
  }, [])

  const recordVisit = useCallback(async (entityType: string, entityName: string) => {
    try {
      const response = await fetch(buildApiUrl(`/catalog/${entityType}/${entityName}/visit`), {
        method: 'POST',
      })
      if (!response.ok) {
        console.warn(`Failed to record visit for ${entityType}:${entityName}`)
      }
    } catch (err) {
      console.warn('Failed to record visit:', err)
    }
  }, [])

  const handleLabelsUpdate = useCallback((entityType: string, entityName: string, labels: string[]) => {
    setEntries(prev =>
      prev.map(entry =>
        entry.type === entityType && entry.name === entityName
          ? { ...entry, labels }
          : entry
      )
    )
    // Refresh available labels
    fetchLabels()
  }, [fetchLabels])

  useEffect(() => {
    fetchCatalog()
    fetchLabels()
  }, [fetchCatalog, fetchLabels])

  // Keyboard shortcuts: Cmd/Ctrl+K to focus search, Escape to clear
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      // Cmd/Ctrl+K: Focus search input
      if ((event.metaKey || event.ctrlKey) && event.key === 'k') {
        event.preventDefault()
        searchInputRef.current?.focus()
        searchInputRef.current?.select()
        return
      }
      // Escape: Clear search if active, otherwise blur
      if (event.key === 'Escape') {
        if (search) {
          event.preventDefault()
          setSearch('')
        }
        searchInputRef.current?.blur()
        return
      }
      // Cmd/Ctrl+Shift+F: Clear all filters
      if ((event.metaKey || event.ctrlKey) && event.shiftKey && event.key === 'F') {
        event.preventDefault()
        setSearch('')
        setTypeFilter('all')
        searchInputRef.current?.focus()
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [search])

  const catalogCounts = useMemo(() => {
    const summary = entries.reduce(
      (acc, entry) => {
        acc.total += 1
        if (entry.type === 'scenario') acc.scenarios += 1
        if (entry.type === 'resource') acc.resources += 1
        if (entry.has_prd) acc.withPrd += 1
        if (entry.has_draft) acc.drafts += 1
        return acc
      },
      { total: 0, scenarios: 0, resources: 0, withPrd: 0, drafts: 0 },
    )
    return { ...summary, missing: summary.total - summary.withPrd }
  }, [entries])

  const coverage = catalogCounts.total ? Math.round((catalogCounts.withPrd / catalogCounts.total) * 100) : 0
  const missingPercentage = catalogCounts.total ? Math.round((catalogCounts.missing / catalogCounts.total) * 100) : 0

  const requirementTotals = useMemo(() => {
    return entries.reduce(
      (acc, entry) => {
        const summary = entry.requirements_summary
        if (!summary) {
          return acc
        }
        acc.total += summary.total
        acc.completed += summary.completed
        acc.inProgress += summary.in_progress
        acc.pending += summary.pending
        acc.p0 += summary.p0
        acc.p1 += summary.p1
        acc.p2 += summary.p2
        return acc
      },
      { total: 0, completed: 0, inProgress: 0, pending: 0, p0: 0, p1: 0, p2: 0 },
    )
  }, [entries])

  const filteredEntries = useMemo(() => {
    const query = search.trim().toLowerCase()
    const filtered = entries.filter(entry => {
      const matchesSearch =
        query === '' ||
        entry.name.toLowerCase().includes(query) ||
        entry.description.toLowerCase().includes(query)
      const matchesType = typeFilter === 'all' || entry.type === typeFilter
      const matchesLabels = selectedLabels.length === 0 ||
        (entry.labels && selectedLabels.every(label => entry.labels?.includes(label)))
      return matchesSearch && matchesType && matchesLabels
    })

    const getCoverageScore = (entry: CatalogEntry) => {
      const summary = entry.requirements_summary
      if (!summary || summary.total === 0) {
        return 0
      }
      return summary.completed / summary.total
    }

    return [...filtered].sort((a, b) => {
      switch (sortMode) {
        case 'name_desc':
          return b.name.localeCompare(a.name)
        case 'coverage':
          return getCoverageScore(b) - getCoverageScore(a)
        case 'recently_viewed': {
          // Sort by visit count first, then by last visited timestamp
          const aVisits = a.visit_count ?? 0
          const bVisits = b.visit_count ?? 0
          if (bVisits !== aVisits) {
            return bVisits - aVisits
          }
          // If visit counts equal, sort by most recently visited
          const aTime = a.last_visited_at ? new Date(a.last_visited_at).getTime() : 0
          const bTime = b.last_visited_at ? new Date(b.last_visited_at).getTime() : 0
          return bTime - aTime
        }
        case 'status': {
          const score = (entry: CatalogEntry) => {
            if (entry.has_prd) return 3
            if (entry.has_draft) return 2
            return 1
          }
          return score(b) - score(a) || a.name.localeCompare(b.name)
        }
        case 'name_asc':
        default:
          return a.name.localeCompare(b.name)
      }
    })
  }, [entries, search, typeFilter, sortMode, selectedLabels])

  const stats = useMemo(
    () => [
      {
        key: 'total',
        label: 'Total entities',
        value: catalogCounts.total,
        description: 'Scenarios & resources in scope',
      },
      {
        key: 'with-prd',
        label: 'With PRD',
        value: catalogCounts.withPrd,
        description: `${coverage}% coverage`,
      },
      {
        key: 'drafts',
        label: 'Active drafts',
        value: catalogCounts.drafts,
        description: 'Awaiting review',
      },
      {
        key: 'missing',
        label: 'Missing PRDs',
        value: catalogCounts.missing,
        description: `${missingPercentage}% need attention`,
      },
    ],
    [catalogCounts, coverage, missingPercentage],
  )

  const requirementCompletion = requirementTotals.total
    ? Math.round((requirementTotals.completed / requirementTotals.total) * 100)
    : 0

  const requirementCards = useMemo(
    () => [
      {
        key: 'requirements-total',
        label: 'Tracked requirements',
        value: requirementTotals.total,
        description: 'Across all catalog entries',
      },
      {
        key: 'requirements-complete',
        label: 'Completed',
        value: requirementTotals.completed,
        description: `${requirementCompletion}% of total`,
      },
      {
        key: 'requirements-progress',
        label: 'In progress',
        value: requirementTotals.inProgress,
        description: `${requirementTotals.pending} pending`,
      },
      {
        key: 'requirements-criticality',
        label: 'Criticality split',
        value: requirementTotals.p0,
        description: `P0 ${requirementTotals.p0} · P1 ${requirementTotals.p1} · P2 ${requirementTotals.p2}`,
      },
    ],
    [requirementTotals, requirementCompletion],
  )

  if (loading) {
    return (
      <div className="app-container space-y-6">
        <TopNav />
        <CatalogSkeleton />
      </div>
    )
  }

  if (error) {
    return (
      <div className="app-container">
        <Card className="border-amber-200 bg-amber-50">
          <CardContent className="flex items-start gap-3 text-amber-900">
            <AlertTriangle size={20} className="mt-1" />
            <div>
              <p className="font-semibold">Error loading catalog</p>
              <p>{error}</p>
              <Button variant="ghost" className="mt-3" onClick={fetchCatalog}>
                Try again
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="app-container">
      <TopNav />

      {/* Floating Action Button - Quick Add Idea */}
      <button
        onClick={() => setQuickAddOpen(true)}
        className="group fixed bottom-6 right-6 sm:bottom-8 sm:right-8 z-50 flex h-14 w-14 sm:h-16 sm:w-16 items-center justify-center rounded-full bg-gradient-to-br from-amber-500 to-amber-600 text-white shadow-xl transition-all hover:scale-110 hover:shadow-2xl focus:outline-none focus:ring-4 focus:ring-amber-300 focus:ring-offset-2 active:scale-95"
        title="Quick add idea to backlog"
        aria-label="Quick add idea to backlog"
      >
        <Sparkles size={24} strokeWidth={2.5} className="sm:w-7 sm:h-7 group-hover:rotate-12 transition-transform duration-200" />
      </button>

      <QuickAddIdeaDialog open={quickAddOpen} onOpenChange={setQuickAddOpen} />

      <header className="rounded-3xl border bg-white/90 p-4 sm:p-6 shadow-soft-lg">
        <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div className="space-y-3 flex-1">
            <div className="flex items-center justify-between gap-3">
              <span className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-500">Product operations</span>
              <div className="hidden lg:flex items-center gap-2 text-[11px] text-slate-500">
                <kbd className="inline-flex h-5 items-center gap-1 rounded border border-slate-200 bg-slate-50 px-1.5 font-mono font-semibold">⌘K</kbd>
                <span>to search</span>
                <span className="text-slate-300 mx-1">·</span>
                <kbd className="inline-flex h-5 items-center gap-1 rounded border border-slate-200 bg-slate-50 px-1.5 font-mono font-semibold">Esc</kbd>
                <span>to clear</span>
              </div>
            </div>
            <div className="flex items-center gap-3 text-2xl sm:text-3xl font-semibold text-slate-900">
              <span className="rounded-2xl bg-violet-100 p-2 sm:p-3 text-violet-600">
                <ClipboardList size={24} strokeWidth={2.5} className="sm:w-7 sm:h-7" />
              </span>
              PRD Control Tower
            </div>
            <p className="max-w-3xl text-sm sm:text-base text-muted-foreground">
              Centralized coverage for {catalogCounts.total.toLocaleString()} tracked entities. {coverage}% already ship with a published PRD.
            </p>
          </div>
          <div className="flex flex-col sm:flex-row gap-2 sm:gap-3 w-full sm:w-auto">
            <Button variant="secondary" size="lg" asChild className="w-full sm:w-auto">
              <Link to="/backlog" className="flex items-center justify-center gap-2">
                <StickyNote size={18} />
                <span className="hidden sm:inline">Capture backlog</span>
                <span className="sm:hidden">Backlog</span>
              </Link>
            </Button>
            <Button size="lg" asChild className="w-full sm:w-auto">
              <Link to="/drafts" className="flex items-center justify-center gap-2">
                <Layers size={18} />
                View drafts ({catalogCounts.drafts})
              </Link>
            </Button>
          </div>
        </div>
        <Separator className="my-6" />
        <div className="flex items-start gap-3 rounded-2xl border bg-slate-50/80 p-4 text-sm">
          <span className="rounded-full bg-violet-100 p-2 text-violet-700">
            <Sparkles size={18} />
          </span>
          <div>
            <p className="font-medium text-slate-900">Quality pulse</p>
            <p className="text-muted-foreground">
              {catalogCounts.withPrd.toLocaleString()} PRDs published · {catalogCounts.missing.toLocaleString()} gaps remain ({missingPercentage}% of catalog)
            </p>
          </div>
        </div>
      </header>

      <section className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {stats.map(stat => (
          <Card key={stat.key} className="bg-white/90">
            <CardHeader className="pb-2">
              <CardDescription>{stat.label}</CardDescription>
              <CardTitle className="text-3xl font-bold">{stat.value.toLocaleString()}</CardTitle>
            </CardHeader>
            <CardContent className="pt-0 text-sm text-muted-foreground">{stat.description}</CardContent>
          </Card>
        ))}
      </section>

      {requirementTotals.total > 0 && (
        <details className="group">
          <summary className="cursor-pointer rounded-2xl border-2 border-violet-100 bg-gradient-to-br from-white to-violet-50/30 p-4 shadow-sm hover:shadow-md transition-all hover:border-violet-200 list-none">
            <div className="flex items-center justify-between gap-3">
              <div className="flex items-center gap-3">
                <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-violet-100 text-violet-600 group-open:bg-violet-500 group-open:text-white transition-colors shrink-0">
                  <Target size={20} />
                </span>
                <div>
                  <h3 className="text-sm font-semibold text-slate-900">Requirements Overview</h3>
                  <p className="text-xs text-slate-600">
                    {requirementTotals.total} total · {requirementCompletion}% complete
                  </p>
                </div>
              </div>
              <div className="flex items-center gap-2 text-xs font-semibold text-violet-600">
                <span className="hidden sm:inline">View details</span>
                <span className="group-open:rotate-180 transition-transform text-base">▼</span>
              </div>
            </div>
          </summary>
          <section className="grid gap-4 md:grid-cols-2 lg:grid-cols-4 mt-4">
            {requirementCards.map((stat) => (
              <Card key={stat.key} className="border border-violet-100 bg-gradient-to-br from-white to-violet-50/60">
                <CardHeader className="pb-2">
                  <CardDescription>{stat.label}</CardDescription>
                  <CardTitle className="text-3xl font-bold">{stat.value.toLocaleString()}</CardTitle>
                </CardHeader>
                <CardContent className="pt-0 text-sm text-muted-foreground">{stat.description}</CardContent>
              </Card>
            ))}
          </section>
        </details>
      )}

      <section className="rounded-3xl border-2 bg-white p-4 sm:p-6 shadow-md">
        <div className="space-y-5">
          <div className="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
            <div className="relative w-full xl:max-w-2xl">
              <Search className="pointer-events-none absolute left-4 top-1/2 -translate-y-1/2 text-slate-400" size={20} />
              <Input
                ref={searchInputRef}
                data-testid={selectors.catalog.searchInput}
                value={search}
                onChange={event => setSearch(event.target.value)}
                placeholder="Search by name or description..."
                className="h-12 sm:h-14 border-2 pl-12 pr-20 sm:pr-24 text-sm sm:text-base shadow-sm transition focus:border-violet-400 focus:ring-4 focus:ring-violet-100"
                aria-label="Search catalog"
              />
              {!search && (
                <div className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 flex items-center gap-2">
                  <kbd className="hidden sm:inline-flex h-7 items-center gap-1 rounded border border-slate-200 bg-slate-50 px-2 text-[11px] font-semibold text-slate-600 shadow-sm">
                    <span className="text-xs">⌘</span>K
                  </kbd>
                </div>
              )}
              {search && (
                <button
                  type="button"
                  className="absolute right-2 sm:right-3 top-1/2 -translate-y-1/2 rounded-lg bg-slate-100 px-3 sm:px-4 py-1.5 sm:py-2 text-xs sm:text-sm font-medium text-slate-600 hover:bg-slate-200 hover:text-slate-800 active:scale-95 transition-all"
                  onClick={() => {
                    setSearch('')
                    searchInputRef.current?.focus()
                  }}
                  aria-label="Clear search"
                >
                  Clear
                </button>
              )}
            </div>
            <div className="flex flex-col sm:flex-row gap-3 w-full xl:w-auto">
              <div className="flex flex-col gap-1.5 w-full sm:w-auto">
                <label htmlFor="catalog-sort" className="flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wider text-slate-500">
                  Sort by
                  <HelpTooltip
                    content="Choose how to order catalog entries. 'Status' shows PRDs first, 'Coverage' sorts by requirement completion percentage."
                    side="right"
                  />
                </label>
                <select
                  id="catalog-sort"
                  value={sortMode}
                  onChange={(event) => setSortMode(event.target.value as CatalogSort)}
                  className="h-12 sm:h-auto rounded-xl border-2 border-slate-200 bg-white px-4 py-2.5 text-sm font-medium shadow-sm transition focus:border-violet-400 focus:ring-2 focus:ring-violet-100 w-full sm:w-auto min-w-0 sm:min-w-[200px]"
                >
                  <option value="status">Status (PRD first)</option>
                  <option value="recently_viewed">Recently viewed</option>
                  <option value="coverage">Requirements coverage</option>
                  <option value="name_asc">Name A → Z</option>
                  <option value="name_desc">Name Z → A</option>
                </select>
              </div>
              {availableLabels.length > 0 && (
                <div className="flex flex-col gap-1.5 w-full sm:w-auto">
                  <label className="flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wider text-slate-500">
                    <Tag size={14} />
                    Filter by labels
                    <HelpTooltip
                      content="Click labels to filter catalog entries. Multiple labels require ALL to match."
                      side="right"
                    />
                  </label>
                  <div className="flex flex-wrap gap-2 items-center min-h-[48px] sm:min-h-[40px]">
                    {availableLabels.map(label => (
                      <Badge
                        key={label}
                        variant={selectedLabels.includes(label) ? "default" : "outline"}
                        className="cursor-pointer hover:scale-105 transition-transform"
                        onClick={() => {
                          setSelectedLabels(prev =>
                            prev.includes(label)
                              ? prev.filter(l => l !== label)
                              : [...prev, label]
                          )
                        }}
                      >
                        {label}
                      </Badge>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>

          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <Tabs value={typeFilter} onValueChange={(value: string) => setTypeFilter(value as CatalogFilter)} className="w-full sm:w-auto">
              <TabsList className="h-10 w-full sm:w-auto grid grid-cols-3 sm:inline-flex">
                <TabsTrigger value="all" className="px-3 sm:px-4 text-xs sm:text-sm">All ({catalogCounts.total})</TabsTrigger>
                <TabsTrigger value="scenario" className="px-3 sm:px-4 text-xs sm:text-sm">Scenarios ({catalogCounts.scenarios})</TabsTrigger>
                <TabsTrigger value="resource" className="px-3 sm:px-4 text-xs sm:text-sm">Resources ({catalogCounts.resources})</TabsTrigger>
              </TabsList>
            </Tabs>
            {search && (
              <span className="text-xs sm:text-sm text-slate-600 px-1">
                Searching in: <span className="font-medium">{typeFilter === 'all' ? 'All types' : `${typeFilter}s only`}</span>
              </span>
            )}
          </div>

          {/* Active filters chips */}
          {(search || typeFilter !== 'all' || selectedLabels.length > 0) && (
            <div className="flex flex-wrap items-center gap-2 pt-1">
              <span className="text-xs font-semibold uppercase tracking-wider text-slate-500">Active filters:</span>
              {search && (
                <button
                  onClick={() => {
                    setSearch('')
                    searchInputRef.current?.focus()
                  }}
                  className="inline-flex items-center gap-1.5 rounded-full bg-violet-100 pl-3 pr-2 py-1.5 text-xs font-medium text-violet-700 hover:bg-violet-200 active:scale-95 transition-all group"
                  aria-label={`Remove search filter: ${search}`}
                >
                  <Search size={12} className="shrink-0" />
                  <span className="max-w-[150px] truncate">{search}</span>
                  <span className="ml-1 text-violet-500 group-hover:text-violet-700 font-bold">×</span>
                </button>
              )}
              {typeFilter !== 'all' && (
                <button
                  onClick={() => setTypeFilter('all')}
                  className="inline-flex items-center gap-1.5 rounded-full bg-blue-100 pl-3 pr-2 py-1.5 text-xs font-medium text-blue-700 hover:bg-blue-200 active:scale-95 transition-all group capitalize"
                  aria-label={`Remove type filter: ${typeFilter}`}
                >
                  <Layers size={12} className="shrink-0" />
                  {typeFilter}
                  <span className="ml-1 text-blue-500 group-hover:text-blue-700 font-bold">×</span>
                </button>
              )}
              {selectedLabels.map(label => (
                <button
                  key={label}
                  onClick={() => setSelectedLabels(prev => prev.filter(l => l !== label))}
                  className="inline-flex items-center gap-1.5 rounded-full bg-green-100 pl-3 pr-2 py-1.5 text-xs font-medium text-green-700 hover:bg-green-200 active:scale-95 transition-all group"
                  aria-label={`Remove label filter: ${label}`}
                >
                  <Tag size={12} className="shrink-0" />
                  {label}
                  <span className="ml-1 text-green-500 group-hover:text-green-700 font-bold">×</span>
                </button>
              ))}
              {(search || typeFilter !== 'all' || selectedLabels.length > 0) && (
                <button
                  onClick={() => {
                    setSearch('')
                    setTypeFilter('all')
                    setSelectedLabels([])
                    searchInputRef.current?.focus()
                  }}
                  className="text-xs font-medium text-slate-500 hover:text-slate-700 underline underline-offset-2 ml-1"
                >
                  Clear all
                </button>
              )}
            </div>
          )}

          {/* Search results feedback */}
          {search && (
            <div className={cn(
              "rounded-lg border px-3 py-2 flex items-center gap-2 text-sm transition-colors",
              filteredEntries.length > 0
                ? "bg-emerald-50 border-emerald-200"
                : "bg-amber-50 border-amber-200"
            )}>
              <span className={cn(
                "font-semibold",
                filteredEntries.length > 0 ? "text-emerald-700" : "text-amber-700"
              )}>
                {filteredEntries.length}
              </span>
              <span className="text-slate-700">
                {filteredEntries.length === 1 ? 'result' : 'results'} found
              </span>
              {filteredEntries.length === 0 && entries.length > 0 && (
                <span className="text-slate-600 ml-auto hidden sm:inline">· Try adjusting your filters</span>
              )}
              {filteredEntries.length > 0 && (
                <span className="text-emerald-600 ml-auto">✓</span>
              )}
            </div>
          )}
        </div>
      </section>

      {filteredEntries.length === 0 ? (
        <Card className="border-2 border-dashed border-slate-200 bg-gradient-to-br from-white via-slate-50/30 to-white">
          <CardContent className="flex flex-col items-center gap-6 py-12 sm:py-16 text-center px-4 sm:px-6">
            <div className="relative">
              <div className="absolute inset-0 animate-pulse rounded-full bg-violet-100 opacity-75 blur-xl" />
              <div className="relative rounded-full bg-gradient-to-br from-slate-100 to-slate-200 p-5 sm:p-6 text-slate-400 shadow-inner">
                <Layers size={48} strokeWidth={1.5} className="sm:w-14 sm:h-14" />
              </div>
            </div>
            <div className="space-y-3 max-w-lg">
              <h3 className="text-lg sm:text-xl font-bold text-slate-900">
                {entries.length === 0 ? 'No Entities Found' : 'No Matching Results'}
              </h3>
              <p className="text-sm sm:text-base text-slate-600 leading-relaxed">
                {entries.length === 0
                  ? 'The catalog appears empty. This may indicate a scanning issue or fresh installation.'
                  : `No entities match "${search}"${typeFilter !== 'all' ? ` in ${typeFilter}s` : ''}. Try different search terms or filters.`}
              </p>
              {entries.length === 0 && (
                <div className="mt-6 rounded-xl border border-violet-100 bg-gradient-to-br from-violet-50/50 to-white p-4 sm:p-5 text-left space-y-3">
                  <div className="flex items-center gap-2">
                    <Sparkles size={16} className="text-violet-600 shrink-0" />
                    <p className="text-xs font-semibold uppercase tracking-wider text-violet-700">Troubleshooting Steps</p>
                  </div>
                  <ol className="space-y-2 text-xs sm:text-sm text-slate-700 pl-1">
                    <li className="flex items-start gap-2">
                      <span className="font-bold text-violet-600 shrink-0">1.</span>
                      <span>Verify the API is running at <code className="bg-slate-100 px-1.5 py-0.5 rounded text-[11px] font-mono">http://localhost:18600</code></span>
                    </li>
                    <li className="flex items-start gap-2">
                      <span className="font-bold text-violet-600 shrink-0">2.</span>
                      <span>Check that scenarios exist in the <code className="bg-slate-100 px-1.5 py-0.5 rounded text-[11px] font-mono">scenarios/</code> directory</span>
                    </li>
                    <li className="flex items-start gap-2">
                      <span className="font-bold text-violet-600 shrink-0">3.</span>
                      <span>Click <strong>Refresh Catalog</strong> below to re-scan the filesystem</span>
                    </li>
                  </ol>
                </div>
              )}
            </div>
            <div className="flex flex-wrap gap-2 sm:gap-3 justify-center">
              {search && (
                <Button variant="outline" size="lg" onClick={() => setSearch('')} className="gap-2 min-h-[44px]">
                  <Search size={16} />
                  <span className="text-sm">Clear search</span>
                </Button>
              )}
              {typeFilter !== 'all' && (
                <Button variant="outline" size="lg" onClick={() => setTypeFilter('all')} className="gap-2 min-h-[44px]">
                  <Layers size={16} />
                  <span className="text-sm">Show all types</span>
                </Button>
              )}
              {entries.length === 0 && (
                <Button variant="default" size="lg" onClick={fetchCatalog} className="gap-2 shadow-md min-h-[44px]">
                  <RefreshCw size={16} />
                  <span className="text-sm">Refresh Catalog</span>
                </Button>
              )}
            </div>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          {filteredEntries.map(entry => (
            <CatalogCard
              key={`${entry.type}:${entry.name}`}
              entry={entry}
              navigate={navigate}
              prepareDraft={prepareDraft}
              preparingId={preparingId}
              onVisit={recordVisit}
              onLabelsUpdate={handleLabelsUpdate}
            />
          ))}
        </div>
      )}
    </div>
  )
}

function CatalogSkeleton() {
  return (
    <div className="space-y-6 animate-in fade-in duration-300">
      {/* Header skeleton */}
      <header className="rounded-3xl border bg-white/90 p-4 sm:p-6 shadow-soft-lg">
        <div className="space-y-4">
          <Skeleton className="h-3 w-40 animate-pulse" />
          <div className="flex items-center gap-3">
            <Skeleton className="h-12 w-12 rounded-2xl animate-pulse" />
            <Skeleton className="h-8 w-72 animate-pulse" />
          </div>
          <Skeleton className="h-4 w-full max-w-3xl animate-pulse" />
          <div className="flex flex-col sm:flex-row gap-2 sm:gap-3 pt-2">
            <Skeleton className="h-11 w-full sm:w-40 animate-pulse" />
            <Skeleton className="h-11 w-full sm:w-40 animate-pulse" />
          </div>
        </div>
        <Separator className="my-6" />
        <div className="flex items-start gap-3 rounded-2xl border bg-slate-50/80 p-4">
          <Skeleton className="h-10 w-10 rounded-full animate-pulse" />
          <div className="flex-1 space-y-2">
            <Skeleton className="h-4 w-32 animate-pulse" />
            <Skeleton className="h-4 w-full max-w-md animate-pulse" />
          </div>
        </div>
      </header>

      {/* Stats skeleton */}
      <section className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {Array.from({ length: 4 }).map((_, idx) => (
          <Card key={`stat-${idx}`} className="bg-white/90 overflow-hidden">
            <CardHeader className="pb-2">
              <Skeleton className="h-4 w-32 mb-2 animate-pulse" style={{ animationDelay: `${idx * 75}ms` }} />
              <Skeleton className="h-9 w-20 animate-pulse" style={{ animationDelay: `${idx * 75}ms` }} />
            </CardHeader>
            <CardContent className="pt-0">
              <Skeleton className="h-4 w-full animate-pulse" style={{ animationDelay: `${idx * 75}ms` }} />
            </CardContent>
          </Card>
        ))}
      </section>

      {/* Search and filters skeleton */}
      <section className="rounded-3xl border-2 bg-white p-4 sm:p-6 shadow-md">
        <div className="space-y-5">
          <div className="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
            <Skeleton className="h-12 sm:h-14 w-full xl:max-w-2xl rounded-xl animate-pulse" />
            <Skeleton className="h-12 sm:h-auto w-full sm:w-auto sm:min-w-[200px] rounded-lg animate-pulse" />
          </div>
          <div className="grid gap-3 sm:grid-cols-3">
            <Skeleton className="h-10 rounded-full animate-pulse" />
            <Skeleton className="h-10 rounded-full animate-pulse" style={{ animationDelay: '100ms' }} />
            <Skeleton className="h-10 rounded-full animate-pulse" style={{ animationDelay: '200ms' }} />
          </div>
          <Skeleton className="h-5 w-48 animate-pulse" />
          <div className="grid gap-4 md:grid-cols-2">
            {Array.from({ length: 6 }).map((_, idx) => (
              <CardSkeleton key={`card-${idx}`} style={{ animationDelay: `${idx * 50}ms` }} />
            ))}
          </div>
        </div>
      </section>
    </div>
  )
}
