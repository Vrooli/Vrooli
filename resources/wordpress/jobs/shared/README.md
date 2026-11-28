# WordPress form adapter source

`vrooli-delivery-destination.php` contains the first-party MU-plugin routes and
success-hook adapters for Contact Form 7 and WPForms Lite. Forminator uses its
free webhook action and the plugin's fixed-port allow-list filter.

Run the offline source-level adapter harness with:

```text
php resources/wordpress/jobs/shared/test-adapters.php
```

The nine-case harness stubs WordPress APIs, `wp_remote_post` and a small
in-memory database. It checks hook arity, payload field mapping, disabled
WPForms entry-ID behavior, the fixed local URL, Unicode codepoint limits,
Forminator's port, rejection of unsafe webhook URLs, and the first-party
destination's create/replay/conflict/over-limit behavior. It does not load
WordPress or the third-party plugins and does not count as plugin runtime proof.
The A3 n8n node-code harness in `jobs/forms` also sends its supported Contact Form 7, WPForms Lite and Forminator normalized events through this stubbed destination in one process; its synthetic Elementor compatibility case is graded but never sent to the actual-provider allowlist.
