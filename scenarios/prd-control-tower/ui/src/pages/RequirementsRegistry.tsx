import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { Search, Loader2, AlertTriangle, ListTree, ExternalLink, CheckCircle2, Clock, XCircle, RefreshCw, BookOpen } from 'lucide-react'
import { Input } from '../components/ui/input'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../components/ui/card'
import { Badge } from '../components/ui/badge'
import { Button } from '../components/ui/button'
import { TopNav } from '../components/ui/top-nav'
import { buildApiUrl } from '../utils/apiClient'
import type { CatalogEntry, CatalogResponse, RequirementGroup, RequirementsResponse } from '../types'
import { selectors } from '../consts/selectors'

interface RequirementsSummary {
  entityType: string
  entityName: string
  total: number
  completed: number
  inProgress: number
  pending: number
  p0Count: number
  p1Count: number
  p2Count: number
}

export default function RequirementsRegistry() {
  const [entries, setEntries] = useState<CatalogEntry[]>([])
  const [summaries, setSummaries] = useState<Map<string, RequirementsSummary>>(new Map())
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [search, setSearch] = useState('')
  const [loadingSummaries, setLoadingSummaries] = useState(false)

  const fetchCatalog = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const response = await fetch(buildApiUrl('/catalog'))
      if (!response.ok) {
        throw new Error(`Failed to fetch catalog: ${response.statusText}`)
      }
      const data: CatalogResponse = await response.json()
      const withRequirements = data.entries.filter(entry => entry.has_requirements)
      setEntries(withRequirements)
      setLoading(false)

      // Fetch summaries for entries with requirements
      if (withRequirements.length > 0) {
        fetchSummaries(withRequirements)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
      setLoading(false)
    }
  }, [])

  const fetchSummaries = async (catalogEntries: CatalogEntry[]) => {
    setLoadingSummaries(true)
    const newSummaries = new Map<string, RequirementsSummary>()

    for (const entry of catalogEntries) {
      try {
        const response = await fetch(buildApiUrl(`/catalog/${entry.type}/${entry.name}/requirements`))
        if (!response.ok) continue

        const data: RequirementsResponse = await response.json()

        // Calculate statistics from requirements groups
        let total = 0
        let completed = 0
        let inProgress = 0
        let pending = 0
        let p0Count = 0
        let p1Count = 0
        let p2Count = 0

        const countRequirements = (groups: RequirementGroup[]) => {
          for (const group of groups) {
            for (const req of group.requirements || []) {
              total++
              if (req.status === 'complete') completed++
              else if (req.status === 'in_progress') inProgress++
              else pending++

              if (req.criticality === 'P0') p0Count++
              else if (req.criticality === 'P1') p1Count++
              else if (req.criticality === 'P2') p2Count++
            }
            if (group.children) {
              countRequirements(group.children)
            }
          }
        }

        countRequirements(data.groups || [])

        newSummaries.set(`${entry.type}:${entry.name}`, {
          entityType: entry.type,
          entityName: entry.name,
          total,
          completed,
          inProgress,
          pending,
          p0Count,
          p1Count,
          p2Count,
        })
      } catch (err) {
        // Skip entries that fail to load
        continue
      }
    }

    setSummaries(newSummaries)
    setLoadingSummaries(false)
  }

  useEffect(() => {
    fetchCatalog()
  }, [fetchCatalog])

  const filteredEntries = useMemo(() => {
    if (!search) return entries
    const searchLower = search.toLowerCase()
    return entries.filter(
      entry =>
        entry.name.toLowerCase().includes(searchLower) ||
        entry.description?.toLowerCase().includes(searchLower) ||
        entry.type.toLowerCase().includes(searchLower)
    )
  }, [entries, search])

  const totalStats = useMemo(() => {
    let total = 0
    let completed = 0
    let inProgress = 0
    let pending = 0
    let p0 = 0
    let p1 = 0
    let p2 = 0

    summaries.forEach(summary => {
      total += summary.total
      completed += summary.completed
      inProgress += summary.inProgress
      pending += summary.pending
      p0 += summary.p0Count
      p1 += summary.p1Count
      p2 += summary.p2Count
    })

    return { total, completed, inProgress, pending, p0, p1, p2 }
  }, [summaries])

  if (loading) {
    return (
      <div className="app-container" data-layout="dual">
        <TopNav />
        <Card className="border-2 border-dashed bg-gradient-to-br from-white via-violet-50/10 to-slate-50/30">
          <CardContent className="flex flex-col items-center gap-4 py-16 text-center">
            <div className="rounded-full bg-violet-100 p-4 shadow-inner">
              <Loader2 size={36} className="animate-spin text-violet-600" />
            </div>
            <div className="space-y-1">
              <p className="text-lg font-semibold text-slate-900">Loading Requirements Registry</p>
              <p className="text-sm text-slate-600">Scanning scenarios and resources for requirements...</p>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (error) {
    return (
      <div className="app-container" data-layout="dual">
        <TopNav />
        <Card className="border-2 border-amber-200 bg-gradient-to-br from-amber-50 via-white to-amber-50/30">
          <CardContent className="flex flex-col items-center gap-4 py-16 text-center">
            <div className="rounded-full bg-amber-100 p-4 shadow-inner">
              <AlertTriangle size={36} className="text-amber-600" />
            </div>
            <div className="space-y-2 max-w-md">
              <p className="text-lg font-semibold text-amber-900">Failed to Load Registry</p>
              <p className="text-sm text-amber-700">{error}</p>
              <p className="text-xs text-slate-600">Check your API connection and try again.</p>
            </div>
            <Button onClick={fetchCatalog} size="lg" className="mt-2 h-12 shadow-md hover:shadow-lg">
              <RefreshCw size={18} className="mr-2" />
              Retry Loading
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="app-container space-y-6" data-layout="dual">
      <TopNav data-testid={selectors.requirements.registryView} />

      <header className="rounded-3xl border bg-white/90 p-6 shadow-soft-lg">
        <div className="space-y-2">
          <span className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-500">Global View</span>
          <div className="flex items-center gap-3 text-3xl font-semibold text-slate-900">
            <span className="rounded-2xl bg-purple-100 p-3 text-purple-600">
              <ListTree size={28} strokeWidth={2.5} />
            </span>
            Requirements Registry
          </div>
          <p className="max-w-3xl text-base text-muted-foreground">
            Browse requirements across all scenarios and resources. Track progress, identify gaps, and ensure comprehensive coverage.
          </p>
        </div>

        <div className="mt-6 grid gap-4 sm:grid-cols-4">
          <Card>
            <CardContent className="pt-4">
              <p className="text-sm text-muted-foreground">Total Requirements</p>
              <p className="text-3xl font-bold text-slate-900">{totalStats.total}</p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-4">
              <p className="text-sm text-muted-foreground">Completed</p>
              <p className="text-3xl font-bold text-green-600">{totalStats.completed}</p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-4">
              <p className="text-sm text-muted-foreground">In Progress</p>
              <p className="text-3xl font-bold text-blue-600">{totalStats.inProgress}</p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-4">
              <p className="text-sm text-muted-foreground">Completion Rate</p>
              <p className="text-3xl font-bold text-violet-600">
                {totalStats.total > 0 ? Math.round((totalStats.completed / totalStats.total) * 100) : 0}%
              </p>
            </CardContent>
          </Card>
        </div>

        <div className="mt-4 flex flex-wrap gap-3 text-sm">
          <div className="flex items-center gap-2">
            <Badge variant="destructive">P0: {totalStats.p0}</Badge>
            <Badge variant="default">P1: {totalStats.p1}</Badge>
            <Badge variant="secondary">P2: {totalStats.p2}</Badge>
          </div>
          {loadingSummaries && (
            <div className="flex items-center gap-2 text-muted-foreground">
              <Loader2 size={14} className="animate-spin" />
              <span className="text-xs">Loading detailed statistics...</span>
            </div>
          )}
        </div>
      </header>

      <div className="relative max-w-xl">
        <Search className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" size={18} />
        <Input
          type="text"
          className="pl-9"
          placeholder="Search by scenario/resource name, type, or description..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
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

      {filteredEntries.length === 0 ? (
        <Card className="border-2 border-dashed bg-gradient-to-br from-white via-slate-50/20 to-white">
          <CardContent className="flex flex-col items-center gap-4 py-16 text-center">
            <div className="rounded-full bg-slate-100 p-5 shadow-inner">
              <ListTree size={42} className="text-slate-400" />
            </div>
            <div className="space-y-2 max-w-md">
              <p className="text-lg font-semibold text-slate-900">
                {search ? 'No matching requirements' : 'No requirements found'}
              </p>
              <p className="text-sm text-slate-600 leading-relaxed">
                {search
                  ? 'Try adjusting your search terms or clear the filter to see all entries.'
                  : 'Scenarios and resources with requirements/ directories will appear here once they\'re created.'}
              </p>
            </div>
            {search && (
              <Button onClick={() => setSearch('')} variant="outline" size="lg" className="mt-2 h-12">
                <XCircle size={18} className="mr-2" />
                Clear Search Filter
              </Button>
            )}
            {!search && (
              <Button variant="outline" size="lg" asChild className="mt-2 h-12">
                <Link to="/catalog">
                  <BookOpen size={18} className="mr-2" />
                  Browse Catalog
                </Link>
              </Button>
            )}
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {filteredEntries.map(entry => {
            const key = `${entry.type}:${entry.name}`
            const summary = summaries.get(key)
            const completionRate = summary && summary.total > 0
              ? Math.round((summary.completed / summary.total) * 100)
              : 0

            return (
              <Card key={key} data-testid={selectors.requirements.requirementCard} className="flex flex-col justify-between hover:border-violet-300 transition">
                <CardHeader>
                  <div className="flex items-start justify-between gap-2">
                    <CardTitle className="text-lg">{entry.name}</CardTitle>
                    <Badge variant="secondary" className="capitalize text-xs">
                      {entry.type}
                    </Badge>
                  </div>
                  <CardDescription className="line-clamp-2 text-sm">
                    {entry.description || 'No description'}
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-3">
                  {summary ? (
                    <>
                      <div className="flex items-center justify-between text-sm">
                        <span className="text-muted-foreground">Progress</span>
                        <span className="font-semibold">
                          {completionRate}%
                        </span>
                      </div>
                      <div className="h-2 w-full rounded-full bg-slate-100">
                        <div
                          className="h-full rounded-full bg-gradient-to-r from-green-500 to-emerald-500 transition-all"
                          style={{ width: `${completionRate}%` }}
                        />
                      </div>
                      <div className="grid grid-cols-3 gap-2 text-xs">
                        <div className="flex items-center gap-1">
                          <CheckCircle2 size={12} className="text-green-600" />
                          <span>{summary.completed}</span>
                        </div>
                        <div className="flex items-center gap-1">
                          <Clock size={12} className="text-blue-600" />
                          <span>{summary.inProgress}</span>
                        </div>
                        <div className="flex items-center gap-1">
                          <XCircle size={12} className="text-slate-400" />
                          <span>{summary.pending}</span>
                        </div>
                      </div>
                      <div className="flex flex-wrap gap-1.5">
                        {summary.p0Count > 0 && (
                          <Badge variant="destructive" className="text-xs">P0: {summary.p0Count}</Badge>
                        )}
                        {summary.p1Count > 0 && (
                          <Badge variant="default" className="text-xs">P1: {summary.p1Count}</Badge>
                        )}
                        {summary.p2Count > 0 && (
                          <Badge variant="secondary" className="text-xs">P2: {summary.p2Count}</Badge>
                        )}
                      </div>
                    </>
                  ) : (
                    <div className="text-xs text-muted-foreground">Loading statistics...</div>
                  )}
                  <Button variant="outline" size="sm" className="w-full" asChild>
                    <Link to={`/scenario/${entry.type}/${encodeURIComponent(entry.name)}?tab=requirements`}>
                      <ExternalLink size={14} className="mr-2" />
                      View Requirements
                    </Link>
                  </Button>
                </CardContent>
              </Card>
            )
          })}
        </div>
      )}
    </div>
  )
}
