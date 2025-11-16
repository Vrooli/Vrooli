import { useParams, Link } from 'react-router-dom'
import { Search, Loader2, AlertTriangle, ListTree } from 'lucide-react'
import { useRequirementsExplorer } from '../hooks/useRequirementsExplorer'
import { RequirementTree } from '../components/requirements/RequirementTree'
import { RequirementDetailPanel } from '../components/requirements/RequirementDetailPanel'
import { Input } from '../components/ui/input'
import { Card, CardContent } from '../components/ui/card'
import { Button } from '../components/ui/button'

export default function RequirementsExplorer() {
  const { entityType, entityName } = useParams<{ entityType?: string; entityName?: string }>()

  const { groups, loading, error, selectedRequirement, filter, setFilter, filteredGroups, handleSelectRequirement, refresh } =
    useRequirementsExplorer({ entityType, entityName })

  if (loading && groups.length === 0) {
    return (
      <div className="app-container" data-layout="dual">
        <Card className="border-dashed bg-white/80">
          <CardContent className="flex items-center gap-3 py-8 text-muted-foreground">
            <Loader2 size={20} className="animate-spin" /> Loading requirements...
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
              <p className="font-medium">Error loading requirements</p>
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

  const totalRequirements = groups.reduce((sum, group) => {
    const countInGroup = (g: typeof group): number => {
      return g.requirements.length + (g.children?.reduce((childSum, child) => childSum + countInGroup(child), 0) ?? 0)
    }
    return sum + countInGroup(group)
  }, 0)

  const completedRequirements = groups.reduce((sum, group) => {
    const countCompleteInGroup = (g: typeof group): number => {
      const complete = g.requirements.filter((r) => r.status === 'complete').length
      const childrenComplete = g.children?.reduce((childSum, child) => childSum + countCompleteInGroup(child), 0) ?? 0
      return complete + childrenComplete
    }
    return sum + countCompleteInGroup(group)
  }, 0)

  return (
    <div className="app-container space-y-6" data-layout="dual">
      <header className="rounded-3xl border bg-white/90 p-6 shadow-soft-lg">
        <div className="space-y-2">
          <span className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-500">Requirements Registry</span>
          <div className="flex items-center gap-3 text-3xl font-semibold text-slate-900">
            <span className="rounded-2xl bg-purple-100 p-3 text-purple-600">
              <ListTree size={28} strokeWidth={2.5} />
            </span>
            Requirements Explorer
          </div>
          <p className="max-w-3xl text-base text-muted-foreground">
            Browse and explore requirements for{' '}
            <span className="font-medium text-slate-700">
              {entityType}/{entityName}
            </span>
          </p>
        </div>

        <div className="mt-6 grid gap-4 sm:grid-cols-3">
          <Card>
            <CardContent className="pt-4">
              <p className="text-sm text-muted-foreground">Total Requirements</p>
              <p className="text-3xl font-bold text-slate-900">{totalRequirements}</p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-4">
              <p className="text-sm text-muted-foreground">Completed</p>
              <p className="text-3xl font-bold text-green-600">{completedRequirements}</p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-4">
              <p className="text-sm text-muted-foreground">Completion Rate</p>
              <p className="text-3xl font-bold text-blue-600">
                {totalRequirements > 0 ? Math.round((completedRequirements / totalRequirements) * 100) : 0}%
              </p>
            </CardContent>
          </Card>
        </div>
      </header>

      <div className="relative max-w-xl">
        <Search className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" size={18} />
        <Input
          id="requirements-search"
          type="text"
          className="pl-9"
          placeholder="Search requirements by title, ID, or description..."
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

      <div className="grid gap-6 lg:grid-cols-[400px,1fr]">
        <Card>
          <CardContent className="p-4">
            <RequirementTree groups={filteredGroups} selectedId={selectedRequirement?.id ?? null} onSelect={handleSelectRequirement} />
          </CardContent>
        </Card>

        <RequirementDetailPanel requirement={selectedRequirement} entityType={entityType} entityName={entityName} />
      </div>

      <div className="flex flex-wrap gap-4 text-sm">
        <Link to={`/prd/${entityType}/${entityName}`} className="text-primary hover:underline">
          ← View PRD
        </Link>
        <Link to={`/targets/${entityType}/${entityName}`} className="text-primary hover:underline">
          → View Operational Targets
        </Link>
        <Link to="/catalog" className="text-primary hover:underline">
          ← Back to Catalog
        </Link>
      </div>
    </div>
  )
}
