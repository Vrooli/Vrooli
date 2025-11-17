import { Link } from 'react-router-dom'
import { AlertCircle, AlertTriangle, CheckCircle2, Loader2, RefreshCw, Shield } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '../ui/card'
import { Button } from '../ui/button'
import { Badge } from '../ui/badge'
import type { ScenarioQualityReport } from '../../types'
import { formatDate } from '../../utils/formatters'

interface QualityInsightsPanelProps {
  report: ScenarioQualityReport | null
  loading: boolean
  error: string | null
  onRefresh?: () => void
}

const STATUS_STYLES: Record<ScenarioQualityReport['status'], string> = {
  healthy: 'bg-emerald-50 text-emerald-700 border border-emerald-100',
  needs_attention: 'bg-amber-50 text-amber-800 border border-amber-100',
  blocked: 'bg-rose-50 text-rose-800 border border-rose-100',
  missing_prd: 'bg-rose-50 text-rose-800 border border-rose-100',
  error: 'bg-slate-100 text-slate-700 border border-slate-200',
}

const STATUS_LABELS: Record<ScenarioQualityReport['status'], string> = {
  healthy: 'Healthy',
  needs_attention: 'Needs attention',
  blocked: 'Blocking issues',
  missing_prd: 'PRD missing',
  error: 'Error',
}

export function QualityInsightsPanel({ report, loading, error, onRefresh }: QualityInsightsPanelProps) {
  const hasIssues = (report?.issue_counts.total ?? 0) > 0

  return (
    <Card className="mt-6">
      <CardHeader className="flex flex-row items-center justify-between space-y-0">
        <div>
          <CardTitle className="flex items-center gap-2 text-xl text-slate-900">
            <Shield size={18} className="text-violet-600" /> Quality Insights
          </CardTitle>
          <p className="text-sm text-muted-foreground">
            Snapshot of structural compliance, requirement coverage, and PRD references.
          </p>
        </div>
        <div className="flex items-center gap-2">
          {report?.validated_at && (
            <span className="text-xs text-muted-foreground">
              Last verified {formatDate(report.validated_at)}
              {report.cache_used ? ' · cached' : ''}
            </span>
          )}
          <Button
            size="sm"
            variant="outline"
            onClick={onRefresh}
            disabled={loading}
            className="gap-2"
          >
            {loading ? <Loader2 size={14} className="animate-spin" /> : <RefreshCw size={14} />}
            {loading ? 'Refreshing' : 'Refresh scan'}
          </Button>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {error && (
          <div className="flex items-start gap-2 rounded-md border border-rose-200 bg-rose-50 p-3 text-sm text-rose-900">
            <AlertCircle size={16} className="mt-0.5" />
            <div>
              <p className="font-medium">Unable to load quality insights</p>
              <p>{error}</p>
            </div>
          </div>
        )}

        {loading && !report && (
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Loader2 size={16} className="animate-spin" />
            Gathering diagnostics...
          </div>
        )}

        {report && (
          <div className="space-y-4">
            <div className={`inline-flex items-center gap-2 rounded-full px-3 py-1 text-sm font-medium ${STATUS_STYLES[report.status]}`}>
              <span>{STATUS_LABELS[report.status]}</span>
              <Badge variant="secondary" className="bg-white text-slate-700">
                {report.issue_counts.total} issue{report.issue_counts.total === 1 ? '' : 's'}
              </Badge>
            </div>

            {!hasIssues && (
              <div className="flex items-center gap-2 rounded-md border border-emerald-200 bg-emerald-50 p-3 text-sm text-emerald-800">
                <CheckCircle2 size={16} />
                {report.message || 'No blocking issues detected.'}
              </div>
            )}

            {hasIssues && (
              <div className="grid gap-4 md:grid-cols-2">
                {report.template_compliance_v2 && report.issue_counts.missing_template_sections > 0 && (
                  <div className="space-y-2 rounded-lg border border-amber-100 bg-amber-50 p-3 text-sm">
                    <p className="flex items-center gap-2 font-semibold text-amber-900">
                      <AlertTriangle size={16} /> Template coverage
                    </p>
                    <ul className="list-inside list-disc text-amber-800">
                      {report.template_compliance_v2.missing_sections.slice(0, 4).map((section: string) => (
                        <li key={section}>{section}</li>
                      ))}
                      {report.template_compliance_v2.missing_sections.length > 4 && (
                        <li className="text-xs text-muted-foreground">
                          +{report.template_compliance_v2.missing_sections.length - 4} more sections
                        </li>
                      )}
                    </ul>
                  </div>
                )}

                {report.target_linkage_issues && report.target_linkage_issues.length > 0 && (
                  <div className="space-y-2 rounded-lg border border-rose-100 bg-rose-50 p-3 text-sm">
                    <p className="flex items-center gap-2 font-semibold text-rose-900">
                      <AlertTriangle size={16} /> Operational targets without requirements
                    </p>
                    <ul className="list-inside list-disc text-rose-800">
                      {report.target_linkage_issues.slice(0, 4).map((issue, idx) => (
                        <li key={`${issue.title}-${idx}`}>
                          <span className="font-medium">{issue.criticality}</span>: {issue.title}
                        </li>
                      ))}
                      {report.target_linkage_issues.length > 4 && (
                        <li className="text-xs text-muted-foreground">
                          +{report.target_linkage_issues.length - 4} more targets
                        </li>
                      )}
                    </ul>
                  </div>
                )}

                {report.requirements_without_targets && report.requirements_without_targets.length > 0 && (
                  <div className="space-y-2 rounded-lg border border-slate-200 bg-slate-50 p-3 text-sm">
                    <p className="flex items-center gap-2 font-semibold text-slate-900">
                      <AlertTriangle size={16} /> Requirements missing PRD linkage
                    </p>
                    <ul className="list-inside list-disc text-slate-700">
                      {report.requirements_without_targets.slice(0, 4).map((req) => (
                        <li key={req.id}>
                          <span className="font-medium">{req.id}</span> · {req.title}
                        </li>
                      ))}
                      {report.requirements_without_targets.length > 4 && (
                        <li className="text-xs text-muted-foreground">
                          +{report.requirements_without_targets.length - 4} more requirements
                        </li>
                      )}
                    </ul>
                  </div>
                )}

                {report.prd_ref_issues && report.prd_ref_issues.length > 0 && (
                  <div className="space-y-2 rounded-lg border border-purple-100 bg-purple-50 p-3 text-sm">
                    <p className="flex items-center gap-2 font-semibold text-purple-900">
                      <AlertTriangle size={16} /> PRD reference mismatches
                    </p>
                    <ul className="list-inside list-disc text-purple-800">
                      {report.prd_ref_issues.slice(0, 4).map((issue, idx) => (
                        <li key={`${issue.requirement_id}-${idx}`}>
                          <span className="font-medium">{issue.requirement_id}</span> · {issue.message}
                        </li>
                      ))}
                      {report.prd_ref_issues.length > 4 && (
                        <li className="text-xs text-muted-foreground">
                          +{report.prd_ref_issues.length - 4} more reference issues
                        </li>
                      )}
                    </ul>
                  </div>
                )}
              </div>
            )}

            <div className="flex flex-wrap gap-4 text-sm text-muted-foreground">
              {report.entity_type && report.entity_name && (
                <Link to={`/scenario/${report.entity_type}/${report.entity_name}?tab=requirements`} className="text-primary hover:underline">
                  Resolve in Scenario Control Center →
                </Link>
              )}
              <Link to="/quality-scanner" className="text-primary hover:underline">
                Open Quality Scanner
              </Link>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
