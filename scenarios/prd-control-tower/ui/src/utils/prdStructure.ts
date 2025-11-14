const REQUIRED_SECTIONS = [
  { token: '# Product Requirements Document (PRD)', label: 'Document title' },
  { token: '## 🎯 Capability Definition', label: 'Capability definition' },
  { token: '## 📊 Success Metrics', label: 'Success metrics' },
  { token: '### Functional Requirements', label: 'Functional requirements' },
  { token: '### Performance Criteria', label: 'Performance criteria' },
  { token: '### Quality Gates', label: 'Quality gates' },
  { token: '## 🏗️ Technical Architecture', label: 'Technical architecture' },
  { token: '### Resource Dependencies', label: 'Resource dependencies' },
  { token: '### Data Models', label: 'Data models' },
  { token: '### API Contract', label: 'API contract' },
  { token: '### Event Interface', label: 'Event interface' },
  { token: '## 🖥️ CLI Interface Contract', label: 'CLI contract' },
]

export interface StructureSummary {
  missingSections: string[]
  presentSections: string[]
  totalRequired: number
}

export function analyzeDraftStructure(content: string): StructureSummary {
  const normalized = content || ''
  const missing: string[] = []
  const present: string[] = []

  REQUIRED_SECTIONS.forEach((section) => {
    if (normalized.includes(section.token)) {
      present.push(section.label)
    } else {
      missing.push(section.label)
    }
  })

  return {
    missingSections: missing,
    presentSections: present,
    totalRequired: REQUIRED_SECTIONS.length,
  }
}
