# Configuration

`MONEY_LEDGER_API_URL` enables the optional typed actuals/posture join. `MONEY_LEDGER_BOOK_ID` selects the ledger book whose position is shown on the board. If either is absent, the board reports the source as unavailable and continues serving catalog state.

The Offer Desk database uses the lifecycle storage resolver and routed test pools. Evaluation is scheduled in-process with the API's `schedule.Clock`; no caller-owned cron job is required.

When `OFFER_DESK_DEPLOYMENT_MANAGER_READINESS_TOKEN` is set, Offer Desk reports
catalog reconciliation and pricing evidence to Deployment Manager. Configure
the matching `OFFER_DESK_DEPLOYMENT_MANAGER_READINESS_SCENARIO`,
`_PROFILE_ID`, `_CANDIDATE_COMMIT`, `_ARTIFACT_DIGEST`, `_TARGETS` (comma-separated),
`_CHANNEL`, and `_POLICY_VERSION` variables. Release-bound reports also require
`_CANDIDATE_ID`, `_DESTINATION_REVISION_ID`, and `_AUTHORIZATION_EPOCH` together.
