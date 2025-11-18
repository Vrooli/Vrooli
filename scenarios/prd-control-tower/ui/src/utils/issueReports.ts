import type { IssueReportCategorySeed, IssueReportSeed, ScenarioQualityReport } from '../types'
import { flattenMissingTemplateSections } from './prdStructure'
import { formatDate } from './formatters'

function makeSelectionId(prefix: string, index: number) {
  return `${prefix}-${index}`
}

function deriveSeverity(criticality?: string): 'critical' | 'high' | 'medium' | 'low' {
  const normalized = (criticality ?? '').toUpperCase()
  if (normalized === 'P0') return 'critical'
  if (normalized === 'P1') return 'high'
  return 'medium'
}

function buildStructureCategories(report: ScenarioQualityReport): IssueReportCategorySeed[] {
  const categories: IssueReportCategorySeed[] = []
  if (report.status === 'missing_prd') {
    categories.push({
      id: 'missing_prd',
      title: 'PRD missing',
      severity: 'critical',
      items: [
        {
          id: 'missing-prd',
          title: 'PRD not detected',
          detail: 'No PRD.md found for this scenario. Create a PRD to re-run validations.',
          category: 'structure',
          severity: 'critical',
        },
      ],
    })
    return categories
  }

  const missingSections = flattenMissingTemplateSections(report.template_compliance_v2)
  if (missingSections.length > 0) {
    categories.push({
      id: 'structure_missing',
      title: 'Missing PRD sections',
      severity: 'high',
      items: missingSections.map((section, index) => ({
        id: makeSelectionId('missing-section', index),
        title: section,
        detail: `${section} section missing from PRD`,
        category: 'structure',
        severity: 'high',
      })),
    })
  }

  const unexpectedSections = report.template_compliance_v2?.unexpected_sections ?? []
  if (unexpectedSections.length > 0) {
    categories.push({
      id: 'structure_unexpected',
      title: 'Unexpected sections',
      severity: 'low',
      items: unexpectedSections.map((section, index) => ({
        id: makeSelectionId('unexpected-section', index),
        title: section,
        detail: `${section} section present but not part of template`,
        category: 'structure',
        severity: 'low',
      })),
    })
  }

  return categories
}

function buildTargetCategories(report: ScenarioQualityReport): IssueReportCategorySeed[] {
  const categories: IssueReportCategorySeed[] = []
  if (report.target_linkage_issues && report.target_linkage_issues.length > 0) {
    categories.push({
      id: 'target_linkage',
      title: 'Operational targets lacking PRD references',
      severity: 'high',
      items: report.target_linkage_issues.map((issue, index) => ({
        id: makeSelectionId('target', index),
        title: issue.title,
        detail: issue.message,
        category: 'targets',
        severity: deriveSeverity(issue.criticality),
      })),
    })
  }

  if (report.requirements_without_targets && report.requirements_without_targets.length > 0) {
    categories.push({
      id: 'requirements_without_targets',
      title: 'Requirements missing target coverage',
      severity: 'medium',
      items: report.requirements_without_targets.map((req, index) => ({
        id: makeSelectionId('requirement-gap', index),
        title: req.title,
        detail: `Requirement ${req.id} has no linked operational target`,
        category: 'requirements',
        severity: 'medium',
        reference: req.id,
      })),
    })
  }

  return categories
}

function buildReferenceCategories(report: ScenarioQualityReport): IssueReportCategorySeed[] {
  if (!report.prd_ref_issues || report.prd_ref_issues.length === 0) {
    return []
  }

  return [
    {
      id: 'prd_reference',
      title: 'PRD reference mismatches',
      severity: 'medium',
      items: report.prd_ref_issues.map((issue, index) => ({
        id: makeSelectionId('prd-ref', index),
        title: issue.requirement_id,
        detail: issue.message,
        category: 'prd_reference',
        severity: 'medium',
      })),
    },
  ]
}

export function buildIssueReportSeedFromQualityReport(
  report: ScenarioQualityReport,
  options?: { source?: string },
): IssueReportSeed {
  const source = options?.source ?? 'quality_scanner'
  const metadata: Record<string, string> = {
    status: report.status.replace('_', ' '),
    last_scanned: formatDate(report.validated_at),
    prd_path: report.prd_path || '',
    requirements_path: report.requirements_path || '',
    cache_used: report.cache_used ? 'true' : 'false',
  }
  if (!metadata.prd_path) delete metadata.prd_path
  if (!metadata.requirements_path) delete metadata.requirements_path

  const categories: IssueReportCategorySeed[] = [
    ...buildStructureCategories(report),
    ...buildTargetCategories(report),
    ...buildReferenceCategories(report),
  ].filter((category) => category.items.length > 0)

  const description = `Quality scan detected ${report.issue_counts.total} structural or linkage issue${
    report.issue_counts.total === 1 ? '' : 's'
  } for ${report.entity_name} on ${formatDate(report.validated_at)}.`

  return {
    entity_type: report.entity_type,
    entity_name: report.entity_name,
    display_name: report.entity_name,
    source,
    title: `Quality review for ${report.entity_name}`,
    description,
    summary: report.message,
    tags: ['quality_scan', report.status],
    metadata,
    categories,
  }
}

export function buildIssueReportSeedForCategories(
  report: ScenarioQualityReport,
  categoryIds: string | string[],
  options?: { source?: string },
): IssueReportSeed | null {
  const ids = Array.isArray(categoryIds) ? categoryIds : [categoryIds]
  const base = buildIssueReportSeedFromQualityReport(report, options)
  const categories = base.categories.filter((entry) => ids.includes(entry.id))
  if (categories.length === 0) {
    return null
  }
  return {
    ...base,
    title: `${base.title} – ${categories.map((entry) => entry.title).join(', ')}`,
    categories,
  }
}
