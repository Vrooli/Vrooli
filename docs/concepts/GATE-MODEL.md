# Catalog gate model

The React Component Library validates per asset. A resolver combines universal
rules, kind-based `appliesTo` declarations, and asset-level quality/capability
declarations. Registered runners receive a `gates.Scope`, and evidence is
cached by asset revision plus resolved rule-set digest.

Every finding exposes `rule_source` and `rule_declared_in`, allowing clients to
trace a result to the declaration that imposed it.
