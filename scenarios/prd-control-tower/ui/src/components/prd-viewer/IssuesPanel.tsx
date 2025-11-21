import { AlertTriangle } from 'lucide-react'
import type { ScenarioQualityReport } from '../../types'
import { formatDate } from '../../utils/formatters'
import { flattenMissingTemplateSections } from '../../utils/prdStructure'
import { IssuesSummaryCard, type IssueCategory, type IssueFooterAction, type IssueTone } from '../issues'

interface IssuesPanelProps {
  report: ScenarioQualityReport | null
  loading: boolean
  error: string | null
  onRefresh?: () => void
  onReportIssues?: () => void
  onReportCategory?: (categoryId: string) => void
}

export const STATUS_LABELS: Record<ScenarioQualityReport['status'], string> = {
  healthy: 'Healthy',
  needs_attention: 'Needs attention',
  blocked: 'Blocking issues',
  missing_prd: 'PRD missing',
  error: 'Error',
}

export const STATUS_TONES: Record<ScenarioQualityReport['status'], IssueTone> = {
  healthy: 'success',
  needs_attention: 'warning',
  blocked: 'critical',
  missing_prd: 'critical',
  error: 'warning',
}

export function IssuesPanel({ report, loading, error, onRefresh, onReportIssues, onReportCategory }: IssuesPanelProps) {
  const hasIssues = (report?.issue_counts.total ?? 0) > 0
  const categories = report ? buildQualityIssueCategories(report, { onReportCategory }) : []

  const footerActions: IssueFooterAction[] = []
  if (report?.entity_type && report.entity_name) {
    footerActions.push({
      label: 'Resolve in Scenario Control Center →',
      to: `/scenario/${report.entity_type}/${report.entity_name}?tab=requirements`,
    })
  }
  footerActions.push({ label: 'Open Quality Scanner', to: '/quality-scanner' })

  return (
    <IssuesSummaryCard
      subtitle="Snapshot of structural compliance, requirement coverage, and PRD references."
      metadata={
        report?.validated_at ? (
          <span>
            Last verified {formatDate(report.validated_at)}
            {report.cache_used ? ' · cached' : ''}
          </span>
        ) : undefined
      }
      loading={!report && loading}
      error={error}
      onRefresh={onRefresh}
      refreshing={loading}
      refreshLabel="Refresh scan"
      statusLabel={report ? STATUS_LABELS[report.status] : undefined}
      statusTone={report ? STATUS_TONES[report.status] : 'info'}
      issueCount={report?.issue_counts.total}
      statusMessage={report && !hasIssues ? report.message || 'No blocking issues detected.' : undefined}
      categories={categories}
      footerActions={footerActions}
      primaryAction={
        onReportIssues
          ? {
              label: 'Report issues',
              onClick: onReportIssues,
            }
          : undefined
      }
    />
  )
}

export function buildQualityIssueCategories(
  report: ScenarioQualityReport,
  options?: { onReportCategory?: (categoryId: string) => void },
): IssueCategory[] {
  const missingTemplateSections = flattenMissingTemplateSections(report.template_compliance_v2)
  const unexpectedSections = report.template_compliance_v2?.unexpected_sections ?? []
  const categories: IssueCategory[] = []

  if (report.template_compliance_v2 && report.issue_counts.missing_template_sections > 0) {
    const sections = []
    if (missingTemplateSections.length > 0) {
      sections.push({
        id: 'missing-sections',
        title: 'Missing sections',
        items: missingTemplateSections,
        maxVisible: 4,
      })
    }
    if (unexpectedSections.length > 0) {
      sections.push({
        id: 'unexpected-sections',
        title: 'Unexpected sections',
        items: unexpectedSections,
        maxVisible: 4,
      })
    }
    categories.push({
      id: 'structure',
      title: (
        <span className="flex items-center gap-2">
          <AlertTriangle size={16} /> PRD structure issues
        </span>
      ),
      tone: 'warning',
      sections,
      action: options?.onReportCategory
        ? { label: 'Report', onClick: () => options.onReportCategory?.('structure') }
        : undefined,
    })
  }

  if (report.target_linkage_issues && report.target_linkage_issues.length > 0) {
    categories.push({
      id: 'targets',
      title: 'Operational targets without requirements',
      tone: 'critical',
      items: report.target_linkage_issues.map((issue) => `${issue.criticality}: ${issue.title}`),
      action: options?.onReportCategory
        ? { label: 'Report', onClick: () => options.onReportCategory?.('targets') }
        : undefined,
    })
  }

  if (report.requirements_without_targets && report.requirements_without_targets.length > 0) {
    categories.push({
      id: 'requirements',
      title: 'Requirements missing PRD linkage',
      tone: 'info',
      items: report.requirements_without_targets.map((req) => `${req.id} · ${req.title}`),
      action: options?.onReportCategory
        ? { label: 'Report', onClick: () => options.onReportCategory?.('requirements') }
        : undefined,
    })
  }

  if (report.prd_ref_issues && report.prd_ref_issues.length > 0) {
    categories.push({
      id: 'references',
      title: 'PRD reference mismatches',
      tone: 'info',
      items: report.prd_ref_issues.map((issue) => `${issue.requirement_id} · ${issue.message}`),
      action: options?.onReportCategory
        ? { label: 'Report', onClick: () => options.onReportCategory?.('references') }
        : undefined,
    })
  }

  // Documentation issues (empty sections, content quality issues)
  const contentIssues = report.template_compliance_v2?.content_issues ?? []
  const requirementsReadmeIssues = report.requirements_readme_issues ?? []
  const allDocIssues = [...contentIssues, ...requirementsReadmeIssues]

  if (allDocIssues.length > 0) {
    const sections = []

    if (contentIssues.length > 0) {
      sections.push({
        id: 'content-issues',
        title: 'PRD content issues',
        items: contentIssues.map((issue) => `${issue.section}: ${issue.message}`),
        maxVisible: 4,
      })
    }

    if (requirementsReadmeIssues.length > 0) {
      sections.push({
        id: 'readme-issues',
        title: 'Requirements README issues',
        items: requirementsReadmeIssues,
        maxVisible: 4,
      })
    }

    categories.push({
      id: 'documentation',
      title: (
        <span className="flex items-center gap-2">
          <AlertTriangle size={16} /> Documentation issues
        </span>
      ),
      tone: 'warning',
      sections,
      action: options?.onReportCategory
        ? { label: 'Report', onClick: () => options.onReportCategory?.('documentation') }
        : undefined,
    })
  }

  return categories
}
