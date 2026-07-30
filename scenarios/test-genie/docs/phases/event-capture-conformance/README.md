# Event Capture Conformance

Vrooli Events owns this opt-in phase. It applies only when a scenario both declares a non-disabled `vrooli-events` dependency and carries a receipt declaration under `.vrooli/vrooli-events/`.

The provider validates each Connect operation, exact response type, and projection path against the committed protobuf descriptor image, then verifies that every declared policy is present in the global policy snapshot. A missing or drifted policy is an error because it would silently leave the declared operation unobserved.

## North Star

Every opt-in receipt declaration is valid, exact, and present in the global policy snapshot before an operator relies on it for analytical evidence.

## The rungs and their gates

`L0` is an invalid declaration or an unreconciled policy. `L1` is descriptor-valid and reconciled. `L2` is continuously clean under this provider phase.

## What each finding means

`event_capture.declaration_invalid` means the source, operation, response type, or projection is invalid. `event_capture.policy_unreconciled` means the declaration has not reached—or no longer matches—the global snapshot.

## The canonical fix

Repair the file under `.vrooli/vrooli-events/`, then run `vrooli-events capture-preview --scenario <scenario>` followed by `vrooli-events capture-reconcile --scenario <scenario>`.

## How to verify

Run `vrooli scenario test <scenario>` and confirm this phase passes. A scenario without both the dependency and declaration remains `not_applicable`.
