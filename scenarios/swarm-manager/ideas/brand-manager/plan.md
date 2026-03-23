# Implementation Plan: Brand Manager – Full Branding Lifecycle for All Scenarios

## Purpose
## What It Is
A single scenario that manages the full branding lifecycle for all Vrooli scenarios — generating, storing, applying, and validating brand identity. Replaces the old Brand Manager and App Personalizer scenarios (now deleted) with a clean rewrite.

## Core Concepts

### Brand
A versioned bundle containing:
- **Identity:** display name, tagline, description, author
- **Visual assets:** logo (SVG + rasterized), favicon (multi-size), app icon, og-image
- **Color system:** primary, secondary, accent, semantic colors, dark/light variants, WCAG-validated
- **Typography:** heading, body, and mono fonts with scale/weights
- **Voice:** tone descriptors and example copy (for LLM consumption by other scenarios)
- **Notes:** freeform user input that influences generation (e.g., "ocean theme", "bold and playful")

### Assignment
A record that brand X is applied to scenario Y, tracking what was applied, when, and what version.

### Asset
An individual generated file with metadata (dimensions, format, purpose, version).

## How It Works

### 1. Discovery (auto-populate)
When a user first opens a scenario in Brand Manager, it scans the scenario existing state — service.json, theme files, static assets, manifests — and populates a draft brand from what it finds. If the display name is just the slug or clearly not a proper name, it is flagged as needing attention, not silently accepted.

### 2. Generation (user-guided)
The user fills in what they know and leaves the rest blank. They might set a specific primary color, or just write "ocean theme" in the notes. Brand Manager uses OpenRouter to generate everything missing — logo concepts, palette, typography pairings, copy — incorporating the user inputs and notes as constraints. The user picks from options, refines, and iterates.

### 3. Storage (Brand Manager is source of truth)
All drafts, versions, and final assets live in Brand Manager SQLite database and file storage. Version history with notes. A brand can exist in the library without being applied to anything yet.

### 4. Application (manual, two-tier)
User explicitly triggers "apply" via UI button or CLI command. Application happens in two tiers:
- **Programmatic:** Standard patterns (CSS custom properties, manifest.json, favicon paths, static asset dirs, service.json fields) are written directly.
- **Agent-assisted:** For complex or non-standard scenarios, Brand Manager spawns an agent to integrate. The agent is instructed to set things up so they are programmatically validatable afterward.

Partial application is supported — you can apply just the logo now and the color system later. Validation will still flag what is missing.

### 5. Validation (strict, programmatic)
Brand Manager registers rules in Scenario Auditor for each branding requirement (has logo, has favicon, has color system, has display name, has typography, etc.). These rules are always checkable programmatically — that is a hard constraint on how application works. Deployment readiness skills in prompt-manager run these rules and can use Brand Manager CLI to remediate violations.

## Brands Are Reusable
A brand is a library item that can be assigned to multiple scenarios. "Vrooli Core" brand across internal tools, or a client-specific brand across their suite.

## Opt-Out
All scenarios need branding — 99% of Vrooli scenarios are intended for monetization. The only exception is scenarios that exist purely for testing (e.g., test-scenario, hello-world). These opt out via a clear mechanism (e.g., a tag or field in service.json). The opt-out is explicitly documented as rare and intentional.

## Resource Dependencies
- **OpenRouter** — image generation (logos, favicons, icons) + LLM for copy/palette suggestions, following agent-inbox established patterns
- **SQLite** — brand metadata, assignments, version history

## Surfaces
- **UI:** Scenario-centric dashboard showing branding status across all scenarios. Wizard for creating/editing brands. Preview before applying.
- **CLI:** brand-manager generate, brand-manager apply, brand-manager status — for automation and agent-driven workflows.
- **API:** RESTful endpoints for brands, assignments, assets, and status — consumed by Scenario Auditor rules and deployment skills.

## Out of Scope
- Digital twin / behavioral personalization
- Multi-tenant white-labeling
- N8n orchestration
- ComfyUI
- A/B testing, analytics dashboards, ML optimization
- Auto-applying on brand update (push model)
