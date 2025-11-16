import { useParams, Link, useSearchParams } from 'react-router-dom'
import { ListTree, Target, AlertTriangle, Loader2, FileEdit, FileText } from 'lucide-react'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '../components/ui/tabs'
import { Card, CardContent } from '../components/ui/card'
import { Button } from '../components/ui/button'
import { TopNav } from '../components/ui/top-nav'
import { Breadcrumbs } from '../components/ui/breadcrumbs'
import { useRequirementsExplorer } from '../hooks/useRequirementsExplorer'
import { useOperationalTargets } from '../hooks/useOperationalTargets'
import { RequirementTree } from '../components/requirements/RequirementTree'
import { RequirementDetailPanel } from '../components/requirements/RequirementDetailPanel'
import { TargetsList } from '../components/targets/TargetsList'
import { TargetDetailPanel } from '../components/targets/TargetDetailPanel'
import { Input } from '../components/ui/input'
import { Badge } from '../components/ui/badge'

/**
 * RequirementsDashboard
 *
 * Unified dashboard for managing both requirements and operational targets for a scenario/resource.
 * Replaces separate RequirementsExplorer and OperationalTargetsExplorer pages.
 *
 * Features:
 * - Tabbed interface: Overview | Requirements | Targets
 * - Overview tab shows linkage status and validation issues
 * - Requirements tab shows tree view with test linkage
 * - Targets tab shows PRD targets with requirement linkage
 * - Validates that P0/P1 targets are linked to requirements
 * - Shows unmatched requirements (no target coverage)
 */
