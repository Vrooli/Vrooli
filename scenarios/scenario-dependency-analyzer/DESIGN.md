# Design Contract

## Intent

Scenario Dependency Analyzer uses a dense operator interface for graph inspection, drift triage, and deployment readiness. Screens should prioritize scanability, stable controls, and direct access to evidence over marketing-style presentation.

## Feedback & State

Loading, empty, degraded, and error states must keep the current scenario context visible. Actions that mutate scenario metadata need clear read-only versus apply affordances.

## Request Lifecycle

UI requests should resolve the API base through `@vrooli/api-base`, display API reachability failures inline, and keep graph/deployment/catalog views usable when one surface is temporarily unavailable.
