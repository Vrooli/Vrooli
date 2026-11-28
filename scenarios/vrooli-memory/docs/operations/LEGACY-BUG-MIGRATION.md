# Scenario QA bug history audit

The November 2026 cutover verified 1,980 published records and 275 closed or retagged case accounts in `agent-memory`. Scenario QA retained the owner and next action for each of the 1,290 open cases. Two investigations retained their original records with no verified report link. The source scope `team:scenario-qa` preserves all historical records, including 59 private draft records. [R4 and R8](../../../../docs/agent-system/RULES.md)

The private audit directory at `~/.vrooli/state/vrooli/vrooli-memory/migrations/legacy-scenario-qa-bugs/` contains the final source-to-memory ID and content-hash mapping in `final-inventory.json`, the original case handoff in `final-triage-seed.json`, and the verification in `source-audit.json` and `cutover-verification.json`.

Use `vrooli-memory recall` and `vrooli-memory recall zoom` to review published evidence. Use a receipt's source entry ID with `source-ledger journal get --scope team:scenario-qa <entry-id>` to recover a specific original record. The inventory identifies private draft IDs in that same scope.

Source Ledger retains historical receipt reads and rejects writes under the retired import namespace. New observations use `vrooli-memory remember`.
