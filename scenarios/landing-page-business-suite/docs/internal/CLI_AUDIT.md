# Landing Page Business Suite CLI Audit

## Last Updated
2026-02-04

## Current State
- [x] Go-based CLI exists
- [x] Uses cli-core package
- [x] Cross-platform installers present (`cli/install.sh`, `cli/install.ps1`)
- [x] CLI install step present in `.vrooli/service.json`
- [x] Commands cover API endpoints

## API Coverage
All routes registered in `api/routes.go` have matching CLI commands in `cli/app.go`, grouped by domain:
- Health, Landing, Auth, Account
- Billing & Payments
- Variants + Content
- Metrics + Engagement (Feedback, Waitlist)
- Credits (API keys, limits, usage)
- AI Gateway
- Admin Core + Admin Commerce + Admin Users
- Docs + Config

## Issues Found
No gaps detected in API parity or cli-core usage.

## Recent Changes
- 2026-02-04: Re-grouped CLI command organization in `cli/app.go` to mirror API domain modules.
- 2026-02-04: Added `remote-profiles-proxy` and `admin-downloads-upload-managed` helper commands for remote upload automation.
