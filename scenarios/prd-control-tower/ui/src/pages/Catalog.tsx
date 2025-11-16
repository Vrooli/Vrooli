import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import {
  AlertTriangle,
  ClipboardList,
  Layers,
  Search,
  Sparkles,
  StickyNote,
} from 'lucide-react'

import { Button } from '../components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../components/ui/card'
import { Input } from '../components/ui/input'
import { Separator } from '../components/ui/separator'
import { Tabs, TabsList, TabsTrigger } from '../components/ui/tabs'
import { TopNav } from '../components/ui/top-nav'
import { buildApiUrl } from '../utils/apiClient'
import { usePrepareDraft } from '../utils/useDraft'
import type { CatalogEntry, CatalogResponse } from '../types'
import { CatalogCard, QuickAddIdeaDialog } from '../components/catalog'

type CatalogFilter = 'all' | 'scenario' | 'resource'

export default function Catalog() {
  const [entries, setEntries] = useState<CatalogEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [search, setSearch] = useState('')
  const [typeFilter, setTypeFilter] = useState<CatalogFilter>('all')
  const [quickAddOpen, setQuickAddOpen] = useState(false)
  const navigate = useNavigate()

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
      const response = await fetch(buildApiUrl('/catalog'))
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

  useEffect(() => {
    fetchCatalog()
  }, [fetchCatalog])

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

  const filteredEntries = useMemo(() => {
    const query = search.trim().toLowerCase()
    return entries.filter(entry => {
      const matchesSearch =
        query === '' ||
        entry.name.toLowerCase().includes(query) ||
        entry.description.toLowerCase().includes(query)
      const matchesType = typeFilter === 'all' || entry.type === typeFilter
      return matchesSearch && matchesType
    })
  }, [entries, search, typeFilter])

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

  if (loading) {
    return (
      <div className="app-container">
        <Card className="border-dashed bg-white/70">
          <CardContent className="flex items-center gap-3 text-muted-foreground">
            <span className="flex h-10 w-10 items-center justify-center rounded-full bg-muted">
              <Layers size={18} />
            </span>
            Loading catalog...
          </CardContent>
        </Card>
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
        className="fixed bottom-8 right-8 z-50 flex h-14 w-14 items-center justify-center rounded-full bg-gradient-to-br from-amber-500 to-amber-600 text-white shadow-lg transition-all hover:scale-110 hover:shadow-xl focus:outline-none focus:ring-2 focus:ring-amber-400 focus:ring-offset-2"
        title="Quick add idea"
      >
        <Sparkles size={24} strokeWidth={2.5} />
      </button>

      <QuickAddIdeaDialog open={quickAddOpen} onOpenChange={setQuickAddOpen} />

      <header className="rounded-3xl border bg-white/90 p-6 shadow-soft-lg">
        <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div className="space-y-3">
            <span className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-500">Product operations</span>
            <div className="flex items-center gap-3 text-3xl font-semibold text-slate-900">
              <span className="rounded-2xl bg-violet-100 p-3 text-violet-600">
                <ClipboardList size={28} strokeWidth={2.5} />
              </span>
              PRD Control Tower
            </div>
            <p className="max-w-3xl text-base text-muted-foreground">
              Centralized coverage for {catalogCounts.total.toLocaleString()} tracked entities. {coverage}% already ship with a published PRD.
            </p>
          </div>
          <div className="flex flex-wrap gap-3">
            <Button variant="secondary" size="lg" asChild>
              <Link to="/backlog" className="flex items-center gap-2">
                <StickyNote size={18} />
                Capture backlog
              </Link>
            </Button>
            <Button size="lg" asChild>
              <Link to="/drafts" className="flex items-center gap-2">
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

      <section className="rounded-3xl border bg-white/90 p-5 shadow-sm">
        <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div className="relative w-full max-w-xl">
            <Search className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" size={18} />
            <Input
              value={search}
              onChange={event => setSearch(event.target.value)}
              placeholder="Search by name or description..."
              className="pl-9"
            />
            {search && (
              <button
                type="button"
                className="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-slate-500"
                onClick={() => setSearch('')}
              >
                Clear
              </button>
            )}
          </div>
          <Tabs value={typeFilter} onValueChange={(value: string) => setTypeFilter(value as CatalogFilter)}>
            <TabsList>
              <TabsTrigger value="all">All ({catalogCounts.total})</TabsTrigger>
              <TabsTrigger value="scenario">Scenarios ({catalogCounts.scenarios})</TabsTrigger>
              <TabsTrigger value="resource">Resources ({catalogCounts.resources})</TabsTrigger>
            </TabsList>
          </Tabs>
        </div>
      </section>

      {filteredEntries.length === 0 ? (
        <Card className="border-dashed bg-white/80">
          <CardContent className="flex flex-col items-center gap-3 py-10 text-center text-muted-foreground">
            <Layers size={40} />
            <p>No entities found matching your filters.</p>
            <Button variant="outline" onClick={() => setSearch('')}>
              Reset filters
            </Button>
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
            />
          ))}
        </div>
      )}
    </div>
  )
}
