# Source distribution security boundary

The source root is treated as untrusted content. Closure analysis refuses
escaping links and private environment/history paths. Publishability scans
secret-like content, nested archives, unknown binary asset licensing, and
resource limits before assembly. Archive verification never extracts into the
source tree and rejects traversal, absolute, duplicate, case-colliding, and
unsupported entries.

No lifecycle, test, CLI, or API path may initialize or mutate Git. Human-only
publication is a control boundary, not a UI convention. Report policy bypasses
with the repository security process and include the exact source, policy,
recipe, closure, and archive identities without including secret contents.
