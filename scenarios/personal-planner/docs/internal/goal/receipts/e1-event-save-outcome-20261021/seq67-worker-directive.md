# Worker directive — seq67 target-axis correction

Feedback `PP-SELECT-PLANNER-SHADOW-POSITIONALLY-NOT-WH-INSTANCE-20261021` corrects the two target axes. Use the normal reachable Workflow Health service with positional Planner target `personal-planner@shadow`; `--instance shadow` selects the Workflow Health service instance. Do not start/restart `workflow-health@shadow` to select Planner shadow.

Continue evidence-first under seq65. Inspect current registered diagnostic cases and current exact-instance logs/artifacts. If one purposeful reproduction remains necessary, use a changed/current capture and intentional new key, then run the supported route:

```text
workflow-health validate start-selected personal-planner@shadow --path /home/matthalloran8/Vrooli/scenarios/personal-planner --case-paths <current registered diagnostic case> --idempotency-key <new intentional diagnostic key> --parent-run-id 5d07e651-eaa6-48ec-89b4-5dc359059193 --json
```

Before interpreting it, verify the returned target, exact run ID/key, fresh managed lease and zero primary writes; wait once for that exact run and retain the response status/body. Do not use the stale-revision case, execute against live, mutate/clear the pool, or patch `ErrEventOwnerRequired` without response evidence. Preserve all original E1/Q1 acceptance gates.
