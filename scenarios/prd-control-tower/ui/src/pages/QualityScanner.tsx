import { Fragment, useCallback, useEffect, useMemo, useState } from 'react'
import toast from 'react-hot-toast'
import {
  AlertTriangle,
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
import { Input } from '../components/ui/input'
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
import { buildQualityIssueCategories, STATUS_LABELS, STATUS_TONES } from '../components/prd-viewer/IssuesPanel'

const SCAN_STORAGE_KEY = 'prd-control-tower:quality-scan'

type ScanFilter = 'all' | 'scenario' | 'resource'
type PRDFilter = 'all' | 'with' | 'missing'
type ResultFilter = 'all' | 'issues' | 'healthy'
type SortKey = 'status' | 'issues' | 'name' | 'targets'

type StoredScanPayload = {
  generated_at: string
  reports: ScenarioQualityReport[]
}

type GroupKey = 'missing' | 'has_prd' | 'resources'

type GroupConfig = {
  key: GroupKey
  title: string
  description: string
  highlight?: boolean
  predicate: (entry: CatalogEntry) => boolean
}

type SortConfig = {
  key: SortKey
  direction: 'asc' | 'desc'
}

const GROUPS: GroupConfig[] = [
  {
    key: 'missing',
    title: 'Missing PRDs',
    description: 'Highest priority scenarios to unblock this week.',
    highlight: true,
    predicate: (entry) => entry.type === 'scenario' && !entry.has_prd,
  },
  {
    key: 'has_prd',
    title: 'Published PRDs',
    description: 'Healthy coverage but worth rescan before launches.',
    predicate: (entry) => entry.type === 'scenario' && entry.has_prd,
  },
  {
    key: 'resources',
    title: 'Resources',
    description: 'Shared infrastructure that still needs documentation discipline.',
    predicate: (entry) => entry.type === 'resource',
  },
]

const STATUS_ORDER: Record<ScenarioQualityReport['status'], number> = {
  blocked: 0,
  missing_prd: 0,
  error: 1,
  needs_attention: 2,
  healthy: 3,
}

export default function QualityScanner() {
  const [entries, setEntries] = useState<CatalogEntry[]>([])
  const [loadingCatalog, setLoadingCatalog] = useState(true)
  const [catalogError, setCatalogError] = useState<string | null>(null)
  const [search, setSearch] = useState('')
  const [typeFilter, setTypeFilter] = useState<ScanFilter>('all')
  const [prdFilter, setPrdFilter] = useState<PRDFilter>('missing')
  const [selected, setSelected] = useState<Record<string, boolean>>({})
  const [scanning, setScanning] = useState(false)
  const [scanError, setScanError] = useState<string | null>(null)
  const [reports, setReports] = useState<ScenarioQualityReport[]>([])
  const [resultFilter, setResultFilter] = useState<ResultFilter>('all')
  const [lastRunAt, setLastRunAt] = useState<string | null>(null)
  const [qualitySummary, setQualitySummary] = useState<QualitySummary | null>(null)
  const [hasStoredScan, setHasStoredScan] = useState(false)
  const [expandedGroups, setExpandedGroups] = useState<Record<GroupKey, boolean>>({
    missing: true,
    has_prd: false,
    resources: false,
  })
  const [sortConfig, setSortConfig] = useState<SortConfig>({ key: 'status', direction: 'asc' })
  const [expandedReportId, setExpandedReportId] = useState<string | null>(null)

  useEffect(() => {
    let mounted = true
    const loadCatalog = async () => {
      setLoadingCatalog(true)
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
      } finally {
        if (mounted) {
          setLoadingCatalog(false)
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
    if (typeof window === 'undefined') {
      return
    }
    setHasStoredScan(Boolean(window.localStorage.getItem(SCAN_STORAGE_KEY)))
  }, [])

  const filteredEntries = useMemo(() => {
    const q = search.trim().toLowerCase()
    return entries.filter((entry) => {
      if (typeFilter !== 'all' && entry.type !== typeFilter) return false
      if (!q) return true
      return entry.name.toLowerCase().includes(q) || entry.description.toLowerCase().includes(q)
    })
  }, [entries, search, typeFilter])

  const groupedEntries = useMemo(() => {
    return GROUPS.map((group) => ({
      ...group,
      entries: filteredEntries.filter(group.predicate),
    }))
  }, [filteredEntries])

  const visibleGroups = useMemo(() => {
    if (prdFilter === 'all') return groupedEntries
    if (prdFilter === 'missing') return groupedEntries.filter((group) => group.key === 'missing')
    if (prdFilter === 'with') return groupedEntries.filter((group) => group.key === 'has_prd')
    return groupedEntries
  }, [groupedEntries, prdFilter])

  const selectedEntities: QualityScanEntity[] = useMemo(() => {
    return entries
      .filter((entry) => selected[`${entry.type}:${entry.name}`])
      .map((entry) => ({ type: entry.type, name: entry.name }))
  }, [entries, selected])

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

  const handleScan = useCallback(() => {
    void runScanForEntities(selectedEntities)
  }, [selectedEntities, runScanForEntities])

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
    setSelected((prev) => {
      const next = { ...prev }
      missing.forEach((entity) => {
        next[`${entity.type}:${entity.name}`] = true
      })
      return next
    })
    void runScanForEntities(missing, 'missing PRDs')
  }, [entries, runScanForEntities])

  const filteredReports = useMemo(() => {
    return reports.filter((report) => {
      if (resultFilter === 'all') return true
      const hasIssues = report.issue_counts.total > 0 || report.status === 'missing_prd'
      if (resultFilter === 'issues') return hasIssues
      return !hasIssues
    })
  }, [reports, resultFilter])

  const sortedReports = useMemo(() => {
    return sortReports(filteredReports, sortConfig)
  }, [filteredReports, sortConfig])

  const qualityStats = qualitySummary
    ? [
        { label: 'Tracked', value: qualitySummary.total_entities },
        { label: 'Issues flagged', value: qualitySummary.with_issues },
        { label: 'Missing PRDs', value: qualitySummary.missing_prd },
      ]
    : []

  const toggleGroupExpansion = (key: GroupKey) => {
    setExpandedGroups((prev) => ({ ...prev, [key]: !prev[key] }))
  }

  const applySelectionForGroup = (entriesInGroup: CatalogEntry[], shouldSelect: boolean) => {
    if (entriesInGroup.length === 0) return
    setSelected((prev) => {
      const next = { ...prev }
      entriesInGroup.forEach((entry) => {
        next[`${entry.type}:${entry.name}`] = shouldSelect
      })
      return next
    })
  }

  const countSelectedForGroup = (entriesInGroup: CatalogEntry[]) => {
    return entriesInGroup.reduce((acc, entry) => {
      const key = `${entry.type}:${entry.name}`
      return acc + (selected[key] ? 1 : 0)
    }, 0)
  }

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

  const totalSelected = selectedEntities.length

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
              Select cohorts, scan for PRD + requirements drift, and export reports. The layout keeps stats, selection, and results inline so you never lose context.
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
            <Button className="gap-2" onClick={handleScan} disabled={scanning || totalSelected === 0}>
              {scanning ? <Loader2 size={16} className="animate-spin" /> : <ScanSearch size={16} />}
              {scanning
                ? 'Scanning...'
                : totalSelected > 0
                  ? `Scan ${totalSelected} selected`
                  : 'Select entities to scan'}
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
            <CardTitle className="text-lg">Choose what to scan</CardTitle>
            <p className="text-sm text-muted-foreground">
              Filter the catalog, review cohorts, and select everything needed with one click.
            </p>
          </CardHeader>
          <CardContent className="space-y-5">
            <div className="grid gap-3 lg:grid-cols-[2fr_1fr]">
              <Input placeholder="Search scenarios..." value={search} onChange={(e) => setSearch(e.target.value)} />
              <div className="flex gap-2 text-sm">
                <select
                  value={typeFilter}
                  onChange={(e) => setTypeFilter(e.target.value as ScanFilter)}
                  className="flex-1 rounded border px-3 py-2"
                >
                  <option value="all">All types</option>
                  <option value="scenario">Scenarios</option>
                  <option value="resource">Resources</option>
                </select>
                <select
                  value={prdFilter}
                  onChange={(e) => setPrdFilter(e.target.value as PRDFilter)}
                  className="flex-1 rounded border px-3 py-2"
                >
                  <option value="missing">Focus: Missing PRDs</option>
                  <option value="with">Focus: Published PRDs</option>
                  <option value="all">Focus: All coverage</option>
                </select>
              </div>
            </div>
            <div className="text-xs text-muted-foreground">
              {totalSelected} selected · {filteredEntries.length} visible of {entries.length}
            </div>
            <div className="space-y-4">
              {loadingCatalog ? (
                <div className="flex items-center gap-2 py-6 text-sm text-muted-foreground">
                  <Loader2 size={16} className="animate-spin" /> Loading catalog...
                </div>
              ) : (
                visibleGroups.map((group) => {
                  const isExpanded = expandedGroups[group.key]
                  const selectedCount = countSelectedForGroup(group.entries)
                  return (
                    <div
                      key={group.key}
                      className={cn('rounded-2xl border p-4', group.highlight ? 'border-rose-200 bg-rose-50/60' : 'bg-white')}
                    >
                      <button
                        type="button"
                        className="flex w-full items-center justify-between"
                        onClick={() => toggleGroupExpansion(group.key)}
                      >
                        <div className="text-left">
                          <p className="text-sm font-semibold text-slate-900 flex items-center gap-2">
                            {isExpanded ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
                            {group.title}
                            <Badge variant="secondary">
                              {group.entries.length} {group.entries.length === 1 ? 'entity' : 'entities'}
                            </Badge>
                          </p>
                          <p className="text-xs text-muted-foreground">{group.description}</p>
                        </div>
                        <p className="text-xs text-slate-600">{selectedCount} selected</p>
                      </button>
                      {isExpanded && (
                        <div className="mt-3 space-y-3">
                          <div className="flex gap-2 text-xs">
                            <Button
                              variant="outline"
                              size="sm"
                              className="text-xs"
                              onClick={() => applySelectionForGroup(group.entries, true)}
                              disabled={group.entries.length === 0}
                            >
                              Select all in group
                            </Button>
                            <Button
                              variant="ghost"
                              size="sm"
                              className="text-xs"
                              onClick={() => applySelectionForGroup(group.entries, false)}
                              disabled={group.entries.length === 0}
                            >
                              Clear group
                            </Button>
                          </div>
                          <div className="divide-y rounded-xl border">
                            {group.entries.length === 0 && (
                              <p className="p-3 text-xs text-muted-foreground">No entries in this group.</p>
                            )}
                            {group.entries.map((entry) => {
                              const key = `${entry.type}:${entry.name}`
                              const isSelected = Boolean(selected[key])
                              return (
                                <label
                                  key={key}
                                  className={cn(
                                    'flex cursor-pointer items-start gap-3 px-3 py-3 text-sm',
                                    isSelected ? 'bg-violet-50' : '',
                                  )}
                                >
                                  <input
                                    type="checkbox"
                                    checked={isSelected}
                                    onChange={() =>
                                      setSelected((prev) => ({ ...prev, [key]: !prev[key] }))
                                    }
                                    className="mt-1"
                                  />
                                  <div className="space-y-1">
                                    <div className="font-medium text-slate-900">
                                      {entry.name}
                                      {!entry.has_prd && (
                                        <Badge variant="destructive" className="ml-2">
                                          No PRD
                                        </Badge>
                                      )}
                                    </div>
                                    <p className="text-xs text-muted-foreground line-clamp-2">
                                      {entry.description || 'No description'}
                                    </p>
                                  </div>
                                </label>
                              )
                            })}
                          </div>
                        </div>
                      )}
                    </div>
                  )
                })
              )}
            </div>
          </CardContent>
        </Card>

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
              <select
                value={resultFilter}
                onChange={(e) => setResultFilter(e.target.value as ResultFilter)}
                className="ml-auto rounded border px-3 py-2 text-sm"
              >
                <option value="all">Show: All</option>
                <option value="issues">Show: Issues only</option>
                <option value="healthy">Show: Healthy only</option>
              </select>
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
                <p className="text-xs">Showing {sortedReports.length} results</p>
              </div>
            )}

            {sortedReports.length > 0 && (
              <div className="overflow-x-auto rounded-2xl border">
                <table className="w-full text-sm">
                  <thead className="bg-slate-50 text-left text-xs uppercase tracking-wide text-slate-500">
                    <tr>
                      <SortableHeader label="Entity" column="name" sortConfig={sortConfig} onSort={setSortConfig} />
                      <SortableHeader label="Status" column="status" sortConfig={sortConfig} onSort={setSortConfig} />
                      <SortableHeader
                        label="Template gaps"
                        column="issues"
                        sortConfig={sortConfig}
                        onSort={setSortConfig}
                      />
                      <SortableHeader
                        label="Targets"
                        column="targets"
                        sortConfig={sortConfig}
                        onSort={setSortConfig}
                      />
                      <th className="px-4 py-3">Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {sortedReports.map((report) => {
                      const id = `${report.entity_type}:${report.entity_name}`
                      const isExpanded = expandedReportId === id
                      return (
                        <Fragment key={id}>
                          <tr className="border-t text-slate-700">
                            <td className="px-4 py-3 font-semibold text-slate-900">
                              {report.entity_name}
                              <Badge variant="secondary" className="ml-2 uppercase">
                                {report.entity_type}
                              </Badge>
                            </td>
                            <td className="px-4 py-3">
                              <StatusPill status={report.status} />
                            </td>
                            <td className="px-4 py-3">
                              {report.issue_counts.missing_template_sections + report.issue_counts.target_coverage +
                                report.issue_counts.requirement_coverage +
                                report.issue_counts.prd_ref}{' '}
                              issues
                            </td>
                            <td className="px-4 py-3">
                              {report.target_count} targets · {report.requirement_count} requirements
                            </td>
                            <td className="px-4 py-3 text-right">
                              <Button
                                variant="link"
                                className="px-0 text-sm"
                                onClick={() =>
                                  setExpandedReportId((prev) => (prev === id ? null : id))
                                }
                              >
                                {isExpanded ? 'Hide details' : 'View details'}
                              </Button>
                            </td>
                          </tr>
                          {isExpanded && (
                            <tr>
                              <td colSpan={5} className="bg-slate-50 px-4 py-4">
                                <ReportDetails report={report} />
                              </td>
                            </tr>
                          )}
                        </Fragment>
                      )
                    })}
                  </tbody>
                </table>
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

function sortReports(reports: ScenarioQualityReport[], sortConfig: SortConfig) {
  const direction = sortConfig.direction === 'asc' ? 1 : -1
  return [...reports].sort((a, b) => {
    switch (sortConfig.key) {
      case 'name':
        return a.entity_name.localeCompare(b.entity_name) * direction
      case 'issues':
        return (getIssueScore(a) - getIssueScore(b)) * direction
      case 'targets':
        return (a.target_count - b.target_count) * direction
      case 'status':
      default:
        return (STATUS_ORDER[a.status] - STATUS_ORDER[b.status]) * direction
    }
  })
}

function getIssueScore(report: ScenarioQualityReport) {
  const baseIssues =
    report.issue_counts.missing_template_sections +
    report.issue_counts.target_coverage +
    report.issue_counts.requirement_coverage +
    report.issue_counts.prd_ref
  return report.status === 'missing_prd' ? baseIssues + 10 : baseIssues
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

function SortableHeader({
  label,
  column,
  sortConfig,
  onSort,
}: {
  label: string
  column: SortKey
  sortConfig: SortConfig
  onSort: (config: SortConfig) => void
}) {
  const isActive = sortConfig.key === column
  const nextDirection = isActive && sortConfig.direction === 'asc' ? 'desc' : 'asc'
  return (
    <th
      scope="col"
      className={cn('px-4 py-3 cursor-pointer select-none', isActive ? 'text-slate-900' : '')}
      onClick={() => onSort({ key: column, direction: isActive ? nextDirection : 'asc' })}
    >
      <span className="flex items-center gap-1">
        {label}
        {isActive && (sortConfig.direction === 'asc' ? '↑' : '↓')}
      </span>
    </th>
  )
}

function ReportDetails({ report }: { report: ScenarioQualityReport }) {
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
    />
  )
}
