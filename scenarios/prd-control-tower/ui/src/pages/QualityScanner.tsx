import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import toast from 'react-hot-toast'
import {
  AlertTriangle,
  CheckCircle2,
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
import type {
  CatalogEntry,
  CatalogResponse,
  QualityScanEntity,
  QualitySummary,
  ScenarioQualityReport,
} from '../types'
import { formatDate } from '../utils/formatters'

const SCAN_STORAGE_KEY = 'prd-control-tower:quality-scan'

type ScanFilter = 'all' | 'scenario' | 'resource'
type PRDFilter = 'all' | 'with' | 'missing'
type ResultFilter = 'all' | 'issues' | 'healthy'

type StoredScanPayload = {
  generated_at: string
  reports: ScenarioQualityReport[]
}

export default function QualityScanner() {
  const [entries, setEntries] = useState<CatalogEntry[]>([])
  const [loadingCatalog, setLoadingCatalog] = useState(true)
  const [catalogError, setCatalogError] = useState<string | null>(null)
  const [search, setSearch] = useState('')
  const [typeFilter, setTypeFilter] = useState<ScanFilter>('all')
  const [prdFilter, setPrdFilter] = useState<PRDFilter>('with')
  const [selected, setSelected] = useState<Record<string, boolean>>({})
  const [scanning, setScanning] = useState(false)
  const [scanError, setScanError] = useState<string | null>(null)
  const [reports, setReports] = useState<ScenarioQualityReport[]>([])
  const [resultFilter, setResultFilter] = useState<ResultFilter>('all')
  const [lastRunAt, setLastRunAt] = useState<string | null>(null)
  const [qualitySummary, setQualitySummary] = useState<QualitySummary | null>(null)
  const [hasStoredScan, setHasStoredScan] = useState(false)

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
      if (prdFilter === 'with' && !entry.has_prd) return false
      if (prdFilter === 'missing' && entry.has_prd) return false
      if (!q) return true
      return entry.name.toLowerCase().includes(q) || entry.description.toLowerCase().includes(q)
    })
  }, [entries, search, typeFilter, prdFilter])

  const toggleSelection = (entry: CatalogEntry) => {
    const key = `${entry.type}:${entry.name}`
    setSelected((prev) => ({ ...prev, [key]: !prev[key] }))
  }

  const selectAllVisible = () => {
    setSelected((prev) => {
      const next = { ...prev }
      filteredEntries.forEach((entry) => {
        next[`${entry.type}:${entry.name}`] = true
      })
      return next
    })
  }

  const deselectAll = () => {
    setSelected({})
  }

  const selectedEntities: QualityScanEntity[] = useMemo(() => {
    return entries
      .filter((entry) => selected[`${entry.type}:${entry.name}`])
      .map((entry) => ({ type: entry.type, name: entry.name }))
  }, [entries, selected])

  const handleScan = useCallback(async () => {
    if (selectedEntities.length === 0) {
      toast.error('Select at least one scenario to scan')
      return
    }
    setScanning(true)
    setScanError(null)
    try {
      const response = await runQualityScan(selectedEntities)
      setReports(response.reports)
      setLastRunAt(response.generated_at)
      persistScan(response)
      setHasStoredScan(true)
      toast.success(`Scan complete · ${response.reports.length} entities`)
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Scan failed'
      setScanError(message)
      toast.error(message)
    } finally {
      setScanning(false)
    }
  }, [selectedEntities])

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

  const filteredReports = useMemo(() => {
    return sortReports(reports).filter((report) => {
      if (resultFilter === 'all') return true
      const hasIssues = report.issue_counts.total > 0 || report.status === 'missing_prd'
      if (resultFilter === 'issues') return hasIssues
      return !hasIssues
    })
  }, [reports, resultFilter])

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

  const qualityStats = qualitySummary
    ? [
        { label: 'Tracked', value: qualitySummary.total_entities },
        { label: 'Issues flagged', value: qualitySummary.with_issues },
        { label: 'Missing PRDs', value: qualitySummary.missing_prd },
      ]
    : []

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
              Select scenarios, scan for PRD + requirements drift, and export reports. No more clicking into every entity to find blockers.
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
            <Button className="gap-2" onClick={handleScan} disabled={scanning || selectedEntities.length === 0}>
              {scanning ? <Loader2 size={16} className="animate-spin" /> : <ScanSearch size={16} />}
              {scanning ? 'Scanning...' : selectedEntities.length > 0 ? `Scan ${selectedEntities.length} selected` : 'Scan selected'}
            </Button>
          </div>
        </div>
      </header>

      {catalogError && (
        <Card className="border-rose-200 bg-rose-50">
          <CardContent className="flex items-center gap-2 py-4 text-rose-900">
            <AlertTriangle size={18} />
            <span>{catalogError}</span>
            <Button variant="ghost" size="sm" className="ml-auto" onClick={() => window.location.reload()}>
              Retry
            </Button>
          </CardContent>
        </Card>
      )}

      <div className="grid gap-6 lg:grid-cols-[360px_1fr]">
        <Card className="h-full">
          <CardHeader>
            <CardTitle className="text-lg">Select scenarios</CardTitle>
            <p className="text-sm text-muted-foreground">Filter the catalog and choose which entities to scan.</p>
          </CardHeader>
          <CardContent className="space-y-4">
            <Input
              placeholder="Search scenarios..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
            <div className="flex gap-2 text-sm">
              <select value={typeFilter} onChange={(e) => setTypeFilter(e.target.value as ScanFilter)} className="flex-1 rounded border px-3 py-2">
                <option value="all">All types</option>
                <option value="scenario">Scenarios</option>
                <option value="resource">Resources</option>
              </select>
              <select value={prdFilter} onChange={(e) => setPrdFilter(e.target.value as PRDFilter)} className="flex-1 rounded border px-3 py-2">
                <option value="all">PRD: all</option>
                <option value="with">PRD: present</option>
                <option value="missing">PRD: missing</option>
              </select>
            </div>
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              {selectedEntities.length} selected · {filteredEntries.length} visible of {entries.length}
              <Button variant="link" size="sm" className="text-xs" onClick={selectAllVisible}>
                Select all
              </Button>
              <Button variant="link" size="sm" className="text-xs" onClick={deselectAll}>
                Clear
              </Button>
            </div>
            <div className="max-h-[420px] overflow-y-auto divide-y">
              {loadingCatalog ? (
                <div className="flex items-center gap-2 py-6 text-sm text-muted-foreground">
                  <Loader2 size={16} className="animate-spin" /> Loading catalog...
                </div>
              ) : (
                filteredEntries.map((entry) => {
                  const key = `${entry.type}:${entry.name}`
                  const isSelected = Boolean(selected[key])
                  return (
                    <label key={key} className={cn('flex cursor-pointer items-start gap-3 px-1 py-3 text-sm', isSelected ? 'bg-violet-50' : '')}>
                      <input
                        type="checkbox"
                        checked={isSelected}
                        onChange={() => toggleSelection(entry)}
                        className="mt-1"
                      />
                      <div className="space-y-1">
                        <div className="font-medium text-slate-900">
                          {entry.name}
                          {!entry.has_prd && <Badge variant="destructive" className="ml-2">No PRD</Badge>}
                        </div>
                        <p className="text-xs text-muted-foreground line-clamp-2">{entry.description || 'No description'}</p>
                      </div>
                    </label>
                  )
                })
              )}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-lg">Scan results</CardTitle>
            <p className="text-sm text-muted-foreground">Run scans to populate this space. Export data for follow-up.</p>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex flex-wrap gap-2">
              <Button variant="outline" size="sm" className="gap-2" onClick={exportJSON}>
                <Download size={14} /> Export JSON
              </Button>
              <Button variant="outline" size="sm" className="gap-2" onClick={exportCSV}>
                <Download size={14} /> Export CSV
              </Button>
              <select value={resultFilter} onChange={(e) => setResultFilter(e.target.value as ResultFilter)} className="ml-auto rounded border px-3 py-2 text-sm">
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
                <p>{reports.length} entities scanned · latest run {lastRunAt ? formatDate(lastRunAt) : 'just now'}</p>
                <p className="text-xs">Filtering {filteredReports.length} results</p>
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {filteredReports.length > 0 && (
        <div className="space-y-4">
          {filteredReports.map((report) => (
            <ReportCard key={`${report.entity_type}-${report.entity_name}`} report={report} />
          ))}
        </div>
      )}
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

