# Credential use in published plugins

Published Agent Plugins carry capability instructions and typed tool metadata.
They never carry a vault item, enrollment token, machine token, browser secret,
or runtime credential value.

Credential work is requested at runtime through the Secrets Manager typed
broker contract. A plugin may request metadata, create a bounded broker
session, execute the operation allowed by that session, and revoke the session.
The result is a bounded projection with an explicit `secret_exposure` label;
the plugin must not turn that result into a generic shell or model tool.

Browser use is a separate protected executor contract. It binds the grant to
an exact origin, account, and document, and rejects cookies, DOM extraction,
arbitrary evaluation, and unrestricted exposure. SSH use returns a signature
and algorithm metadata, never a private key.

The plugin conformance and attestation pipeline scans emitted artifacts for
credential literals. A package that needs a credential must document the
runtime capability and use the broker surface after installation.
