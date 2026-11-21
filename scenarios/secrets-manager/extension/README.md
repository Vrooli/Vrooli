# {{APP_NAME}}

This MV3 extension uses the Secrets Manager native-messaging host for
grant-bound, origin-bound field filling. It keeps owner tokens, native session
capabilities, and returned credential values in service-worker or call-stack
memory only. Approved origins and non-sensitive item metadata may be stored in
browser local storage.

The extension does not submit forms. A human approves the current site,
selects safe account metadata, and chooses the item revision and field before
a fill. Save and update read the visible login fields only after an explicit
popup action and send them through the native authority; returned credentials
are never placed in browser storage.
