# WordPress Site Workshop resource

Current F11 evidence, source links, measured hours, and open proof gates are
collected in the [category review](review/README.md).

`wordpress` owns disposable local WordPress environments for freelance forms,
shop and content work. WordPress, MariaDB, and WP-CLI run in digest-pinned
containers. Site and database content live in resource-owned Docker volumes;
each site uses an internal Docker network with no container egress. No host
port is published. The owner returns the site's private address for local proof
and approved companion connections.

## Owner lifecycle

Use the resource-owned command family rather than running Docker directly:

```text
vrooli resource wordpress acquire
vrooli resource wordpress start --site-id <site-id>
vrooli resource wordpress status --site-id <site-id>
vrooli resource wordpress apply-recipe --site-id <site-id> --recipe-id <recipe-id>
vrooli resource wordpress verify-content-ground --site-id <site-id>
vrooli resource wordpress audit-core-content --site-id <site-id> --json
vrooli resource wordpress verify-content-audit-ground --site-id <site-id> --json
vrooli resource wordpress update-core-content-metric --site-id <site-id> --slug <post-slug> --value <integer>
 vrooli resource wordpress set-core-content-term --site-id <site-id> --slug <post-slug> --taxonomy category|tag --term <name> --present true|false
vrooli resource wordpress plan-content-migration --source-site-id <site-id> --target-site-id <site-id>
vrooli resource wordpress migrate-content --source-site-id <site-id> --target-site-id <site-id>
vrooli resource wordpress install-delivery-destination --site-id <site-id>
vrooli resource wordpress verify-delivery-destination --site-id <site-id>
vrooli resource wordpress grade-ground
vrooli resource wordpress snapshot --site-id <site-id>
vrooli resource wordpress reset --site-id <site-id>
vrooli resource wordpress stop --site-id <site-id>
```

`start` requires the three images to have been acquired. It generates local
database credentials under the resource's private state directory and never
prints them. `stop` leaves the named site's durable data intact. `reset`
removes only that owner's containers, network, named volumes and credentials, then
recreates a clean site. `snapshot` writes a bounded database dump and content
archive beneath `snapshots/`; reset does not delete snapshots.

No service is started automatically on install. The first-party destination
installer validates and installs only the checked-in MU plugin; its live
verification writes synthetic rows and checks create, duplicate, identity
conflict, and new-order-revision responses. The grader executes all hard
fixtures and compares results with known answers. Third-party recipe
dependencies must be staged by Scenario Dependency Analyzer; this resource
does not download or install third-party packages itself.

The core blog recipe is an explicit degraded-infrastructure path. It creates
two synthetic posts with bounded slugs, statuses, sanitized HTML body, excerpt,
core categories and tags, and typed custom fields, then verifies an idempotent
rerun against the matching site-owned recipe marker.
It does not create a Pods custom post type and does not replace the Pods recipe.
`verify-content-ground` independently inserts and removes a temporary synthetic
post while proving field create/read/update/repeat behavior in the live
WordPress database.

`migrate-content` exercises a separate local content-transfer task. It exports
only posts tagged by `content-core-blog` whose client type is `synthetic`,
validates a versioned, bounded archive preserving sanitized body, excerpt, categories, tags and
custom fields, hashes canonical content and taxonomy, then preflights every destination slug before writing. A destination slug already
owned by another recipe or source is rejected before changes. Identical replay
is counted as duplicate; changed content from the same source key updates in
place. `update-core-content-metric` provides a controlled synthetic content
revision. `set-core-content-term` adds or removes one bounded category/tag from
a recipe-owned synthetic post, preserving the other core taxonomy terms; this
provides a controlled real taxonomy revision for migration planning and replay. Neither operation accepts arbitrary content or
client records. Multi-post writes use a conflict preflight, but database
failures after the write phase begins may leave a partial transfer; rerunning
is safe for completed records.
`plan-content-migration` returns per-slug create/update/duplicate/conflict
decisions and the archive checksum using read-only WordPress queries.
`export-core-content` returns that same bounded, versioned synthetic archive
directly for reviewer inspection or a downstream local job; its schema is the
content-migration job contract.

## Approved free recipes

| Recipe | Free packages | Purpose |
|---|---|---|
| `forms-contact-form-7` | Contact Form 7 | Validate a real visitor-style contact submission |
| `forms-wpforms-lite` | WPForms Lite | Validate the free Lite hook and payload path |
| `forms-forminator` | Forminator | Validate a second submission-storage model and hook |
| `shop-woocommerce` | WooCommerce | Synthetic products and orders, with payments disabled |
| `content-pods-blog` | Pods | Blog and custom-field management |
| `content-core-blog` | WordPress core | Synthetic posts and verified `post_meta` fields without third-party plugins |

The packages use exact WordPress.org release versions declared in
`recipes/plugins.lock.json`. No premium add-ons, themes or paid plugins are
included. The owner refuses to apply a recipe if an exact package is not
present in its verified local package cache.

## Composition

The resource's stable interface is the versioned recipe and site API. F7 may
declare a site instance as a Workspace Sandbox companion when its owner runtime
is ready and approves the exact network relationship. Until then, F7 integration
is a separate, unqualified edge. The resource itself can be used independently
through its managed control-plane command and private WordPress URL.

## Proof boundaries

Focused Go tests and `vrooli resource wordpress grade-ground --json` execute
the 44 synthetic hard answers (forms 9/9, shop 7/7, content, migration and audit 28/28). The live
`k3-ground` Docker proof installs the site and first-party MU plugin, verifies
form callback statuses 201/200/409 and order statuses 201/200/201 against
WordPress and MariaDB, confirms both unauthenticated REST writes return 401,
snapshots the site, and stops both containers. Third-party plugins, their real
form hooks, n8n import/execution, and the shared F7 network path remain open;
see the K3 checkpoint and `jobs/README.md`.
