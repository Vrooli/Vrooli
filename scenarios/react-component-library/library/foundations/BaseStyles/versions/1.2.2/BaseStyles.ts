/**
 * @libraryId react-component-library:BaseStyles
 * @displayName Base Styles
 * @description Shared control reset, accessibility treatments, and sizing tokens.
 * @version 1.2.2
 * @tags ["styles","controls","accessibility"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @version 1.2.0 */
/** @vrooliComponentSource react-component-library:BaseStyles */
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1";

/**
 * The published control reset, focus-visible ring, reduced-motion policy,
 * forced-colors policy, control-size ramp, direct-child icon scale, the
 * visually-hidden utility, and the host viewport contract.
 *
 * The viewport contract is the seam a host uses to tell the library what the
 * usable viewport actually is. A browser's `env(safe-area-inset-*)` and
 * `100dvh` describe the *layout* viewport, which an application that manages
 * its own scrolling and keyboard handling has usually already narrowed. When
 * the two disagree, an overlay that trusts the raw environment lands on
 * neither edge. The defaults below are the raw environment; a host that knows
 * better assigns these six properties on the document element and every
 * library surface follows it.
 */
export const baseStyles = `
/* rcl:canonical-tokens:begin */
@layer rcl.tokens {
  :root {
    /* @tier Expression */
    --app-background: var(--color-background);
    /* @tier Expression */
    --app-foreground: var(--color-foreground);
    /* @tier Expression */
    --app-muted-foreground: var(--color-muted-foreground);
    /* @tier Expression */
    --app-shell: var(--color-shell);
    /* @tier Expression */
    --app-surface: var(--color-surface);
    /* @tier Expression */
    --app-surface-muted: var(--color-surface-muted);
    /* @tier Expression */
    --app-surface-raised: var(--color-surface-raised);
    /* @tier Expression */
    --badge-border: var(--color-border);
    /* @tier Contract */
    --border-focus: 2px;
    /* @tier Rhythm */
    --border-hairline: 1px;
    /* @tier Rhythm */
    --border-medium: 2px;
    /* @tier Rhythm */
    --border-strong: 2px;
    /* @tier Rhythm */
    --border-thin: 1px;
    /* @tier Expression */
    --color-accent: #0891b2;
    /* @tier Expression */
    --color-accent-subtle: color-mix(in srgb, var(--color-accent) 14%, transparent);
    /* @tier Expression */
    --color-background: #f8fafc;
    /* @tier Expression */
    --color-border: #cbd5e1;
    /* @tier Expression */
    --color-border-strong: color-mix(in srgb, var(--color-border) 72%, var(--color-foreground));
    /* @tier Expression */
    --color-danger: #dc2626;
    /* @tier Expression */
    --color-danger-border: color-mix(in srgb, var(--color-danger) 38%, var(--color-border));
    /* @tier Expression */
    --color-danger-foreground: color-mix(in srgb, var(--color-danger) 78%, var(--color-foreground));
    /* @tier Expression */
    --color-danger-foreground-inverse: var(--color-primary-foreground);
    /* @tier Expression */
    --color-danger-subtle: color-mix(in srgb, var(--color-danger) 12%, var(--color-surface));
    /* @tier Expression */
    --color-field: var(--color-surface);
    /* @tier Contract */
    --color-focus: #2563eb;
    /* @tier Contract */
    --color-focus-ring: var(--color-focus);
    /* @tier Expression */
    --color-foreground: #0f172a;
    /* @tier Expression */
    --color-info: #0284c7;
    /* @tier Expression */
    --color-muted-foreground: #64748b;
    /* @tier Expression */
    --color-on-primary: var(--color-primary-foreground);
    /* @tier Expression */
    --color-overlay: var(--color-shell);
    /* @tier Expression */
    --color-primary: #2563eb;
    /* @tier Expression */
    --color-primary-foreground: #ffffff;
    /* @tier Expression */
    --color-primary-hover: color-mix(in srgb, var(--color-primary) 88%, var(--color-foreground));
    /* @tier Expression */
    --color-primary-strong: var(--color-primary);
    /* @tier Expression */
    --color-scrim: color-mix(in srgb, var(--color-shell) 52%, transparent);
    /* @tier Expression */
    --color-shell: #020617;
    /* @tier Expression */
    --color-success: #16a34a;
    /* @tier Expression */
    --color-success-foreground: color-mix(in srgb, var(--color-success) 76%, var(--color-foreground));
    /* @tier Expression */
    --color-surface: #ffffff;
    /* @tier Expression */
    --color-surface-muted: #f1f5f9;
    /* @tier Expression */
    --color-surface-raised: #ffffff;
    /* @tier Expression */
    --color-surface-sunken: color-mix(in srgb, var(--color-surface-muted) 72%, var(--color-background));
    /* @tier Expression */
    --color-warning: #d97706;
    /* @tier Expression */
    --color-warning-foreground: color-mix(in srgb, var(--color-warning) 72%, var(--color-foreground));
    /* @tier Expression */
    --color-warning-subtle: color-mix(in srgb, var(--color-warning) 16%, var(--color-surface));
    /* @tier Rhythm */
    --content-min-height: 12rem;
    /* @tier Expression */
    --control-border: 1px solid var(--color-border);
    /* @tier Rhythm */
    --control-height: 2.75rem;
    /* @tier Rhythm */
    --control-height-lg: 3.25rem;
    /* @tier Rhythm */
    --control-height-sm: 2.25rem;
    /* @tier Rhythm */
    --control-padding: var(--space-sm);
    /* @tier Expression */
    --control-radius: var(--radius-control);
    /* @tier Expression */
    --dur-deliberate: 400ms;
    /* @tier Expression */
    --dur-enter: var(--dur-quick);
    /* @tier Expression */
    --dur-fast: var(--dur-instant);
    /* @tier Expression */
    --dur-instant: 120ms;
    /* @tier Expression */
    --dur-moderate: 280ms;
    /* @tier Expression */
    --dur-normal: var(--dur-moderate);
    /* @tier Expression */
    --dur-quick: 180ms;
    /* @tier Expression */
    --dur-slow: var(--dur-deliberate);
    /* @tier Expression */
    --ease-enter: cubic-bezier(0, 0, 0, 1);
    /* @tier Expression */
    --ease-exit: cubic-bezier(.3, 0, 1, 1);
    /* @tier Expression */
    --ease-standard: cubic-bezier(.2, 0, 0, 1);
    /* @tier Expression */
    --elev-flat: none;
    /* @tier Expression */
    --elev-floating: 0 4px 12px rgba(9, 18, 22, 0.12);
    /* @tier Expression */
    --elev-modal: 0 4px 12px rgba(9, 18, 22, .10), 0 16px 48px rgba(9, 18, 22, .18);
    /* @tier Expression */
    --elev-overlay: 0 2px 4px rgba(9, 18, 22, .06), 0 4px 12px rgba(9, 18, 22, .10);
    /* @tier Expression */
    --elev-raised: 0 1px 2px rgba(9, 18, 22, .06), 0 1px 3px rgba(9, 18, 22, .10);
    /* @tier Expression */
    --elev-subtle: 0 1px 2px rgba(9, 18, 22, .06);
    /* @tier Contract */
    --focus-ring: 0 0 0 3px color-mix(in srgb, var(--color-focus) 35%, transparent);
    /* @tier Contract */
    --focus-ring-color: var(--color-focus);
    /* @tier Contract */
    --focus-ring-offset: 2px;
    /* @tier Contract */
    --focus-ring-width: 2px;
    /* @tier Expression */
    --font-mono: "JetBrains Mono", "Fira Code", "SF Mono", Consolas, "Liberation Mono", Menlo, monospace;
    /* @tier Expression */
    --font-sans: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    /* @tier Expression */
    --font-size-lg: 18px;
    /* @tier Expression */
    --font-size-sm: 14px;
    /* @tier Rhythm */
    --icon-size-lg: 24px;
    /* @tier Rhythm */
    --icon-size-md: 20px;
    /* @tier Rhythm */
    --icon-size-sm: 16px;
    /* @tier Rhythm */
    --icon-size-xs: 12px;
    /* @tier Contract */
    --layer-alert: 700;
    /* @tier Contract */
    --layer-base: 0;
    /* @tier Contract */
    --layer-dropdown: 200;
    /* @tier Contract */
    --layer-menu: 610;
    /* @tier Contract */
    --layer-modal: 400;
    /* @tier Contract */
    --layer-overlay: 300;
    /* @tier Contract */
    --layer-popover: 200;
    /* @tier Contract */
    --layer-raised: 150;
    /* @tier Contract */
    --layer-sticky: 100;
    /* @tier Contract */
    --layer-toast: 500;
    /* @tier Contract */
    --layer-tooltip: 600;
    /* @tier Expression */
    --motion-fast: var(--dur-quick);
    /* @tier Expression */
    --motion-normal: var(--dur-moderate);
    /* @tier Contract */
    --opacity-disabled: .40;
    /* @tier Expression */
    --opacity-muted: .64;
    /* @tier Expression */
    --opacity-scrim: .72;
    /* @tier Rhythm */
    --overlay-dialog-lg: 48rem;
    /* @tier Rhythm */
    --overlay-dialog-md: 36rem;
    /* @tier Rhythm */
    --overlay-dialog-sm: 24rem;
    /* @tier Rhythm */
    --overlay-drawer-top-gap: 32px;
    /* @tier Rhythm */
    --overlay-grabber-block: 4px;
    /* @tier Rhythm */
    --overlay-grabber-inline: 36px;
    /* @tier Rhythm */
    --overlay-menu-align: 0px;
    /* @tier Rhythm */
    --panel-padding: var(--space-md);
    /* @tier Expression */
    --panel-radius: var(--radius-panel);
    /* @tier Expression */
    --radius-control: 0.375rem;
    /* @tier Expression */
    --radius-overlay: 1rem;
    /* @tier Expression */
    --radius-panel: 0.5rem;
    /* @tier Expression */
    --radius-pill: 9999px;
    /* @tier Expression */
    --radius-sheet: 1rem;
    /* @tier Expression */
    --scrollbar-thumb: #94a3b8;
    /* @tier Expression */
    --scrollbar-thumb-hover: #64748b;
    /* @tier Rhythm */
    --sidebar-max-width: 30rem;
    /* @tier Rhythm */
    --sidebar-min-width: 16.25rem;
    /* @tier Rhythm */
    --sidebar-width: 20rem;
    /* @tier Rhythm */
    --space-2xl: 48px;
    /* @tier Rhythm */
    --space-2xs: 8px;
    /* @tier Rhythm */
    --space-3xs: 4px;
    /* @tier Rhythm */
    --space-4xl: 80px;
    /* @tier Rhythm */
    --space-4xs: 4px;
    /* @tier Rhythm */
    --space-lg: 32px;
    /* @tier Rhythm */
    --space-md: 24px;
    /* @tier Rhythm */
    --space-sm: 16px;
    /* @tier Rhythm */
    --space-xl: 40px;
    /* @tier Rhythm */
    --space-xs: 12px;
    /* @tier Expression */
    --spring-expressive: cubic-bezier(.16, 1.2, .3, 1.05);
    /* @tier Expression */
    --spring-subtle: cubic-bezier(.2, .8, .2, 1.05);
    /* @tier Contract */
    --tap-target-min: 44px;
    /* @tier Expression */
    --text-body: 400 var(--text-body-size) / var(--text-body-line) var(--font-sans);
    /* @tier Expression */
    --text-body-line: 22px;
    /* @tier Expression */
    --text-body-size: 14px;
    /* @tier Expression */
    --text-body-sm: 400 var(--text-body-sm-size) / var(--text-body-sm-line) var(--font-sans);
    /* @tier Expression */
    --text-body-sm-line: 20px;
    /* @tier Expression */
    --text-body-sm-size: 13px;
    /* @tier Expression */
    --text-caption: 600 var(--text-caption-size) / var(--text-caption-line) var(--font-sans);
    /* @tier Expression */
    --text-caption-line: 16px;
    /* @tier Expression */
    --text-caption-size: 11px;
    /* @tier Expression */
    --text-code: 500 0.8125rem/1.25rem var(--font-mono);
    /* @tier Expression */
    --text-display: 700 var(--text-display-size) / var(--text-display-line) var(--font-sans);
    /* @tier Expression */
    --text-display-line: 38px;
    /* @tier Expression */
    --text-display-size: 32px;
    /* @tier Expression */
    --text-heading: 600 var(--text-heading-size) / var(--text-heading-line) var(--font-sans);
    /* @tier Expression */
    --text-heading-lg: 600 20px / 26px var(--font-sans);
    /* @tier Expression */
    --text-heading-line: 24px;
    /* @tier Expression */
    --text-heading-size: 18px;
    /* @tier Expression */
    --text-heading-sm: 600 16px / 22px var(--font-sans);
    /* @tier Expression */
    --text-label: 500 var(--text-label-size) / var(--text-label-line) var(--font-sans);
    /* @tier Expression */
    --text-label-line: 16px;
    /* @tier Expression */
    --text-label-size: 12px;
    /* @tier Expression */
    --text-label-tracking: 0.005em;
    /* @tier Expression */
    --text-overline: 700 var(--text-caption-size) / var(--text-caption-line) var(--font-sans);
    /* @tier Expression */
    --text-subheading-line: 20px;
    /* @tier Expression */
    --text-subheading-size: 15px;
    /* @tier Expression */
    --text-subtitle: 600 var(--text-subheading-size) / var(--text-subheading-line) var(--font-sans);
    /* @tier Expression */
    --text-subtitle-tracking: 0;
    /* @tier Expression */
    --text-title: 700 var(--text-title-size) / var(--text-title-line) var(--font-sans);
    /* @tier Expression */
    --text-title-line: 30px;
    /* @tier Expression */
    --text-title-size: 24px;
    /* @tier Expression */
    --text-title-tracking: -.01em;
    /* @tier Expression */
    --text-xs: 12px;
    /* @tier Contract */
    --touch-target: 44px;
    /* @tier Expression */
    --tracking-caps: .08em;
    /* @tier Expression */
    --tracking-tight: -.02em;
    /* @tier Expression */
    --weight-medium: 500;
  }
}
/* rcl:canonical-tokens:end */
:root { --control-size-xs: 32px; --control-size-sm: 36px; --control-size-md: 40px; --control-size-lg: 44px; --control-size-xl: 48px; --control-size-icon: 40px; --control-icon-size-xs: 12px; --control-icon-size-sm: 14px; --control-icon-size-md: 16px; --control-icon-size-lg: 18px; --control-icon-size-xl: 20px; --control-icon-size-icon: 16px; }
:root { --rcl-safe-top: env(safe-area-inset-top, 0px); --rcl-safe-right: env(safe-area-inset-right, 0px); --rcl-safe-bottom: env(safe-area-inset-bottom, 0px); --rcl-safe-left: env(safe-area-inset-left, 0px); --rcl-keyboard-inset: 0px; --rcl-viewport-height: 100dvh; }
[data-rcl-control] { appearance: none; box-sizing: border-box; border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-control); font: inherit; }
:where([data-rcl-control], [data-rcl-asset-detail], [data-rcl-asset-detail] *, [data-rcl-bottom-nav], [data-rcl-bottom-nav] *, [data-rcl-bottom-sheet], [data-rcl-bottom-sheet] *, [data-rcl-card], [data-rcl-card] *, [data-rcl-command-center], [data-rcl-command-center] *, [data-rcl-dialog], [data-rcl-dialog] *, [data-rcl-drawer-shell], [data-rcl-drawer-shell] *, [data-rcl-full-page-drawer], [data-rcl-full-page-drawer] *, [data-rcl-inspector-layout], [data-rcl-inspector-layout] *, [data-rcl-overlay-grabber], [data-rcl-resize-handle], [data-rcl-resize-handle] *, [data-rcl-responsive-dialog], [data-rcl-responsive-dialog] *, [data-rcl-responsive-panel], [data-rcl-responsive-panel] *, [data-rcl-sidebar-shell], [data-rcl-sidebar-shell] *, [data-rcl-sidebar-backdrop], [data-rcl-tabs], [data-rcl-tabs] *, [data-rcl-workspace-header], [data-rcl-workspace-header] *, [data-rcl-code-block], [data-rcl-code-block] *, [data-rcl-inline], [data-rcl-inline] *, [data-rcl-markdown], [data-rcl-markdown] *, [data-rcl-mermaid], [data-rcl-mermaid] *):focus-visible { outline: 3px solid color-mix(in srgb, var(--color-focus) 38%, transparent); outline-offset: 2px; }
[data-rcl-control] > svg, [data-rcl-control] > [data-rcl-icon] { inline-size: var(--control-icon-size); block-size: var(--control-icon-size); }
[data-rcl-control][data-control-size="xs"] { --control-icon-size: var(--control-icon-size-xs); } [data-rcl-control][data-control-size="sm"] { --control-icon-size: var(--control-icon-size-sm); } [data-rcl-control][data-control-size="md"], [data-rcl-control][data-control-size="default"] { --control-icon-size: var(--control-icon-size-md); } [data-rcl-control][data-control-size="lg"] { --control-icon-size: var(--control-icon-size-lg); } [data-rcl-control][data-control-size="xl"] { --control-icon-size: var(--control-icon-size-xl); } [data-rcl-control][data-control-size="icon"] { --control-icon-size: var(--control-icon-size-icon); }
[data-rcl-visually-hidden], .rcl-visually-hidden { position: absolute !important; inline-size: 1px !important; block-size: 1px !important; padding: 0 !important; margin: -1px !important; overflow: hidden !important; clip-path: inset(50%) !important; white-space: nowrap !important; border: 0 !important; }
@media (prefers-reduced-motion: reduce) { *, *::before, *::after { animation-duration: .01ms !important; animation-iteration-count: 1 !important; scroll-behavior: auto !important; transition-duration: .01ms !important; } }
@media (forced-colors: active) { :where([data-rcl-control], [data-rcl-asset-detail], [data-rcl-asset-detail] *, [data-rcl-bottom-nav], [data-rcl-bottom-nav] *, [data-rcl-bottom-sheet], [data-rcl-bottom-sheet] *, [data-rcl-card], [data-rcl-card] *, [data-rcl-command-center], [data-rcl-command-center] *, [data-rcl-dialog], [data-rcl-dialog] *, [data-rcl-drawer-shell], [data-rcl-drawer-shell] *, [data-rcl-full-page-drawer], [data-rcl-full-page-drawer] *, [data-rcl-inspector-layout], [data-rcl-inspector-layout] *, [data-rcl-overlay-grabber], [data-rcl-resize-handle], [data-rcl-resize-handle] *, [data-rcl-responsive-dialog], [data-rcl-responsive-dialog] *, [data-rcl-responsive-panel], [data-rcl-responsive-panel] *, [data-rcl-sidebar-shell], [data-rcl-sidebar-shell] *, [data-rcl-sidebar-backdrop], [data-rcl-tabs], [data-rcl-tabs] *, [data-rcl-workspace-header], [data-rcl-workspace-header] *, [data-rcl-code-block], [data-rcl-code-block] *, [data-rcl-inline], [data-rcl-inline] *, [data-rcl-markdown], [data-rcl-markdown] *, [data-rcl-mermaid], [data-rcl-mermaid] *) { border-color: CanvasText; color: CanvasText; } [data-rcl-control] { background: ButtonFace; color: ButtonText; } }
`;

/**
 * Mounts {@link baseStyles} once for the whole page and renders no DOM node.
 * The hook owns a single head node per key, so repeated mounts are free.
 */
export function BaseStyles() {
  useLibraryStyleSheet("base-styles-1.2.1", baseStyles);
  return null;
}
