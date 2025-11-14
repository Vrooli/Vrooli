// PRD template structure validation aligned with scripts/scenarios/templates/react-vite/PRD.md
// This validation focuses on CORE required sections while making subsections optional/recommended

const REQUIRED_SECTIONS = [
  // Document structure
  { token: '# Product Requirements Document (PRD)', label: 'Document title', criticality: 'required' },

  // Core sections (required)
  { token: '## 🎯 Capability Definition', label: 'Capability Definition section', criticality: 'required' },
  { token: '## 📊 Success Metrics', label: 'Success Metrics section', criticality: 'required' },
  { token: '## 🏗️ Technical Architecture', label: 'Technical Architecture section', criticality: 'required' },
  { token: '## 🖥️ CLI Interface Contract', label: 'CLI Interface Contract section', criticality: 'required' },
  { token: '## 🔄 Integration Requirements', label: 'Integration Requirements section', criticality: 'required' },
  { token: '## 💰 Value Proposition', label: 'Value Proposition section', criticality: 'required' },
  { token: '## 🧬 Evolution Path', label: 'Evolution Path section', criticality: 'required' },
  { token: '## 🔄 Scenario Lifecycle Integration', label: 'Scenario Lifecycle Integration section', criticality: 'required' },
  { token: '## 🚨 Risk Mitigation', label: 'Risk Mitigation section', criticality: 'required' },
  { token: '## ✅ Validation Criteria', label: 'Validation Criteria section', criticality: 'required' },
  { token: '## 📝 Implementation Notes', label: 'Implementation Notes section', criticality: 'required' },
  { token: '## 🔗 References', label: 'References section', criticality: 'required' },

  // Important subsections (recommended but not required)
  { token: '### Core Capability', label: 'Core Capability subsection', criticality: 'recommended' },
  { token: '### Intelligence Amplification', label: 'Intelligence Amplification subsection', criticality: 'recommended' },
  { token: '### Recursive Value', label: 'Recursive Value subsection', criticality: 'recommended' },
  { token: '### Functional Requirements', label: 'Functional Requirements subsection', criticality: 'recommended' },
  { token: '- **Must Have (P0)**', label: 'P0 requirements checklist', criticality: 'recommended' },
  { token: '### Performance Criteria', label: 'Performance Criteria subsection', criticality: 'recommended' },
  { token: '### Quality Gates', label: 'Quality Gates subsection', criticality: 'recommended' },
  { token: '### Resource Dependencies', label: 'Resource Dependencies subsection', criticality: 'recommended' },
  { token: '### Command Structure', label: 'CLI Command Structure subsection', criticality: 'recommended' },
  { token: '### Business Value', label: 'Business Value subsection', criticality: 'recommended' },
  { token: '### Version 1.0 (Current)', label: 'Version 1.0 subsection', criticality: 'recommended' },
  { token: '### Technical Risks', label: 'Technical Risks subsection', criticality: 'recommended' },
  { token: '### Design Decisions', label: 'Design Decisions subsection', criticality: 'recommended' },
  { token: '### Documentation', label: 'Documentation references subsection', criticality: 'recommended' },
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
