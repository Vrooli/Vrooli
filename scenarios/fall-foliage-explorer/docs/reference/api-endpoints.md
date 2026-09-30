# API Endpoints

Base URL is assigned by the lifecycle system. In local fallback mode the UI expects `http://127.0.0.1:17175`.

| Method | Path | Purpose | Requirement | Implementation |
| --- | --- | --- | --- | --- |
| GET | `/health` | Standard health response with database state | [REQ: REQ-P0-001] | [CODE: api/main.go#healthHandler] |
| GET | `/api/regions` | List foliage regions and response metadata | [REQ: REQ-P0-003] | [CODE: api/main.go#regionsHandler] |
| GET | `/api/foliage?region_id=N` | Current foliage status for one region | [REQ: REQ-P0-003] | [CODE: api/main.go#foliageHandler] |
| POST | `/api/predict` | Generate and store peak foliage prediction | [REQ: REQ-P0-005], [REQ: REQ-P2-001] | [CODE: api/main.go#predictHandler] |
| GET | `/api/weather?region_id=N&date=YYYY-MM-DD` | Weather data for prediction context | [REQ: REQ-P0-004] | [CODE: api/main.go#weatherHandler] |
| GET | `/api/reports?region_id=N` | List crowd-sourced reports for a region | [REQ: REQ-P1-001], [REQ: REQ-P1-004] | [CODE: api/main.go#reportsHandler] |
| POST | `/api/reports` | Submit report with optional `photo_url` | [REQ: REQ-P1-001], [REQ: REQ-P1-004] | [CODE: api/main.go#reportsHandler] |
| GET | `/api/trips` | List saved trip plans | [REQ: REQ-P1-003] | [CODE: api/main.go#tripsHandler] |
| POST | `/api/trips` | Save a trip plan | [REQ: REQ-P1-003] | [CODE: api/main.go#tripsHandler] |

## Response Envelope

Most endpoints return:

```json
{
  "status": "success",
  "message": "optional details",
  "data": {},
  "error": "optional error"
}
```

The shape is defined by [CODE: api/main.go#Response] and consumed by [CODE: cli/internal/support/support.go].

## Health Check

Health checks must keep meaningful JSON responses and lifecycle timeout values at or below five seconds. The configured endpoints are in [CODE: .vrooli/service.json].
