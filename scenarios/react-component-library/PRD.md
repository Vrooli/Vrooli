# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md`
> **Validation**: Enforced by `prd-control-tower` + `scenario-auditor`
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

**Purpose**: Central UI for designing, previewing, editing, and tracking shared React UI components across the Vrooli ecosystem. This scenario provides a comprehensive workflow hub for component creation, AI-powered refinement, multi-viewport testing, and adoption tracking.

**Primary users/verticals**:
- Frontend developers building and maintaining UI components
- Design system managers coordinating shared component libraries
- Coding agents implementing component adoption via app-issue-tracker
- Product teams ensuring consistent UI/UX across scenarios

**Deployment surfaces**:
- Web UI with live preview, multi-viewport emulator, and code editor
- REST API for component registry, versioning, and adoption tracking
- CLI for component queries and adoption workflow automation
- Integration with app-issue-tracker for adoption issue generation

**Value promise**: Eliminates component duplication across scenarios, accelerates UI development with AI-powered editing, ensures design consistency through shared library, and enables systematic component evolution with version tracking and diff views.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Component Registry with Metadata | Maintain registry of shared components with ID, name, description, tags, version, file path, and tech stack
- [ ] OT-P0-002 | Header Comment Enforcement | Enforce standardized header comment block in all library components containing metadata and adoption tracking info
- [ ] OT-P0-003 | Component Indexing from Disk | Re-index library components from disk using header comment metadata without DB-only reliance
- [ ] OT-P0-004 | Code Editor with TSX Support | Provide full-featured code editor with syntax highlighting, lint/type feedback, and undo/redo
- [ ] OT-P0-005 | Live Preview Renderer | Render components in isolated iframe with configurable props/variants and auto-refresh on changes
- [ ] OT-P0-006 | Multi-Frame Emulator | Support simultaneous preview frames with independent viewport settings (desktop, mobile, tablet presets)
- [ ] OT-P0-007 | iframe-bridge Element Selection | Enable element selection mode allowing users to click and capture stable element identifiers
- [ ] OT-P0-008 | AI Editing via resource-openrouter | Connect AI chat panel to editor with context-aware code suggestions, refactors, and styling changes
- [ ] OT-P0-009 | Search and Tag Filtering | Provide searchable component library with free-text search and category/tag filtering
- [ ] OT-P0-010 | Apply to Scenario Workflow | Generate detailed adoption reports via app-issue-tracker with absolute paths, dependencies, and step-by-step integration instructions
- [ ] OT-P0-011 | Adoption Tracking Records | Track which scenarios have adopted which components, at which version, with timestamps
- [ ] OT-P0-012 | Component Version Management | Track current version for each component using semver or incremental versioning

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | AI Patch Review and Merge | Allow users to review AI-suggested diffs/patches before accepting and applying changes
- [ ] OT-P1-002 | Multi-Element Selection | Enable attaching multiple selected elements from different frames to a single AI chat message
- [ ] OT-P1-003 | Per-Frame Props Configuration | Support different props/variants per emulator frame with persistent settings per component
- [ ] OT-P1-004 | Change Detection and Diff Views | Detect divergence between library version and adopted files, show diff views (library vs adopted)
- [ ] OT-P1-005 | Adoption Status Dashboard | Show which scenarios use which components, their versions, and update status (current/behind/modified)
- [ ] OT-P1-006 | Component Categories and Organization | Organize components by categories (form, CTA, hero, card, dashboard, etc.)
- [ ] OT-P1-007 | Preview Shell Configuration | Support wrapping previews in configurable shell (theme providers, routers, context)
- [ ] OT-P1-008 | Multi-File Component Support | Support editing and previewing components with helper files and related modules
- [ ] OT-P1-009 | Automated Adoption Verification | Confirm adopted files exist and validate header comments during adoption tracking
- [ ] OT-P1-010 | AI Context Enhancement | Include emulator viewport context and selected element info in AI requests for better suggestions

### 🟢 P2 – Future / expansion ideas

- [ ] OT-P2-001 | Component Story/Demo Files | Support creating and editing story files for component variants and usage examples
- [ ] OT-P2-002 | Visual Component Diff Viewer | Show side-by-side visual comparison of component versions in preview
- [ ] OT-P2-003 | Bulk Component Updates | Push library component updates to multiple adopted scenarios with automatic PR creation
- [ ] OT-P2-004 | Component Usage Analytics | Track which components are most adopted, least modified, and highest ROI
- [ ] OT-P2-005 | Design Token Integration | Sync with brand-manager or design token systems for consistent theming
- [ ] OT-P2-006 | Screenshot Testing Integration | Auto-capture screenshots across viewports for visual regression testing
- [ ] OT-P2-007 | Component Playground Sharing | Generate shareable URLs for component previews with specific props/viewports
- [ ] OT-P2-008 | AI-Powered Component Generation | Generate new components from natural language descriptions with preview
- [ ] OT-P2-009 | Component Dependency Graph | Visualize which components depend on or are composed from other components
- [ ] OT-P2-010 | Export to Figma/Design Tools | Generate design assets or Figma components from library components

