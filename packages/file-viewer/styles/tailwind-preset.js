// Tailwind preset for @vrooli/file-viewer. It maps the viewer's `fv-*` utility
// names onto the `--fv-*` CSS variables (RGB channel triplets) that
// `styles.css` defines, so a host themes the viewer by setting variables, not
// by rebuilding classes. Add the package's dist to Tailwind `content`.

/** A color utility whose alpha is adjustable: `bg-fv-accent/10`. */
const channel = (name) => `rgb(var(--fv-${name}) / <alpha-value>)`;
/** A token whose alpha is already part of the variable (borders). */
const composed = (name) => `rgb(var(--fv-${name}))`;

/** @type {import("tailwindcss").Config} */
export default {
  theme: {
    extend: {
      colors: {
        fv: {
          "surface-base": channel("surface-base"),
          "surface-raised": channel("surface-raised"),
          "surface-input": channel("surface-input"),
          "text-primary": channel("text-primary"),
          "text-secondary": channel("text-secondary"),
          "text-muted": channel("text-muted"),
          "text-faint": channel("text-faint"),
          accent: channel("accent"),
          "accent-fg": channel("accent-fg"),
          "error-text": channel("error-text"),
          success: channel("success"),
        },
      },
      borderColor: {
        "fv-default": composed("border-default"),
        "fv-hover": composed("border-hover"),
        "fv-accent": composed("accent-border"),
      },
      zIndex: {
        // In-pane chrome that must sit above its siblings (a settings popover).
        "fv-chrome": "var(--fv-z-chrome)",
      },
    },
  },
};
