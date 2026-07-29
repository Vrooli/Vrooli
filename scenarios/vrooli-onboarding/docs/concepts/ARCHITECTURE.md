# Architecture

The API is a read model over repository manifests and credential-authority
metadata. It is also the sole atomic writer of operator state. The UI and CLI
are thin clients; browser navigation and database progress are not
configuration authority.

Credential values enter through the control-plane provisioning boundary and
never appear in status responses. Integration Hub is a deferred dependency.
