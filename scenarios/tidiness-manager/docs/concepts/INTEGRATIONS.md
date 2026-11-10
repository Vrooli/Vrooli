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

Code Facts owns Go syntax graphs and complexity facts. Tidiness projects those facts into metrics and reduction recommendations. Dupl and jscpd still supply token-level duplication findings pending their migration into the shared registry. Missing source providers or tools produce explicit degraded metadata.
