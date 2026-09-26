# React Vite Tailwind Adapter

This adapter translates `templates/design/vrooli-command-display/DESIGN.md` into assets consumed by React/Vite/Tailwind scenarios.

- `tokens.css` defines dark-first display tokens, kiosk utilities, theme-specific CSS variables, motion helpers, focus behavior, reduced-motion fallbacks, and safe-area helpers.
- `tailwind.theme.json` mirrors the shared token names for Tailwind configuration.
- `adapter.json` owns copy rules. Scenario templates declare the adapter they consume; they should not duplicate these copy rules.

Generated scenarios may extend local page themes, scenes, and chart tokens, but root-level `DESIGN.md` remains the design contract.
