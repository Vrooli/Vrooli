# experience/ — UX Contract

This folder is Portal's experience contract. Requirements say what Portal must
do; experience specs say what the UI must communicate and make operable.

Current pages:

- `dashboard`: chat workspace, grouped sidebar, omnibox, mode indicator, and
  message tree.
- `settings`: integration readiness and behavior override controls.

Use `experience-manager spec validate portal --json` after edits. Raise page
depth only when the UI has stable selectors and the claim can be checked by the
experience phase. Manual claims need attestations with expiry; aspirational
claims are useful intent but never gate.
