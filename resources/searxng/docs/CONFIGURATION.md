# SearXNG configuration

Settings live at `${RESOURCE_CONFIG_DIR}/settings.yml`. The Go CLI owns a
small policy overlay while preserving unknown upstream YAML fields and existing
session secrets.

```bash
resource-searxng config-apply
resource-searxng config-show
resource-searxng config-validate
```

`config-apply` imports an existing document, backs it up next to the file with
a timestamp, validates it, then atomically replaces it. Its report never
prints a secret. A fresh configuration generates a cryptographically random
session secret; use `--secret-file` only when supplying a secret from an
approved secret-management path.

The owned contract keeps upstream defaults enabled, materializes a session
secret, enables metrics for engine diagnostics, and requires JSON plus HTML
search formats. Invalid YAML, a missing secret, or removal of JSON format is
rejected without changing the current file.
