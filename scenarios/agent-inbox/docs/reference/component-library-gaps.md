# Component-library gaps

Agent Inbox is a conversation-family product. Its dual-pane chat workspace,
mobile chat/list transitions, message composer, and operation drawers are the
product surface; the current AppShell/2 contract cannot carry those simultaneous
conversation interactions without moving work owned by the sibling conversation
family plan.

```shell-ejection
{"archetype":"navigated-console","reason":"Agent Inbox is the conversation-family surface. Its chat workspace, mobile list/detail transition, message composer, operation drawers, and settings overlays require simultaneous product-specific regions that AppShell/2 cannot carry without the sibling conversation-family redesign. The exact local frame and its dependent chrome are listed here so the exception is file-scoped and reviewable.","files":["ui/src/App.tsx","ui/src/components/ErrorBoundary.tsx","ui/src/components/chat/AgentStartModal.tsx","ui/src/components/chat/AsyncOperationDrawer.tsx","ui/src/components/chat/ChatHeader.tsx","ui/src/components/chat/MessageAttachments.tsx","ui/src/components/chat/Suggestions.tsx","ui/src/components/chat/agent/tools/ToolCardShell.tsx","ui/src/components/chat/templateEditor/TemplateEditorModal.tsx","ui/src/components/chat/templateEditor/UnsavedChangesDialog.tsx","ui/src/components/layout/Sidebar.tsx","ui/src/components/layout/SidebarPanel.tsx","ui/src/components/layout/sidebar/CollapsedSidebar.tsx","ui/src/components/layout/sidebar/Sidebar.tsx","ui/src/components/scenarios/ScenarioViewer.tsx","ui/src/components/settings/Settings.tsx","ui/src/components/settings/SkillEditorModal.tsx","ui/src/components/settings/UnsavedChangesDialog.tsx","ui/src/components/ui/dialog.tsx"]}
```
