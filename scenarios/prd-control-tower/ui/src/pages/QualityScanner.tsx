import { Fragment, useCallback, useEffect, useMemo, useState } from 'react'
import toast from 'react-hot-toast'
import {
  AlertTriangle,
  ArrowDown,
  ArrowUp,
  ChevronDown,
  ChevronRight,
  Download,
  Loader2,
  ListChecks,
  ScanSearch,
  ShieldAlert,
} from 'lucide-react'
import { TopNav } from '../components/ui/top-nav'
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/card'
import { Button } from '../components/ui/button'
import { Badge } from '../components/ui/badge'
import { cn } from '../lib/utils'
import { buildApiUrl } from '../utils/apiClient'
import { runQualityScan, fetchQualitySummary } from '../utils/quality'
import { flattenMissingTemplateSections } from '../utils/prdStructure'
import type {
  CatalogEntry,
  CatalogResponse,
  QualityScanEntity,
  QualitySummary,
  ScenarioQualityReport,
} from '../types'
import { formatDate } from '../utils/formatters'
import { IssuesSummaryCard } from '../components/issues'
import { useReportIssueActions } from '../components/issues/ReportIssueProvider'
import { buildQualityIssueCategories, STATUS_LABELS, STATUS_TONES } from '../components/prd-viewer/IssuesPanel'
import { buildIssueReportSeedFromQualityReport } from '../utils/issueReports'

const SCAN_STORAGE_KEY = 'prd-control-tower:quality-scan'

type ResultGroupKey = 'scenarios' | 'resources'

type StoredScanPayload = {
  generated_at: string
  reports: ScenarioQualityReport[]
}


