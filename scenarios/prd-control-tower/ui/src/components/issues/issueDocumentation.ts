export interface DocumentationLink {
  title: string
  path: string
  summary: string
}

const canonicalTemplateDoc: DocumentationLink = {
  title: 'Canonical PRD Template',
  path: 'scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md',
  summary: 'Defines the required PRD.md sections, emoji headings, and editing guardrails enforced by the quality scanner.',
}

const requirementsGuideDoc: DocumentationLink = {
  title: 'Requirements Registry README',
  path: 'scenarios/prd-control-tower/requirements/README.md',
  summary: 'Explains how requirements/index.json, imports, and operational target coverage flow are wired for PRD Control Tower.',
}

const integrationRequirementsDoc: DocumentationLink = {
  title: 'Operational Target Integration Requirements',
  path: 'scenarios/prd-control-tower/requirements/requirements/integration.json',
  summary: 'Captures PCT-REQ-TARGETS and PCT-REQ-LINKAGE expectations for parsing targets, requirements, and reference alignment.',
}

export const ISSUE_DOCUMENTATION_LIBRARY: Record<string, DocumentationLink[]> = {
  missing_prd: [canonicalTemplateDoc, requirementsGuideDoc],
  structure_missing: [canonicalTemplateDoc],
  structure_unexpected: [canonicalTemplateDoc],
  target_linkage: [requirementsGuideDoc, integrationRequirementsDoc],
  requirements_without_targets: [requirementsGuideDoc, integrationRequirementsDoc],
  prd_reference: [requirementsGuideDoc, integrationRequirementsDoc, canonicalTemplateDoc],
}
