# Component-library gaps

```shell-ejection
{"archetype":"navigated-console","reason":"AppShell/2 owns the Agent Manager application frame. These files are independent run-detail and dialog surfaces with their own viewport or overlay geometry; treating them as shell chrome would change feature behavior. They remain scoped product surfaces until the corresponding dialog/detail primitives are proven.","files":["ui/src/components/RunDetail.tsx","ui/src/components/patterns/MasterDetail/DetailModal.tsx","ui/src/components/ui/dialog.tsx"]}
```