export default function QualityScanner() {
  const [entries, setEntries] = useState<CatalogEntry[]>([])
  const [catalogError, setCatalogError] = useState<string | null>(null)
  const [scanning, setScanning] = useState(false)
  const [scanError, setScanError] = useState<string | null>(null)
  const [reports, setReports] = useState<ScenarioQualityReport[]>([])
  const [selectedReports, setSelectedReports] = useState<Set<string>>(new Set())
  const [lastRunAt, setLastRunAt] = useState<string | null>(null)
  const [qualitySummary, setQualitySummary] = useState<QualitySummary | null>(null)
  const [hasStoredScan, setHasStoredScan] = useState(false)
  const [expandedResultGroups, setExpandedResultGroups] = useState<Record<ResultGroupKey, boolean>>({
    scenarios: true,
    resources: true,
  })
  const [expandedReportId, setExpandedReportId] = useState<string | null>(null)
  const { openIssueDialog, openBulkIssueDialog } = useReportIssueActions()
  const selectedCount = selectedReports.size

  const toggleReportSelection = useCallback((report: ScenarioQualityReport) => {
    if (!canReportScenario(report)) {
      return
    }
    const key = buildReportKey(report)
    setSelectedReports((prev) => {
      const next = new Set(prev)
      if (next.has(key)) {
        next.delete(key)
      } else {
        next.add(key)
      }
      return next
    })
  }, [])

  const handleReportSingle = useCallback(
    (report: ScenarioQualityReport) => {
      const seed = buildIssueReportSeedFromQualityReport(report)
      if (!seed.categories.length) {
        toast.error('No actionable issues to report for this scenario')
        return
      }
      openIssueDialog(seed)
    },
    [openIssueDialog],
  )

  const handleBulkReport = useCallback(() => {
    if (selectedReports.size === 0) {
      return
    }
    const seeds = reports
      .filter((report) => selectedReports.has(buildReportKey(report)))
      .map((report) => buildIssueReportSeedFromQualityReport(report))
      .filter((seed) => seed.categories.length > 0)

    if (seeds.length === 0) {
      toast.error('Selected scenarios do not have actionable issues')
      return
    }

    openBulkIssueDialog(seeds)
  }, [openBulkIssueDialog, reports, selectedReports])

  useEffect(() => {
    let mounted = true
    const loadCatalog = async () => {
      setCatalogError(null)
      try {
        const response = await fetch(buildApiUrl('/catalog'))
        if (!response.ok) {
          throw new Error(`Failed to load catalog: ${response.statusText}`)
        }
        const data: CatalogResponse = await response.json()
        if (mounted) {
          setEntries(data.entries)
        }
      } catch (err) {
        if (mounted) {
          setCatalogError(err instanceof Error ? err.message : 'Failed to load catalog')
        }
      }
    }

    void loadCatalog()
    return () => {
      mounted = false
    }
  }, [])

  useEffect(() => {
    let mounted = true
    fetchQualitySummary()
      .then((data) => {
        if (mounted) {
          setQualitySummary(data)
        }
      })
      .catch(() => {
        if (mounted) {
          setQualitySummary(null)
        }
      })

    return () => {
      mounted = false
    }
  }, [])

  useEffect(() => {
    setSelectedReports((prev) => {
      const next = new Set<string>()
      reports.forEach((report) => {
        const key = buildReportKey(report)
        if (prev.has(key)) {
          next.add(key)
        }
      })
      return next
    })
  }, [reports])

  useEffect(() => {
    if (typeof window === 'undefined') {
      return
    }
    setHasStoredScan(Boolean(window.localStorage.getItem(SCAN_STORAGE_KEY)))
  }, [])

  const allEntities: QualityScanEntity[] = useMemo(() => {
    return entries.map((entry) => ({ type: entry.type, name: entry.name }))
  }, [entries])

  const runScanForEntities = useCallback(
    async (entities: QualityScanEntity[], label?: string) => {
      if (entities.length === 0) {
        toast.error('Select at least one entity to scan')
        return
      }
      setScanning(true)
      setScanError(null)
      try {
        const response = await runQualityScan(entities)
        setReports(response.reports)
        setLastRunAt(response.generated_at)
        persistScan(response)
        setHasStoredScan(true)
        toast.success(
          label ? `Scan complete · ${label}` : `Scan complete · ${response.reports.length} entities`,
        )
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Scan failed'
        setScanError(message)
        toast.error(message)
      } finally {
        setScanning(false)
      }
    },
    [],
  )

  const handleScanAll = useCallback(() => {
    void runScanForEntities(allEntities, 'full catalog scan')
  }, [allEntities, runScanForEntities])

  const loadPreviousScan = () => {
    const payload = loadStoredScan()
    if (!payload) {
      toast.error('No stored scan found')
      return
    }
    setReports(payload.reports)
    setLastRunAt(payload.generated_at)
    toast.success('Loaded previous scan')
  }

  const quickScanMissing = useCallback(() => {
    const missing = entries
      .filter((entry) => entry.type === 'scenario' && !entry.has_prd)
      .map((entry) => ({ type: entry.type, name: entry.name }))
    if (missing.length === 0) {
      toast.success('No missing PRDs detected right now')
      return
    }
    void runScanForEntities(missing, 'missing PRDs')
  }, [entries, runScanForEntities])

  const filteredReports = useMemo(() => {
    return reports.filter((report) => {
      // Always exclude scenarios/resources without issues
      const hasIssues = report.issue_counts.total > 0 || report.status === 'missing_prd'
      return hasIssues
    })
  }, [reports])

  const visibleSelectableReportIds = useMemo(() => {
    return filteredReports.filter((report) => canReportScenario(report)).map((report) => buildReportKey(report))
  }, [filteredReports])

  const visibleSelectedCount = useMemo(() => {
    return visibleSelectableReportIds.reduce((count, id) => (selectedReports.has(id) ? count + 1 : count), 0)
  }, [selectedReports, visibleSelectableReportIds])

  const allVisibleSelected = visibleSelectableReportIds.length > 0 && visibleSelectedCount === visibleSelectableReportIds.length

  const qualityStats = qualitySummary
    ? [
        { label: 'Tracked', value: qualitySummary.total_entities },
        { label: 'Issues flagged', value: qualitySummary.with_issues },
        { label: 'Missing PRDs', value: qualitySummary.missing_prd },
      ]
    : []

  const groupedReports = useMemo(() => {
    const scenarios = filteredReports.filter((r) => r.entity_type === 'scenario')
    const resources = filteredReports.filter((r) => r.entity_type === 'resource')
    return {
      scenarios,
      resources,
    }
  }, [filteredReports])

  const toggleResultGroupExpansion = (key: ResultGroupKey) => {
    setExpandedResultGroups((prev) => ({ ...prev, [key]: !prev[key] }))
  }

  const toggleVisibleReportSelection = useCallback(
    (select: boolean) => {
      setSelectedReports((prev) => {
        const next = new Set(prev)
        visibleSelectableReportIds.forEach((id) => {
          if (select) {
            next.add(id)
          } else {
            next.delete(id)
          }
        })
        return next
      })
    },
    [visibleSelectableReportIds],
  )

  const exportJSON = () => {
    if (reports.length === 0) {
      toast.error('Run a scan first')
      return
    }
    const payload = {
      generated_at: lastRunAt ?? new Date().toISOString(),
      reports,
    }
    downloadBlob(JSON.stringify(payload, null, 2), 'quality-scan.json', 'application/json')
  }

  const exportCSV = () => {
    if (reports.length === 0) {
      toast.error('Run a scan first')
      return
    }
    const csv = buildCsv(reports)
    downloadBlob(csv, 'quality-scan.csv', 'text/csv')
  }

  return (
    <div className="app-container space-y-6">
      <TopNav />

      <header className="rounded-3xl border bg-white/90 p-6 shadow-soft-lg">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div className="space-y-3">
            <span className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-500">Quality Ops</span>
            <div className="flex items-center gap-3 text-3xl font-semibold text-slate-900">
              <span className="rounded-2xl bg-rose-100 p-3 text-rose-600">
                <ShieldAlert size={28} />
              </span>
              Quality Scanner
            </div>
            <p className="max-w-2xl text-base text-muted-foreground">
              Scan all scenarios and resources for PRD + requirements drift. Only entities with issues are shown in the results.
            </p>
            {qualityStats.length > 0 && (
              <div className="flex flex-wrap gap-4 text-sm text-slate-600">
                {qualityStats.map((stat) => (
                  <span key={stat.label}>
                    <strong className="text-slate-900">{stat.value}</strong> {stat.label}
                  </span>
                ))}
                {qualitySummary?.last_generated && (
                  <span className="text-xs text-muted-foreground">
                    Summary refreshed {formatDate(qualitySummary.last_generated)}
                  </span>
                )}
              </div>
            )}
          </div>
          <div className="flex flex-wrap gap-2">
            <Button variant="outline" className="gap-2" onClick={loadPreviousScan} disabled={!hasStoredScan}>
              <ListChecks size={16} />
              Load previous scan
            </Button>
            <Button variant="outline" className="gap-2" onClick={quickScanMissing} disabled={scanning}>
              <ShieldAlert size={16} />
              Quick scan missing PRDs
            </Button>
            <Button className="gap-2" onClick={handleScanAll} disabled={scanning || allEntities.length === 0}>
              {scanning ? <Loader2 size={16} className="animate-spin" /> : <ScanSearch size={16} />}
              {scanning ? 'Scanning...' : `Scan all (${allEntities.length})`}
            </Button>
          </div>
        </div>
        {catalogError && (
          <div className="mt-4 rounded-xl border border-rose-200 bg-rose-50 p-4 text-sm text-rose-900">
            <div className="flex items-center gap-2">
              <AlertTriangle size={16} />
              <span>{catalogError}</span>
              <Button variant="ghost" size="sm" className="ml-auto" onClick={() => window.location.reload()}>
                Retry
              </Button>
            </div>
          </div>
        )}
      </header>

      <section className="space-y-6">
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">Scan results</CardTitle>
            <p className="text-sm text-muted-foreground">
              Export follow-ups or drill into detail without leaving the page.
            </p>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex flex-wrap gap-2">
              <Button variant="outline" size="sm" className="gap-2" onClick={exportJSON}>
                <Download size={14} /> Export JSON
              </Button>
              <Button variant="outline" size="sm" className="gap-2" onClick={exportCSV}>
                <Download size={14} /> Export CSV
              </Button>
              <Button
                variant={allVisibleSelected ? 'ghost' : 'outline'}
                size="sm"
                onClick={() => toggleVisibleReportSelection(!allVisibleSelected)}
                disabled={visibleSelectableReportIds.length === 0}
              >
                {allVisibleSelected
                  ? 'Clear visible selection'
                  : `Select all visible (${visibleSelectableReportIds.length})`}
              </Button>
              <Button
                size="sm"
                className="gap-2"
                disabled={selectedCount === 0}
                onClick={handleBulkReport}
              >
                <ListChecks size={14} /> Report selected ({selectedCount})
              </Button>
            </div>

            {scanError && (
              <div className="rounded-md border border-rose-200 bg-rose-50 p-3 text-sm text-rose-900">
                <AlertTriangle size={14} className="mr-2 inline" /> {scanError}
              </div>
            )}

            {reports.length === 0 ? (
              <div className="rounded-lg border border-dashed p-8 text-center text-sm text-muted-foreground">
                Run a scan to see detailed results.
              </div>
            ) : (
              <div className="space-y-2 text-sm text-muted-foreground">
                <p>
                  {reports.length} entities scanned · latest run {lastRunAt ? formatDate(lastRunAt) : 'just now'}
                </p>
                <p className="text-xs">
                  Showing {filteredReports.length} results ({groupedReports.scenarios.length} scenarios, {groupedReports.resources.length} resources)
                </p>
              </div>
            )}

            {reports.length > 0 && (
              <div className="space-y-4">
                {/* Scenarios Group */}
                {groupedReports.scenarios.length > 0 && (
                  <div className="rounded-2xl border bg-white">
                    <button
                      type="button"
                      className="flex w-full items-center justify-between p-4"
                      onClick={() => toggleResultGroupExpansion('scenarios')}
                    >
                      <div className="flex items-center gap-2 text-left">
                        {expandedResultGroups.scenarios ? <ChevronDown size={18} /> : <ChevronRight size={18} />}
                        <span className="text-lg font-semibold text-slate-900">Scenarios</span>
                        <Badge variant="secondary">
                          {groupedReports.scenarios.length} {groupedReports.scenarios.length === 1 ? 'result' : 'results'}
                        </Badge>
                      </div>
                    </button>
                    {expandedResultGroups.scenarios && (
                      <div className="border-t">
                        <ReportTable
                          reports={groupedReports.scenarios}
                          expandedReportId={expandedReportId}
                          setExpandedReportId={setExpandedReportId}
                          selectedReports={selectedReports}
                          toggleReportSelection={toggleReportSelection}
                          handleReportSingle={handleReportSingle}
                        />
                      </div>
                    )}
                  </div>
                )}

                {/* Resources Group */}
                {groupedReports.resources.length > 0 && (
                  <div className="rounded-2xl border bg-white">
                    <button
                      type="button"
                      className="flex w-full items-center justify-between p-4"
                      onClick={() => toggleResultGroupExpansion('resources')}
                    >
                      <div className="flex items-center gap-2 text-left">
                        {expandedResultGroups.resources ? <ChevronDown size={18} /> : <ChevronRight size={18} />}
                        <span className="text-lg font-semibold text-slate-900">Resources</span>
                        <Badge variant="secondary">
                          {groupedReports.resources.length} {groupedReports.resources.length === 1 ? 'result' : 'results'}
                        </Badge>
                      </div>
                    </button>
                    {expandedResultGroups.resources && (
                      <div className="border-t">
                        <ReportTable
                          reports={groupedReports.resources}
                          expandedReportId={expandedReportId}
                          setExpandedReportId={setExpandedReportId}
                          selectedReports={selectedReports}
                          toggleReportSelection={toggleReportSelection}
                          handleReportSingle={handleReportSingle}
                        />
                      </div>
                    )}
                  </div>
                )}
              </div>
            )}
          </CardContent>
        </Card>
      </section>
    </div>
  )
}

