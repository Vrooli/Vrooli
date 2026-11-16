# WordPress resource design

## Placement

`wordpress` is a managed external-integration resource with a programmatic
operator surface. Its core job is to create local, disposable WordPress sites
that other delivery workflows can call. The stable composition seam is a
versioned recipe document: plugin selection, site settings, content, expected
answers, and resource image profile travel together and can be reused by a
standalone client workflow or an F7 sandbox. There is no independent UI.

Lifecycle must be owner-controlled and label-verified: acquire the declared
images, start a site and private database on an internal, no-egress network,
stop only the owner's containers, reset only its dedicated Docker volumes, and snapshot a
bounded, consistent site plus database. A site recipe never contains an
arbitrary image, command, bind mount, host port, or remote destination. The
private network IP is returned only for local proof and authorized companion use;
the site has no published host port.

The first-party `vrooli-delivery-destination.php` MU plugin exposes the
versioned contract `vrooli.wordpress.delivery.v1`: `POST
/wp-json/vrooli/v1/form-intake` and `POST
/wp-json/vrooli/v1/shop-order-sync`. Both routes require an authenticated
administrator. A unique-key MySQL receipt table gives durable
create/duplicate/conflict behavior across restarts, unlike an in-memory
rehearsal ledger. The owner installs the checked-in plugin on an owned running
site and can exercise its callbacks through the local WP-CLI PHP runtime without
creating credentials or making network calls.

## Recipe families

- `forms-contact-form-7`: a practical business contact form with required
  name/email/message, validation and a known-answer submission.
- `forms-wpforms-lite`: same intake goal, using the free WPForms Lite plugin.
- `forms-forminator`: same intake goal, using Forminator's free plugin.
- `shop-woocommerce`: WooCommerce with a synthetic catalog and order fixtures;
  payment gateways remain disabled.
- `content-pods-blog`: a blog with a Pods custom content type and custom fields.

The original A3 case also asks for WPForms and Elementor-shaped compatibility.
The Elementor Pro form itself is not installed or licensed. The A3 mapper should
test an explicitly labeled Elementor-shaped synthetic payload while each
installed recipe exercises its actual free plugin's request and submission
hook. Those real plugin archives remain unstaged until SDA supports the
WordPress package ecosystem.

## Plugin license and fit evidence (2026-11-14)

- Contact Form 7 describes itself as open source and lists its WordPress.org
  release and compatibility. Its documentation supports reading the submitted
  fields from the submission object: <https://wordpress.org/plugins/contact-form-7/>,
  <https://contactform7.com/2020/07/28/accessing-user-input-data/>.
- WPForms Lite is distributed on WordPress.org; its vendor documents the
  successful `wpforms_process_complete` hook and notes that Lite does not
  assign a persistent entry ID. WPForms' own terms say the plugin is GPL 2.0+:
  <https://wordpress.org/plugins/wpforms-lite/>,
  <https://wpforms.com/developers/wpforms_process_complete/>,
  <https://wpforms.com/terms/>.
- Forminator is free/open source in the official WordPress.org directory:
  <https://wordpress.org/plugins/forminator/>.
- WooCommerce describes its core platform as open-source and free:
  <https://wordpress.org/plugins/woocommerce/>.
- Forminator's vendor documents its free webhook integration with n8n and the
  POST setup. The same documentation explains the `forminator_webhook_allowed_ports`
  filter used by F11 to allow the local n8n port: <https://wpmudev.com/docs/wpmu-dev-plugins/forminator/#webhook>,
  <https://gist.github.com/wpmu-docs/385fba1a133c96f4e64ee79f9728fcad>.
- WooCommerce documents order webhooks and the HMAC-SHA256 signature header:
  <https://developer.woocommerce.com/docs/apis/rest-api/v2/webhooks/>.
- Pods advertises all custom-field types free in WordPress.org's directory:
  <https://wordpress.org/plugins/pods/>.
- Ninja Forms' free core stores submissions, but its webhook add-on is a
  separate paid extension; it is not selected for the three recipes:
  <https://wordpress.org/plugins/ninja-forms/>,
  <https://ninjaforms.com/extensions/webhooks/>.

Versions and their license claims are rechecked against the plugin owner before
recipe install. The recipes must not install premium add-ons, external SaaS
connectors, themes or extensions. Actual WPForms Lite entry identity needs a
site-generated idempotency token; a zero entry ID is not a stable retry key.

## F7 boundary

The WordPress resource and A3/site artifacts are independently owned by F11.
No file or runtime under `client-rehearsal`, Workspace Sandbox, or n8n is owned
here. A connection test against F7 is admitted only after K1 confirms its live
runtime path is ready and the owners agree the exact synthetic service wiring.
