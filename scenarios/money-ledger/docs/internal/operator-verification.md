# Operator verification rehearsal

This document is addressed to the monetization operator. It is a rehearsal,
not an adoption decision. Run it against copies of the source documents. Never
write to the live monetization team tree or its operator-inputs file.

## Safety and scratch setup

```bash
scratch="$(mktemp -d)"
cp scenarios/prompt-manager/store/teams/monetization/shared/operator-inputs.json "$scratch/operator-inputs.json"
cp -R docs/monetization "$scratch/monetization"
find docs/monetization -type f -print0 | sort -z | xargs -0 sha256sum | sha256sum
```

The scratch copies are synthetic rehearsal inputs. Delete `$scratch` at the
end. The protected-source digests for this run are recorded in both scenario
`PROGRESS.md` files and must be unchanged afterward.

## Money Ledger rehearsal

1. Start the pair with `vrooli scenario start money-ledger` and
   `vrooli scenario start offer-desk`.
2. In Money Ledger **Accounts**, create a rehearsal book and cash account.
3. Open **Journal** and record one current monetary event. Confirm the row
   shows amount, ISO currency, operator-asserted basis, source, and date.
4. Repeat its external event ID. Confirm the UI/CLI reports a duplicate and
   does not create a second posting.
5. Reverse the event with a reason. Confirm the original and linked reversal
   both remain visible.
6. Use `money-ledger ingest operator-import --source-path "$scratch/operator-inputs.json" --source-mode dry-run --adapter-id <adapter> --book-id <book> --account-id <account> --json`.
   Confirm pending and not-applicable fields are absent, not zero.
7. Populate only the scratch copy with synthetic values, rerun dry-run, and
   inspect the report before any apply. Confirm monetary values are postings,
   time/hours are measures, and MRR is a derived-rate finding rather than a
   posting.
8. Declare the four canonical financial rules through the dashboard goal form.
   Confirm each goal shows threshold, unit, sustain window, and verdict.

The durable classification result is
[`operator-input-rehearsal.json`](evidence/operator-input-rehearsal.json).

## Offer Desk rehearsal

9. In Offer Desk **Offers**, create or inspect a node and its typed edges.
10. Use **Triggers** to declare a trigger, record a fact, and run dry-run
    evaluation. Confirm UNKNOWN, unsatisfied, and satisfied have different
    text explanations.
11. Promote once as an agent. Confirm the node remains non-active and a
    proposal records proposer, requested status, reason, and evidence.
12. In **Proposals**, accept as operator or decline with a reason. Confirm the
    decline history is visible.
13. Run `offer-desk offers catalog-import --source-path "$scratch/monetization" --source-mode dry-run --json`.
    Confirm file counts and findings before apply. Introduce a malformed
    status in the scratch copy and repeat; confirm apply is blocked and no
    catalog write occurs.
14. Run `offer-desk offers board-show --json` and
    `offer-desk offers space --projection offers --json`. Confirm catalog,
    trigger, actuals, posture, and obligation cells remain distinct.
15. Stop Money Ledger through `vrooli scenario stop money-ledger`. Refresh the
    board. Confirm it names the unavailable source and shows no synthetic zero.
16. Restart Money Ledger through `vrooli scenario start money-ledger` and
    confirm the board recovers with source and evaluation age.

The durable product-side outage recording is
[`outage-demo-20260816.mp4`](evidence/outage-demo-20260816.mp4). It was captured
at 1440×1000, 10 fps, and includes the healthy, stopped-source, and recovered
states. A side-by-side alternative capture is intentionally not claimed until an
actual alternative environment is available.

The durable board and projection artifacts are
[`board-degradation-rehearsal.json`](../../../offer-desk/docs/internal/evidence/board-degradation-rehearsal.json)
and [`space-projection-rehearsal.json`](../../../offer-desk/docs/internal/evidence/space-projection-rehearsal.json).
The twelve-row sufficiency manifest is
[`operator-sufficiency-rehearsal.json`](../../../offer-desk/docs/internal/evidence/operator-sufficiency-rehearsal.json).

## Closeout

```bash
vrooli scenario status money-ledger
vrooli scenario status offer-desk
rm -rf -- "$scratch"
```

Before deciding on adoption, compare the protected-tree digests with the
Phase 1 values. This rehearsal does not edit `docs/monetization`,
`operator-inputs.json`, `team.json`, or member files. Narrative judgment,
positioning, and philosophy remain human-owned.