function persistScan(response: { generated_at: string; reports: ScenarioQualityReport[] }) {
  if (typeof window === 'undefined') return
  window.localStorage.setItem(SCAN_STORAGE_KEY, JSON.stringify(response))
}

function buildReportKey(report: ScenarioQualityReport) {
  return `${report.entity_type}:${report.entity_name}`
}

function canReportScenario(report: ScenarioQualityReport) {
  return report.status === 'missing_prd' || report.issue_counts.total > 0
}

function loadStoredScan(): StoredScanPayload | null {
  if (typeof window === 'undefined') return null
  const raw = window.localStorage.getItem(SCAN_STORAGE_KEY)
  if (!raw) return null
  try {
    return JSON.parse(raw) as StoredScanPayload
  } catch (err) {
    console.warn('Failed to parse stored quality scan', err)
    return null
  }
}


function buildCsv(reports: ScenarioQualityReport[]) {
  const header = ['entity_type', 'entity_name', 'status', 'issue_type', 'detail']
  const rows: string[][] = [header]

  reports.forEach((report) => {
    if (report.status === 'missing_prd') {
      rows.push([report.entity_type, report.entity_name, report.status, 'missing_prd', report.message || 'PRD missing'])
      return
    }

    if (report.issue_counts.total === 0) {
      rows.push([report.entity_type, report.entity_name, report.status, 'healthy', 'No issues'])
      return
    }

    const missingSections = flattenMissingTemplateSections(report.template_compliance_v2)
    missingSections.forEach((section: string) => {
      rows.push([report.entity_type, report.entity_name, report.status, 'template_section', section])
    })

    const unexpectedSections = report.template_compliance_v2?.unexpected_sections ?? []
    unexpectedSections.forEach((section: string) => {
      rows.push([report.entity_type, report.entity_name, report.status, 'unexpected_section', section])
    })

    report.target_linkage_issues?.forEach((issue) => {
      rows.push([report.entity_type, report.entity_name, report.status, 'target_linkage', `${issue.criticality}: ${issue.title}`])
    })

    report.requirements_without_targets?.forEach((req) => {
      rows.push([report.entity_type, report.entity_name, report.status, 'requirement_coverage', `${req.id}: ${req.title}`])
    })

    report.prd_ref_issues?.forEach((issue) => {
      rows.push([report.entity_type, report.entity_name, report.status, 'prd_reference', `${issue.requirement_id}: ${issue.message}`])
    })
  })

  return rows.map((row) => row.map(csvEscape).join(',')).join('\n')
}

