# WordPress delivery jobs

These exports and the graded fixture corpus define three unlike client jobs.
Both destinations are owner-local WordPress REST routes backed by the site's
unique-key MySQL ledger. `vrooli-delivery-destination.php` is a Vrooli MU plugin
owned by F11; it is independent of the paid/free third-party plugin cache.

After `vrooli resource wordpress start --site-id <slug>`, install the first-party
destination with `vrooli resource wordpress install-delivery-destination
--site-id <slug> --json`. The owner validates the exact checked-in plugin
signature, copies it only into the matching running site's MU-plugin directory,
and runs PHP syntax validation inside the container. The returned
`vrooli.wordpress.delivery.v1` receipt declares the two REST endpoint paths.
This does not install third-party recipe dependencies. The installer and
verifier ran on local `k3-ground` WordPress 7.1.4/MariaDB 11.4.4: form callback
statuses were 201/200/409 and order statuses were 201/200/201. Unauthenticated
REST POSTs to both routes returned 401.

## A3 forms

`forms/a3-n8n-workflow.json` accepts normalized hook payloads from Contact Form
7, WPForms Lite, or Forminator and validates a stable hidden `event_id`, name, email and message. Field limits count Unicode code points (200/320/4000); the whole request remains byte-bounded at 64 KiB. Elementor-shaped payloads are synthetic compatibility
fixtures only. The destination's idempotency scope is
`provider + formId + eventId`; it returns `created`, `duplicate`, or 409 when
the same key arrives with different content. The WPForms Lite entry ID is not
used because the Lite hook reports zero when entry storage is disabled.

Each form recipe defines a hidden `event_id` field with CSS class
`vrooli-event-id`. The first-party `vrooli-form-event-id.js` gives it a random
ID once per displayed form and rotates it after successful AJAX submission.
Network retries for the same submission therefore carry the same ID, while a
new successful submission receives another ID. The exported n8n HTTP Request
node has no credential or secret. After import, an administrator must bind the
local WordPress HTTP Basic Auth credential backed by an Application Password;
that credential is outside the current F11 proof scope and requires the
operator decision recorded by K3. Do not use a real client destination during
ground runs.

Contact Form 7 (`wpcf7_mail_sent`) and WPForms Lite (`wpforms_process_complete`)
are observed on their documented success hooks;
Forminator's free webhook integration is configured to POST the mapped hidden
event ID and required fields. Its URL query identifies the provider and form ID,
while the body contains Forminator's submitted fields. The MU plugin maps WPForms' field labels to stable
wire keys, and allows only Forminator's documented webhook-port filter to add
the local n8n port 5678. Set Forminator's destination to
`http://n8n:5678/webhook/wordpress-form-intake?provider=forminator&form_id=<id>`;
the query parameters identify its otherwise provider-neutral payload. The n8n
normalizer accepts this form plus the normalized Contact Form 7/WPForms hook
payloads. These source adapters are implemented but real plugin archives and
live payloads still require Scenario Dependency Analyzer support before they
can be exercised.

`shared/README.md` documents the first-party adapters and a nine-case offline
PHP harness. It exercises field mapping, fixed-local-URL enforcement and
destination create/replay/conflict/over-limit behavior with stubbed WordPress
and network functions plus a small in-memory database. It is not plugin or live
WordPress hook proof. The K3 receipt records a URL-guard defect found and fixed
by the harness.

## WooCommerce order sync

`shop-order-sync/order-n8n-workflow.json` verifies WooCommerce's HMAC-SHA256
webhook signature, normalizes only the pinned recipe's synthetic SKUs and sends
one canonical order revision to the local WordPress destination. Configure the
free WooCommerce webhooks for `order.created` and `order.updated`, point them to
the n8n webhook URL, and use a generated local secret in the WooCommerce
webhook and n8n Crypto credential. Bind the local WordPress Basic Auth
credential on the HTTP Request node. The site's unique idempotency key is
`siteId + orderId + date_modified_gmt`; status changes at a new revision are
accepted once, retries of the same revision return `duplicate`, and changed
content under one revision returns 409.

`siteId` and the allowed SKU list in the export are recipe-specific. Update them
only when binding the workflow to a different site's recipe, then rerun the
full fixture suite. Payments, shipping and taxes are disabled in the ground.

## Synthetic content migration

`content-migration/README.md` defines `vrooli.wordpress.content-migration.v1`.
The owner exports only synthetic posts owned by the core blog recipe, validates
their versioned archive, and upserts those records into a second owner-managed
local WordPress site. Canonical content and taxonomy hashes and source keys make retries
idempotent and revisions update in place. Category and tag names are bounded,
assigned through WordPress core taxonomies, and compared on destination; the
receipt lists the verified names per post. A destination slug claimed by another
recipe or source blocks migration before writes. The owner-side job can be
driven with `update-core-content-metric` or `set-core-content-term`, followed by
a read-only plan and `migrate-content`; the live
proof is limited to synthetic local sites and does not transfer client data or
call public WordPress endpoints.

## Synthetic content audit

`content-audit/README.md` defines `vrooli.wordpress.content-audit.v1`. The
read-only owner command audits only synthetic posts owned by `content-core-blog`
for title/excerpt coverage, body length and heading presence, and core category
and tag coverage. Each finding includes a concrete next action; the report
returns per-post findings and a bounded score. The rubric
is a small editorial preflight, not an SEO or accessibility certification.
Eleven graded known answers cover a ready post, a deficient post, a mixed
corpus, isolated failures for each rubric rule, encoded HTML, and an empty corpus. The live `verify-content-audit-ground` command checks and removes a temporary deficient post under its own versioned receipt schema. Live results on the real local WordPress installation are recorded in the K3 content-audit receipt.

## Ground and boundaries

`../ground/levels.json` lists 51 hard known-answer levels across forms, shop,
core content, content migration, and content audit. `../ground/fixtures/*.json` contains
synthetic-only data.
`GradeFixtures` executes all fixture cases against provider normalization,
form/order idempotency ledgers, custom-field operations, read-only migration
plan dispositions, migration create/replay/revision/source collision,
client-data rejection and taxonomy checks, content audit quality cases, and
content hash checks, then compares actual outputs to the
independent expected answers. Go tests and the ground grader require 51/51;
the n8n code nodes are also executed against the same fixture payloads by
`node forms/test-normalizer.js` (18/18, including the 11-case destination bridge) and
`node shop-order-sync/test-normalizer.js` (7/7, including a fixed HMAC-SHA256
known answer). Those Node harnesses execute the exported code-node source but do
not import or run n8n. PHP syntax is separately checked. The local WordPress site, snapshot,
first-party MU-plugin install, and live destination callback behavior are
proven. Third-party plugins, n8n import/execution, and the shared F7 network
path are not proven yet.
K1/F7 owns its runtime and has asked F11 to wait until the versioned companion
contract has live validation.
