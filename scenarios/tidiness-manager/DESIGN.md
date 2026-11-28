# Tidiness Manager Design Contract

## Intent

Tidiness Manager is an operational workbench for maintainability debt. The UI should help agents and humans scan, compare, prioritize, and resolve code-cleanliness issues without implying that lint/type/static-quality policy belongs here.

Use dense, scan-friendly layouts: tables, filters, status badges, and direct commands. Avoid marketing layouts and decorative card-heavy composition.

## Feedback & State

- Show scan status, campaign status, issue severity, and stale/visited state as first-class data.
- Distinguish maintainability findings from Quality Health static-quality findings in labels and empty states.
- Keep failure messages actionable: name the command or endpoint that failed and point to retry or troubleshooting docs.

## Request Lifecycle

- Long scans and smart scans must show pending, success, failed, and partial-result states.
- Do not block the whole UI when one scenario or campaign fails to load.
- Mutating actions such as resolve, ignore, pause, resume, and terminate need visible confirmation and a way to recover through refresh or retry.

## Visual System

- Prefer restrained contrast and clear hierarchy over illustration.
- Use compact controls for filters and command actions.
- Use accessible severity colors and text labels together; color alone is not enough.
