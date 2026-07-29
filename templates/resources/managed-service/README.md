# {{RESOURCE_DISPLAY_NAME}}

{{RESOURCE_DESCRIPTION}}

This scaffold was generated from the `managed-service` resource template on {{CURRENT_DATE}}.

The shared Vrooli managed-service driver owns start, stop, status, logs, artifact verification, and provider authority. Keep this CLI limited to resource-specific commands; do not add a resource-local supervisor or shell lifecycle path.

The `control-plane` target defaults to a Vrooli-owned `managed-shared` host;
the `desktop-bundle` target defaults to `managed-private`. A desktop app may
select shared reuse only with explicit consent. Choose `managed-shared` only
when the resource can isolate every app scope and its bootstrap material can
live in the OS credential store. Declare
`external_access_capabilities` for attach-only use. External write access does
not grant lifecycle authority. Use `managed-discovered` only for an explicit,
verified executable candidate. Vrooli must never adopt a running host process
or endpoint; after verification it may launch that candidate under its own
supervision, configuration, and state.
