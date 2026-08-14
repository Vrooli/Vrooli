# CLI commands

All commands support the standard report and `--json` output. The `ledger` group covers `books-list`, `books-create`, `accounts-list`, `accounts-create`, `journal-list`, `journal-reverse`, `position-show`, `statement-show`, `goals-list`, and `goal-declare`. The `ingest` group covers `adapters-list`, `adapter-register`, `event-ingest`, `adapter-run`, and `file-import`.

Human output keeps basis beside figures. JSON preserves `partial`, `availability`, adapter reasons, receipt counts, and reversal links. Manual events are admitted through `event-ingest` with the same typed contract as file and future upstream adapters.
