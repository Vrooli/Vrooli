# SearXNG resource contract

SearXNG provides the local metasearch capability consumed by `web-search`.
The selected architecture is a managed native service composed from a
checksum-pinned CPython runtime, locked wheels, and a reviewed SearXNG source
tree. It is portable across the declared Linux, macOS, and Windows targets;
the runtime has no external daemon dependency.

The resource contract is deliberately narrow:

- shared Go control-plane lifecycle, status, and logs
- Go-native `config-apply`, `config-show`, and `config-validate`
- Go-native `engine-health` diagnostic
- SearXNG HTTP JSON search for consumers

Configuration and cache remain independently mounted and durable. Existing
YAML is imported with a backup, unknown upstream keys and existing secrets are
preserved, and output redacts secrets. There is no Compose, Redis, shell, or
terminal-search compatibility surface.
