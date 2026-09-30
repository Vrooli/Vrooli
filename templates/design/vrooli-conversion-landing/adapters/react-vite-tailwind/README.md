# React Vite Tailwind Adapter

This adapter translates `templates/design/vrooli-conversion-landing/DESIGN.md` into assets consumed by `templates/scenarios/landing-page-react-vite`.

- `tokens.css` defines the canonical landing-page CSS custom properties for generated scenarios.
- `tailwind.theme.json` exposes those tokens through Tailwind extension keys.
- `.vrooli/styling.json` remains the runtime style-pack/config layer for variants and admin customization.
- Root-level `DESIGN.md` remains the design contract; runtime styling config should instantiate it, not replace it.
