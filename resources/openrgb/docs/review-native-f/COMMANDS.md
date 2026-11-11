# Whole native source review F commands

All commands ran in `/home/matthalloran8/Vrooli-native-lighting` unless named otherwise. Every source-aware command supplied:

```sh
export VROOLI_ROOT=/home/matthalloran8/Vrooli-native-lighting
export VROOLI_SOURCE_ROOT=/home/matthalloran8/Vrooli-native-lighting
export VROOLI_PROTO_ARTIFACT_ROOT=/home/matthalloran8/Vrooli-native-lighting/.vrooli/work/proto-artifacts
```

Exact commands, cwd, exit codes, selected test names/counts, complete logs and SHA256 receipts are in `log-receipts.json`. `proto-preflight.log` is the canonical read-only `packages/proto` verifier, exit0; no generation occurred. The new recovery test was submitted while that preflight attachment was in flight; both completed successfully, and no generated-symbol failure or drift occurred. Subsequent broad commands began after the verifier completed.

Private fixture homes/state are supplied by existing maintained fixture owners. The added overlay uses the public native fixture's private resource/intent/fence roots. The full runtime run explicitly supplies the preexisting approved offline AppImage fixture, which is copied/extracted only into its own temporary home by the maintained test and cleaned up afterward. Vendor executable/SDK/hardware are never invoked. The source regression uses an overlay new virtual `_test.go` file, not a production seam.

The existing shared search-hub/Prompt Manager services were read for recall/skills. Recall's source-ledger provider timed out; discovery was not treated as complete history. An accidental broad read-only `rg --files` lookup under `/tmp` emitted protected-directory errors; it was abandoned and the exact approved fixture path was read from the prior D report instead. No artifact, service, target or device mutation followed that lookup. No raw dependency installation, generation or Git mutation occurred.
