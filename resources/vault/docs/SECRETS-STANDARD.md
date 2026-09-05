# Credential boundary

Resource and scenario manifests are the sole credential declaration source.
Each `credentials.descriptors` entry supplies a backend-neutral `logical_id`, a
field, the runtime injection environment name, and metadata safe to render to
an operator. Credentials are provisioned through `vrooli credentials provision`
using standard input; their values are never passed in command arguments.

Vault is a capability-specific service and may be configured as a scoped
mirror. It is not the authority for ordinary local or desktop credentials, and
the Vault CLI does not inventory, export, or provision resource credentials.

The retired `config/secrets.yaml` files are migration inputs only. Runtime code
must not read them, and no shell-export workflow is supported.
