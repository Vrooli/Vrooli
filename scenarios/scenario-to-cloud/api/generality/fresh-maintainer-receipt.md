# Fresh maintainer receipt: edge rollback contract

## Finding

The generality fake does not model the target owner's edge rollback lifecycle. After two successful route applies for the same deployment (an update), `fakeTarget` stores only the latest route. Its `cloud-target edge route-rollback` handler deletes every route owned by that deployment. The actual owner retains the displaced snippet and restores it on rollback. The new regression is intentionally baseline-red and isolates this contract mismatch in `faketarget_contract_test.go`.

## Expected behavior and operator impact

After an update changes an existing deployment route, rollback restores the route that was active before the update. It must not remove the deployment's only route. Otherwise the generality harness can misrepresent the rollback behavior an operator relies on to recover public traffic after an update.

## Independent owner evidence

- `internal/cloudtarget/edge_route.go:311-361` defines `EdgeRouteRollback`, reads the retained previous snippet, installs it as the active snippet, and retains the displaced current snippet.
- `internal/cloudtarget/edge_test.go:196-224` exercises two applies followed by rollback and asserts the v1 snippet is restored; it also verifies rollback without a displaced snippet is refused.
- The owner test passed independently:

```text
$ go test ./internal/cloudtarget -run '^TestEdgeRouteRollbackRestoresDisplacedSnippet$' -count=1
ok  github.com/vrooli/vrooli/internal/cloudtarget  0.010s
```

## Regression evidence

Added a new test only; no existing tests or production files were edited. It applies an initial route and an updated route through `fakeTarget.Exec`, then asserts rollback restores the initial upstream port (3000).

```text
$ cd scenarios/scenario-to-cloud/api && go test ./generality -run '^TestFakeTargetEdgeRollbackRestoresDisplacedRoute$' -count=1
--- FAIL: TestFakeTargetEdgeRollbackRestoresDisplacedRoute (0.00s)
    faketarget_contract_test.go:48: rollback route = {UpstreamPort:0 ListenerID: Deployment:}, want displaced v1 upstream on port 3000
FAIL
FAIL    scenario-to-cloud/generality  0.010s
FAIL
```

## Limits

This proves the fake's state model diverged from the owner's tested state transition. It does not exercise Caddy, network traffic, or a full deployment operation failing after route application. The owner fake now retains the route set displaced by an update, replaces routes by deployment as a whole, and restores the prior set on rollback; the independent regression passes after that repair.

## Owner follow-through

The focused owner regression and full `go test ./generality -count=1` pass; repeat×5 and `go vet ./generality` pass. The independent real-owner rollback test still passes. Full-package race×5 detected a separate race between `operations.worker.heartbeat` and operation reload/transition code. Those paths are outside this frozen harness boundary and were already dirty in the shared tree, so no sibling fix is claimed. A single isolated race run passed, while five repetitions of `TestHostedBackendRoutesSurviveUpdate` reproduce the race; see the retained owner race log.
