# Components Directory Structure

This directory contains React components organized by feature area.

## Naming Conventions

### Suffix Meanings

- **`*Section`** - UI components that represent visual containers/regions of the page
  - Example: `PreflightSection`, `ConfigurationSection`, `BundleSection`
  - These are presentational components focused on layout and visual grouping

- **`*Stage`** - Components representing pipeline execution stages
  - Example: Components in `sections/generate/` that map to pipeline stage execution
  - These are tied to the pipeline's stage-based workflow

### Folder Organization

```
components/
├── distribution/     # Distribution page components
├── docs/             # Documentation viewer components
├── generator/        # Generator form and related components
├── layout/           # Layout primitives (sidebar, navigation)
├── modals/           # Modal dialogs
├── pipeline/         # Pipeline status and error display
├── preflight/        # Preflight validation components
├── runtime/          # Runtime configuration (bundle, servers)
├── scenario-inventory/ # Scenario listing and management
├── sections/         # Section-based page layouts
│   ├── configuration/  # Configuration section
│   ├── generate/       # Generate section (maps to pipeline stages)
│   └── shared/         # Shared section components
├── signing/          # Code signing page components
├── state/            # State display components (build status, alerts)
├── ui/               # Primitive UI components (button, input, card, etc.)
└── wine/             # Wine runtime components
```

### Import Patterns

Components should import from:

- `../../lib/api` for API functions and types
- `../../domain/*` for domain logic and types
- `../../hooks` for React hooks
- `../../store` for Zustand stores
- `../ui/*` for primitive UI components
