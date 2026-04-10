# ASSUMPTIONS

- App preview iframe URLs are either same-origin proxy URLs or reachable scenario-local URLs.
- Bridge capabilities can vary by preview target; UI must gate screenshot/inspect actions by caps.
- Pane-local state persistence is safe to store in browser localStorage for operator workflows.
