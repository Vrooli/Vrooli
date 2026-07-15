# BAS calibration (planted-error fixtures)

Playbooks here are **expected to fail**. They live outside `bas/cases/` so
`test-genie registry build` never registers them into the green suite. Their job
is to prove that a corresponding registered playbook can actually detect a broken
component — otherwise a green run of the real playbook proves nothing.

## drawershell-focus-trap-broken.json

Mirror of `bas/cases/interaction/drawershell-focus-trap.json`, but it opens the
DrawerShell **@0.1.0** preview harness. That historical version predates
`useFocusTrap` (its version folder has no `useFocusTrap.ts`), so keyboard focus
leaks out of the panel on Tab. The shared `assert-focus-trapped` step therefore
**must fail**.

Run it directly against BAS (not through the test suite):

```bash
# From scenarios/react-component-library
~/.vrooli/bin/browser-automation-studio workflows execute-adhoc \
  --flow-file bas/calibration/drawershell-focus-trap-broken.json --wait --json
```

Expected result: the run **FAILS** with `"status": "EXECUTION_STATUS_FAILED"` and
`"error": "step 5 failed: expected element to exist"` (the `assert-focus-trapped`
step). If it ever COMPLETES, the focus-trap playbook is not discriminating and the
green suite result is not trustworthy.

The playbook uses only literal selectors and a literal scenario name, so no
`project_root`/selector-manifest context is needed for the adhoc run.
