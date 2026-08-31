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

## Blocking policy

The blocking set is limited to gates that have an executable in-process
runner, attribute findings to an asset or the corpus, and have a calibration
fixture. The five experience gates (`unit`, `interaction`, `accessibility`,
`responsive`, and `visual`) remain declared and available to the external
experience runner, but are advisory in the deterministic release path because
their registry definitions do not execute that external runner in-process.

`catalogvalidate` enforces this boundary: a catalog gate with `blocking: true`
must resolve to a registered definition with a non-nil runner. Adding a
runnerless blocking gate is therefore a validation error, not a silently
unmeasured release check.

The current blocking set is intentionally explicit. Each member has a
different release-safety reason; the shared requirement is that the runner
can produce attributable evidence.

| Gate | Reason it blocks | Gate | Reason it blocks |
| --- | --- | --- | --- |
| `graph-reconciled` | release graph is coherent | `dependency-rank` | lower-rank dependency direction |
| `self-hosting` | library remains adoptable | `bas-genericity` | workflows remain asset-neutral |
| `token-vocabulary` | public token names are valid | `fallback-parity` | fallback behavior stays aligned |
| `kit-compatibility` | adopted kit contract holds | `affinity-compatible` | design affinity is respected |
| `token-ramp-complete` | token migration is complete | `scenario-token-requirements` | host token contract is present |
| `released-version-immutable` | released authored bytes are frozen | `release-provenance` | release origin is auditable |
| `version-mirror-integrity` | durable release bytes are readable | `version-liveness` | imports resolve to live versions |
| `specifier-shape` | new imports use governed major lines | `types` | published types compile |
| `api` | public API contract is valid | `performance` | declared budgets are enforced |
| `console-clean` | release emits no console defects | `composition-contract` | composition claims remain consistent |
| `composition-contract` | composition claims are evidenced | `examples` | every release has usable examples |
| `fixture-adversarial` | fixtures exercise failure boundaries | `tokens` | token usage is governed |
| `lifecycle` | version lifecycle metadata is consistent | `i18n` | user-facing text is localized |
| `selector-coverage` | automation selectors are stable | `restyle-contract` | consumers have a styling seam |
| `manifest-identity` | catalog and library identities join | `manifest-metadata` | manifest metadata is complete |
| `overlay-surface-composition` | overlays use the shared surface seam | `shared-style-ownership` | styles have one owner |
| `style-injection` | styles are injected through the contract | `foreign-token-classes` | foreign token classes are rejected |
| `utility-class` | utility-class debt stays bounded | `consumer-pin` | consumers do not pin unsafe releases |
| `deprecated-import` | imports avoid superseded versions | `provenance-stamp` | authored source carries provenance |
| `story-grammar` | story contracts remain parseable | `story-distinctness` | stories provide distinct states |
| `evidence-freshness` | evidence matches current source |  |  |

The advisory set also includes `rtl`, `reduced-motion`, and `composition`:
their current corpus evidence is incomplete or not predominantly measured.
They remain runnable and visible in the matrix, but blocking them would turn
missing evidence into release enforcement. The five experience gates (`unit`,
`interaction`, `accessibility`, `responsive`, and `visual`) are likewise
advisory because their browser/test-genie runners are external to the
in-process catalog runner.
