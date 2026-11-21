# Storage-manager typed reader

Infrastructure-manager reads storage-manager through the typed
`/api/v1/infra-health/storage` feed. The source adapter projects the feed onto
headroom cells H1 through H6 and preserves the feed's trust verdict.

The adapter reports the source as unavailable when storage-manager cannot be
reached or when a required observation is stale or unattributed. It does not
walk storage roots and it does not infer a healthy projection from missing
values.

H2 is untrusted when the slope is stale or when unattributed growth exceeds
the configured tolerance. H3 reports declared-ceiling coverage. H4, H5, and
H6 report recovery efficacy, budget truth, and hot governed-root writers.
