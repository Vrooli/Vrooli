# WordPress core content migration job

Contract: `vrooli.wordpress.content-migration.v1` (`contract.schema.json`).
The resource owner exports only posts owned by the `content-core-blog` recipe
and tagged `client_type=synthetic`. It validates field bounds and source slug
uniqueness, computes a canonical SHA-256 per record and a SHA-256 for the full
archive, then runs a destination slug ownership preflight before any writes.
The archive preserves sanitized post body, excerpt, category and tag names,
publish/draft status, and three typed custom fields together. Destination plans
compare existing taxonomy as well as the stored hash; the verified migration
receipt lists the exact category and tag names per post. Term names are
bounded and validated, and the hash changes when content, metadata or taxonomy
changes.

```text
vrooli resource wordpress start --site-id source-site
vrooli resource wordpress apply-recipe --site-id source-site --recipe-id content-core-blog
vrooli resource wordpress start --site-id destination-site
vrooli resource wordpress export-core-content --site-id source-site --json
vrooli resource wordpress plan-content-migration --source-site-id source-site --target-site-id destination-site --json
vrooli resource wordpress migrate-content --source-site-id source-site --target-site-id destination-site --json
# Retry the same migration to exercise duplicate handling.
vrooli resource wordpress migrate-content --source-site-id source-site --target-site-id destination-site --json
vrooli resource wordpress stop --site-id destination-site
vrooli resource wordpress stop --site-id source-site
```

The plan performs read-only source and destination queries and returns one
`create`, `update`, `duplicate`, or `conflict` disposition per slug plus counts
and the archive checksum. Review this receipt before applying.

`export-core-content` returns the exact versioned archive consumed by the plan
and migration operations. It is useful for review, handoff and reproducible
storage before a destination is chosen. The output uses this job's
`contract.schema.json`; it remains limited to recipe-owned synthetic records.

For synthetic changed-content passes, update the source recipe-owned field with
`update-core-content-metric` or add/remove a bounded category/tag with
`set-core-content-term`, inspect the read-only plan, and migrate again. The destination reports one
update for changed content and duplicates for unchanged records. Replays with
the same source key and content hash do not create posts. A slug owned by a
different recipe/source blocks the full migration before writes. The graded
ground covers plan/create/retry/revision update, source collision, client-data
rejection, taxonomy hash changes and a bad content hash.

This prototype deliberately does not export arbitrary/client content or call a
public API. Both local sites must be running under the WordPress resource owner.
The multi-record transfer performs a complete conflict preflight but does not
wrap WordPress APIs in one database transaction; a database failure after the
write phase begins can leave a partial transfer. Rerunning is safe for records
whose content hash was committed.