function sortReports(reports: ScenarioQualityReport[]) {
  const order: Record<ScenarioQualityReport['status'], number> = {
    blocked: 0,
    missing_prd: 0,
    error: 1,
    needs_attention: 2,
    healthy: 3,
  }
  return [...reports].sort((a, b) => {
    const aOrder = order[a.status] ?? 4
    const bOrder = order[b.status] ?? 4
    if (aOrder !== bOrder) return aOrder - bOrder
    return a.entity_name.localeCompare(b.entity_name)
  })
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

    report.template_compliance_v2?.missing_sections.forEach((section: string) => {
      rows.push([report.entity_type, report.entity_name, report.status, 'template_section', section])
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

interface ReportCardProps {
  report: ScenarioQualityReport
}

function ReportCard({ report }: ReportCardProps) {
  const hasIssues = report.status !== 'healthy'
  const statusClasses: Record<ScenarioQualityReport['status'], string> = {
    healthy: 'bg-emerald-50 text-emerald-700',
    needs_attention: 'bg-amber-50 text-amber-800',
    blocked: 'bg-rose-50 text-rose-800',
    missing_prd: 'bg-rose-50 text-rose-800',
    error: 'bg-slate-100 text-slate-700',
  }

  return (
    <Card>
      <CardHeader className="flex flex-col gap-2 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <CardTitle className="text-xl text-slate-900">
            {report.entity_name}
            <Badge variant="secondary" className="ml-2 uppercase">
              {report.entity_type}
            </Badge>
          </CardTitle>
          <p className="text-sm text-muted-foreground">
            {report.requirement_count} requirements · {report.target_count} targets
          </p>
        </div>
        <div className={`rounded-full px-4 py-1 text-sm font-medium ${statusClasses[report.status]}`}>
          {report.status.replace('_', ' ')}
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex flex-wrap gap-2 text-sm">
          <Badge variant="outline">Template gaps: {report.issue_counts.missing_template_sections}</Badge>
          <Badge variant="outline">Target coverage: {report.issue_counts.target_coverage}</Badge>
          <Badge variant="outline">Requirements w/o targets: {report.issue_counts.requirement_coverage}</Badge>
          <Badge variant="outline">PRD ref: {report.issue_counts.prd_ref}</Badge>
        </div>

        {!hasIssues && (
          <div className="flex items-center gap-2 rounded-md border border-emerald-200 bg-emerald-50 p-3 text-sm text-emerald-800">
            <CheckCircle2 size={16} /> Healthy · No issues detected
          </div>
        )}

        {report.target_linkage_issues && report.target_linkage_issues.length > 0 && (
          <IssueList title="Operational targets" items={report.target_linkage_issues.map((issue) => `${issue.criticality} · ${issue.title}`)} />
        )}

        {report.requirements_without_targets && report.requirements_without_targets.length > 0 && (
          <IssueList title="Requirements missing targets" items={report.requirements_without_targets.map((req) => `${req.id} · ${req.title}`)} />
        )}

        {report.prd_ref_issues && report.prd_ref_issues.length > 0 && (
          <IssueList title="PRD references" items={report.prd_ref_issues.map((issue) => `${issue.requirement_id} · ${issue.message}`)} />
        )}

        <div className="flex flex-wrap gap-3 text-sm text-muted-foreground">
          <Link to={`/prd/${report.entity_type}/${report.entity_name}`} className="text-primary hover:underline">
            View PRD →
          </Link>
          <Link to={`/requirements/${report.entity_type}/${report.entity_name}`} className="text-primary hover:underline">
            Requirements dashboard →
          </Link>
        </div>
      </CardContent>
    </Card>
  )
}

interface IssueListProps {
  title: string
  items: string[]
}

function IssueList({ title, items }: IssueListProps) {
  return (
    <div className="space-y-2 rounded-lg border border-slate-200 bg-slate-50 p-3 text-sm">
      <p className="flex items-center gap-2 font-semibold text-slate-900">
        <AlertTriangle size={16} className="text-amber-600" /> {title}
      </p>
      <ul className="list-inside list-disc text-slate-700">
        {items.slice(0, 5).map((item, idx) => (
          <li key={`${title}-${idx}`}>{item}</li>
        ))}
        {items.length > 5 && (
          <li className="text-xs text-muted-foreground">+{items.length - 5} more</li>
        )}
      </ul>
    </div>
  )
}
