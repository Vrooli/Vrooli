# Offer Desk runbook

Use `make setup`, `make start`, `make logs`, and `make stop`. Do not launch the API binary directly. The board is expected to remain useful if Money Ledger is stopped; its availability row should explain the missing source.

The SQLite catalog is reconstructible only from deliberate exports/imports, so back up the database using the same verified storage procedure as Money Ledger: stop the scenario for a file copy, include WAL/SHM companions, checksum the copy, open it read-only, and query nodes, triggers, facts, evaluations, and proposals. Keep dated copies on separate storage.

If an evaluation is stuck, inspect the latest evaluation row and the fact's observed time/staleness window. Missing and stale facts are expected unknowns, not false results. If a transition is refused, use the named legal transitions in the response. If promotion is refused, use the operator role or create a proposal for review.