export default function RequirementsDashboard() {
  const { entityType, entityName } = useParams<{ entityType?: string; entityName?: string }>()
  const [searchParams, setSearchParams] = useSearchParams()
  const defaultTab = searchParams.get('tab') || 'overview'

  // Requirements data
  const {
    groups: requirementGroups,
    loading: requirementsLoading,
    error: requirementsError,
    selectedRequirement,
    filter: reqFilter,
    setFilter: setReqFilter,
    filteredGroups,
    handleSelectRequirement,
    refresh: refreshRequirements,
  } = useRequirementsExplorer({ entityType, entityName })

  // Targets data
  const {
    targets,
    unmatchedRequirements,
    loading: targetsLoading,
    error: targetsError,
    selectedTarget,
    filter: targetFilter,
    setFilter: setTargetFilter,
    categoryFilter,
    setCategoryFilter,
    statusFilter,
    setStatusFilter,
    filteredTargets,
    handleSelectTarget,
    refresh: refreshTargets,
  } = useOperationalTargets({ entityType, entityName })

  const loading = requirementsLoading || targetsLoading
  const error = requirementsError || targetsError

  // Calculate metrics
  const totalRequirements = requirementGroups.reduce((sum, group) => {
    const countInGroup = (g: typeof group): number => {
      return g.requirements.length + (g.children?.reduce((childSum, child) => childSum + countInGroup(child), 0) ?? 0)
    }
    return sum + countInGroup(group)
  }, 0)

  const completedRequirements = requirementGroups.reduce((sum, group) => {
    const countCompleteInGroup = (g: typeof group): number => {
      const complete = g.requirements.filter((r) => r.status === 'complete').length
      const childrenComplete = g.children?.reduce((childSum, child) => childSum + countCompleteInGroup(child), 0) ?? 0
      return complete + childrenComplete
    }
    return sum + countCompleteInGroup(group)
  }, 0)

  const totalTargets = targets.length
  const completedTargets = targets.filter((t) => t.status === 'complete').length
  const p0Targets = targets.filter((t) => t.criticality === 'P0')
  const p1Targets = targets.filter((t) => t.criticality === 'P1')
  const unlinkedP0Targets = p0Targets.filter((t) => !t.linked_requirement_ids || t.linked_requirement_ids.length === 0)
  const unlinkedP1Targets = p1Targets.filter((t) => !t.linked_requirement_ids || t.linked_requirement_ids.length === 0)

  const breadcrumbItems = [
    { label: 'Catalog', to: '/catalog' },
    { label: `${entityType}/${entityName}`, to: `/prd/${entityType}/${entityName}` },
    { label: 'Requirements & Targets' },
  ]

  const handleTabChange = (value: string) => {
    setSearchParams({ tab: value })
  }

  if (loading && totalRequirements === 0 && totalTargets === 0) {
    return (
      <div className="app-container" data-layout="dual">
        <TopNav />
        <Card className="border-dashed bg-white/80">
          <CardContent className="flex items-center gap-3 py-8 text-muted-foreground">
            <Loader2 size={20} className="animate-spin" /> Loading requirements and targets...
          </CardContent>
        </Card>
      </div>
    )
  }

  if (error) {
    return (
      <div className="app-container" data-layout="dual">
        <TopNav />
        <Card className="border-amber-200 bg-amber-50">
          <CardContent className="flex items-center gap-3 py-8 text-amber-900">
            <AlertTriangle size={20} />
            <div>
              <p className="font-medium">Error loading data</p>
              <p className="text-sm">{error}</p>
            </div>
            <Button
              onClick={() => {
                refreshRequirements()
                refreshTargets()
              }}
              variant="outline"
              size="sm"
              className="ml-auto"
            >
              Retry
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="app-container space-y-6" data-layout="dual">
      <TopNav />
      <Breadcrumbs items={breadcrumbItems} />

      {/* Header */}
      <header className="rounded-3xl border bg-white/90 p-6 shadow-soft-lg">
        <div className="flex items-start justify-between gap-4">
          <div className="space-y-2 flex-1">
            <span className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-500">Command Center</span>
            <div className="flex items-center gap-3 text-3xl font-semibold text-slate-900">
              <span className="rounded-2xl bg-gradient-to-br from-purple-100 to-blue-100 p-3 text-purple-600">
                <ListTree size={28} strokeWidth={2.5} />
              </span>
              Requirements & Targets
            </div>
            <p className="max-w-3xl text-base text-muted-foreground">
              Comprehensive view of requirements, operational targets, and their linkage for{' '}
              <span className="font-medium text-slate-700">
                {entityType}/{entityName}
              </span>
            </p>
          </div>
          <div className="flex flex-col gap-2 pt-6">
            <Link to={`/draft/${entityType}/${entityName}`}>
              <Button variant="outline" size="sm" className="gap-2 w-full">
                <FileEdit size={14} />
                Edit Draft
              </Button>
            </Link>
            <Link to={`/prd/${entityType}/${entityName}`}>
              <Button variant="ghost" size="sm" className="gap-2 w-full">
                <FileText size={14} />
                View PRD
              </Button>
            </Link>
          </div>
        </div>
      </header>

      {/* Tabs */}
      <Tabs value={defaultTab} onValueChange={handleTabChange} className="space-y-6">
        <TabsList className="grid w-full max-w-2xl grid-cols-3">
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="requirements">
            Requirements
            {totalRequirements > 0 && <Badge className="ml-2">{totalRequirements}</Badge>}
          </TabsTrigger>
          <TabsTrigger value="targets">
            Targets
            {totalTargets > 0 && <Badge className="ml-2">{totalTargets}</Badge>}
          </TabsTrigger>
        </TabsList>

        {/* Overview Tab */}
        <TabsContent value="overview" className="space-y-6">
          {/* Metrics Grid */}
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <Card>
              <CardContent className="pt-4">
                <p className="text-sm text-muted-foreground">Requirements</p>
                <p className="text-3xl font-bold text-slate-900">{totalRequirements}</p>
                <p className="mt-1 text-xs text-green-600">
                  {completedRequirements} completed ({totalRequirements > 0 ? Math.round((completedRequirements / totalRequirements) * 100) : 0}%)
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardContent className="pt-4">
                <p className="text-sm text-muted-foreground">Targets (All)</p>
                <p className="text-3xl font-bold text-slate-900">{totalTargets}</p>
                <p className="mt-1 text-xs text-green-600">
                  {completedTargets} completed ({totalTargets > 0 ? Math.round((completedTargets / totalTargets) * 100) : 0}%)
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardContent className="pt-4">
                <p className="text-sm text-muted-foreground">P0 Targets</p>
                <p className="text-3xl font-bold text-blue-600">{p0Targets.length}</p>
                <p className="mt-1 text-xs text-amber-600">{unlinkedP0Targets.length} unlinked</p>
              </CardContent>
            </Card>

            <Card>
              <CardContent className="pt-4">
                <p className="text-sm text-muted-foreground">P1 Targets</p>
                <p className="text-3xl font-bold text-purple-600">{p1Targets.length}</p>
                <p className="mt-1 text-xs text-amber-600">{unlinkedP1Targets.length} unlinked</p>
              </CardContent>
            </Card>
          </div>

          {/* Validation Issues */}
          {(unlinkedP0Targets.length > 0 || unlinkedP1Targets.length > 0 || unmatchedRequirements.length > 0) && (
            <Card className="border-amber-200 bg-amber-50">
              <CardContent className="space-y-4 pt-6">
                <div className="flex items-center gap-2">
                  <AlertTriangle className="text-amber-600" size={20} />
                  <h3 className="font-semibold text-amber-900">Validation Issues</h3>
                </div>

                {unlinkedP0Targets.length > 0 && (
                  <div className="rounded-lg bg-white p-4">
                    <p className="mb-2 text-sm font-medium text-amber-900">
                      {unlinkedP0Targets.length} P0 target(s) not linked to requirements
                    </p>
                    <ul className="space-y-1 text-sm text-amber-800">
                      {unlinkedP0Targets.slice(0, 5).map((target) => (
                        <li key={target.id}>
                          • {target.title}
                          <Button
                            variant="link"
                            size="sm"
                            className="ml-2 h-auto p-0 text-xs text-blue-600"
                            onClick={() => handleTabChange('targets')}
                          >
                            Link now →
                          </Button>
                        </li>
                      ))}
                      {unlinkedP0Targets.length > 5 && (
                        <li className="text-xs text-muted-foreground">... and {unlinkedP0Targets.length - 5} more</li>
                      )}
                    </ul>
                  </div>
                )}

                {unlinkedP1Targets.length > 0 && (
                  <div className="rounded-lg bg-white p-4">
                    <p className="mb-2 text-sm font-medium text-amber-900">
                      {unlinkedP1Targets.length} P1 target(s) not linked to requirements
                    </p>
                    <ul className="space-y-1 text-sm text-amber-800">
                      {unlinkedP1Targets.slice(0, 3).map((target) => (
                        <li key={target.id}>• {target.title}</li>
                      ))}
                      {unlinkedP1Targets.length > 3 && (
                        <li className="text-xs text-muted-foreground">... and {unlinkedP1Targets.length - 3} more</li>
                      )}
                    </ul>
                  </div>
                )}

                {unmatchedRequirements.length > 0 && (
                  <div className="rounded-lg bg-white p-4">
                    <p className="mb-2 text-sm font-medium text-amber-900">
                      {unmatchedRequirements.length} requirement(s) without target coverage
                    </p>
                    <p className="text-xs text-muted-foreground">
                      These requirements exist but aren't linked to any operational targets in the PRD.
                    </p>
                  </div>
                )}
              </CardContent>
            </Card>
          )}

          {/* Quick Actions */}
          <Card>
            <CardContent className="space-y-4 pt-6">
              <h3 className="font-semibold text-slate-900">Quick Actions</h3>
              <div className="grid gap-3 sm:grid-cols-2">
                <Button variant="outline" className="justify-start" onClick={() => handleTabChange('requirements')}>
                  <ListTree className="mr-2" size={16} />
                  View Requirements Tree
                </Button>
                <Button variant="outline" className="justify-start" onClick={() => handleTabChange('targets')}>
                  <Target className="mr-2" size={16} />
                  Manage Operational Targets
                </Button>
                <Link to={`/draft/${entityType}/${entityName}`} className="contents">
                  <Button variant="outline" className="justify-start">
                    Edit PRD Draft
                  </Button>
                </Link>
                <Button
                  variant="outline"
                  className="justify-start"
                  onClick={() => {
                    refreshRequirements()
                    refreshTargets()
                  }}
                >
                  Refresh Data
                </Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Requirements Tab */}
        <TabsContent value="requirements" className="space-y-6">
          <Card>
            <CardContent className="space-y-4 pt-6">
              <div className="flex items-center gap-4">
                <Input
                  type="search"
                  placeholder="Filter requirements by title or ID..."
                  value={reqFilter}
                  onChange={(e) => setReqFilter(e.target.value)}
                  className="max-w-md"
                />
                <p className="text-sm text-muted-foreground">
                  Showing {filteredGroups.length} of {requirementGroups.length} groups
                </p>
              </div>

              <div className="grid gap-6 lg:grid-cols-[1fr_400px]">
                <div className="space-y-4">
                  {filteredGroups.length === 0 ? (
                    <p className="py-12 text-center text-muted-foreground">
                      {reqFilter ? 'No requirements match your filter' : 'No requirements found'}
                    </p>
                  ) : (
                    <RequirementTree
                      groups={filteredGroups}
                      selectedId={selectedRequirement?.id || null}
                      onSelect={handleSelectRequirement}
                    />
                  )}
                </div>

                {selectedRequirement && (
                  <div className="sticky top-6 h-fit">
                    <RequirementDetailPanel
                      requirement={selectedRequirement}
                      entityType={entityType}
                      entityName={entityName}
                    />
                  </div>
                )}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Targets Tab */}
        <TabsContent value="targets" className="space-y-6">
          <Card>
            <CardContent className="space-y-4 pt-6">
              <div className="flex flex-wrap items-center gap-4">
                <Input
                  type="search"
                  placeholder="Filter targets by title..."
                  value={targetFilter}
                  onChange={(e) => setTargetFilter(e.target.value)}
                  className="max-w-md"
                />
                <div className="flex gap-2">
                  <select value={categoryFilter} onChange={(e) => setCategoryFilter(e.target.value)} className="rounded border px-3 py-2 text-sm">
                    <option value="all">All Categories</option>
                    {Array.from(new Set(targets.map((t) => t.category))).map((cat) => (
                      <option key={cat} value={cat}>
                        {cat}
                      </option>
                    ))}
                  </select>
                  <select
                    value={statusFilter}
                    onChange={(e) => setStatusFilter(e.target.value as 'all' | 'pending' | 'complete')}
                    className="rounded border px-3 py-2 text-sm"
                  >
                    <option value="all">All Statuses</option>
                    <option value="complete">Complete</option>
                    <option value="pending">Pending</option>
                  </select>
                </div>
              </div>

              <div className="grid gap-6 lg:grid-cols-[1fr_400px]">
                <div className="space-y-4">
                  <TargetsList
                    targets={filteredTargets}
                    selectedId={selectedTarget?.id || null}
                    onSelect={handleSelectTarget}
                  />
                </div>

                {selectedTarget && (
                  <div className="sticky top-6 h-fit">
                    <TargetDetailPanel target={selectedTarget} />
                  </div>
                )}
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
