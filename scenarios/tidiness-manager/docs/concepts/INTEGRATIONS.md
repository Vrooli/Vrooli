# Integrations

## Test Genie

Test Genie uses Tidiness Manager for the `tidiness` phase. It uses Quality Health for the `quality` phase. This split is the current ownership contract.

## Quality Health

Quality Health validates Tidiness Manager's own lint/type/static-quality setup. It also owns the static-quality policy that used to be mixed into Tidiness Manager and Scenario Auditor.

## visited-tracker

visited-tracker can provide visit history for smart prioritization and campaign handoffs. Tidiness Manager wraps some visited-tracker workflows through the `tracking` CLI domain.

## resource-claude-code And resource-codes

Smart scans can use AI resources to produce deeper refactor recommendations. The light and tidiness scan paths must continue to work without those resources.

## PostgreSQL

The API uses PostgreSQL for issues, metrics, campaigns, and history. Database configuration comes from lifecycle-managed environment variables.

## Optional Analyzer Tools

Tools such as gocyclo, dupl, and jscpd enrich complexity and duplication findings. Missing tools should produce degraded metadata, not failed basic scans.
