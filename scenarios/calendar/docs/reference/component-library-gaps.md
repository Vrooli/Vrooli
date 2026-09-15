# Calendar component-library gaps

Calendar is a specialized calendar workspace. Its left rail owns event search,
event-type and status filters, account context, and collapse state; its header
owns date navigation, view switching, and event creation. These controls are
the workspace's primary interaction model rather than application-destination
navigation, so replacing the frame with `AppShell/2` would remove product
behavior. The workspace remains scenario-owned until the library provides a
calendar workspace archetype that can host these controls directly.

```shell-ejection
{"archetype":"navigated-console","reason":"Calendar's product frame owns event filters, account context, date navigation, view switching, event creation, and responsive sidebar state. AppShell/2 provides destination navigation but no calendar-workspace rail or header contract; forcing it here would discard those controls, so the working frame remains scenario-owned until a compatible archetype exists.","files":["ui/src/App.tsx","ui/src/components/Header.tsx","ui/src/components/Sidebar.tsx"]}
```
