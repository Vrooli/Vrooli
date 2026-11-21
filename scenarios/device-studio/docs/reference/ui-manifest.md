# UI Manifest Reference

## Current versus target routes

The UI currently implements the generated `/`, `/notes` and `/settings` routes. [Experience specifications](../../experience/README.md) distinguish these from 13 planned product pages and eight planned journeys. Draft design routes are not registered as callable APIs or fake live UI bindings. The future manifest must advertise actual surfaces only as their implementation lands, including unavailable-capability states on desktop and smaller screens.

## Shared contract

Read the [UI Manifest Reference shared guide](/scenarios/template-manager/docs/reference/ui-manifest.md). Template Manager owns
the common contract and its implementation references.

## Scenario details

Read the scenario’s `.vrooli/ui-manifest.json` overlay when present.
Record local path overrides here only when they need rationale beyond the
declaration itself.