## 🧱 Tech Direction Snapshot

**Preferred stacks / frameworks**:
- Frontend: React 18+, TypeScript, TailwindCSS, shadcn/ui, Lucide icons
- Backend: Go API server for registry, versioning, and adoption tracking
- Code Editor: Monaco Editor or CodeMirror with TSX language support
- Preview: iframe-based isolation with iframe-bridge for element selection
- AI Integration: resource-openrouter for code editing and refactoring
- Build: Vite for fast HMR and preview recompilation

**Data + storage expectations**:
- PostgreSQL: Component registry, adoption records, version history
- File system: Source of truth for component files with header comment metadata
- Component metadata: Indexed from header comments, persisted in DB for query performance

**Integration strategy**:
- iframe-bridge: Element selection and inspection in preview frames
- resource-openrouter: AI-powered code editing and refactoring
- app-issue-tracker: Adoption workflow issue generation for coding agents
- Emulator inspired by app-monitor but extended for multi-frame support

**Non-goals / guardrails**:
- Not a full design tool replacement (Figma, Sketch) - focus on code-first workflow
- Not supporting non-React frameworks in initial release
- Not managing runtime component hot-swapping in production scenarios
- Not handling binary assets or image optimization (delegate to image-tools)

## 🤝 Dependencies & Launch Plan

**Required resources**:
- postgres: Component registry, adoption tracking, version history
- resource-openrouter: AI-powered code editing and component refactoring

**Scenario dependencies**:
- app-issue-tracker: Integration for generating adoption workflow issues
- app-monitor: Reference for iframe-bridge and emulator patterns

**Operational risks**:
- AI hallucination in code suggestions could break components (mitigate with diff review)
- Header comment deletion by developers would break adoption tracking (enforce with linting/validation)
- Component version drift across scenarios could create inconsistency (address with P1 change detection)
- Large component files may cause editor/preview performance issues (optimize with lazy loading)

**Launch sequencing**:
1. P0 core (registry, editor, preview, emulator, AI editing, adoption workflow)
2. Basic component library with 5-10 common components seeded
3. Integration testing with 2-3 existing scenarios adopting components
4. P1 features (diff views, adoption status, multi-element selection)
5. P2 expansion (analytics, bulk updates, design token integration)

## 🎨 UX & Branding

**Look & feel**:
- Modern, professional component library UI with clean typography
- Side-by-side editor and preview layout with resizable panels
- Multi-viewport emulator inspired by browser dev tools and app-monitor
- Light/dark theme support with preference persistence
- Syntax-highlighted code editor with subtle glow on focus
- Smooth transitions between component selection and preview updates

**Accessibility**:
- WCAG 2.1 AA minimum for all UI controls
- Keyboard navigation for editor, preview, and emulator controls
- Screen reader announcements for preview updates and AI suggestions
- Focus indicators on all interactive elements
- Accessible color contrast in both light and dark themes

**Voice & messaging**:
- Professional and tool-focused: "Design, preview, and track shared components"
- Clear action verbs: "Apply to scenario", "Review changes", "Accept patch"
- Minimal jargon, clear onboarding for new library contributors
- Error messages guide users toward fixes (e.g., "Add header comment to enable tracking")

**Branding hooks**:
- Component library icon: layered squares or building blocks
- Primary color: Tailwind blue-600 for primary actions
- Accent: green for success states, amber for warnings (version drift)
- Typography: Inter or similar modern sans-serif for UI, monospace for code
- Iconography: Lucide icons throughout for consistency with other scenarios

## 📎 Appendix

**Related Scenarios**:
- app-monitor: Reference for iframe-bridge integration and emulator patterns
- app-issue-tracker: Integration target for adoption workflow
- brand-manager: Future integration for design tokens and theming
- tidiness-manager: Could leverage component library for UI consistency

**External References**:
- shadcn/ui: Component architecture and structure patterns
- Storybook: Component story/demo file patterns for future enhancement
- Radix UI: Accessible component primitives used by shadcn
- Monaco Editor: Potential code editor implementation
