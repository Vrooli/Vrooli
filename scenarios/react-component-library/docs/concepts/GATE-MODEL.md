# Catalog gate model

Catalog validation is asset-first. The resolver selects a rule set for each
asset from three declaration layers: universal gates, the gate's `appliesTo`
kind list, and asset-level `qualityGates`/capability declarations. Opt-outs are
explicit and reasoned.

Each executable gate is registered in `api/internal/gates`. Its runner accepts a
`gates.Scope`; an empty asset list is the cold full-corpus path, while a listed
asset id is the targeted path. Corpus gates opt into `CorpusScoped` and run once.

An asset verdict is reusable only when both its source revision and resolved
rule-set digest match. The digest includes the resolved bindings and the
registry's determinism inputs. This makes a source edit invalidate that asset
and a rule declaration edit invalidate precisely the affected rule sets.

Findings carry `rule_source` (`universal`, `kind`, `asset`, or `corpus`) and
`rule_declared_in`, so the API and component test panel can explain both the
defect and the declaration that imposed it.
