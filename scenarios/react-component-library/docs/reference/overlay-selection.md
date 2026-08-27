# Overlay selection

Choose the presentation from the user's task, not from the shape a particular viewport happens to
need.

- Use `FullPageDrawer` for a page-scale task that temporarily replaces the workspace while keeping
  the route and source context intact.
- Use `ResponsiveDialog` for a bounded form or decision that should become a bottom sheet on small
  screens and a centered dialog on larger screens.
- Use `AlertDialog` for destructive or consequential confirmation. The cancel action is initially
  focused, Escape cancels, and neither backdrop nor swipe dismisses it.
- Use `SidebarShell` for durable navigation or inspection beside the main content. It is a persistent,
  optionally resizable rail on desktop and an overlay drawer on mobile.
- Use `ContextMenu` for commands attached to a trigger or pointer location. It is anchored on desktop
  and becomes a touch-friendly action sheet on mobile.

Do not start a new overlay from `DrawerShell`, `ResponsivePanel`, `Drawer`, `FocusTrapPanel`, or
`InspectorPanel`. `DrawerShell` is a compatibility delegate to `FullPageDrawer` and `BottomSheet`;
`ResponsivePanel` is replaced by `SidebarShell`; the three stubs are retired. Compose lower-level
behavior only through `useOverlaySurface`, and keep appearance in a version-owned stylesheet as
described by [style ownership](style-ownership.md).

Consumer classes are optional overrides. They may map a scenario's named z tiers or local layout
seams, but an overlay must remain usable and visually complete when no consumer class is supplied.
