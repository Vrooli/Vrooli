# SearXNG resource contract

SearXNG provides the local metasearch capability consumed by `web-search`.
The selected architecture is a single Docker container because the official
SearXNG image is the supported local runtime; Docker is an explicit host
requirement, not a portability claim.

The resource contract is deliberately narrow:

- shared Go control-plane lifecycle, status, and logs
- Go-native `config-apply`, `config-show`, and `config-validate`
- Go-native `engine-health` diagnostic
- SearXNG HTTP JSON search for consumers

Configuration and cache remain independently mounted and durable. Existing
YAML is imported with a backup, unknown upstream keys and existing secrets are
preserved, and output redacts secrets. There is no Compose, Redis, shell, or
terminal-search compatibility surface.
