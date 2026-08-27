/**
 * @libraryId react-component-library:BaseStyles
 * @displayName Base Styles
 * @description Shared control reset, accessibility treatments, and sizing tokens.
 * @version 1.2.0
 * @tags ["styles","controls","accessibility"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @version 1.2.0 */
/** @vrooliComponentSource react-component-library:BaseStyles */
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";

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
  useLibraryStyleSheet("base-styles-1.2.0", baseStyles);
  return null;
}
