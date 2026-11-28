# agentbrief-go

`agentbrief-go` builds bounded context briefs from search-hub results and never
stores them. It is intentionally a leaf Go module: Portal and other scenario
runtimes may import it, but shared foundation packages must not.

The package owns normalization, the precision gate, provenance-derived trust
classification, consumer redaction, and the two rendered forms used by Portal.
