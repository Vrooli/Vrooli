import { type Config } from "tailwindcss";

/**
 * Design token mapping: CSS custom properties → Tailwind utility classes.
 *
 * Tokens are defined in src/styles.css as `--wc-*` variables.
 * Components use semantic class names (e.g. `bg-wc-surface-base`)
 * instead of raw Tailwind palette classes (e.g. `bg-slate-950`).
 */
export default {
  content: [
    "./index.html",
    "./src/**/*.{ts,tsx}",
    // BRIDGE (remove when the library emits no utility classes — see gate
    // catalog.utility-class). Some published RCL versions still ship Tailwind class strings; without
    // this glob their styling is purged from this app's bundle with no error.
    "./node_modules/@vrooli/react-component-library/dist/**/*.js"
  ],
  theme: {
    extend: {
      colors: {
        wc: {
          "surface-base": "rgb(var(--wc-surface-base) / <alpha-value>)",
          "surface-raised": "rgb(var(--wc-surface-raised) / <alpha-value>)",
          "surface-header": "rgb(var(--wc-surface-header) / <alpha-value>)",
          "surface-input": "rgb(var(--wc-surface-input) / <alpha-value>)",
          "text-primary": "rgb(var(--wc-text-primary) / <alpha-value>)",
          "text-secondary": "rgb(var(--wc-text-secondary) / <alpha-value>)",
          "text-muted": "rgb(var(--wc-text-muted) / <alpha-value>)",
          "text-faint": "rgb(var(--wc-text-faint) / <alpha-value>)",
          accent: "rgb(var(--wc-accent) / <alpha-value>)",
          "accent-fg": "rgb(var(--wc-accent-fg) / <alpha-value>)",
          "error-text": "rgb(var(--wc-error-text) / <alpha-value>)",
          "error-detail": "rgb(var(--wc-error-detail) / <alpha-value>)",
          success: "rgb(var(--wc-success) / <alpha-value>)",
          warning: "rgb(var(--wc-warning) / <alpha-value>)",
        },
        // Border/overlay tokens use pre-composed alpha, reference directly
      },
      borderColor: {
        "wc-default": "rgb(var(--wc-border-default))",
        "wc-hover": "rgb(var(--wc-border-hover))",
        "wc-accent": "rgb(var(--wc-accent-border))",
        "wc-error": "rgb(var(--wc-error-border))",
      },
      height: {
        /* h-wc-app: the viewport-aware app height.
           --wc-app-height is set by the useAppViewport hook to
           visualViewport.height (the real visible area, excluding
           browser chrome and virtual keyboard). Use this instead of
           h-screen (100vh) which overshoots on mobile.
           Fallback 100dvh handles pre-hook-initialization (loading/error
           states that render before the Workspace mounts the hook). */
        "wc-app": "var(--wc-app-height, 100dvh)",
      },
      backgroundColor: {
        "wc-backdrop": "rgb(var(--wc-backdrop))",
        "wc-backdrop-heavy": "rgb(var(--wc-backdrop-heavy))",
        "wc-error-surface": "rgb(var(--wc-error-surface))",
        "wc-accent-active": "rgb(var(--wc-accent-active))",
      },
      /* Overlay z-layer scale — the SSOT for every overlay z value in
         src/components. Components must use these tokens (z-wc-*) and never
         arbitrary z-[..] or raw z-40/z-50 utilities. Tiers, bottom → top:

           chrome          in-pane chrome (sticky headers, resize handles,
                           upload overlays, scroll-to-bottom buttons)
           chrome-raised   chrome that must sit above sibling chrome
           popover         anchored quick-pick popovers + their backdrops
           drawer          DrawerShell + full-screen drawer surfaces
           menu            context menus (ContextMenuBase) + their backdrops.
                           Above drawer: menus spawn from direct presses on
                           whatever surface is topmost (e.g. long-pressing a
                           session inside the mobile sidebar drawer)
           confirm         ConfirmDialog (must layer over drawers and menus)
           toolbar         FloatingToolbar (draggable, above all surfaces)
           tooltip         transient tooltips (always visible)

         Backdrop tokens sit one step below their surface so the surface
         paints above its own backdrop regardless of DOM order. */
		// Every TerminalPane carries `isolate`, so these chrome values are
		// scoped to their pane and cannot paint over a sibling view such as
		// MessagesPane.
      zIndex: {
        "wc-chrome": "var(--layer-sticky)",
        "wc-chrome-raised": "var(--layer-dropdown)",
        "wc-popover-backdrop": "var(--layer-overlay)",
        "wc-popover": "var(--layer-overlay)",
        "wc-drawer": "var(--layer-modal)",
        "wc-menu-backdrop": "var(--layer-overlay)",
        "wc-menu": "var(--layer-menu)",
        "wc-confirm": "var(--layer-alert)",
        "wc-toolbar": "var(--layer-menu)",
        "wc-tooltip": "var(--layer-tooltip)",
      },
    },
  },
  plugins: [],
} satisfies Config;
