# Prompt Manager component-library gaps

This product keeps its specialized workspace frame because the library shell does not
carry the simultaneous workflow regions described in the review record.

```shell-ejection
{"archetype":"navigated-console","reason":"Skill Manager is a dual-pane, resizable tree/editor product with entity-specific routes, graph mode, unsaved-change dialogs, settings overlays, and renderer-owned world/board surfaces. AppShell/2 cannot own the simultaneous tree, editor, graph, world, and domain inspector contract without redesigning the product workspace.","files":["ui/src/components/layout/SkillManagerLayout.tsx","ui/src/components/tree/SkillTreeSidebar.tsx","ui/src/components/graph/FlowShell.tsx","ui/src/components/ErrorBoundary.tsx","ui/src/components/editor/tabs/FilesTab.tsx","ui/src/components/editor/teamTabs/MembersTab.tsx","ui/src/components/editor/teamTabs/TeamDashboardTab.tsx","ui/src/components/editor/teamTabs/TeamFilesTab.tsx","ui/src/components/graph/GraphNodePopover.tsx","ui/src/components/shared/ViewOverlay.tsx","ui/src/components/team/CCTeamImportModal.tsx"]}
```
