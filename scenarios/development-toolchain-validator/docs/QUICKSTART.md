# Quick Start

## What is development-toolchain-validator?

This scenario validates that the Vrooli development ecosystem works correctly by testing development tools and steer skills against known-good **reference scenarios**. It answers three questions:

1. **Are steer skills consistent with each other?** When multiple skills are applied to the same scenario, do they create conflicts?
2. **Are development tools accurate?** Does scenario-auditor, test-genie, and scenario-completeness-scoring produce correct results on known-good code?
3. **How mature are steer skills?** Can we programmatically describe what a skill does, or is it too vague?

## Prerequisites

- PostgreSQL running (resource dependency)
- prompt-manager running (API dependency for reading skills)
- For tooling baselines (P1): scenario-auditor, test-genie, and scenario-completeness-scoring running

## Setup

```bash
cd scenarios/development-toolchain-validator
make start
```

## Core Workflow

### 1. Register a reference scenario

```bash
development-toolchain-validator references add reference-react-vite --template react-vite
```

### 2. Connect a steer skill

```bash
development-toolchain-validator skills connect api-steer --reference reference-react-vite
```

This connects the skill with its current version pinned. No config is needed initially — unconfigured connections represent skills that need structured expectations defined.

### 3. Add structural expectations (optional, but this is where value comes from)

```bash
development-toolchain-validator expectations add structural \
  --connection api-steer:reference-react-vite \
  --type folder --path "api/handlers/{domain}/" --required true \
  --description "API handlers organized by domain module"
```

### 4. Add CLI tool assertions (optional)

```bash
development-toolchain-validator expectations add cli-tool \
  --connection api-steer:reference-react-vite \
  --command "scenario-auditor audit reference-react-vite --json" \
  --path "$.total" --op eq --value 0
```

### 5. Run validation

```bash
development-toolchain-validator validate reference-react-vite
```

### 6. View the report

```bash
development-toolchain-validator report reference-react-vite
```

## What's Next

- Read [Concepts: Vision & Purpose](concepts/VISION.md) to understand why this exists
- Read [Concepts: Architecture](concepts/ARCHITECTURE.md) for the technical design
- Read [Guides: Connecting Skills](guides/connecting-skills.md) for the full workflow
- Read [Guides: Writing Assertions](guides/writing-assertions.md) for assertion syntax
