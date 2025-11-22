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

/**
 * Extracts a specific section from PRD content by section name.
 * Returns the content of that section (excluding the heading itself) or null if not found.
 */
export function extractSectionContent(prdContent: string, sectionName: string): { content: string; startLine: number; endLine: number } | null {
  if (!prdContent || !sectionName) {
    return null
  }

  // Normalize section name by removing emojis and standardizing
  const normalizedSectionName = sectionName.replace(/[🎯🧱🤝🎨📎🔴🟠🟢]/g, '').trim()

  // Split content into lines for processing
  const lines = prdContent.split('\n')
  let sectionStartIndex = -1
  let sectionLevel = 0

  // Find the section header
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i].trim()

    // Check if this line is a markdown header containing our section name
    const headerMatch = line.match(/^(#{1,6})\s+(.*)$/)
    if (headerMatch) {
      const level = headerMatch[1].length
      const headerText = headerMatch[2].replace(/[🎯🧱🤝🎨📎🔴🟠🟢]/g, '').trim()

      // Check if this header matches our target section
      if (headerText.toLowerCase().includes(normalizedSectionName.toLowerCase())) {
        sectionStartIndex = i
        sectionLevel = level
        break
      }
    }
  }

  if (sectionStartIndex === -1) {
    return null // Section not found
  }

  // Find the end of this section (next header of same or higher level)
  let sectionEndIndex = lines.length

  for (let i = sectionStartIndex + 1; i < lines.length; i++) {
    const line = lines[i].trim()
    const headerMatch = line.match(/^(#{1,6})\s+/)

    if (headerMatch) {
      const level = headerMatch[1].length
      // If we find a header of same or higher level (lower number), this is the end
      if (level <= sectionLevel) {
        sectionEndIndex = i
        break
      }
    }
  }

  // Extract the section content (excluding the header line itself)
  const sectionContent = lines.slice(sectionStartIndex + 1, sectionEndIndex).join('\n').trim()

  return {
    content: sectionContent,
    startLine: sectionStartIndex,
    endLine: sectionEndIndex
  }
}
