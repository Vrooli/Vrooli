const REQUIRED_SECTIONS = [
  // Document structure
  { token: '# Product Requirements Document (PRD)', label: 'Document title', criticality: 'required' },

  // Capability definition
  { token: '## 🎯 Capability Definition', label: 'Capability definition section', criticality: 'required' },
  { token: '### Core Capability', label: 'Core capability subsection', criticality: 'required' },
  { token: '### Intelligence Amplification', label: 'Intelligence amplification subsection', criticality: 'required' },
  { token: '### Recursive Value', label: 'Recursive value subsection', criticality: 'required' },

  // Success metrics
  { token: '## 📊 Success Metrics', label: 'Success metrics section', criticality: 'required' },
  { token: '### Functional Requirements', label: 'Functional requirements subsection', criticality: 'required' },
  { token: '- **Must Have (P0)**', label: 'P0 requirements', criticality: 'required' },
  { token: '- **Should Have (P1)**', label: 'P1 requirements', criticality: 'recommended' },
  { token: '### Performance Criteria', label: 'Performance criteria subsection', criticality: 'required' },
  { token: '### Quality Gates', label: 'Quality gates subsection', criticality: 'required' },

  // Technical architecture
  { token: '## 🏗️ Technical Architecture', label: 'Technical architecture section', criticality: 'required' },
  { token: '### Resource Dependencies', label: 'Resource dependencies subsection', criticality: 'required' },
  { token: '### Resource Integration Standards', label: 'Resource integration standards subsection', criticality: 'required' },
  { token: '### Data Models', label: 'Data models subsection', criticality: 'recommended' },
  { token: '### API Contract', label: 'API contract subsection', criticality: 'recommended' },
  { token: '### Event Interface', label: 'Event interface subsection', criticality: 'recommended' },

  // CLI contract
  { token: '## 🖥️ CLI Interface Contract', label: 'CLI contract section', criticality: 'required' },
  { token: '### Command Structure', label: 'CLI command structure', criticality: 'required' },
  { token: '### CLI-API Parity Requirements', label: 'CLI-API parity requirements', criticality: 'required' },
  { token: '### Implementation Standards', label: 'CLI implementation standards', criticality: 'recommended' },

  // Integration requirements
  { token: '## 🔄 Integration Requirements', label: 'Integration requirements section', criticality: 'required' },
  { token: '### Upstream Dependencies', label: 'Upstream dependencies subsection', criticality: 'required' },
  { token: '### Downstream Enablement', label: 'Downstream enablement subsection', criticality: 'required' },
  { token: '### Cross-Scenario Interactions', label: 'Cross-scenario interactions subsection', criticality: 'recommended' },

  // Style and branding
  { token: '## 🎨 Style and Branding Requirements', label: 'Style and branding section', criticality: 'recommended' },
  { token: '### UI/UX Style Guidelines', label: 'UI/UX style guidelines subsection', criticality: 'recommended' },
  { token: '### Target Audience Alignment', label: 'Target audience alignment subsection', criticality: 'recommended' },
  { token: '### Brand Consistency Rules', label: 'Brand consistency rules subsection', criticality: 'recommended' },

  // Value proposition
  { token: '## 💰 Value Proposition', label: 'Value proposition section', criticality: 'required' },
  { token: '### Business Value', label: 'Business value subsection', criticality: 'required' },
  { token: '### Technical Value', label: 'Technical value subsection', criticality: 'recommended' },

  // Evolution path
  { token: '## 🧬 Evolution Path', label: 'Evolution path section', criticality: 'required' },
  { token: '### Version 1.0 (Current)', label: 'Version 1.0 subsection', criticality: 'required' },
  { token: '### Version 2.0 (Planned)', label: 'Version 2.0 subsection', criticality: 'recommended' },
  { token: '### Long-term Vision', label: 'Long-term vision subsection', criticality: 'required' },

  // Lifecycle integration
  { token: '## 🔄 Scenario Lifecycle Integration', label: 'Lifecycle integration section', criticality: 'required' },
  { token: '### Direct Scenario Deployment', label: 'Direct scenario deployment subsection', criticality: 'required' },
  { token: '### Capability Discovery', label: 'Capability discovery subsection', criticality: 'required' },
  { token: '### Version Management', label: 'Version management subsection', criticality: 'required' },

  // Risk mitigation
  { token: '## 🚨 Risk Mitigation', label: 'Risk mitigation section', criticality: 'required' },
  { token: '### Technical Risks', label: 'Technical risks subsection', criticality: 'required' },
  { token: '### Operational Risks', label: 'Operational risks subsection', criticality: 'required' },

  // Validation criteria
  { token: '## ✅ Validation Criteria', label: 'Validation criteria section', criticality: 'required' },
  { token: '### Declarative Test Specification', label: 'Declarative test specification subsection', criticality: 'required' },
  { token: '### Test Execution Gates', label: 'Test execution gates subsection', criticality: 'recommended' },
  { token: '### Performance Validation', label: 'Performance validation subsection', criticality: 'recommended' },
  { token: '### Integration Validation', label: 'Integration validation subsection', criticality: 'recommended' },
  { token: '### Capability Verification', label: 'Capability verification subsection', criticality: 'required' },

  // Implementation notes
  { token: '## 📝 Implementation Notes', label: 'Implementation notes section', criticality: 'required' },
  { token: '### Design Decisions', label: 'Design decisions subsection', criticality: 'required' },
  { token: '### Known Limitations', label: 'Known limitations subsection', criticality: 'required' },
  { token: '### Security Considerations', label: 'Security considerations subsection', criticality: 'required' },

  // References
  { token: '## 🔗 References', label: 'References section', criticality: 'required' },
  { token: '### Documentation', label: 'Documentation subsection', criticality: 'required' },
  { token: '### Related PRDs', label: 'Related PRDs subsection', criticality: 'recommended' },
  { token: '### External Resources', label: 'External resources subsection', criticality: 'recommended' },
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
