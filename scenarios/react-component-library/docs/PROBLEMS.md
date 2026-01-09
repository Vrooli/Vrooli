# Known Problems & Deferred Ideas

This document tracks open issues, technical debt, and deferred ideas for react-component-library.

## Open Issues

### Header Comment Enforcement
- **Issue**: Developers may delete or modify header comments, breaking adoption tracking
- **Impact**: Loss of adoption tracking data, inability to detect version drift
- **Mitigation**: Implement linting rules, editor warnings, and validation on save
- **Status**: Not yet implemented - P0 requirement

### AI Code Hallucination
- **Issue**: AI-generated code suggestions may introduce bugs or break components
- **Impact**: Component quality degradation, potential runtime errors
- **Mitigation**: Require diff review before applying AI patches, add automated testing
- **Status**: P1 feature planned - patch review and merge workflow

### Editor/Preview Performance
- **Issue**: Large component files may cause editor lag or slow preview refresh
- **Impact**: Degraded developer experience
- **Mitigation**: Implement lazy loading, code splitting, optimized recompilation
- **Status**: Monitor during implementation, optimize as needed

### Component Version Drift
- **Issue**: Adopted components may diverge from library versions without detection
- **Impact**: Inconsistent UI across scenarios, maintenance burden
- **Mitigation**: P1 change detection and diff views, automated adoption verification
- **Status**: Planned for P1 phase

## Deferred Ideas

### Bulk Component Updates
- **Description**: Push library component updates to multiple adopted scenarios with automatic PR creation
- **Reason**: Complex workflow requiring git integration and multi-scenario coordination
- **Target**: P2

### Visual Regression Testing
- **Description**: Auto-capture screenshots across viewports for visual comparison
- **Reason**: Requires integration with screenshot testing infrastructure
- **Target**: P2

### Component Playground Sharing
- **Description**: Generate shareable URLs for component previews with specific props/viewports
- **Reason**: Requires URL state management and component serialization
- **Target**: P2

### Figma/Design Tool Export
- **Description**: Generate design assets or Figma components from library components
- **Reason**: Complex integration requiring Figma API and asset generation pipeline
- **Target**: P2

### Multi-Framework Support
- **Description**: Support Vue, Svelte, or other frameworks beyond React
- **Reason**: Significant architectural changes required, React-first approach for MVP
- **Target**: Future consideration, not on current roadmap

## Technical Debt

_This section will be populated during implementation as technical debt is identified._

## Questions & Uncertainties

### Code Editor Choice
- **Question**: Monaco Editor vs CodeMirror for code editing?
- **Considerations**:
  - Monaco: Full VS Code features, larger bundle size
  - CodeMirror: Lighter weight, more customizable
- **Decision**: To be determined during P0 implementation

### Version Strategy
- **Question**: Semver (1.2.3) vs incremental (v1, v2) for component versioning?
- **Considerations**:
  - Semver: Industry standard, semantic meaning
  - Incremental: Simpler, easier to manage
- **Decision**: Support both, let library maintainers choose

### Component File Location
- **Question**: Where should library component files live on disk?
- **Options**:
  - `/scenarios/react-component-library/library/components/`
  - `/scenarios/react-component-library/components/`
  - Separate top-level `/component-library/` directory
- **Decision**: To be determined, likely within scenario directory for simplicity

## Notes for Future Improvers

- Monitor AI suggestion quality and iterate on system prompts to reduce hallucinations
- Consider integration with brand-manager for design token synchronization
- Explore integration with test-genie for automated component test generation
- Evaluate need for component dependency graph visualization (P2 feature)
