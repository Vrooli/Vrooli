# Configuration

`MONEY_LEDGER_API_URL` enables the optional typed actuals/posture join. `MONEY_LEDGER_BOOK_ID` selects the ledger book whose position is shown on the board. If either is absent, the board reports the source as unavailable and continues serving catalog state.

The Offer Desk database uses the lifecycle storage resolver and routed test pools. Evaluation is scheduled in-process with the API's `schedule.Clock`; no caller-owned cron job is required.
