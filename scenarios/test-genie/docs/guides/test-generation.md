# Evidence-driven remediation

Test Genie does not generate tests from coverage percentages, source-file
targets, or arbitrary prompts. Start with a completed scenario execution.

1. Open the scenario overview and inspect structured findings, their stable
   IDs, severity/class, locations, remediation guidance, and descriptor docs.
2. Select a coherent bundle of findings and, when relevant, requirement
   evidence from that same scenario snapshot.
3. Select an Agent Manager role and launch the single remediation job.
4. Treat the agent result as provisional. Request verification; Test Genie
   performs a fresh server-owned run and records resolved, remaining, and new
   stable findings.

The CLI equivalent is:

```bash
test-genie remediate my-scenario \
  --execution <completed-execution-uuid> \
  --findings afid:example \
  --role code.default
```

Use the execution API or dashboard to discover valid selectors. Test Genie
rejects arbitrary agent prompts and client-supplied policy controls.
