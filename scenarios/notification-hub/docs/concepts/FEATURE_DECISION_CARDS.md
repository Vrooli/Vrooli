# Feature loss decision cards

Feature loss cards are a decision packet for a feature seen in a served build and missing
from a later served build. The Hub checks the card file and its explainer together. A card
does not prove that the feature was useless or that the source revision served the claimed
build; it binds evidence and the operator's answer to measured identities supplied by the
producer.

## Card contract

Attach one `research-feature-decision/v2` JSON card as evidence and its self-contained HTML
page as the ask explainer. The card includes:

- a stable card ID and served feature ID;
- the screen, taps to reach it, and component/source location when known;
- a one-sentence summary and embedded before/after or component/crawl images;
- `keep` and `drop`, each with a plain-language effect;
- a recommended choice and a plain-language reason;
- short definitions for any technical terms;
- a finding version, kind, and distinct measured `before_build` and `after_build` IDs;
- a stable `subject_key` derived from the card ID, feature ID and finding version;
- `pending` with no answer, or `resolved` with exactly one `keep`/`drop` answer.

The HTML page carries a `<main>` element with `data-research-feature-card="v2"`,
`data-card-id`, `data-feature-id`, `data-finding-version`, `data-before-build`, and
`data-after-build` matching the JSON card. The ask uses that `subject_key`, names the card's
feature in its question, and offers exactly the same `keep`/`drop` labels and recommendation.
This keeps one open Hub ask for the exact finding version and binds the operator's answer to
that finding. The `subject_key` is a stable hash of the card ID, feature ID and finding
version; the Hub returns that key with the answer so the chosen disposition can be joined to
the exact served evidence. The page shows the summary, recommendation reason,
both option effects, the same images, and buttons with `data-od-answer="keep"` and
`data-od-answer="drop"`. Those buttons prefill the Hub answer form; the Hub records the
answer on the exact ask/card, while the producer's disposition record binds that answer to
the finding and evidence version.

`research-feature-decision/v1` cards remain readable during migration, but the packet check
advises that they lack exact finding/build binding. A malformed or incomplete v2 card
produces `feature_card_invalid`; a valid model whose rendered page is absent or mismatched
produces `feature_card_incomplete`. These are packet advice, consistent with the Hub's
existing non-blocking check policy. Producers should treat either warning as a failed card
readiness check before sending it for an answer.

## Disposition closure

An answer is not proof that the code change is complete. `keep` closes only after a later
served inventory finds the feature present. `drop` closes only after source/dead-code review
and a later inventory confirms that the feature is absent. Until then the finding remains
open and tied to its original build pair. A knip report is evidence to inspect; it does not
choose `drop` automatically.
