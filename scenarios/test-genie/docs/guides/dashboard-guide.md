# Dashboard guide

The Test Genie dashboard has four operator surfaces: Dashboard, Runs, Docs,
and Self-Health.

The Dashboard points operators to scenario execution evidence. Runs lists
scenarios and execution history. A scenario overview renders structured phase
findings from its latest completed execution and exposes the remediation
builder.

The builder ranks deterministic gating findings before advisory findings,
retains stable IDs and locations, and explains each bundle. Operators can add
requirement evidence from the same scenario snapshot, select one Agent Manager
role, and launch one remediation job. While a job is active, the dashboard
shows that job instead of another launch control.

Agent completion is not a success claim. The job panel labels it provisional
until the operator requests verification. Verification is a server-owned Test
Genie execution; the final job view reports the stable-ID delta and retains
links to source run, Agent Manager run, and verification run.

There is no Generate tab, coverage target form, file picker, generic prompt
editor, or Test Genie security-policy control. Agent Manager profiles enforce
execution policy.
