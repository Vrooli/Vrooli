# Notifications: who gets told, and on which channel

An incident that no person sees is an incident that did not happen. This page
names the two settings that make a host's incident notifications reach
someone, and states honestly which channels can deliver on which operating
system.

## The recipient

notification-hub receives every durable event (autoheal incidents among them)
and needs a recipient subject to route them to. It resolves one in this
order:

1. `VROOLI_NOTIFICATION_RECIPIENT` in the hub's environment: the explicit
   override.
2. `notifications.recipient` in `operator-state.json`: the durable operator
   setting. Set it with:

   ```bash
   vrooli-onboarding host set-recipient --subject <subject>
   ```

3. Neither set: the notification is still recorded, owned by the source
   scenario, and its single delivery attempt is `unroutable` with the reason
   naming both settings. Nothing is dropped silently, and
   `coverage-delivery-reach` in vrooli-autoheal reports the gap.

The subject must have at least one channel address registered with the hub:

```bash
notification-hub recipients address-upsert --help
```

## Channels by operating system

| Channel | Linux | macOS | Windows | Requires |
| --- | --- | --- | --- | --- |
| `linux_notification` | host-verified (notify-send over the session bus) | not offered | not offered | a session bus (`DBUS_SESSION_BUS_ADDRESS` or `$XDG_RUNTIME_DIR/bus`), `DISPLAY` or `WAYLAND_DISPLAY`, `notify-send` installed; a headless host reports the channel unavailable, never delivered |
| `macos_notification`, `imessage` | not offered | adapter present, fixture-verified | not offered | a logged-in desktop session |
| `web_push` | yes | yes | yes | a registered browser push subscription |
| `email` | yes | yes | yes | SMTP settings in the hub's environment |
| `in-app`, `telegram`, `slack` | yes | yes | yes | the shared switchboard channel registry |

## What "reached" means

`coverage-delivery-reach` counts an open critical incident as reached only
when notification-hub recorded a `delivered` attempt for it. `unroutable` and
`failed` attempts prove the pipeline ran; they are listed under
`undeliveredOutcomes` with the outcome so the gap names its cause. The check
reads the hub's projection at `/api/v1/integrations/deliveries`; an
unreachable hub makes the check `undetermined`, never `ok`.
