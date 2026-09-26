# Monetization operator verification

This is the reproducible rehearsal command list. It operates on copies and keeps the source trees untouched. Operator mode is dry-run unless `--apply true` is supplied.

```bash
make -C scenarios/money-ledger start
make -C scenarios/offer-desk start

mkdir -p scenarios/offer-desk/tmp/rehearsal scenarios/money-ledger/tmp/rehearsal
cp -a docs/monetization scenarios/offer-desk/tmp/rehearsal/monetization-copy
cp scenarios/prompt-manager/store/teams/monetization/shared/operator-inputs.json \
  scenarios/money-ledger/tmp/rehearsal/operator-inputs.json

offer-desk offers catalog-import \
  --source-path "$PWD/scenarios/offer-desk/tmp/rehearsal/monetization-copy" \
  --source-mode operator --json
offer-desk offers catalog-import \
  --source-path "$PWD/scenarios/offer-desk/tmp/rehearsal/monetization-copy" \
  --source-mode operator --apply true --json

BOOK_ID=$(money-ledger ledger books-create --name "Monetization rehearsal" --currency USD --json | jq -r '.book.id')
ACCOUNT_ID=$(money-ledger ledger accounts-create --book-id "$BOOK_ID" --name "Rehearsal cash" --kind asset --json | jq -r '.account.id')
money-ledger ingest operator-import \
  --source-path "$PWD/scenarios/money-ledger/tmp/rehearsal/operator-inputs.json" \
  --source-mode operator --book-id "$BOOK_ID" --account-id "$ACCOUNT_ID" \
  --adapter-id operator-inputs --json
money-ledger ingest operator-import \
  --source-path "$PWD/scenarios/money-ledger/tmp/rehearsal/operator-inputs.json" \
  --source-mode operator --apply true --book-id "$BOOK_ID" --account-id "$ACCOUNT_ID" \
  --adapter-id operator-inputs --json

money-ledger ledger position-show --book-id "$BOOK_ID" --json
money-ledger ledger goals-list --book-id "$BOOK_ID" --json
offer-desk offers board-show --json
offer-desk offers catalog-list --json
offer-desk offers gates-evaluate --json
offer-desk offers space --projection offers --json
```

The verification report to retain is the JSON from each import command. Review per-file or per-field counts before any later adoption plan removes a source. A nonzero blocking finding or a read/write mismatch must stop the apply step.
