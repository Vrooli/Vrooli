# Data

## Data Model

Tidiness Manager persists scenario-level maintainability data in PostgreSQL. The primary durable entities are issues, file metrics, scan history, and campaigns.

## Issues

Issues describe a maintainability finding for a scenario and optional file. They include category, severity, message, tool/source, status, and resolution metadata. Resolved and ignored status changes remain part of the audit trail.

## File Metrics

File metrics capture path, line count, language/extension, import/function metrics, complexity summaries, duplication summaries, and scan timestamps when available.

## Scan History

Scan history records completed scan attempts and supports UI recency, agent reads, and troubleshooting. Light and tidiness scans should remain usable when optional tools are missing.

## Campaigns

Campaigns track scenario, status, current session, file coverage, limits, priority rules, errors, and completion timestamps. Campaign state is independent from Quality Health static-quality validation.

## Data Ownership

Tidiness Manager owns maintainability data only. Quality Health owns static-quality audit data, rule contracts, and config-fix previews.

## Retention And Privacy

Stored data should avoid secrets and raw environment values. Paths should be scenario-relative where possible, and API responses should not expose database credentials or internal resource configuration.
