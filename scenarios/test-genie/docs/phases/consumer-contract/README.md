# React Consumer Contracts

**Phase:** `consumer-contract`  
**Provider:** `react-component-library`  
**Selection:** default when the React Component Library contracts or fleet checker change  
**Runtime:** static source scan; does not start consumer apps or a browser

## North Star

Every React UI call site follows the usage contracts declared by its pinned RCL
asset versions. The full fleet scan is complete, findings are attributed to
their contracts, and no contract count exceeds its downward-only baseline.

## The rungs and their gates

| Level | Meaning | Gate |
|---|---|---|
| L0 | Fleet contract scan failed or could not produce a trustworthy report. | The command, JSON report, or ratchet is invalid; or a contract count rose. |
| L1 | The React UI fleet was scanned and the contract ratchet passed. | Every discovered React UI was checked and each per-contract count stayed at or below baseline. |

## What each finding means

| Finding | Meaning | Effect |
|---|---|---|
| `USAGE_CONTRACT_RATCHET_FAILED` | A call site violates a published asset contract, a per-contract count increased, or the report is inconsistent. | The phase fails and includes the ratchet's reported reasons. |
| System validation error | The scanner did not return a complete machine-readable result. | The phase fails closed; an unavailable or malformed scan is not a pass. |

The returned native detail includes the number of UI trees and source files
scanned, total and per-contract finding counts, baseline location, and each
contract's baseline/current/delta comparison.

## The canonical fix

Run the fleet checker and follow the reported call sites. Correct component
usage at the consumer, or repair the asset's contract and its types, development
warning, and adversarial story together when the declaration is wrong. Do not
raise a ratchet baseline to hide new violations; a baseline change needs
measured evidence that the contract or scanner intentionally changed.

## How to verify

From `scenarios/react-component-library/ui`, run:

```bash
pnpm usage-contracts:fleet
```

The Test Genie phase invokes that same command with JSON output. Verify the
registered provider route with:

```bash
test-genie phases inspect consumer-contract --json
test-genie execute react-component-library --phases consumer-contract
```

The phase is evaluated only when execution is requested. Without an execution
request, the provider reports `SKIPPED` with an L0 assessment rather than
claiming the fleet is clean.
