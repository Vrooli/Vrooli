# Browser Automation Studio component-library gaps

Browser Automation Studio is a workflow editor. Its workflow route owns a canvas header, resizable node sidebar, keyboard shortcut surface, and editor-specific breadcrumbs; these controls coordinate the canvas rather than provide application destination navigation.

```shell-ejection
{"archetype":"navigated-console","reason":"WorkflowEditorView's Header and Sidebar are editor-owned canvas controls: breadcrumbs, editable workflow identity, node palette, resizing, and keyboard shortcuts. AppShell/2 does not carry this canvas interaction model, so the editor chrome remains scenario-owned until a matching archetype exists.","files":["ui/src/views/WorkflowEditorView/index.tsx","ui/src/shared/layout/Header.tsx","ui/src/shared/layout/Sidebar.tsx"]}
```
