# Troubleshooting

- A candidate creation or transition refused for `candidate_requires_trigger` needs a trigger attached to that node.
- `UNKNOWN` means a fact is absent or stale; add a dated fact or shorten the investigation, rather than treating it as false.
- A refused lifecycle transition includes the legal next states. Follow that response rather than bypassing the API.
- A board availability row names the Money Ledger error. Do not interpret unavailable actuals as zero earnings.
- To test the scheduled evaluator, create an idea, declare its trigger, transition it to candidate, add a fresh satisfying fact, and wait for the scheduler or call the typed evaluation RPC.
- If generated types or the UI bundle are stale, run `make setup` through the lifecycle, then `pnpm type-check` and `pnpm build` from `ui/`.
