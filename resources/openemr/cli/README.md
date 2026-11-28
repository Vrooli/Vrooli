# OpenEMR CLI ownership

The resource uses the shared `vrooli resource` CLI entrypoint; OpenEMR-specific operations are implemented by `internal/cli/resourcehandlers/handlers_openemr.go` and `internal/resources/openemr`. There is no separate executable or shell wrapper to drift from the resource contract.

Run `vrooli resource openemr --help` for the owner-managed isolated-instance command family.
