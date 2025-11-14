import { useParams, Link } from 'react-router-dom'
import { Search, Loader2, AlertTriangle, Target, Filter } from 'lucide-react'
import { useOperationalTargets } from '../hooks/useOperationalTargets'
import { TargetsList } from '../components/targets/TargetsList'
import { TargetDetailPanel } from '../components/targets/TargetDetailPanel'
import { Input } from '../components/ui/input'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '../components/ui/card'
import { Button } from '../components/ui/button'
import { Badge } from '../components/ui/badge'
import { TopNav } from '../components/ui/top-nav'
import { Breadcrumbs } from '../components/ui/breadcrumbs'

export default function OperationalTargetsExplorer() {
  const { entityType, entityName } = useParams<{ entityType?: string; entityName?: string }>()

  const {
    targets,
    unmatchedRequirements,
    loading,
    error,
    selectedTarget,
    filter,
    setFilter,
    categoryFilter,
    setCategoryFilter,
    statusFilter,
    setStatusFilter,
    filteredTargets,
    handleSelectTarget,
    refresh,
  } = useOperationalTargets({ entityType, entityName })

  if (loading && targets.length === 0) {
    return (
      <div className="app-container">
        <Card className="border-dashed bg-white/80">
          <CardContent className="flex items-center gap-3 py-8 text-muted-foreground">
            <Loader2 size={20} className="animate-spin" /> Loading operational targets...
          </CardContent>
        </Card>
      </div>
    )
  }

  if (error) {
    return (
      <div className="app-container">
        <Card className="border-amber-200 bg-amber-50">
          <CardContent className="flex items-center gap-3 py-8 text-amber-900">
            <AlertTriangle size={20} />
            <div>
              <p className="font-medium">Error loading operational targets</p>
              <p className="text-sm">{error}</p>
            </div>
            <Button onClick={refresh} variant="outline" size="sm" className="ml-auto">
              Retry
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  const completedTargets = targets.filter((t) => t.status === 'complete').length
  const targetsByCategory = targets.reduce(
    (acc, target) => {
      const cat = target.category || 'Uncategorized'
      acc[cat] = (acc[cat] || 0) + 1
      return acc
    },
    {} as Record<string, number>,
  )

  const categories = ['all', ...Object.keys(targetsByCategory).sort()]

  const breadcrumbItems = [
    { label: 'Catalog', to: '/' },
    { label: `${entityType}/${entityName}`, to: `/prd/${entityType}/${entityName}` },
    { label: 'Operational Targets' },
  ]

  return (
    <div className="app-container space-y-6">
      <TopNav />
      <Breadcrumbs items={breadcrumbItems} />
      <header className="rounded-3xl border bg-white/90 p-6 shadow-soft-lg">
        <div className="space-y-2">
          <span className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-500">Operational Targets</span>
          <div className="flex items-center gap-3 text-3xl font-semibold text-slate-900">
            <span className="rounded-2xl bg-blue-100 p-3 text-blue-600">
              <Target size={28} strokeWidth={2.5} />
            </span>
            Targets Explorer
          </div>
          <p className="max-w-3xl text-base text-muted-foreground">
            Track progress on operational targets for{' '}
            <span className="font-medium text-slate-700">
              {entityType}/{entityName}
            </span>
          </p>
        </div>

        <div className="mt-6 grid gap-4 sm:grid-cols-4">
          <Card>
            <CardContent className="pt-4">
              <p className="text-sm text-muted-foreground">Total Targets</p>
              <p className="text-3xl font-bold text-slate-900">{targets.length}</p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-4">
              <p className="text-sm text-muted-foreground">Completed</p>
              <p className="text-3xl font-bold text-green-600">{completedTargets}</p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-4">
              <p className="text-sm text-muted-foreground">Pending</p>
              <p className="text-3xl font-bold text-amber-600">{targets.length - completedTargets}</p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-4">
              <p className="text-sm text-muted-foreground">Completion</p>
              <p className="text-3xl font-bold text-blue-600">{targets.length > 0 ? Math.round((completedTargets / targets.length) * 100) : 0}%</p>
            </CardContent>
          </Card>
        </div>
      </header>

      {unmatchedRequirements.length > 0 && (
        <Card className="border-amber-300 bg-amber-50">
          <CardHeader>
            <CardTitle className="text-lg text-amber-900">⚠️ Unmatched Requirements ({unmatchedRequirements.length})</CardTitle>
            <CardDescription className="text-amber-700">
              These requirements have no linked operational targets. Update their prd_ref fields to link them.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2 max-h-40 overflow-y-auto">
              {unmatchedRequirements.map((req) => (
                <div key={req.id} className="flex items-center gap-2 text-sm">
                  <Badge variant="outline" className="font-mono">
                    {req.id}
                  </Badge>
                  <span className="text-amber-900">{req.title}</span>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      <div className="space-y-3">
        <div className="relative max-w-xl">
          <Search className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" size={18} />
          <Input
            id="targets-search"
            type="text"
            className="pl-9"
            placeholder="Search targets by title, ID, or notes..."
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
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

        <div className="flex flex-wrap gap-2 items-center">
          <Filter size={16} className="text-slate-500" />
          <span className="text-sm text-slate-600">Category:</span>
          {categories.map((cat) => (
            <Button
              key={cat}
              variant={categoryFilter === cat ? 'default' : 'outline'}
              size="sm"
              onClick={() => setCategoryFilter(cat)}
              className="capitalize"
            >
              {cat}
              {cat !== 'all' && targetsByCategory[cat] && ` (${targetsByCategory[cat]})`}
            </Button>
          ))}
        </div>

        <div className="flex flex-wrap gap-2 items-center">
          <Filter size={16} className="text-slate-500" />
          <span className="text-sm text-slate-600">Status:</span>
          <Button variant={statusFilter === 'all' ? 'default' : 'outline'} size="sm" onClick={() => setStatusFilter('all')}>
            All ({targets.length})
          </Button>
          <Button variant={statusFilter === 'complete' ? 'default' : 'outline'} size="sm" onClick={() => setStatusFilter('complete')}>
            Complete ({completedTargets})
          </Button>
          <Button variant={statusFilter === 'pending' ? 'default' : 'outline'} size="sm" onClick={() => setStatusFilter('pending')}>
            Pending ({targets.length - completedTargets})
          </Button>
        </div>
      </div>

      <div className="grid gap-6 lg:grid-cols-[450px,1fr]">
        <Card>
          <CardContent className="p-4">
            <TargetsList targets={filteredTargets} selectedId={selectedTarget?.id ?? null} onSelect={handleSelectTarget} />
          </CardContent>
        </Card>

        <TargetDetailPanel target={selectedTarget} />
      </div>

      <div className="flex flex-wrap gap-4 text-sm">
        <Link to={`/prd/${entityType}/${entityName}`} className="text-primary hover:underline">
          ← View PRD
        </Link>
        <Link to={`/requirements/${entityType}/${entityName}`} className="text-primary hover:underline">
          → View Requirements
        </Link>
        <Link to="/" className="text-primary hover:underline">
          ← Back to Catalog
        </Link>
      </div>
    </div>
  )
}
