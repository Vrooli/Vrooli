# React Vite Tailwind Adapter

This adapter translates `templates/design/vrooli-default/DESIGN.md` into assets consumed by `templates/scenarios/react-vite`.

- `tokens.css` defines CSS custom properties for global styles and component primitives.
- `tailwind.theme.json` mirrors those tokens for Tailwind configuration.
- `adapter.json` owns copy rules. Scenario templates declare the adapter they consume; they should not duplicate these copy rules.

Generated scenarios may extend local UI details, but root-level `DESIGN.md` remains the design contract.
