# Host pressure capture

This directory is the evidence capture for the host-pressure plan's Problem
section. It is intentionally immutable after Phase 12 remediation: rediscovery
and regression tests must continue to run against the captured state rather
than silently replacing it with a cleaner host.

The manifest records the sampling intervals and counter values. The files are
sanitized snapshots; they contain no credentials and use repository-relative
fixture names instead of operator home paths.
