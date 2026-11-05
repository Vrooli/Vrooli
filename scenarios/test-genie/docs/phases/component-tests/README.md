# Component Tests

## North Star

Every selected React Component Library asset version has a safe, version-pinned story contract and browser-observed evidence for its declared behavior.

## The rungs and their gates

L0 means the catalog contract cannot be evaluated. L1 requires every selected version to parse, resolve its dependency closure, and complete its declared story checks without a `COMPONENT_TEST_FAILED` finding.

## What each finding means

`COMPONENT_TEST_FAILED` means an indexed story contract is invalid, its versioned source is incomplete, the preview harness could not render it, or a declared browser assertion failed. A blocked result means the browser executor is unavailable and is not behavioral evidence.

## The canonical fix

Repair the versioned `story.json` contract or the versioned component implementation, reindex the catalog if necessary, then rerun the component-test phase. If Chrome is unavailable, configure `RCL_CHROME_BIN` to a usable browser before treating the result as a behavior verdict.

## How to verify

Run `vrooli scenario test react-component-library`, wait for its server-owned run, and confirm that the `component-tests` phase is passed with no `COMPONENT_TEST_FAILED` findings. The report records durable RCL component-test evidence.

The React Component Library provider evaluates each latest versioned catalog asset's declarative test contract. It resolves manifest-pinned closures, rejects unsafe contract input, checks declared component traces against indexed examples, and records durable RCL reports. Test Genie owns the enclosing provider lifecycle and exposes the normal run wait, follow, cancellation, and artifact semantics.
