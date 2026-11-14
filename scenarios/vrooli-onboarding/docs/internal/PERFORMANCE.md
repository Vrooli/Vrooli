# Performance

Read models derive only the manifests needed for selected scenarios. Operator
state writes use a single atomic file replacement. Readiness checks use bounded
metadata queries and must not resolve or scan credential values.
