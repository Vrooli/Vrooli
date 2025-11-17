import { useCallback, useEffect, useState } from 'react'
import { useParams, Link, useSearchParams } from 'react-router-dom'
import { ListTree, Target, AlertTriangle, FileEdit, ArrowRight, RefreshCw, Loader2 } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import ReactMarkdown from 'react-markdown'
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
import { usePublishedPRD } from '../hooks/usePublishedPRD'
import { EcosystemTaskPanel } from '../components/ecosystem/EcosystemTaskPanel'
import { DiagnosticsPanel, IssuesPanel } from '../components/prd-viewer'
import { fetchQualityReport } from '../utils/quality'
import type { ScenarioQualityReport } from '../types'
import { buildApiUrl } from '../utils/apiClient'

interface DiagnosticsState {
  entityType: string
  entityName: string
  validatedAt?: string
  cacheUsed?: boolean
  violations: unknown
}

type QuickAction = {
  key: string
  title: string
  description: string
  icon: LucideIcon
  onClick?: () => void
  to?: string
  cta?: string
}

/**
 * ScenarioControlCenter
 *
 * Unified view for scenario health, PRD content, requirements, and targets.
 * Acts as the single landing page for understanding any scenario's state.
 */
export default function ScenarioControlCenter() {
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

  const [qualityReport, setQualityReport] = useState<ScenarioQualityReport | null>(null)
  const [qualityLoading, setQualityLoading] = useState(false)
  const [qualityError, setQualityError] = useState<string | null>(null)
  const [diagnostics, setDiagnostics] = useState<DiagnosticsState | null>(null)
  const [diagnosticsLoading, setDiagnosticsLoading] = useState(false)
  const [diagnosticsError, setDiagnosticsError] = useState<string | null>(null)

  const loadQualityReport = useCallback(
    async (force = false) => {
      if (!entityType || !entityName) {
        setQualityReport(null)
        return
      }

      setQualityLoading(true)
      if (force) {
        setQualityError(null)
      }

      try {
        const data = await fetchQualityReport(entityType, entityName, { useCache: !force })
        setQualityReport(data)
      } catch (err) {
        setQualityError(err instanceof Error ? err.message : 'Failed to load issues')
        setQualityReport(null)
      } finally {
        setQualityLoading(false)
      }
    },
    [entityType, entityName],
  )

  const handleRunDiagnostics = useCallback(async () => {
    if (!entityType || !entityName) {
      return
    }

    setDiagnosticsLoading(true)
    setDiagnosticsError(null)

    try {
      const response = await fetch(buildApiUrl('/drafts/validate'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ entity_type: entityType, entity_name: entityName }),
      })

      if (!response.ok) {
        const body = await response.text()
        throw new Error(body || `Failed to run diagnostics: ${response.statusText}`)
      }

      const data: Record<string, unknown> = await response.json()
      setDiagnostics({
        entityType: typeof data.entity_type === 'string' ? (data.entity_type as string) : entityType,
        entityName: typeof data.entity_name === 'string' ? (data.entity_name as string) : entityName,
        validatedAt: typeof data.validated_at === 'string' ? (data.validated_at as string) : undefined,
        cacheUsed: data.cache_used === true,
        violations: data.violations,
      })
    } catch (err) {
      setDiagnosticsError(err instanceof Error ? err.message : 'Unknown error running diagnostics.')
    } finally {
      setDiagnosticsLoading(false)
    }
  }, [entityType, entityName])

  useEffect(() => {
    loadQualityReport(true)
  }, [loadQualityReport])

  const loading = requirementsLoading || targetsLoading
  const error = requirementsError || targetsError

  const entityLabel = entityType && entityName ? `${entityType}/${entityName}` : 'scenario'

  const {
    data: prdData,
    loading: prdLoading,
    error: prdError,
  } = usePublishedPRD(entityType, entityName, { treatNotFoundAsEmpty: true })
  const hasPRDContent = Boolean(prdData?.content && prdData.content.trim().length > 0)

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

  const refreshAllData = () => {
    refreshRequirements()
    refreshTargets()
    loadQualityReport(true)
  }

  const quickActions: QuickAction[] = [
    {
      key: 'requirements',
      title: 'Requirements tree',
      description: 'Trace hierarchy & coverage',
      icon: ListTree,
      onClick: () => handleTabChange('requirements'),
      cta: 'Open requirements view',
    },
    {
      key: 'targets',
      title: 'Operational targets',
      description: 'Link to requirements',
      icon: Target,
      onClick: () => handleTabChange('targets'),
      cta: 'Jump to targets tab',
    },
  ]

  if (entityType && entityName) {
    quickActions.push({
      key: 'edit-draft',
      title: 'Edit PRD draft',
      description: 'Polish the source document',
      icon: FileEdit,
      to: `/draft/${entityType}/${entityName}`,
      cta: 'Open draft workspace',
    })
  }

  const quickActionClasses =
    'group block h-full rounded-2xl border border-slate-200 bg-white/90 p-4 text-left shadow-soft-sm transition hover:-translate-y-0.5 hover:border-slate-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-violet-200'

  const renderQuickAction = (action: QuickAction) => {
    const Icon = action.icon
    const body = (
      <>
        <div className="flex items-center justify-between gap-4">
          <div>
            <p className="text-xs font-medium uppercase tracking-[0.2em] text-muted-foreground">{action.description}</p>
            <p className="text-lg font-semibold text-slate-900">{action.title}</p>
          </div>
          <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-slate-100 text-slate-600">
            <Icon size={18} />
          </span>
        </div>
        <div className="mt-4 flex items-center gap-2 text-sm text-slate-500">
          <ArrowRight size={14} />
          <span>{action.cta || (action.to ? 'Open view' : 'Run action')}</span>
        </div>
      </>
    )

    if (action.to) {
      return (
        <Link key={action.key} to={action.to} className={quickActionClasses}>
          {body}
        </Link>
      )
    }

    return (
      <button key={action.key} type="button" onClick={action.onClick} className={quickActionClasses}>
        {body}
      </button>
    )
  }

  const breadcrumbItems = [
    { label: 'Catalog', to: '/catalog' },
    { label: `${entityType}/${entityName}`, to: `/scenario/${entityType}/${entityName}` },
    { label: 'Scenario Control Center' },
  ]

  const handleTabChange = (value: string) => {
    setSearchParams({ tab: value })
  }

  if (loading && totalRequirements === 0 && totalTargets === 0) {
    return (
      <div className="app-container space-y-6" data-layout="dual">
        <TopNav />
        <ControlCenterSkeleton />
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
              Scenario Control Center
            </div>
            <p className="max-w-3xl text-base text-muted-foreground">
              Comprehensive view of PRD content, requirements, and operational targets for{' '}
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
            <Button variant="ghost" size="sm" className="gap-2 w-full" onClick={() => handleTabChange('prd')}>
              <ArrowRight size={14} />
              Jump to PRD tab
            </Button>
          </div>
        </div>
      </header>

      {/* Tabs */}
      <Tabs value={defaultTab} onValueChange={handleTabChange} className="space-y-6">
        <TabsList className="grid w-full max-w-3xl grid-cols-4">
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="prd">PRD</TabsTrigger>
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

          {entityType && entityName && (
            <IssuesPanel
              report={qualityReport}
              loading={qualityLoading}
              error={qualityError}
              onRefresh={() => loadQualityReport(true)}
            />
          )}

          {entityType && entityName && (
            <div className="rounded-3xl border bg-white/90 p-6 shadow-soft-lg">
              <DiagnosticsPanel
                diagnostics={diagnostics}
                loading={diagnosticsLoading}
                error={diagnosticsError}
                onRunDiagnostics={handleRunDiagnostics}
                entityLabel={entityLabel}
              />
            </div>
          )}

          <EcosystemTaskPanel
            entityType={entityType}
            entityName={entityName}
            criticalIssues={unlinkedP0Targets.length}
            coverageIssues={unmatchedRequirements.length}
          />

          <Card>
            <CardContent className="space-y-4 pt-6">
              <div className="flex items-center justify-between">
                <h3 className="text-lg font-semibold text-slate-900">Quick actions</h3>
                <Button variant="ghost" size="sm" className="gap-2" onClick={refreshAllData}>
                  <RefreshCw size={14} className="text-slate-500" /> Refresh all
                </Button>
              </div>
              <div className="grid gap-4 md:grid-cols-2">
                {quickActions.map((action) => renderQuickAction(action))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* PRD Tab */}
        <TabsContent value="prd" className="space-y-6">
          <Card className="overflow-hidden border bg-white/90 shadow-soft-lg">
            <CardContent className="p-0">
              {prdLoading ? (
                <div className="flex items-center gap-3 px-6 py-6 text-muted-foreground">
                  <Loader2 size={18} className="animate-spin" /> Loading PRD...
                </div>
              ) : prdError ? (
                <div className="rounded-none border-0 px-6 py-6 text-sm text-red-700">
                  {prdError}
                </div>
              ) : hasPRDContent ? (
                <div className="prd-content px-6 py-6">
                  <ReactMarkdown>{prdData?.content ?? ''}</ReactMarkdown>
                </div>
              ) : (
                <div className="px-6 py-6 text-sm text-muted-foreground">
                  No published PRD yet. Publish a draft to see it here.
                </div>
              )}
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

function ControlCenterSkeleton() {
  return (
    <div className="space-y-6">
      <header className="rounded-3xl border bg-white/90 p-6 shadow-soft-lg">
        <div className="flex items-start justify-between gap-4">
          <div className="space-y-3 flex-1">
            <div className="h-4 w-32 rounded-full bg-slate-200" />
            <div className="h-8 w-64 rounded-full bg-slate-200" />
            <div className="h-4 w-full max-w-xl rounded-full bg-slate-200" />
          </div>
          <div className="flex flex-col gap-2 pt-6">
            <div className="h-9 w-36 rounded-lg bg-slate-200" />
            <div className="h-9 w-36 rounded-lg bg-slate-100" />
          </div>
        </div>
      </header>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {[...Array(4)].map((_, idx) => (
          <Card key={idx} className="animate-pulse">
            <CardContent className="space-y-3 pt-4">
              <div className="h-3 w-24 rounded-full bg-slate-200" />
              <div className="h-7 w-20 rounded-lg bg-slate-300" />
            </CardContent>
          </Card>
        ))}
      </div>

      <div className="rounded-3xl border bg-white/90 p-6 shadow-soft-lg">
        <div className="space-y-3 animate-pulse">
          <div className="h-4 w-28 rounded-full bg-slate-200" />
          <div className="h-5 w-72 rounded-full bg-slate-200" />
          <div className="h-32 rounded-2xl bg-slate-100" />
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        {[...Array(2)].map((_, idx) => (
          <Card key={`quick-${idx}`} className="animate-pulse">
            <CardContent className="space-y-3 pt-6">
              <div className="h-4 w-48 rounded-full bg-slate-200" />
              <div className="h-4 w-60 rounded-full bg-slate-200" />
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}
