# Vrooli Scenario Templates

> **Canonical templates for creating production-grade scenarios. Always start from one of these; never create a new template folder.**

## 📁 Directory Structure

```
templates/
├── react-vite/     # React + TypeScript + Vite UI + Go API archetype
└── requirements/   # Shared registry scaffolds (used by `vrooli scenario requirements init`)
```

### `react-vite/` (use this for every new scenario)
- React + TypeScript + Vite + shadcn/ui + lucide UI
- Go API server, CLI, deployment scripts, and phased tests
- `.vrooli/service.json` pre-wired for lifecycle + resource metadata
- `requirements/index.json` seeded with an example entry so operational targets map to technical validations

**Copy command (from the repo root):**
```bash
vrooli scenario generate react-vite --id my-scenario --display-name "My Scenario" --description "One sentence summary"
```

### `requirements/`
Shared modular requirement registry samples consumed by `vrooli scenario requirements init`. This stays separate so templates and requirement tooling evolve independently.

## Usage Guidelines
- ✅ Pick an existing template (currently `react-vite/`).
- ✅ Customize within your scenario folder after copying.
- ✅ Submit PRs to enhance existing templates.
- ❌ Do not create new template folders without explicit platform approval.

By standardizing on a single high-quality template, every new scenario immediately benefits from the same architecture, testing harness, and lifecycle wiring.
