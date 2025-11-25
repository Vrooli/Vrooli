import { useCallback, useEffect, useState } from 'react'
import toast from 'react-hot-toast'
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
import { RequirementCreateDialog } from '../components/requirements/RequirementCreateDialog'
import { TargetsList } from '../components/targets/TargetsList'
import { TargetDetailPanel } from '../components/targets/TargetDetailPanel'
import { TargetCreateDialog } from '../components/targets/TargetCreateDialog'
import { Input } from '../components/ui/input'
import { Badge } from '../components/ui/badge'
import { usePublishedPRD } from '../hooks/usePublishedPRD'
import { EcosystemTaskPanel } from '../components/ecosystem/EcosystemTaskPanel'
import { DiagnosticsPanel, IssuesPanel } from '../components/prd-viewer'
import { fetchQualityReport } from '../utils/quality'
import type { ScenarioQualityReport } from '../types'
import { buildApiUrl } from '../utils/apiClient'
import { useReportIssueActions } from '../components/issues/ReportIssueProvider'
import { buildIssueReportSeedForCategories, buildIssueReportSeedFromQualityReport } from '../utils/issueReports'

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
  const targetIdParam = searchParams.get('target')
  const targetSearchParam = searchParams.get('search')

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
  } = useOperationalTargets({
    entityType,
    entityName,
    autoSelectTargetId: targetIdParam,
    autoSelectTargetSearch: targetSearchParam
  })

  const [qualityReport, setQualityReport] = useState<ScenarioQualityReport | null>(null)
  const [qualityLoading, setQualityLoading] = useState(false)
  const [qualityError, setQualityError] = useState<string | null>(null)
  const [diagnostics, setDiagnostics] = useState<DiagnosticsState | null>(null)
  const [diagnosticsLoading, setDiagnosticsLoading] = useState(false)
  const [diagnosticsError, setDiagnosticsError] = useState<string | null>(null)
  const { openIssueDialog } = useReportIssueActions()

  const handleReportIssues = useCallback(() => {
    if (!qualityReport) return
    const seed = buildIssueReportSeedFromQualityReport(qualityReport)
    if (!seed.categories.length) {
      toast.error('No actionable issues to report for this scenario')
      return
    }
    openIssueDialog(seed)
  }, [openIssueDialog, qualityReport])

  const handleReportCategory = useCallback(
    (categoryId: string) => {
      if (!qualityReport) return
      const categoryMap: Record<string, string | string[]> = {
        structure: ['structure_missing', 'structure_unexpected'],
        targets: ['target_linkage'],
        requirements: ['requirements_without_targets'],
        references: ['prd_reference'],
      }
      const targetIds = categoryMap[categoryId] ?? categoryId
      const seed = buildIssueReportSeedForCategories(qualityReport, targetIds) ?? buildIssueReportSeedFromQualityReport(qualityReport)
      if (!seed.categories.length) {
        toast.error('No actionable issues found for this category')
        return
      }
      openIssueDialog(seed)
    },
    [openIssueDialog, qualityReport],
  )

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

  const handleRequirementUpdate = () => {
    refreshRequirements()
  }

  const handleRequirementDelete = () => {
    refreshRequirements()
  }

  const handleTargetUpdate = () => {
    refreshTargets()
  }

  const handleTargetDelete = () => {
    refreshTargets()
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
    'group block h-full rounded-2xl border-2 border-slate-200 bg-white/90 p-5 text-left shadow-soft-md transition hover:-translate-y-1 hover:border-violet-300 hover:shadow-lg focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-violet-200'

  const renderQuickAction = (action: QuickAction) => {
    const Icon = action.icon
    const body = (
      <>
        <div className="flex items-start justify-between gap-4">
          <div className="space-y-1 flex-1">
            <p className="text-xs font-semibold uppercase tracking-[0.2em] text-violet-600">{action.description}</p>
            <p className="text-lg font-bold text-slate-900 leading-tight group-hover:text-violet-700 transition-colors">{action.title}</p>
          </div>
          <span className="flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-xl bg-gradient-to-br from-violet-100 to-purple-100 text-violet-600 shadow-inner group-hover:from-violet-200 group-hover:to-purple-200 transition-all">
            <Icon size={20} strokeWidth={2.5} />
          </span>
        </div>
        <div className="mt-4 flex items-center gap-2 text-sm font-medium text-slate-600 group-hover:text-violet-600 transition-colors">
          <ArrowRight size={16} className="group-hover:translate-x-1 transition-transform" />
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
        <Card className="border-2 border-amber-200 bg-gradient-to-br from-amber-50 via-white to-amber-50/30">
          <CardContent className="flex flex-col items-center gap-4 py-16 text-center">
            <div className="rounded-full bg-amber-100 p-4 shadow-inner">
              <AlertTriangle size={36} className="text-amber-600" />
            </div>
            <div className="space-y-2 max-w-md">
              <p className="text-lg font-semibold text-amber-900">Failed to Load Scenario Data</p>
              <p className="text-sm text-amber-700">{error}</p>
              <p className="text-xs text-slate-600">Ensure the scenario exists and the API is accessible.</p>
            </div>
            <div className="flex gap-3 mt-2">
              <Button
                onClick={() => {
                  refreshRequirements()
                  refreshTargets()
                }}
                size="lg"
                className="h-12 shadow-md hover:shadow-lg"
              >
                <RefreshCw size={18} className="mr-2" />
                Retry Loading
              </Button>
              <Button variant="outline" size="lg" asChild className="h-12">
                <Link to="/catalog">
                  <ArrowRight size={18} className="mr-2" />
                  Back to Catalog
                </Link>
              </Button>
            </div>
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
      <header className="rounded-3xl border bg-gradient-to-br from-white via-violet-50/20 to-white p-6 sm:p-8 shadow-soft-lg">
        <div className="flex flex-col gap-6 lg:flex-row lg:items-start lg:justify-between">
          <div className="space-y-3 flex-1">
            <div className="inline-flex items-center gap-2 rounded-full bg-white/80 px-3 py-1.5 shadow-sm w-fit">
              <div className="h-2 w-2 rounded-full bg-purple-500 animate-pulse" />
              <span className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-700">Command Center</span>
            </div>
            <div className="flex items-center gap-3 text-2xl sm:text-3xl font-semibold text-slate-900">
              <span className="rounded-2xl bg-gradient-to-br from-purple-100 to-blue-100 p-2.5 sm:p-3 text-purple-600 shadow-md">
                <ListTree size={24} strokeWidth={2.5} className="sm:w-7 sm:h-7" />
              </span>
              <span className="leading-tight">Scenario Control Center</span>
            </div>
            <p className="max-w-3xl text-sm sm:text-base text-slate-600 leading-relaxed">
              Comprehensive view of PRD content, requirements, and operational targets for{' '}
              <span className="font-semibold text-violet-700 bg-violet-50 px-2 py-0.5 rounded">
                {entityType}/{entityName}
              </span>
            </p>
          </div>
          <div className="flex flex-col sm:flex-row lg:flex-col gap-2.5 sm:pt-0 lg:pt-6 min-w-0 lg:min-w-[200px]">
            <Button
              variant="outline"
              size="lg"
              className="gap-2.5 h-11 font-semibold hover:bg-violet-50 hover:border-violet-300 hover:text-violet-700 transition-all shadow-sm hover:shadow-md"
              asChild
            >
              <Link to={`/draft/${entityType}/${entityName}`}>
                <FileEdit size={16} />
                <span>Edit Draft</span>
              </Link>
            </Button>
            <Button
              variant="ghost"
              size="lg"
              className="gap-2.5 h-11 font-semibold hover:bg-slate-100 transition-colors"
              onClick={() => handleTabChange('prd')}
            >
              <ArrowRight size={16} />
              <span className="hidden sm:inline lg:hidden">PRD</span>
              <span className="sm:hidden lg:inline">View PRD</span>
            </Button>
          </div>
        </div>
      </header>

      {/* Tabs */}
      <Tabs value={defaultTab} onValueChange={handleTabChange} className="space-y-6">
        <div className="sticky top-0 z-10 bg-gradient-to-b from-white via-white to-transparent pb-3 -mt-3 pt-3">
          <TabsList className="grid w-full max-w-4xl grid-cols-2 sm:grid-cols-4 gap-2 h-auto bg-white/80 backdrop-blur-sm p-1.5 shadow-md border">
            <TabsTrigger
              value="overview"
              className="data-[state=active]:bg-violet-600 data-[state=active]:text-white data-[state=active]:shadow-md h-10 font-semibold transition-all"
            >
              Overview
            </TabsTrigger>
            <TabsTrigger
              value="prd"
              className="data-[state=active]:bg-violet-600 data-[state=active]:text-white data-[state=active]:shadow-md h-10 font-semibold transition-all"
            >
              PRD
            </TabsTrigger>
            <TabsTrigger
              value="requirements"
              className="data-[state=active]:bg-violet-600 data-[state=active]:text-white data-[state=active]:shadow-md h-10 font-semibold transition-all"
            >
              <span>Requirements</span>
              {totalRequirements > 0 && (
                <Badge className="ml-1.5 bg-violet-100 text-violet-700 data-[state=active]:bg-white data-[state=active]:text-violet-700 text-xs px-1.5 py-0.5">
                  {totalRequirements}
                </Badge>
              )}
            </TabsTrigger>
            <TabsTrigger
              value="targets"
              className="data-[state=active]:bg-violet-600 data-[state=active]:text-white data-[state=active]:shadow-md h-10 font-semibold transition-all"
            >
              <span>Targets</span>
              {totalTargets > 0 && (
                <Badge className="ml-1.5 bg-violet-100 text-violet-700 data-[state=active]:bg-white data-[state=active]:text-violet-700 text-xs px-1.5 py-0.5">
                  {totalTargets}
                </Badge>
              )}
            </TabsTrigger>
          </TabsList>
        </div>

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
              onReportIssues={handleReportIssues}
              onReportCategory={handleReportCategory}
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
                {entityType && entityName && (
                  <RequirementCreateDialog
                    entityType={entityType}
                    entityName={entityName}
                    groups={requirementGroups}
                    onSuccess={refreshRequirements}
                  />
                )}
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
                      onRequirementUpdate={handleRequirementUpdate}
                      onRequirementDelete={handleRequirementDelete}
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
                {entityType && entityName && (
                  <TargetCreateDialog
                    entityType={entityType}
                    entityName={entityName}
                    onSuccess={refreshTargets}
                  />
                )}
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
                    <TargetDetailPanel
                      target={selectedTarget}
                      entityType={entityType}
                      entityName={entityName}
                      onTargetUpdate={handleTargetUpdate}
                      onTargetDelete={handleTargetDelete}
                    />
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
