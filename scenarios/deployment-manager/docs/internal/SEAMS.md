# SEAMS - deployment-manager

Architectural seams and variation points for the deployment-manager scenario.

## Visual Validation Seams (Mar 2026)

### ValidationRepository Seam
- **Interface**: `validation.Repository` (`api/validation/repository.go`)
- **Default Implementation**: `SQLRepository` — PostgreSQL persistence for validation records
- **Variation Point**: Could be replaced with in-memory store for tests or file-backed store for single-node deployments
- **Test Double**: `inMemoryRepo` in `api/validation/handler_test.go`

### VideoTransport Seam
- **Interface**: `DesktopPackagerClient.DownloadVideo()` (`api/deployments/desktop_client.go`)
- **Default Implementation**: HTTP streaming from scenario-to-desktop `/api/v1/smoketest/{id}/video`
- **Variation Point**: Could be replaced with MinIO/S3 download for multi-server deployments
- **Test Strategy**: Mock HTTP server in client tests

### DesktopPackagerClient Seam (extended)
- **Interface**: Methods on `DesktopPackagerClient` (`api/deployments/desktop_client.go`)
- **New Methods**: `RunSmokeTest`, `GetSmokeTestStatus`, `WaitForSmokeTest`, `DownloadVideo`
- **Default Implementation**: HTTP calls to scenario-to-desktop API
- **Variation Point**: Already an established seam; smoke test methods follow the same HTTP polling pattern as build methods