function csvEscape(value: string) {
  if (value == null) return ''
  if (/[",\n]/.test(value)) {
    return `"${value.replace(/"/g, '""')}"`
  }
  return value
}

function downloadBlob(content: string, filename: string, mime: string) {
  const blob = new Blob([content], { type: mime })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

function StatusPill({ status }: { status: ScenarioQualityReport['status'] }) {
  const statusClasses: Record<ScenarioQualityReport['status'], string> = {
    healthy: 'bg-emerald-50 text-emerald-700',
    needs_attention: 'bg-amber-50 text-amber-800',
    blocked: 'bg-rose-50 text-rose-800',
    missing_prd: 'bg-rose-50 text-rose-800',
    error: 'bg-slate-100 text-slate-700',
  }
  return <span className={cn('rounded-full px-3 py-1 text-xs font-semibold', statusClasses[status])}>{status.replace('_', ' ')}</span>
}

type SortColumn = 'entity' | 'status' | 'issues' | 'targets' | 'requirements'
type SortDirection = 'asc' | 'desc'

function ReportTable({
  reports,
  expandedReportId,
  setExpandedReportId,
  selectedReports,
  toggleReportSelection,
  handleReportSingle,
}: {
  reports: ScenarioQualityReport[]
  expandedReportId: string | null
  setExpandedReportId: (id: string | null) => void
  selectedReports: Set<string>
  toggleReportSelection: (report: ScenarioQualityReport) => void
  handleReportSingle: (report: ScenarioQualityReport) => void
}) {
  const [sortColumn, setSortColumn] = useState<SortColumn>('issues')
  const [sortDirection, setSortDirection] = useState<SortDirection>('desc')

  const handleSort = (column: SortColumn) => {
    if (sortColumn === column) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc')
    } else {
      setSortColumn(column)
      setSortDirection('desc')
    }
  }

  const sortedReports = useMemo(() => {
    const sorted = [...reports]
    sorted.sort((a, b) => {
      let comparison = 0
      switch (sortColumn) {
        case 'entity':
          comparison = a.entity_name.localeCompare(b.entity_name)
          break
        case 'status':
          comparison = a.status.localeCompare(b.status)
          break
        case 'issues':
          comparison = a.issue_counts.total - b.issue_counts.total
          break
        case 'targets':
          comparison = a.target_count - b.target_count
          break
        case 'requirements':
          comparison = a.requirement_count - b.requirement_count
          break
      }
      return sortDirection === 'asc' ? comparison : -comparison
    })
    return sorted
  }, [reports, sortColumn, sortDirection])

  const SortIcon = ({ column }: { column: SortColumn }) => {
    if (sortColumn !== column) {
      return <ArrowDown size={14} className="ml-1 inline opacity-30" />
    }
    return sortDirection === 'asc' ? (
      <ArrowUp size={14} className="ml-1 inline" />
    ) : (
      <ArrowDown size={14} className="ml-1 inline" />
    )
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead className="bg-slate-50 text-left text-xs uppercase tracking-wide text-slate-500">
          <tr>
            <th className="px-4 py-3">Select</th>
            <th className="px-4 py-3">
              <button
                type="button"
                className="flex items-center hover:text-slate-700"
                onClick={() => handleSort('entity')}
              >
                Entity
                <SortIcon column="entity" />
              </button>
            </th>
            <th className="px-4 py-3">
              <button
                type="button"
                className="flex items-center hover:text-slate-700"
                onClick={() => handleSort('status')}
              >
                Status
                <SortIcon column="status" />
              </button>
            </th>
            <th className="px-4 py-3">
              <button
                type="button"
                className="flex items-center hover:text-slate-700"
                onClick={() => handleSort('issues')}
              >
                Issues
                <SortIcon column="issues" />
              </button>
            </th>
            <th className="px-4 py-3">
              <button
                type="button"
                className="flex items-center hover:text-slate-700"
                onClick={() => handleSort('targets')}
              >
                Targets
                <SortIcon column="targets" />
              </button>
            </th>
            <th className="px-4 py-3">
              <button
                type="button"
                className="flex items-center hover:text-slate-700"
                onClick={() => handleSort('requirements')}
              >
                Requirements
                <SortIcon column="requirements" />
              </button>
            </th>
            <th className="px-4 py-3">Actions</th>
          </tr>
        </thead>
        <tbody>
          {sortedReports.map((report) => {
            const id = buildReportKey(report)
            const isExpanded = expandedReportId === id
            const isSelectable = canReportScenario(report)
            const isSelected = selectedReports.has(id)
            return (
              <Fragment key={id}>
                <tr className="border-t text-slate-700">
                  <td className="px-4 py-3">
                    <input
                      type="checkbox"
                      className="h-4 w-4"
                      checked={isSelected}
                      disabled={!isSelectable}
                      onChange={() => toggleReportSelection(report)}
                    />
                  </td>
                  <td className="px-4 py-3 font-semibold text-slate-900">
                    {report.entity_name}
                  </td>
                  <td className="px-4 py-3">
                    <StatusPill status={report.status} />
                  </td>
                  <td className="px-4 py-3">
                    {report.issue_counts.missing_template_sections +
                      report.issue_counts.target_coverage +
                      report.issue_counts.requirement_coverage +
                      report.issue_counts.prd_ref}
                  </td>
                  <td className="px-4 py-3">
                    {report.target_count}
                  </td>
                  <td className="px-4 py-3">
                    {report.requirement_count}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <div className="flex items-center justify-end gap-3">
                      <Button
                        variant="link"
                        className="px-0 text-sm"
                        onClick={() => setExpandedReportId(isExpanded ? null : id)}
                      >
                        {isExpanded ? 'Hide details' : 'View details'}
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        disabled={!isSelectable}
                        onClick={() => handleReportSingle(report)}
                      >
                        Report
                      </Button>
                    </div>
                  </td>
                </tr>
                {isExpanded && (
                  <tr>
                    <td colSpan={7} className="bg-slate-50 px-4 py-4">
                      <ReportDetails report={report} onReport={handleReportSingle} />
                    </td>
                  </tr>
                )}
              </Fragment>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

function ReportDetails({ report, onReport }: { report: ScenarioQualityReport; onReport?: (report: ScenarioQualityReport) => void }) {
  const hasIssues = report.status !== 'healthy'
  const categories = buildQualityIssueCategories(report)
  const overviewBadges = (
    <div className="flex flex-wrap gap-2 text-xs">
      <Badge variant="outline">Template gaps: {report.issue_counts.missing_template_sections}</Badge>
      <Badge variant="outline">Target coverage: {report.issue_counts.target_coverage}</Badge>
      <Badge variant="outline">Reqs without targets: {report.issue_counts.requirement_coverage}</Badge>
      <Badge variant="outline">PRD ref: {report.issue_counts.prd_ref}</Badge>
    </div>
  )

  return (
    <IssuesSummaryCard
      className="mt-0"
      title="Issues overview"
      subtitle={`${report.entity_type} · ${report.entity_name}`}
      overview={overviewBadges}
      statusLabel={STATUS_LABELS[report.status]}
      statusTone={STATUS_TONES[report.status]}
      issueCount={report.issue_counts.total}
      statusMessage={!hasIssues ? 'Healthy · No issues detected' : undefined}
      categories={categories}
      footerActions={[
        { label: 'View PRD →', to: `/scenario/${report.entity_type}/${report.entity_name}?tab=prd` },
        { label: 'Requirements dashboard →', to: `/scenario/${report.entity_type}/${report.entity_name}?tab=requirements` },
      ]}
      primaryAction={
        onReport
          ? {
              label: 'Report issues',
              onClick: () => onReport(report),
            }
          : undefined
      }
    />
  )
}
