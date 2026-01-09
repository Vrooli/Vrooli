# Research Notes

This document captures research findings, uniqueness checks, and external references for react-component-library.

## Uniqueness Check

### Within Vrooli Repository
- **Search Date**: 2025-11-22
- **Search Pattern**: `react-component-library`, `component library`, `design system`, `UI library`
- **Findings**:
  - No existing `react-component-library` scenario found
  - Found artifacts in `app-issue-tracker/artifacts/` suggesting prior planning or testing
  - No other scenarios provide centralized component library management
  - Several scenarios could benefit from shared component library (tidiness-manager, landing-manager, deployment-manager)

### Related Vrooli Scenarios

#### app-monitor
- **Relevance**: Reference implementation for iframe-bridge integration and emulator patterns
- **Overlap**: Uses iframe-bridge for element inspection, has preview/emulator functionality
- **Differentiation**: app-monitor focuses on monitoring running scenarios, not component editing/management
- **Learnings**:
  - iframe-bridge implementation patterns
  - Multi-viewport emulator architecture
  - Element selection and inspection UI patterns

#### app-issue-tracker
- **Relevance**: Integration target for component adoption workflow
- **Overlap**: Issue creation and tracking
- **Differentiation**: react-component-library generates adoption issues, app-issue-tracker manages them
- **Integration**: Will use app-issue-tracker API to create detailed adoption reports for coding agents

#### browser-automation-studio
- **Relevance**: Template reference for React + Vite + Go stack
- **Overlap**: Similar tech stack, lifecycle patterns
- **Differentiation**: Focuses on browser automation, not component management
- **Learnings**: UI structure, API patterns, testing organization

## External References

### Component Library Patterns

#### shadcn/ui
- **URL**: https://ui.shadcn.com/
- **Relevance**: Component architecture and structure patterns
- **Learnings**:
  - Component registry with metadata
  - Copy/paste adoption model (similar to our "apply to scenario")
  - Header comment pattern for component metadata
  - Radix UI primitives for accessibility

#### Storybook
- **URL**: https://storybook.js.org/
- **Relevance**: Component story/demo file patterns
- **Learnings**:
  - Story-based component development
  - Multi-viewport preview
  - Component documentation patterns
- **Future Integration**: P2 feature for component story files

#### Radix UI
- **URL**: https://www.radix-ui.com/
- **Relevance**: Accessible component primitives used by shadcn
- **Learnings**:
  - Accessibility-first component design
  - Headless component architecture
  - Keyboard navigation patterns

### Code Editor Solutions

#### Monaco Editor
- **URL**: https://microsoft.github.io/monaco-editor/
- **Relevance**: Potential code editor implementation
- **Pros**: Full VS Code features, excellent TypeScript/TSX support, widely used
- **Cons**: Larger bundle size (~2-3MB), more complex integration
- **Use Cases**: VS Code, Azure DevOps, StackBlitz

#### CodeMirror 6
- **URL**: https://codemirror.net/
- **Relevance**: Alternative code editor option
- **Pros**: Lighter weight, highly customizable, modern architecture
- **Cons**: Less feature-complete than Monaco out of box
- **Use Cases**: Replit, Observable, many lightweight editors

### AI Code Editing References

#### GitHub Copilot
- **Relevance**: AI-powered code suggestions and completions
- **Learnings**: Context-aware suggestions, inline diff display, accept/reject workflow

#### Cursor
- **Relevance**: AI-first code editor with chat and inline editing
- **Learnings**: Chat-to-code workflow, patch review UI, multi-file context

#### Codeium
- **Relevance**: AI code completion and chat
- **Learnings**: Free AI code assistance, diff view patterns

## Domain Research

### Component Library Management
- **Problem**: Component duplication across projects leads to inconsistency, maintenance burden
- **Solution**: Centralized library with version tracking, adoption workflow, change detection
- **Industry Examples**: Design systems at Airbnb (React Dates), Shopify (Polaris), Adobe (Spectrum)

### Multi-Viewport Testing
- **Problem**: Responsive design requires testing across many viewport sizes
- **Industry Tools**: Browser DevTools, Responsively App, BrowserStack
- **Our Approach**: Simultaneous multi-frame preview inspired by browser dev tools and app-monitor

### AI-Powered Code Editing
- **Problem**: Manual component refactoring is time-consuming and error-prone
- **Industry Trend**: AI-assisted development (GitHub Copilot, Cursor, Codeium)
- **Our Approach**: Context-aware AI editing via resource-openrouter with diff review workflow

## Technical Constraints

### iframe-bridge Integration
- **Requirement**: Components must render in iframes for isolation and element selection
- **Implication**: Need iframe-friendly build configuration, cross-origin communication
- **Reference**: app-monitor implementation

### Header Comment Parsing
- **Requirement**: Components must have standardized header comments
- **Implication**: Need robust parsing, validation, and enforcement
- **Challenge**: Preventing developer deletion or modification

### Version Tracking
- **Requirement**: Track component versions and detect drift
- **Implication**: Need file system scanning, content hashing, diff generation
- **Complexity**: Balancing performance with accuracy

## Open Research Questions

1. **Code Editor Performance**: How to handle large component files (>1000 lines) without lag?
2. **AI Context Limits**: What's the optimal context size for AI code suggestions? How to handle large components?
3. **Adoption Verification**: How to reliably verify component adoption across scenarios without tight coupling?
4. **Version Strategy**: Should we support automatic version bumping based on changes, or require manual version updates?

## Related Literature & Blog Posts

_To be populated as relevant articles and resources are discovered during implementation._

## Future Research Areas

- Component dependency graph generation and visualization
- Design token integration with brand-manager or similar systems
- Automated component test generation with test-genie
- Component usage analytics and ROI tracking
- Visual regression testing integration
