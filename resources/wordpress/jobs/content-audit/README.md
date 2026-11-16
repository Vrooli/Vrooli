# WordPress synthetic content audit

Report contract: `vrooli.wordpress.content-audit.v1` (`contract.schema.json`).
Live-verifier receipt contract: `vrooli.wordpress.content-audit-ground.v1`
(`ground-receipt.schema.json`).
`vrooli resource wordpress audit-core-content --site-id <site-id> --json` reads
only `content-core-blog` posts that carry the `client_type=synthetic` marker.
It does not change posts, install plugins, inspect other post types, or send
content over the network.

The editorial preflight checks a small repeatable set of basics: title has at
least three words, excerpt is present and has at least six words, body has at
least twenty words and an HTML heading, and the post has at least one category
and one tag. Each failed check adds a named finding with a concrete next action and a fixed
penalty to a 100-point score. Any finding sets `needs-review`; no findings sets `ready`.
This score is a local checklist result, not an SEO, accessibility, brand, or
editorial-quality certification.

The receipt includes per-post word and heading counts, score, disposition, and
finding codes, plus aggregate ready/review counts. Hard ground exercises a
clean post, a deliberately deficient post, a mixed corpus, each rule in isolation,
HTML entities and nested tags, and an empty corpus. Known answers grade counts,
headings, word counts, score penalties, dispositions, findings, and next actions. `verify-content-audit-ground` creates one temporary deficient synthetic post,
runs the same live audit against it, verifies expected findings, and deletes the
post in a `finally` cleanup path. The ordinary live report on `content-core-blog`
posts is also recorded. No client text is used.
