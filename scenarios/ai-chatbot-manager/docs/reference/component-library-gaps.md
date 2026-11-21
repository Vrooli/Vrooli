# AI Chatbot Manager component-library gaps

AI Chatbot Manager's application frame is owned by `AppShell/2`. The page and
card headers below are feature content: they label chatbot lists, editor
sections, analytics panels, and the conversation sandbox. They are not a
second application shell, so they remain scenario-owned under a scoped
ejection while the library owns the routed frame.

```shell-ejection
{"archetype":"navigated-console","reason":"The App.tsx connection state is a transient loading/error fallback, and the remaining headers label feature pages, editor sections, analytics panels, and conversation-sandbox content inside the AppShell/2 main pane; moving these into application chrome would change feature semantics.","files":["ui/src/App.tsx","ui/src/components/Analytics.tsx","ui/src/components/ChatbotEditor.tsx","ui/src/components/ChatbotList.tsx","ui/src/components/TestChat.tsx"]}
```
