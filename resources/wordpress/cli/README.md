# WordPress command ownership

This task-scoped resource is controlled by the Vrooli root command surface:
`vrooli resource wordpress <acquire|start|stop|reset|status|snapshot|apply-recipe|install-delivery-destination|verify-delivery-destination|grade-ground>`.
It intentionally has no separate resource CLI binary. The control-plane owner in
`internal/resources/wordpress` applies the validated manifest and image profile
before it runs Docker operations. `grade-ground` executes the checked-in
synthetic cases and known answers; `verify-delivery-destination` exercises the
first-party plugin's PHP callbacks using only synthetic data and the owned
site's local WP-CLI runtime.
