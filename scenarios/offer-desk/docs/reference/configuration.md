# Configuration

`MONEY_LEDGER_API_URL` is an optional fixed service address for the typed actuals/posture join. Without it, the board discovers Money Ledger at request time. `MONEY_LEDGER_BOOK_ID` selects the ledger book whose position is shown on the board. If the peer or book is unavailable, the board reports the source as unavailable and continues serving catalog state.

`TREASURY_API_URL` and `DEPLOYMENT_MANAGER_API_BASE_URL` optionally pin the corresponding peer addresses for deployments. Without these overrides, Offer Desk uses local scenario discovery when each operation runs. Swarm Manager uses local discovery.

The Offer Desk database uses the lifecycle storage resolver and routed test pools. Evaluation is scheduled in-process with the API's `schedule.Clock`; no caller-owned cron job is required.

When `OFFER_DESK_DEPLOYMENT_MANAGER_READINESS_TOKEN` is set, Offer Desk reports
catalog reconciliation and pricing evidence to Deployment Manager. The reporter
resolves the current peer address for each report; the optional base URL above
remains a fixed deployment override. Configure
the matching `OFFER_DESK_DEPLOYMENT_MANAGER_READINESS_SCENARIO`,
`_PROFILE_ID`, `_CANDIDATE_COMMIT`, `_ARTIFACT_DIGEST`, `_TARGETS` (comma-separated),
`_CHANNEL`, and `_POLICY_VERSION` variables. Release-bound reports also require
`_CANDIDATE_ID`, `_DESTINATION_REVISION_ID`, and `_AUTHORIZATION_EPOCH` together.
