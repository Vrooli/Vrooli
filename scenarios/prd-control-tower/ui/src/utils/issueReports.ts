import type { IssueReportCategorySeed, IssueReportSeed, ScenarioQualityReport } from '../types'
import { flattenMissingTemplateSections } from './prdStructure'
import { formatDate } from './formatters'

function makeSelectionId(prefix: string, index: number) {
  return `${prefix}-${index}`
}

function summarizeList(items: string[], limit = 3) {
  if (items.length === 0) return ''
  const displayed = items.slice(0, limit)
  const remainder = items.length - displayed.length
  return `${displayed.join(', ')}${remainder > 0 ? ` +${remainder} more` : ''}`
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

  const issueCount = report.issue_counts.total
  const summaryLine =
    report.status === 'missing_prd'
      ? `Quality scanner could not find a PRD or requirements file for ${report.entity_name} during the run on ${formatDate(report.validated_at)}.`
      : `Quality scanner flagged ${issueCount} structural or linkage issue${issueCount === 1 ? '' : 's'} for ${report.entity_name} on ${formatDate(report.validated_at)}.`

  const missingSections = flattenMissingTemplateSections(report.template_compliance_v2)
  const unexpectedSections = report.template_compliance_v2?.unexpected_sections ?? []
  const targetIssues = report.target_linkage_issues ?? []
  const orphanRequirements = report.requirements_without_targets ?? []
  const prdReferenceIssues = report.prd_ref_issues ?? []

  const keyFindings: string[] = []
  if (report.status === 'missing_prd') {
    keyFindings.push('PRD.md was not detected in the scenario directory, so deeper structural checks were skipped.')
    if (!report.has_requirements) {
      keyFindings.push('requirements.json is also missing which blocks requirement-to-target coverage checks.')
    }
  } else {
    if (missingSections.length > 0) {
      keyFindings.push(
        `Template coverage incomplete: ${missingSections.length} required section${missingSections.length === 1 ? '' : 's'} missing (${summarizeList(missingSections)}).`,
      )
    }
    if (unexpectedSections.length > 0) {
      keyFindings.push(
        `Unexpected sections present (${summarizeList(unexpectedSections)}). Align PRD.md with the canonical template to keep navigation deterministic.`,
      )
    }
    if (targetIssues.length > 0) {
      keyFindings.push(
        `Operational targets referencing invalid or missing requirements (${targetIssues.length} impacted, e.g. ${summarizeList(targetIssues.map((issue) => issue.title))}).`,
      )
    }
    if (orphanRequirements.length > 0) {
      keyFindings.push(
        `${orphanRequirements.length} requirement${orphanRequirements.length === 1 ? '' : 's'} have no operational target coverage (e.g. ${summarizeList(orphanRequirements.map((req) => req.id || req.title))}).`,
      )
    }
    if (prdReferenceIssues.length > 0) {
      keyFindings.push(
        `PRD reference mismatches detected for ${prdReferenceIssues.length} requirement${prdReferenceIssues.length === 1 ? '' : 's'} (sample: ${summarizeList(prdReferenceIssues.map((issue) => issue.requirement_id))}).`,
      )
    }
  }

  const remediationSteps: string[] = []
  if (report.status === 'missing_prd') {
    remediationSteps.push('Create PRD.md (use the canonical template in docs/prd-template) under the scenario root and commit it to the repo.')
    remediationSteps.push('Add requirements.json so quality scanner can map requirements to operational targets before the next release gate.')
  } else {
    if (missingSections.length > 0 || !report.has_prd) {
      remediationSteps.push('Restore every required section in PRD.md and keep headings identical to the template so future diffs stay machine-readable.')
    }
    if (targetIssues.length > 0 || orphanRequirements.length > 0) {
      remediationSteps.push('Link every requirement to at least one operational target (Operational Targets panel → link requirement ID) and update targets missing references.')
    }
    if (prdReferenceIssues.length > 0) {
      remediationSteps.push('Normalize requirement checkboxes / IDs in the PRD references table so they exactly match requirements.json entries.')
    }
  }

  remediationSteps.push('Rerun PRD Control Tower → Quality Scanner after fixes to confirm the issue counter reaches zero.')

  const metadataLines: string[] = []
  if (report.prd_path) metadataLines.push(`- PRD path: ${report.prd_path}`)
  if (report.requirements_path) metadataLines.push(`- Requirements path: ${report.requirements_path}`)
  metadataLines.push(`- Cache used: ${report.cache_used ? 'yes' : 'no'}`)

  const descriptionSections = [summaryLine]
  if (report.message) {
    descriptionSections.push(`Latest status: ${report.message}`)
  }
  if (keyFindings.length > 0) {
    descriptionSections.push(['### Key findings', '', ...keyFindings.map((finding) => `- ${finding}`)].join('\n'))
  }
  if (remediationSteps.length > 0) {
    descriptionSections.push([
      '### Required remediation',
      '',
      ...remediationSteps.map((step, index) => `${index + 1}. ${step}`),
    ].join('\n'))
  }
  descriptionSections.push(['### Validation context', '', ...metadataLines].join('\n'))

  const description = descriptionSections.filter(Boolean).join('\n\n')

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
