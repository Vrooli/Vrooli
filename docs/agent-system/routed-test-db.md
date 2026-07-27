# Routed Test Storage

Mutating browser workflows run against a live scenario process only when its
leased test storage is installed and verified. This applies to both database
and filesystem writes; restarting a scenario is not an isolation mechanism.

The test-mode request header routes eligible persistence operations to the
lease. When either storage leg is unavailable, workflow validation refuses the
mutating case rather than falling back to primary storage. The lease teardown
reports test-write and primary-leak counters as durable run evidence.

Use `storage-health validate scenario <name>` before enabling mutating E2E for
a scenario. It verifies the routed database/file seams and namespaced external
stores. Use `vrooli scenario test <name>` for the server-owned suite; Test
Genie records whether the scenario source remained stable.

For workflow authors, use explicit execution-mode and reset labels. Observer
workflows must contain only read-only nodes; mutating workflows require the
routed isolation contract and are rejected if it cannot be proven.
