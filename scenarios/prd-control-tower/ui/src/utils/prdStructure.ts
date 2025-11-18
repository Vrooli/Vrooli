import type { PRDValidationResultV2 } from '../types'

// PRD template structure validation aligned with scripts/scenarios/templates/react-vite/PRD.md
// This validation focuses on CORE required sections while making subsections optional/recommended

const REQUIRED_SECTIONS = [
  { token: '# Product Requirements Document (PRD)', label: 'Document title', criticality: 'required' },
  { token: '## 🎯 Overview', label: 'Overview section', criticality: 'required' },
  { token: '## 🎯 Operational Targets', label: 'Operational Targets section', criticality: 'required' },
  { token: '### 🔴 P0 – Must ship for viability', label: 'P0 checklist', criticality: 'required' },
  { token: '### 🟠 P1 – Should have post-launch', label: 'P1 checklist', criticality: 'required' },
  { token: '### 🟢 P2 – Future / expansion', label: 'P2 checklist', criticality: 'required' },
  { token: '## 🧱 Tech Direction Snapshot', label: 'Tech Direction section', criticality: 'required' },
  { token: '## 🤝 Dependencies & Launch Plan', label: 'Dependencies & Launch Plan section', criticality: 'required' },
  { token: '## 🎨 UX & Branding', label: 'UX & Branding section', criticality: 'required' },
  { token: '## 📎 Appendix', label: 'Appendix (optional)', criticality: 'recommended' },
] as const

export interface SectionStatus {
  label: string
  criticality: 'required' | 'recommended'
  present: boolean
}

export interface StructureSummary {
  missingSections: string[]
  presentSections: string[]
  totalRequired: number
  missingRequired: SectionStatus[]
  missingRecommended: SectionStatus[]
  completenessPercent: number
}

export function analyzeDraftStructure(content: string): StructureSummary {
  const normalized = content || ''
  const missing: string[] = []
  const present: string[] = []
  const missingRequired: SectionStatus[] = []
  const missingRecommended: SectionStatus[] = []

  let requiredCount = 0
  let requiredPresent = 0

  REQUIRED_SECTIONS.forEach((section) => {
    const isPresent = normalized.includes(section.token)

    if (isPresent) {
      present.push(section.label)
      if (section.criticality === 'required') {
        requiredPresent++
      }
    } else {
      missing.push(section.label)
      const status: SectionStatus = {
        label: section.label,
        criticality: section.criticality,
        present: false,
      }

      if (section.criticality === 'required') {
        missingRequired.push(status)
      } else {
        missingRecommended.push(status)
      }
    }

    if (section.criticality === 'required') {
      requiredCount++
    }
  })

  const completenessPercent = requiredCount > 0 ? Math.round((requiredPresent / requiredCount) * 100) : 0

  return {
    missingSections: missing,
    presentSections: present,
    totalRequired: REQUIRED_SECTIONS.length,
    missingRequired,
    missingRecommended,
    completenessPercent,
  }
}

export function flattenMissingTemplateSections(result?: PRDValidationResultV2 | null): string[] {
  if (!result) {
    return []
  }

  const missing: string[] = [...result.missing_sections]
  const subsectionEntries = Object.entries(result.missing_subsections || {})

  subsectionEntries.forEach(([parent, subsections]) => {
    subsections.forEach((subsection) => {
      missing.push(`${parent} → ${subsection}`)
    })
  })

  return missing
}
