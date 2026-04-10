## Problem
Standards report UI server compliance failures: custom server detected, missing proxyToApi, and missing standard server function. These are blocking the standards phase.

## Top Violations
- scenarios/vrooli-bridge/ui/server.js: Custom server detected (medium)
- scenarios/vrooli-bridge/ui/server.js: Missing proxyToApi in custom server (high)
- scenarios/vrooli-bridge/ui/server.js: Missing standard server function (medium)
- scenarios/vrooli-bridge/ui/package.json: Missing @vrooli/api-base (critical)
- scenarios/vrooli-bridge/requirements/index.json: P0 target missing requirements (critical)

## Impact
Custom server patterns bypass the Vrooli tunnel and API proxy expectations, risking broken /api routing and non-standard deployment behavior. Standards failures also block readiness and testing.

## Reproduction
git-control-tower review-run vrooli-bridge --json

## Success Criteria
- Standards blockingViolations = 0
- ui/server.js uses startScenarioServer/createScenarioServer
- proxyToApi is defined and routes /api requests through the Vrooli tunnel

## Proposed Action
1. Replace the custom UI server with startScenarioServer/createScenarioServer.
2. Implement proxyToApi to forward /api requests to the scenario API endpoint.
3. Re-run standards check to verify server compliance.
