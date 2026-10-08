# Remediation agent ownership

Test Genie creates remediation intent from immutable execution evidence. Agent
Manager owns every execution-policy decision: protected workspace, sandbox
mode, tool permissions, network access, review/apply behavior, and resource
selection.

The only portable operator choice is an Agent Manager `roleRef`. Test Genie
builds one task packet for one scenario containing selected stable finding IDs,
selected requirement evidence, source-run provenance, descriptor documentation,
and server-owned verification instructions. It does not accept prompts,
preambles, tool lists, command allowlists, network toggles, runner/model names,
or workspace policy overrides.

Agent completion is provisional. A remediation job becomes verified only after
Test Genie records a fresh execution and compares the selected stable finding
IDs with its new findings artifact. The job record links source execution,
Agent Manager task/run, verification run, and the resolved/remaining/new delta.
